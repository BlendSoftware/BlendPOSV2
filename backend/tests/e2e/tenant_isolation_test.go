//go:build integration

package e2e

// tenant_isolation_test.go
// F2-3: Integration tests for tenant isolation end-to-end.
// Creates 2 tenants (A and B) with products, sales, and fiscal config,
// then verifies that NEVER can one tenant access data from the other.
//
// Run with: go test -tags integration ./tests/e2e/... -v -run TestTenantIsolation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"blendpos/internal/config"
	"blendpos/internal/infra"
	"blendpos/internal/middleware"
	"blendpos/internal/repository"
	"blendpos/internal/router"
	"blendpos/internal/service"
	"blendpos/internal/worker"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcPostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcRedis "github.com/testcontainers/testcontainers-go/modules/redis"
	"gorm.io/gorm"
	gormPostgres "gorm.io/driver/postgres"
)

// ── Multi-Tenant Test Environment ───────────────────────────────────────────

type tenantEnv struct {
	token     string // admin JWT for this tenant
	tenantID  string // tenant UUID
	userID    string // admin user UUID
	productID string // created product UUID
	cajaID    string // open caja session UUID
	ventaID   string // created venta UUID
}

type isolationTestEnv struct {
	server  *httptest.Server
	db      *gorm.DB
	cfg     *config.Config
	tenantA *tenantEnv
	tenantB *tenantEnv
}

const testJWTSecret = "test-secret-key-must-be-at-least-32-chars-long"

