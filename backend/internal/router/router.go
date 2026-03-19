package router

import (
	"strings"
	"time"

	"blendpos/internal/config"
	"blendpos/internal/handler"
	"blendpos/internal/infra"
	"blendpos/internal/middleware"
	"blendpos/internal/repository"
	"blendpos/internal/service"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Deps bundles every dependency the router needs.
// All repositories, services and infrastructure are created in main.go
// (the sole composition root) and injected here.
type Deps struct {
	Cfg    *config.Config
	DB     *gorm.DB
	// DBRead is the read replica DB for analytics queries (F1-9).
	// Falls back to DB (primary) when no replica is configured.
	DBRead *gorm.DB
	RDB    *redis.Client
	AfipCB *infra.CircuitBreaker

	// Services
	AuthSvc         service.AuthService
	ProductoSvc     service.ProductoService
	InventarioSvc   service.InventarioService
	VentaSvc        service.VentaService
	CajaSvc         service.CajaService
	FacturacionSvc  service.FacturacionService
	ConfigFiscalSvc service.ConfiguracionFiscalService
	ProveedorSvc    service.ProveedorService
	CategoriaSvc    service.CategoriaService
	AuditSvc        service.AuditService
	CompraSvc       service.CompraService
	PromocionSvc    service.PromocionService
	TenantSvc       service.TenantService
	BillingSvc      service.BillingService
	ReportesSvc     service.ReportesService
	LoteSvc         service.LoteService
	ClienteSvc      service.ClienteService
	SucursalSvc     service.SucursalService

	// Repos still needed by handlers that bypass the service layer
	ProductoRepo        repository.ProductoRepository
	HistorialPrecioRepo repository.HistorialPrecioRepository
	AuditRepo           repository.AuditRepository
	ComprobanteRepo     repository.ComprobanteRepository
	VentaRepo           repository.VentaRepository
	TenantRepo          repository.TenantRepository

	// Worker dispatcher for email/facturacion jobs
	Dispatcher interface{}
}

// New wires handlers and registers routes. It does NOT create infrastructure,
// repositories or services — that is the responsibility of main.go (S-05).
func New(d Deps) *gin.Engine {
	cfg := d.Cfg

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Global middleware chain (order matters)
	r.Use(middleware.MaxBodySize(10 << 20))   // 10 MB default body limit (S-07)
	r.Use(gzip.Gzip(gzip.DefaultCompression)) // 7.5 — compress JSON responses (saves ~70% on product lists)
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	origins := strings.Split(cfg.AllowedOrigins, ",")
	r.Use(middleware.CORS(origins))
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.GlobalTimeout(30 * time.Second))
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.RateLimiter(d.RDB, 1000, time.Minute)) // 1000 req/min per IP (Redis-backed)

	// ── Handlers ─────────────────────────────────────────────────────────────
	authH := handler.NewAuthHandler(d.AuthSvc)
	tenantsH := handler.NewTenantsHandler(d.TenantSvc)
	productosH := handler.NewProductosHandler(d.ProductoSvc)
	inventarioH := handler.NewInventarioHandler(d.InventarioSvc)
	ventasH := handler.NewVentasHandler(d.VentaSvc)
	cajaH := handler.NewCajaHandler(d.CajaSvc)
	facturacionH := handler.NewFacturacionHandler(d.FacturacionSvc, cfg.PDFStoragePath, d.ComprobanteRepo, d.VentaRepo, d.ConfigFiscalSvc)
	facturacionH.SetDispatcher(d.Dispatcher) // Inyectar dispatcher para envío de emails
	proveedoresH := handler.NewProveedoresHandler(d.ProveedorSvc)
	usuariosH := handler.NewUsuariosHandler(d.AuthSvc)
	consultaH := handler.NewConsultaPreciosHandler(d.ProductoRepo, d.RDB)
	historialPreciosH := handler.NewHistorialPreciosHandler(d.HistorialPrecioRepo)
	categoriasH := handler.NewCategoriasHandler(d.CategoriaSvc)
	auditH := handler.NewAuditHandler(d.AuditRepo)
	configFiscalH := handler.NewConfiguracionFiscalHandler(d.ConfigFiscalSvc)
	comprasH := handler.NewCompraHandler(d.CompraSvc)
	promocionesH := handler.NewPromocionHandler(d.PromocionSvc)
	ventaReporteH := handler.NewVentaReporteHandler(d.DBRead)
	billingH := handler.NewBillingHandler(d.BillingSvc)
	reportesH := handler.NewReportesHandler(d.ReportesSvc)
	vencimientosH := handler.NewVencimientosHandler(d.LoteSvc)
	clientesH := handler.NewClientesHandler(d.ClienteSvc)
	sucursalesH := handler.NewSucursalesHandler(d.SucursalSvc)

	// ── Routes ───────────────────────────────────────────────────────────────

	// Public
	r.GET("/health", handler.Health(d.DB, d.RDB, d.AfipCB, cfg))

	// Public tenant registration + plan listing + presets
	public := r.Group("/v1/public")
	{
		public.POST("/register", tenantsH.Register)
		public.GET("/planes", tenantsH.ListarPlanes)
		public.GET("/presets", tenantsH.ListarPresets)
		public.GET("/presets/:tipo", tenantsH.ObtenerPreset)
	}

	// Billing webhook — public, no JWT (MercadoPago calls this)
	r.POST("/v1/billing/webhook", billingH.Webhook)

	// Auth (public)
	auth := r.Group("/v1/auth")
	{
		auth.POST("/login", middleware.LoginRateLimiter(d.RDB), authH.Login)
		auth.POST("/refresh", middleware.RefreshRateLimiter(d.RDB), authH.Refresh)
	}

	// Price check — no auth required (RF-27)
	// Dedicated rate limit: 60 req/min per IP to prevent catalog scraping.
	r.GET("/v1/precio/:barcode", middleware.RateLimiter(d.RDB, 60, time.Minute), consultaH.GetPrecioPorBarcode)

	// Protected routes
	jwtMW := middleware.JWTAuth(cfg.JWTSecret, d.RDB)

	// Authenticated logout — requires a valid (non-revoked) token
	auth.POST("/logout", jwtMW, authH.Logout)
	auth.POST("/change-password", jwtMW, authH.ChangePassword)

	// IDOR guard helpers — validate resource ownership before reaching handler (F2-4).
	// Not applied to: superadmin routes (cross-tenant), public routes, list endpoints.
	idor := func(table, param string) gin.HandlerFunc {
		return middleware.ValidateResourceOwnership(d.DB, table, param)
	}

	v1 := r.Group("/v1", jwtMW, middleware.TenantMiddleware(d.DB))
	v1.Use(middleware.RateLimitPerTenant(d.TenantRepo, d.RDB))
	v1.Use(middleware.TenantAuditMiddleware())
	v1.Use(middleware.AuditMiddleware(d.AuditSvc))
	{
		// Roles: cajero, supervisor, administrador — declared per-endpoint
		v1.POST("/ventas", middleware.RequireRole("cajero", "supervisor", "administrador"), ventasH.RegistrarVenta)
		v1.GET("/ventas", middleware.RequireRole("cajero", "supervisor", "administrador"), ventasH.ListarVentas)
		v1.DELETE("/ventas/:id", middleware.RequireRole("supervisor", "administrador"), idor("ventas", "id"), ventasH.AnularVenta)

		// GET /v1/productos — cajero/supervisor/administrador can read (catalog sync)
		v1.GET("/productos", middleware.RequireRole("cajero", "supervisor", "administrador"), productosH.Listar)
		v1.GET("/productos/:id", middleware.RequireRole("cajero", "supervisor", "administrador"), idor("productos", "id"), productosH.ObtenerPorID)
		v1.GET("/productos/:id/historial-precios", middleware.RequireRole("cajero", "supervisor", "administrador"), idor("productos", "id"), historialPreciosH.ListarPorProducto)
		// PATCH stock — supervisor or administrador
		v1.PATCH("/productos/:id/stock", middleware.RequireRole("supervisor", "administrador"), idor("productos", "id"), productosH.AjustarStock)
		// Write operations — administrador only (plan limit applied to POST)
		prods := v1.Group("/productos", middleware.RequireRole("administrador"))
		{
			prods.POST("", middleware.EnforcePlanLimitProductos(d.TenantRepo, d.RDB), productosH.Crear)
			prods.POST("/bulk", middleware.EnforcePlanLimitProductos(d.TenantRepo, d.RDB), productosH.CrearBulk)
			prods.PUT("/:id", idor("productos", "id"), productosH.Actualizar)
			prods.DELETE("/:id", idor("productos", "id"), productosH.Desactivar)
			prods.PATCH("/:id/reactivar", idor("productos", "id"), productosH.Reactivar)
		}

		inv := v1.Group("/inventario", middleware.RequireRole("administrador", "supervisor"))
		{
			inv.POST("/vinculos", inventarioH.CrearVinculo)
			inv.GET("/vinculos", inventarioH.ListarVinculos)
			inv.POST("/desarme", inventarioH.DesarmeManual)
			inv.GET("/alertas", inventarioH.ObtenerAlertas)
			inv.GET("/movimientos", inventarioH.ListarMovimientos)
		}

		// Lotes de producto y alertas de vencimiento
		lotes := v1.Group("/lotes", middleware.RequireRole("administrador", "supervisor"))
		{
			lotes.POST("", vencimientosH.CrearLote)
			lotes.GET("", vencimientosH.ListarLotes)
			lotes.DELETE("/:id", idor("lotes_producto", "id"), vencimientosH.EliminarLote)
		}
		v1.GET("/vencimientos/alertas", middleware.RequireRole("administrador", "supervisor"), vencimientosH.ObtenerAlertasVencimiento)

		caja := v1.Group("/caja")
		{
			caja.POST("/abrir", middleware.RequireRole("cajero", "supervisor", "administrador"), middleware.EnforcePlanLimitTerminales(d.TenantRepo, d.RDB), cajaH.Abrir)
			caja.POST("/arqueo", middleware.RequireRole("cajero", "supervisor", "administrador"), cajaH.Arqueo)
			caja.GET("/:id/reporte", middleware.RequireRole("cajero", "supervisor", "administrador"), idor("sesiones_caja", "id"), cajaH.ObtenerReporte)
			caja.POST("/movimiento", middleware.RequireRole("cajero", "supervisor", "administrador"), cajaH.RegistrarMovimiento)
			caja.GET("/activa", middleware.RequireRole("cajero", "supervisor", "administrador"), cajaH.GetActiva)
			caja.GET("/historial", middleware.RequireRole("supervisor", "administrador"), cajaH.Historial)
		}

		// Read-only: cajero can check their own comprobante status and download it
		factR := v1.Group("/facturacion", middleware.RequireRole("cajero", "supervisor", "administrador"))
		{
			factR.GET("/:venta_id", idor("ventas", "venta_id"), facturacionH.ObtenerComprobante)
			factR.GET("/pdf/:id", idor("comprobantes", "id"), facturacionH.DescargarPDF)
			factR.GET("/html/:id", idor("comprobantes", "id"), facturacionH.ObtenerHTML)
			factR.POST("/:id/enviar-email", idor("comprobantes", "id"), facturacionH.EnviarEmailComprobante)
		}
		// Write operations: admin/supervisor only
		factW := v1.Group("/facturacion", middleware.RequireRole("administrador", "supervisor"))
		{
			factW.DELETE("/:id", idor("comprobantes", "id"), facturacionH.AnularComprobante)
			factW.POST("/:id/reintentar", idor("comprobantes", "id"), facturacionH.ReintentarComprobante)
			factW.POST("/:id/regen-pdf", idor("comprobantes", "id"), facturacionH.RegenerarPDF)
			factW.POST("/cancelar-pendientes", middleware.RequireRole("administrador"), facturacionH.CancelarPendientes)
		}

		prov := v1.Group("/proveedores", middleware.RequireRole("administrador"))
		{
			prov.POST("", proveedoresH.Crear)
			prov.GET("", proveedoresH.Listar)
			prov.GET("/:id", idor("proveedores", "id"), proveedoresH.ObtenerPorID)
			prov.PUT("/:id", idor("proveedores", "id"), proveedoresH.Actualizar)
			prov.DELETE("/:id", idor("proveedores", "id"), proveedoresH.Eliminar)
			prov.POST("/:id/precios/masivo", idor("proveedores", "id"), proveedoresH.ActualizarPreciosMasivo)
		}

		v1.POST("/csv/import", middleware.RequireRole("administrador"), proveedoresH.ImportarCSV)

		usuarios := v1.Group("/usuarios", middleware.RequireRole("administrador"))
		{
			usuarios.POST("", usuariosH.Crear)
			usuarios.GET("", usuariosH.Listar)
			usuarios.PUT("/:id", idor("usuarios", "id"), usuariosH.Actualizar)
			usuarios.DELETE("/:id", idor("usuarios", "id"), usuariosH.Desactivar)
			usuarios.PATCH("/:id/reactivar", idor("usuarios", "id"), usuariosH.Reactivar)
		}

		// Clientes / Fiado (cuenta corriente)
		// Deudores list must be registered BEFORE the /:id routes to avoid Gin treating "deudores" as an :id param.
		v1.GET("/clientes/deudores", middleware.RequireRole("cajero", "supervisor", "administrador"), clientesH.ListarDeudores)
		v1.GET("/clientes", middleware.RequireRole("cajero", "supervisor", "administrador"), clientesH.Listar)
		v1.GET("/clientes/:id", middleware.RequireRole("cajero", "supervisor", "administrador"), idor("clientes", "id"), clientesH.ObtenerPorID)
		v1.GET("/clientes/:id/movimientos", middleware.RequireRole("cajero", "supervisor", "administrador"), idor("clientes", "id"), clientesH.ListarMovimientos)
		clientesW := v1.Group("/clientes", middleware.RequireRole("supervisor", "administrador"))
		{
			clientesW.POST("", clientesH.Crear)
			clientesW.PUT("/:id", idor("clientes", "id"), clientesH.Actualizar)
			clientesW.POST("/:id/pago", idor("clientes", "id"), clientesH.RegistrarPago)
		}

		// Offline sync endpoint (PWA SyncEngine)
		v1.POST("/ventas/sync-batch", middleware.RequireRole("cajero", "supervisor", "administrador"), ventasH.SyncBatch)

		// Analytics reporte de ventas (read replica — F1-9)
		v1.GET("/ventas/reporte", middleware.RequireRole("administrador", "supervisor"), ventaReporteH.GetReporte)

		// Analytics — reportes agregados (T5.1+T5.2, read replica)
		// resumen + medios-pago: available to all plans (basic dashboard)
		// top-productos + ventas-periodo: require analytics_avanzados feature flag
		reportes := v1.Group("/reportes", middleware.RequireRole("administrador", "supervisor"))
		{
			reportes.GET("/resumen", reportesH.GetResumen)
			reportes.GET("/top-productos", middleware.RequireFeature("analytics_avanzados", d.TenantRepo, d.RDB), reportesH.GetTopProductos)
			reportes.GET("/medios-pago", reportesH.GetMediosPago)
			reportes.GET("/ventas-periodo", middleware.RequireFeature("analytics_avanzados", d.TenantRepo, d.RDB), reportesH.GetVentasPeriodo)
			reportes.GET("/cajeros", middleware.RequireFeature("analytics_avanzados", d.TenantRepo, d.RDB), reportesH.GetCajeros)
			reportes.GET("/turnos", middleware.RequireFeature("analytics_avanzados", d.TenantRepo, d.RDB), reportesH.GetTurnos)
		}

		// Categorías — administrador can write, all authenticated can read
		v1.GET("/categorias", middleware.RequireRole("cajero", "supervisor", "administrador"), categoriasH.Listar)
		categorias := v1.Group("/categorias", middleware.RequireRole("administrador"))
		{
			categorias.POST("", categoriasH.Crear)
			categorias.PUT("/:id", idor("categorias", "id"), categoriasH.Actualizar)
			categorias.DELETE("/:id", idor("categorias", "id"), categoriasH.Desactivar)
		}

		// Audit log — read-only, admin only (Q-03)
		v1.GET("/audit", middleware.RequireRole("administrador"), auditH.List)

		// Configuración fiscal — admin only (AFIP parameters)
		configFiscal := v1.Group("/configuracion/fiscal", middleware.RequireRole("administrador"))
		{
			configFiscal.GET("", configFiscalH.Obtener)
			configFiscal.PUT("", configFiscalH.Actualizar)
		}

		// Compras — administrador can write, supervisor can read
		v1.GET("/compras", middleware.RequireRole("supervisor", "administrador"), comprasH.Listar)
		v1.GET("/compras/:id", middleware.RequireRole("supervisor", "administrador"), idor("compras", "id"), comprasH.ObtenerPorID)
		compras := v1.Group("/compras", middleware.RequireRole("administrador"))
		{
			compras.POST("", comprasH.Crear)
			compras.PATCH(":id/estado", idor("compras", "id"), comprasH.ActualizarEstado)
		}

		// Promociones - lectura para todos los roles autenticados del POS;
		// escritura solo para administrador.
		v1.GET("/promociones", middleware.RequireRole("cajero", "supervisor", "administrador"), promocionesH.Listar)
		v1.GET("/promociones/:id", middleware.RequireRole("cajero", "supervisor", "administrador"), idor("promociones", "id"), promocionesH.ObtenerPorID)
		promos := v1.Group("/promociones", middleware.RequireRole("administrador"))
		{
			promos.POST("", promocionesH.Crear)
			promos.PUT(":id", idor("promociones", "id"), promocionesH.Actualizar)
			promos.DELETE(":id", idor("promociones", "id"), promocionesH.Eliminar)
		}

		// Sucursales — admin only, all roles can read
		v1.GET("/sucursales", middleware.RequireRole("cajero", "supervisor", "administrador"), sucursalesH.Listar)
		v1.GET("/sucursales/:id", middleware.RequireRole("cajero", "supervisor", "administrador"), idor("sucursales", "id"), sucursalesH.ObtenerPorID)
		sucursales := v1.Group("/sucursales", middleware.RequireRole("administrador"))
		{
			sucursales.POST("", sucursalesH.Crear)
			sucursales.PUT("/:id", idor("sucursales", "id"), sucursalesH.Actualizar)
			sucursales.DELETE("/:id", idor("sucursales", "id"), sucursalesH.Desactivar)
		}

		// Billing — subscription management (F1-5)
		billing := v1.Group("/billing", middleware.RequireRole("administrador"))
		{
			billing.POST("/subscribe", billingH.Subscribe)
			billing.GET("/status", billingH.GetStatus)
		}

		// Tenant self-service (F1-1)
		tenantMe := v1.Group("/tenant")
		{
			tenantMe.GET("/me", tenantsH.ObtenerTenantActual)
			tenantMe.PUT("/me", middleware.RequireRole("administrador"), tenantsH.ActualizarTenantActual)
			tenantMe.GET("/plan", tenantsH.ObtenerPlanActual)
		}

		// Superadmin routes (F1-10) — require superadmin role
		superadmin := v1.Group("/superadmin", middleware.RequireSuperAdmin())
		{
			superadmin.GET("/tenants", tenantsH.ListarTodos)
			superadmin.GET("/tenants/:id", tenantsH.ObtenerTenantDetalle)
			superadmin.PUT("/tenants/:id", tenantsH.ToggleActivo)
			superadmin.PUT("/tenants/:id/plan", tenantsH.CambiarPlan)
			superadmin.GET("/metrics", tenantsH.ObtenerMetricas)
		}
	}

	// Swagger UI — only enabled outside production
	if cfg.Env != "production" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	return r
}