func setupIsolationEnv(t *testing.T) *isolationTestEnv {
	t.Helper()
	ctx := context.Background()

	// Start Postgres container
	pgC, err := tcPostgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		tcPostgres.WithDatabase("blendpos_isolation_test"),
		tcPostgres.WithUsername("blendpos"),
		tcPostgres.WithPassword("blendpos"),
		tcPostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pgC.Terminate(ctx) })

	pgURL, err := pgC.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Start Redis container
	rdC, err := tcRedis.RunContainer(ctx,
		testcontainers.WithImage("redis:7-alpine"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdC.Terminate(ctx) })

	rdURL, err := rdC.ConnectionString(ctx)
	require.NoError(t, err)

	cfg := &config.Config{
		Port:               8000,
		Env:                "test",
		JWTSecret:          testJWTSecret,
		JWTExpirationHours: 8,
		JWTRefreshHours:    24,
		DatabaseURL:        pgURL,
		RedisURL:           rdURL,
		AFIPSidecarURL:     "http://localhost:9999",
		WorkerPoolSize:     1,
		PDFStoragePath:     t.TempDir(),
	}

	// Open a raw GORM connection first to run SQL migrations BEFORE
	// infra.NewDatabase (which runs schema patches that expect tables to exist).
	rawDB, err := gorm.Open(gormPostgres.Open(pgURL), &gorm.Config{})
	require.NoError(t, err)

	runSQLMigrations(t, rawDB)

	// Fix: migration 000025 drops categorias_nombre_key but not idx_categorias_nombre
	// (the actual index created by 000005). Drop it here so per-tenant unique works.
	_ = rawDB.Exec("DROP INDEX IF EXISTS idx_categorias_nombre").Error

	// Fix: migration 000025 didn't update the partial unique index on sesion_cajas
	// to include tenant_id. Replace it with a tenant-scoped version.
	_ = rawDB.Exec("DROP INDEX IF EXISTS uq_caja_abierta_por_punto").Error
	_ = rawDB.Exec(`CREATE UNIQUE INDEX uq_caja_abierta_por_punto
		ON sesion_cajas (tenant_id, punto_de_venta)
		WHERE estado = 'abierta'`).Error

	// Close raw connection - NewDatabase will open its own with full config
	rawSqlDB, _ := rawDB.DB()
	_ = rawSqlDB.Close()

	db, err := infra.NewDatabase(cfg.DatabaseURL)
	require.NoError(t, err)

	rdb, err := infra.NewRedis(cfg.RedisURL)
	require.NoError(t, err)

	// Register tenant audit callback (F2-2)
	middleware.RegisterTenantAuditCallback(db)

	// Build all repos
	afipCB := infra.NewCircuitBreaker(infra.DefaultCBConfig())
	afipClient := infra.NewAFIPClient(cfg.AFIPSidecarURL, cfg.InternalAPIToken)
	mailer := infra.NewMailer(cfg)
	dispatcher := worker.NewDispatcher(rdb, afipClient, mailer)

	usuarioRepo := repository.NewUsuarioRepository(db)
	productoRepo := repository.NewProductoRepository(db)
	ventaRepo := repository.NewVentaRepository(db)
	cajaRepo := repository.NewCajaRepository(db)
	comprobanteRepo := repository.NewComprobanteRepository(db)
	proveedorRepo := repository.NewProveedorRepository(db)
	historialPrecioRepo := repository.NewHistorialPrecioRepository(db)
	movimientoStockRepo := repository.NewMovimientoStockRepository(db)
	categoriaRepo := repository.NewCategoriaRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	compraRepo := repository.NewCompraRepository(db)
	promocionRepo := repository.NewPromocionRepository(db)
	configFiscalRepo := repository.NewConfiguracionFiscalRepository(db)
	tenantRepo := repository.NewTenantRepository(db)
	subscriptionRepo := repository.NewSubscriptionRepository(db)

	// Build services
	authSvc := service.NewAuthService(usuarioRepo, nil, cfg, rdb)
	tenantSvc := service.NewTenantService(tenantRepo, usuarioRepo, cfg, rdb)
	productoSvc := service.NewProductoService(productoRepo, movimientoStockRepo, categoriaRepo, rdb, nil, nil)
	inventarioSvc := service.NewInventarioService(productoRepo, movimientoStockRepo, nil)
	cajaSvc := service.NewCajaService(cajaRepo, usuarioRepo)
	ventaSvc := service.NewVentaService(ventaRepo, inventarioSvc, cajaSvc, cajaRepo, productoRepo, dispatcher, comprobanteRepo, configFiscalRepo)
	facturacionSvc := service.NewFacturacionService(comprobanteRepo, dispatcher)
	proveedorSvc := service.NewProveedorService(proveedorRepo, productoRepo, categoriaRepo)
	categoriaSvc := service.NewCategoriaService(categoriaRepo)
	auditSvc := service.NewAuditService(auditRepo)
	compraSvc := service.NewCompraService(compraRepo)
	promocionSvc := service.NewPromocionService(promocionRepo)
	configFiscalSvc := service.NewConfiguracionFiscalService(configFiscalRepo, afipClient)
	billingSvc := service.NewBillingService(subscriptionRepo, tenantRepo, &noopMPClient{})

	dbRead := infra.NewDatabaseReadReplica(db, cfg.DatabaseReadReplicaURL)

	r := router.New(router.Deps{
		Cfg:                 cfg,
		DB:                  db,
		DBRead:              dbRead,
		RDB:                 rdb,
		AfipCB:              afipCB,
		AuthSvc:             authSvc,
		TenantSvc:           tenantSvc,
		BillingSvc:          billingSvc,
		ProductoSvc:         productoSvc,
		InventarioSvc:       inventarioSvc,
		VentaSvc:            ventaSvc,
		CajaSvc:             cajaSvc,
		FacturacionSvc:      facturacionSvc,
		ConfigFiscalSvc:     configFiscalSvc,
		ProveedorSvc:        proveedorSvc,
		CategoriaSvc:        categoriaSvc,
		AuditSvc:            auditSvc,
		CompraSvc:           compraSvc,
		PromocionSvc:        promocionSvc,
		ProductoRepo:        productoRepo,
		HistorialPrecioRepo: historialPrecioRepo,
		AuditRepo:           auditRepo,
		ComprobanteRepo:     comprobanteRepo,
		VentaRepo:           ventaRepo,
		TenantRepo:          tenantRepo,
		Dispatcher:          dispatcher,
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	env := &isolationTestEnv{
		server: srv,
		db:     db,
		cfg:    cfg,
	}

	// ── Create Tenant A ─────────────────────────────────────────────────
	env.tenantA = registerTenant(t, srv, "Kiosco Alpha", "kiosco-alpha", "admin-a", "password123456")

	// ── Create Tenant B ─────────────────────────────────────────────────
	env.tenantB = registerTenant(t, srv, "Kiosco Beta", "kiosco-beta", "admin-b", "password123456")

	// Create products for each tenant
	env.tenantA.productID = createProduct(t, srv, env.tenantA.token, "Gaseosa Alpha", "7890000000001", 50)
	env.tenantB.productID = createProduct(t, srv, env.tenantB.token, "Gaseosa Beta", "7890000000002", 50)

	// Open caja for each tenant
	env.tenantA.cajaID = openCaja(t, srv, env.tenantA.token)
	env.tenantB.cajaID = openCaja(t, srv, env.tenantB.token)

	// Create a sale for each tenant
	env.tenantA.ventaID = createSale(t, srv, env.tenantA.token, env.tenantA.cajaID, env.tenantA.productID)
	env.tenantB.ventaID = createSale(t, srv, env.tenantB.token, env.tenantB.cajaID, env.tenantB.productID)

	// Create fiscal config for each tenant
	createFiscalConfig(t, db, env.tenantA.tenantID, "20111111111", "Kiosco Alpha SRL")
	createFiscalConfig(t, db, env.tenantB.tenantID, "20222222222", "Kiosco Beta SRL")

	return env
}

// ── Setup Helpers ──────────────────────────────────────────────────────────

func registerTenant(t *testing.T, srv *httptest.Server, nombre, slug, username, password string) *tenantEnv {
	t.Helper()

	resp := do(t, srv, "POST", "/v1/public/register", jsonBody(t, map[string]string{
		"nombre_negocio": nombre,
		"slug":           slug,
		"username":       username,
		"password":       password,
		"nombre":         "Admin " + slug,
	}), "")
	require.Equal(t, http.StatusCreated, resp.StatusCode, "register tenant %s failed", slug)

	var body struct {
		Tenant struct {
			ID string `json:"id"`
		} `json:"tenant"`
		User struct {
			ID string `json:"id"`
		} `json:"user"`
		AccessToken string `json:"access_token"`
	}
	decodeJSON(t, resp, &body)
	require.NotEmpty(t, body.AccessToken)
	require.NotEmpty(t, body.Tenant.ID)

	return &tenantEnv{
		token:    body.AccessToken,
		tenantID: body.Tenant.ID,
		userID:   body.User.ID,
	}
}

func createProduct(t *testing.T, srv *httptest.Server, token, nombre, barcode string, stock int) string {
	t.Helper()

	resp := do(t, srv, "POST", "/v1/productos", jsonBody(t, map[string]any{
		"nombre":        nombre,
		"codigo_barras": barcode,
		"categoria":     "bebidas",
		"precio_costo":  100.0,
		"precio_venta":  200.0,
		"stock_actual":  stock,
	}), token)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "create product %s failed", nombre)

	var body struct {
		ID string `json:"id"`
	}
	decodeJSON(t, resp, &body)
	require.NotEmpty(t, body.ID)
	return body.ID
}

func openCaja(t *testing.T, srv *httptest.Server, token string) string {
	t.Helper()

	resp := do(t, srv, "POST", "/v1/caja/abrir", jsonBody(t, map[string]any{
		"punto_de_venta": 1,
		"monto_inicial":  1000.0,
	}), token)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "open caja failed")

	var body struct {
		SesionCajaID string `json:"sesion_caja_id"`
	}
	decodeJSON(t, resp, &body)
	require.NotEmpty(t, body.SesionCajaID)
	return body.SesionCajaID
}

func createSale(t *testing.T, srv *httptest.Server, token, cajaID, productID string) string {
	t.Helper()

	resp := do(t, srv, "POST", "/v1/ventas", jsonBody(t, map[string]any{
		"sesion_caja_id": cajaID,
		"items": []map[string]any{
			{"producto_id": productID, "cantidad": 2, "descuento": 0},
		},
		"pagos": []map[string]any{
			{"metodo": "efectivo", "monto": 400.0},
		},
	}), token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var body struct {
		ID string `json:"id"`
	}
	decodeJSON(t, resp, &body)
	require.NotEmpty(t, body.ID)
	return body.ID
}

func createFiscalConfig(t *testing.T, db *gorm.DB, tenantIDStr, cuit, razonSocial string) {
	t.Helper()

	// Use a transaction so set_config and INSERT share the same connection.
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT set_config('app.tenant_id', ?, true)", tenantIDStr).Error; err != nil {
			return err
		}
		return tx.Exec(`INSERT INTO configuracion_fiscal
			(id, tenant_id, cuit_emisor, razon_social, condicion_fiscal, punto_de_venta, modo, created_at, updated_at)
			VALUES (gen_random_uuid(), ?, ?, ?, 'Responsable Inscripto', 1, 'homologacion', NOW(), NOW())`,
			tenantIDStr, cuit, razonSocial).Error
	})
	require.NoError(t, err)
}

// signIsolationToken creates a signed JWT with specific claims for testing.
func signIsolationToken(t *testing.T, tenantID, userID, role string, expiry time.Duration) string {
	t.Helper()
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": "testuser",
		"rol":      role,
		"tid":      tenantID,
		"did":      uuid.New().String(),
		"type":     "access",
		"exp":      time.Now().Add(expiry).Unix(),
		"iat":      time.Now().Unix(),
		"jti":      uuid.New().String(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)
	return s
}

// noopMPClient is a placeholder MercadoPago client for tests.
type noopMPClient struct{}

func (c *noopMPClient) CreateSubscription(_ context.Context, _ service.MPCreateSubscriptionRequest) (*service.MPCreateSubscriptionResponse, error) {
	return nil, fmt.Errorf("MP not implemented in tests")
}

// ── Test Suite ──────────────────────────────────────────────────────────────

func TestTenantIsolation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	env := setupIsolationEnv(t)

	// ── Isolation básica ────────────────────────────────────────────────

	t.Run("TenantA_cannot_list_TenantB_products", func(t *testing.T) {
		resp := do(t, env.server, "GET", "/v1/productos", nil, env.tenantA.token)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var listResp struct {
			Data []idHolder `json:"data"`
		}
		decodeJSON(t, resp, &listResp)

		// Only Tenant A's product should appear
		ids := extractProductIDs(listResp.Data)
		assert.Contains(t, ids, env.tenantA.productID, "Tenant A should see own product")
		assert.NotContains(t, ids, env.tenantB.productID, "Tenant A must NOT see Tenant B product")
	})

	t.Run("TenantA_cannot_list_TenantB_sales", func(t *testing.T) {
		resp := do(t, env.server, "GET",
			fmt.Sprintf("/v1/ventas?fecha=%s", time.Now().Format("2006-01-02")),
			nil, env.tenantA.token)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body json.RawMessage
		decodeJSON(t, resp, &body)

		assert.Contains(t, string(body), env.tenantA.ventaID, "Tenant A should see own sale")
		assert.NotContains(t, string(body), env.tenantB.ventaID, "Tenant A must NOT see Tenant B sale")
	})

	t.Run("TenantA_cannot_get_TenantB_product_by_ID", func(t *testing.T) {
		resp := do(t, env.server, "GET", "/v1/productos/"+env.tenantB.productID, nil, env.tenantA.token)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode, "IDOR guard should block cross-tenant product access")
		resp.Body.Close()
	})

	t.Run("TenantA_cannot_see_TenantB_fiscal_config", func(t *testing.T) {
		respA := do(t, env.server, "GET", "/v1/configuracion/fiscal", nil, env.tenantA.token)
		require.Equal(t, http.StatusOK, respA.StatusCode)

		var wrapperA struct {
			Data struct {
				CUITEmisor string `json:"cuit_emisor"`
			} `json:"data"`
		}
		decodeJSON(t, respA, &wrapperA)
		assert.Equal(t, "20111111111", wrapperA.Data.CUITEmisor, "Tenant A should see own fiscal config")

		respB := do(t, env.server, "GET", "/v1/configuracion/fiscal", nil, env.tenantB.token)
		require.Equal(t, http.StatusOK, respB.StatusCode)

		var wrapperB struct {
			Data struct {
				CUITEmisor string `json:"cuit_emisor"`
			} `json:"data"`
		}
		decodeJSON(t, respB, &wrapperB)
		assert.Equal(t, "20222222222", wrapperB.Data.CUITEmisor, "Tenant B should see own fiscal config")

		// Cross check: A's config is NOT B's config
		assert.NotEqual(t, wrapperA.Data.CUITEmisor, wrapperB.Data.CUITEmisor,
			"Tenants must see different fiscal configs")
	})

	// ── Sync-batch isolation ────────────────────────────────────────────

	t.Run("sync_batch_with_TenantA_JWT_does_not_affect_TenantB", func(t *testing.T) {
		// Get Tenant B stock before
		respBefore := do(t, env.server, "GET", "/v1/productos/"+env.tenantB.productID, nil, env.tenantB.token)
		require.Equal(t, http.StatusOK, respBefore.StatusCode)
		var beforeProd struct {
			StockActual int `json:"stock_actual"`
		}
		decodeJSON(t, respBefore, &beforeProd)

		// Sync-batch a sale for Tenant A using Tenant A's product
		offlineID := uuid.New().String()
		batch := map[string]any{
			"ventas": []map[string]any{
				{
					"sesion_caja_id": env.tenantA.cajaID,
					"offline_id":     offlineID,
					"items":          []map[string]any{{"producto_id": env.tenantA.productID, "cantidad": 1, "descuento": 0}},
					"pagos":          []map[string]any{{"metodo": "efectivo", "monto": 200.0}},
				},
			},
		}
		batchResp := do(t, env.server, "POST", "/v1/ventas/sync-batch", jsonBody(t, batch), env.tenantA.token)
		require.Equal(t, http.StatusOK, batchResp.StatusCode)
		batchResp.Body.Close()

		// Verify Tenant B stock is unchanged
		respAfter := do(t, env.server, "GET", "/v1/productos/"+env.tenantB.productID, nil, env.tenantB.token)
		require.Equal(t, http.StatusOK, respAfter.StatusCode)
		var afterProd struct {
			StockActual int `json:"stock_actual"`
		}
		decodeJSON(t, respAfter, &afterProd)
		assert.Equal(t, beforeProd.StockActual, afterProd.StockActual,
			"Tenant B stock must NOT change after Tenant A sync-batch")
	})

	t.Run("sync_batch_ignores_tenant_id_in_body", func(t *testing.T) {
		offlineID := uuid.New().String()
		batch := map[string]any{
			"ventas": []map[string]any{
				{
					"sesion_caja_id": env.tenantA.cajaID,
					"offline_id":     offlineID,
					"tenant_id":      env.tenantB.tenantID, // Malicious — should be ignored
					"items":          []map[string]any{{"producto_id": env.tenantA.productID, "cantidad": 1, "descuento": 0}},
					"pagos":          []map[string]any{{"metodo": "efectivo", "monto": 200.0}},
				},
			},
		}
		resp := do(t, env.server, "POST", "/v1/ventas/sync-batch", jsonBody(t, batch), env.tenantA.token)
		// Should succeed and the sale should belong to Tenant A, not B
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// Verify sale was NOT created in Tenant B
		ventasB := do(t, env.server, "GET",
			fmt.Sprintf("/v1/ventas?fecha=%s", time.Now().Format("2006-01-02")),
			nil, env.tenantB.token)
		require.Equal(t, http.StatusOK, ventasB.StatusCode)
		var bodyB json.RawMessage
		decodeJSON(t, ventasB, &bodyB)
		assert.NotContains(t, string(bodyB), offlineID,
			"Sale with injected tenant_id must NOT appear in Tenant B")
	})

	// ── IDOR attempts ───────────────────────────────────────────────────

	t.Run("IDOR_GET_product_of_other_tenant_returns_404", func(t *testing.T) {
		resp := do(t, env.server, "GET", "/v1/productos/"+env.tenantB.productID, nil, env.tenantA.token)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("IDOR_PUT_product_of_other_tenant_returns_404", func(t *testing.T) {
		resp := do(t, env.server, "PUT", "/v1/productos/"+env.tenantB.productID,
			jsonBody(t, map[string]any{
				"nombre":        "Hacked Name",
				"codigo_barras": "7890000000002",
				"precio_costo":  1.0,
				"precio_venta":  1.0,
				"stock_actual":  999,
			}),
			env.tenantA.token)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode, "PUT to other tenant's product should 404")
		resp.Body.Close()

		// Verify product was NOT modified
		check := do(t, env.server, "GET", "/v1/productos/"+env.tenantB.productID, nil, env.tenantB.token)
		require.Equal(t, http.StatusOK, check.StatusCode)
		var p struct {
			Nombre string `json:"nombre"`
		}
		decodeJSON(t, check, &p)
		assert.Equal(t, "Gaseosa Beta", p.Nombre, "Product name must be unchanged")
	})

	t.Run("IDOR_DELETE_user_of_other_tenant_returns_404", func(t *testing.T) {
		resp := do(t, env.server, "DELETE", "/v1/usuarios/"+env.tenantB.userID, nil, env.tenantA.token)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode, "DELETE of other tenant's user should 404")
		resp.Body.Close()
	})

	// ── SQL injection in parameters ─────────────────────────────────────

	t.Run("SQL_injection_in_offline_id_no_breaks_isolation", func(t *testing.T) {
		maliciousOfflineID := "'; DROP TABLE ventas; --"
		saleBody := map[string]any{
			"sesion_caja_id": env.tenantA.cajaID,
			"offline_id":     maliciousOfflineID,
			"items":          []map[string]any{{"producto_id": env.tenantA.productID, "cantidad": 1, "descuento": 0}},
			"pagos":          []map[string]any{{"metodo": "efectivo", "monto": 200.0}},
		}
		resp := do(t, env.server, "POST", "/v1/ventas", jsonBody(t, saleBody), env.tenantA.token)
		// Should either succeed (parameterized queries handle it) or fail gracefully — NOT drop tables
		resp.Body.Close()

		// Verify ventas table still exists and Tenant B data is intact
		checkB := do(t, env.server, "GET",
			fmt.Sprintf("/v1/ventas?fecha=%s", time.Now().Format("2006-01-02")),
			nil, env.tenantB.token)
		require.Equal(t, http.StatusOK, checkB.StatusCode,
			"ventas table must still exist after SQL injection attempt")
		checkB.Body.Close()
	})

	t.Run("SQL_injection_in_search_params_no_breaks", func(t *testing.T) {
		resp := do(t, env.server, "GET", "/v1/productos?search='+DROP+TABLE+productos;+--", nil, env.tenantA.token)
		// Should return 200 (empty result or error) — NOT drop the table
		resp.Body.Close()

		// Verify productos table still works
		check := do(t, env.server, "GET", "/v1/productos", nil, env.tenantB.token)
		require.Equal(t, http.StatusOK, check.StatusCode,
			"productos table must still exist after SQL injection attempt")
		check.Body.Close()
	})

	// ── JWT manipulation ────────────────────────────────────────────────

	t.Run("JWT_with_tampered_tenant_id_returns_401", func(t *testing.T) {
		// Create a valid-looking JWT but sign with a DIFFERENT secret
		claims := jwt.MapClaims{
			"user_id":  env.tenantA.userID,
			"username": "admin-a",
			"rol":      "administrador",
			"tid":      env.tenantB.tenantID, // Trying to access Tenant B
			"did":      uuid.New().String(),
			"type":     "access",
			"exp":      time.Now().Add(time.Hour).Unix(),
			"iat":      time.Now().Unix(),
		}
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tamperedToken, err := tok.SignedString([]byte("wrong-secret-that-doesnt-match-server"))
		require.NoError(t, err)

		resp := do(t, env.server, "GET", "/v1/productos", nil, tamperedToken)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
			"Tampered JWT must be rejected")
		resp.Body.Close()
	})

	t.Run("JWT_expired_returns_401", func(t *testing.T) {
		// Sign with the correct secret but expired 1 hour ago
		expiredToken := signIsolationToken(t, env.tenantA.tenantID, env.tenantA.userID, "administrador", -1*time.Hour)

		resp := do(t, env.server, "GET", "/v1/productos", nil, expiredToken)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
			"Expired JWT must be rejected")
		resp.Body.Close()
	})

	// ── SuperAdmin bypass ───────────────────────────────────────────────

	t.Run("SuperAdmin_can_see_all_tenants", func(t *testing.T) {
		saToken := signIsolationToken(t, uuid.New().String(), uuid.New().String(), "superadmin", time.Hour)

		resp := do(t, env.server, "GET", "/v1/superadmin/tenants", nil, saToken)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Tenants []struct {
				ID string `json:"id"`
			} `json:"tenants"`
			Total int64 `json:"total"`
		}
		decodeJSON(t, resp, &body)
		assert.GreaterOrEqual(t, body.Total, int64(2), "SuperAdmin should see at least 2 tenants")

		ids := make([]string, len(body.Tenants))
		for i, t := range body.Tenants {
			ids[i] = t.ID
		}
		assert.Contains(t, ids, env.tenantA.tenantID, "SuperAdmin should see Tenant A")
		assert.Contains(t, ids, env.tenantB.tenantID, "SuperAdmin should see Tenant B")
	})

	t.Run("Non_superadmin_cannot_access_superadmin_routes", func(t *testing.T) {
		// Use Tenant A's admin token (role = administrador)
		resp := do(t, env.server, "GET", "/v1/superadmin/tenants", nil, env.tenantA.token)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode,
			"Non-superadmin should get 403 on /superadmin routes")
		resp.Body.Close()
	})

	// ── Cross-verification: each tenant only sees their own data ────────

	t.Run("TenantB_cannot_list_TenantA_products", func(t *testing.T) {
		resp := do(t, env.server, "GET", "/v1/productos", nil, env.tenantB.token)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var listResp struct {
			Data []idHolder `json:"data"`
		}
		decodeJSON(t, resp, &listResp)

		ids := extractProductIDs(listResp.Data)
		assert.Contains(t, ids, env.tenantB.productID)
		assert.NotContains(t, ids, env.tenantA.productID)
	})

	t.Run("TenantB_cannot_get_TenantA_product_by_ID", func(t *testing.T) {
		resp := do(t, env.server, "GET", "/v1/productos/"+env.tenantA.productID, nil, env.tenantB.token)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("TenantB_cannot_delete_TenantA_sale", func(t *testing.T) {
		resp := do(t, env.server, "DELETE", "/v1/ventas/"+env.tenantA.ventaID,
			jsonBody(t, map[string]any{"motivo": "Cross-tenant attack"}),
			env.tenantB.token)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode,
			"Deleting other tenant's sale should return 404")
		resp.Body.Close()
	})
}

// ── Utility ────────────────────────────────────────────────────────────────

type idHolder struct {
	ID string `json:"id"`
}

func extractProductIDs(items []idHolder) []string {
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	return ids
}

// Redefine jsonBody/do/decodeJSON as they are in e2e_test.go — Go test
// files in the same package share scope, so these are accessible. But we
// need to make sure the helpers exist. If they conflict at compile time,
// remove these redefinitions.
// NOTE: These are already defined in e2e_test.go in the same package,
// so we do NOT redefine them here.

// runSQLMigrations applies all *.up.sql migration files in order.
// This replaces the disabled GORM AutoMigrate for e2e tests.
func runSQLMigrations(t *testing.T, db *gorm.DB) {
	t.Helper()

	migrationsDir := filepath.Join("..", "..", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	require.NoError(t, err, "failed to read migrations directory")

	var upFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			upFiles = append(upFiles, e.Name())
		}
	}
	sort.Strings(upFiles) // lexicographic order = numeric order for 000001_xxx format

	for _, f := range upFiles {
		content, err := os.ReadFile(filepath.Join(migrationsDir, f))
		require.NoError(t, err, "failed to read migration %s", f)

		err = db.Exec(string(content)).Error
		require.NoError(t, err, "failed to execute migration %s", f)
	}
}

// Silence unused import warnings — the helpers from e2e_test.go are used.
var (
	_ = bytes.NewBuffer
	_ = json.Marshal
)
