package service

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"blendpos/internal/config"
	"blendpos/internal/dto"
	"blendpos/internal/model"
	"blendpos/internal/repository"
	"blendpos/internal/tenantctx"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// basicoPlanID is the fixed UUID of the Basico (free) plan seeded in migration 000027/000040.
const basicoPlanID = "00000000-0000-0000-0000-000000000001"

// TenantService manages tenant lifecycle and self-service operations.
type TenantService interface {
	// Registrar creates a new tenant + admin user. Returns a ready-to-use JWT.
	Registrar(ctx context.Context, req dto.RegisterTenantRequest) (*dto.RegisterTenantResponse, error)

	// ObtenerActual returns the tenant info for the caller's JWT.
	ObtenerActual(ctx context.Context) (*dto.TenantResponse, error)

	// ActualizarActual updates mutable fields of the caller's tenant.
	ActualizarActual(ctx context.Context, req dto.ActualizarTenantRequest) (*dto.TenantResponse, error)

	// ListarTodos returns paginated tenants with metrics (superadmin use only).
	ListarTodos(ctx context.Context, req dto.TenantListRequest) (*dto.TenantListResponse, error)

	// ObtenerTenantDetalle returns a single tenant with metrics (superadmin use only).
	ObtenerTenantDetalle(ctx context.Context, tenantID uuid.UUID) (*dto.SuperadminTenantListItem, error)

	// CambiarPlan assigns a new plan to a tenant (superadmin use only).
	CambiarPlan(ctx context.Context, tenantID uuid.UUID, planID uuid.UUID) (*dto.TenantResponse, error)

	// ToggleActivo activates or deactivates a tenant (superadmin use only).
	ToggleActivo(ctx context.Context, tenantID uuid.UUID, activo bool) (*dto.TenantResponse, error)

	// ObtenerMetricas returns global usage metrics (superadmin use only).
	ObtenerMetricas(ctx context.Context) (*dto.SuperadminMetricsResponse, error)

	// GetPlanActual returns the plan limits for the caller's tenant.
	GetPlanActual(ctx context.Context) (*dto.PlanResponse, error)

	// ListarPlanes returns all available plans (public).
	ListarPlanes(ctx context.Context) ([]dto.PlanResponse, error)
}

type tenantService struct {
	repo        repository.TenantRepository
	usuRepo     repository.UsuarioRepository
	sucursalSvc SucursalService
	cfg         *config.Config
	rdb         *redis.Client
}

func NewTenantService(
	repo repository.TenantRepository,
	usuRepo repository.UsuarioRepository,
	cfg *config.Config,
	rdb *redis.Client,
	sucursalSvc ...SucursalService,
) TenantService {
	svc := &tenantService{repo: repo, usuRepo: usuRepo, cfg: cfg, rdb: rdb}
	if len(sucursalSvc) > 0 {
		svc.sucursalSvc = sucursalSvc[0]
	}
	return svc
}

// slugRegex strips everything that is not lowercase alphanumeric or hyphen.
var slugRegex = regexp.MustCompile(`[^a-z0-9-]+`)

// GenerateSlug creates a URL-safe slug from a business name.
// Exported so tests can validate the algorithm independently.
func GenerateSlug(nombre string) string {
	s := strings.ToLower(strings.TrimSpace(nombre))
	s = strings.ReplaceAll(s, " ", "-")
	s = slugRegex.ReplaceAllString(s, "")
	// Collapse multiple hyphens
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	if len(s) > 63 {
		s = s[:63]
	}
	return s
}

// ── Registration ──────────────────────────────────────────────────────────────

func (s *tenantService) Registrar(ctx context.Context, req dto.RegisterTenantRequest) (*dto.RegisterTenantResponse, error) {
	// Auto-generate slug from nombre_negocio if not provided
	slug := req.Slug
	if slug == "" {
		slug = GenerateSlug(req.NombreNegocio)
	}
	if slug == "" {
		return nil, errors.New("no se pudo generar un slug válido a partir del nombre del negocio")
	}

	// Check slug uniqueness
	if existing, err := s.repo.FindTenantBySlug(ctx, slug); err == nil && existing.ID != uuid.Nil {
		return nil, errors.New("el slug ya está en uso")
	}

	// Hash password before entering the transaction
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, err
	}

	planID := uuid.MustParse(basicoPlanID)

	tipoNegocio := req.TipoNegocio
	if tipoNegocio == "" {
		tipoNegocio = "kiosco"
	}

	tenant := &model.Tenant{
		Slug:        slug,
		Nombre:      req.NombreNegocio,
		PlanID:      &planID,
		TipoNegocio: tipoNegocio,
		Activo:      true,
	}
	if req.CUIT != "" {
		tenant.CUIT = &req.CUIT
	}

	admin := &model.Usuario{
		Username:           req.Username,
		Nombre:             req.Nombre,
		PasswordHash:       string(hash),
		Rol:                "administrador",
		Activo:             true,
		MustChangePassword: true, // Force password change on first login (SEC-03)
	}
	if req.Email != "" {
		admin.Email = &req.Email
	}

	// Use a GORM transaction when a real DB is available (production path).
	// When DB() is nil (unit tests with stubs), fall back to sequential repo calls.
	if db := s.repo.DB(); db != nil {
		txErr := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(tenant).Error; err != nil {
				log.Error().Err(err).Str("slug", slug).Msg("failed to create tenant")
				return err
			}
			admin.TenantID = tenant.ID
			if err := tx.Create(admin).Error; err != nil {
				log.Error().Err(err).Str("slug", slug).Str("username", req.Username).Msg("failed to create admin user")
				return err
			}
			return nil
		})
		if txErr != nil {
			return nil, txErr
		}
	} else {
		// Stub/test path — no real DB, use repo interface methods
		if err := s.repo.CreateTenant(ctx, tenant); err != nil {
			return nil, err
		}
		tenantCtx := context.WithValue(ctx, tenantctx.Key, tenant.ID)
		admin.TenantID = tenant.ID
		if err := s.usuRepo.Create(tenantCtx, admin); err != nil {
			return nil, err
		}
	}

	// Reload with Plan association
	tenant, err = s.repo.FindTenantByID(ctx, tenant.ID)
	if err != nil {
		return nil, err
	}

	// Issue JWT for immediate login
	accessToken, refreshToken, err := s.generateTokenPair(admin)
	if err != nil {
		return nil, err
	}

	log.Info().
		Str("tenant_id", tenant.ID.String()).
		Str("slug", slug).
		Str("admin_user", admin.Username).
		Msg("tenant registered successfully")

	// Create default sucursal "Casa Central" for the new tenant.
	// This runs synchronously (fast, single INSERT) so the JWT and
	// login response can include sucursal context from the start.
	if db := s.repo.DB(); db != nil && s.sucursalSvc != nil {
		if err := s.sucursalSvc.EnsureDefaultSucursal(db, tenant.ID); err != nil {
			log.Error().Err(err).Str("tenant_id", tenant.ID.String()).Msg("failed to create default sucursal")
			// Non-fatal: registration continues
		}
	}

	// Seed preset categories and sample products in background.
	// Best-effort: registration succeeds regardless of seeding outcome.
	if db := s.repo.DB(); db != nil {
		go SeedPresets(db, tenant.ID, tipoNegocio)
	}

	return &dto.RegisterTenantResponse{
		Tenant:       toTenantResponse(tenant),
		User: dto.UsuarioResponse{
			ID:       admin.ID.String(),
			Username: admin.Username,
			Nombre:   admin.Nombre,
			Email:    admin.Email,
			Rol:      admin.Rol,
			Activo:   admin.Activo,
		},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "bearer",
		ExpiresIn:    s.cfg.JWTExpirationHours * 3600,
	}, nil
}

// ── Tenant self-service ───────────────────────────────────────────────────────

func (s *tenantService) ObtenerActual(ctx context.Context) (*dto.TenantResponse, error) {
	tid, err := tenantctx.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	tenant, err := s.repo.FindTenantByID(ctx, tid)
	if err != nil {
		return nil, errors.New("tenant no encontrado")
	}
	resp := toTenantResponse(tenant)
	return &resp, nil
}

func (s *tenantService) ActualizarActual(ctx context.Context, req dto.ActualizarTenantRequest) (*dto.TenantResponse, error) {
	tid, err := tenantctx.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	tenant, err := s.repo.FindTenantByID(ctx, tid)
	if err != nil {
		return nil, errors.New("tenant no encontrado")
	}
	if req.Nombre != "" {
		tenant.Nombre = req.Nombre
	}
	if req.CUIT != nil {
		tenant.CUIT = req.CUIT
	}
	if err := s.repo.UpdateTenant(ctx, tenant); err != nil {
		return nil, err
	}
	resp := toTenantResponse(tenant)
	return &resp, nil
}

func (s *tenantService) GetPlanActual(ctx context.Context) (*dto.PlanResponse, error) {
	tid, err := tenantctx.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	tenant, err := s.repo.FindTenantByID(ctx, tid)
	if err != nil || tenant.Plan == nil {
		return nil, errors.New("plan no encontrado")
	}
	resp := toPlanResponse(tenant.Plan)
	return &resp, nil
}

func (s *tenantService) ListarPlanes(ctx context.Context) ([]dto.PlanResponse, error) {
	plans, err := s.repo.ListPlans(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.PlanResponse, len(plans))
	for i, p := range plans {
		pc := p
		result[i] = toPlanResponse(&pc)
	}
	return result, nil
}

// ── Superadmin operations ─────────────────────────────────────────────────────

func (s *tenantService) ListarTodos(ctx context.Context, req dto.TenantListRequest) (*dto.TenantListResponse, error) {
	req.Defaults()

	filter := repository.TenantListFilter{
		Page:     req.Page,
		PageSize: req.PageSize,
		Search:   req.Search,
		Status:   req.Status,
		PlanID:   req.PlanID,
	}

	tenants, total, err := s.repo.ListAllPaginated(ctx, filter)
	if err != nil {
		return nil, err
	}

	items := make([]dto.SuperadminTenantListItem, len(tenants))
	for i, tw := range tenants {
		items[i] = toSuperadminTenantItem(&tw)
	}

	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	return &dto.TenantListResponse{
		Tenants:    items,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *tenantService) ObtenerTenantDetalle(ctx context.Context, tenantID uuid.UUID) (*dto.SuperadminTenantListItem, error) {
	tw, err := s.repo.FindTenantWithMetrics(ctx, tenantID)
	if err != nil {
		return nil, errors.New("tenant no encontrado")
	}
	item := toSuperadminTenantItem(tw)
	return &item, nil
}

func (s *tenantService) CambiarPlan(ctx context.Context, tenantID uuid.UUID, planID uuid.UUID) (*dto.TenantResponse, error) {
	tenant, err := s.repo.FindTenantByID(ctx, tenantID)
	if err != nil {
		return nil, errors.New("tenant no encontrado")
	}
	plan, err := s.repo.FindPlanByID(ctx, planID)
	if err != nil {
		return nil, errors.New("plan no encontrado")
	}
	tenant.PlanID = &plan.ID
	if err := s.repo.UpdateTenant(ctx, tenant); err != nil {
		return nil, err
	}
	// Reload with plan
	tenant, _ = s.repo.FindTenantByID(ctx, tenantID)
	resp := toTenantResponse(tenant)
	return &resp, nil
}

func (s *tenantService) ToggleActivo(ctx context.Context, tenantID uuid.UUID, activo bool) (*dto.TenantResponse, error) {
	tenant, err := s.repo.FindTenantByID(ctx, tenantID)
	if err != nil {
		return nil, errors.New("tenant no encontrado")
	}
	tenant.Activo = activo
	if err := s.repo.UpdateTenant(ctx, tenant); err != nil {
		return nil, err
	}
	resp := toTenantResponse(tenant)
	return &resp, nil
}

func (s *tenantService) ObtenerMetricas(ctx context.Context) (*dto.SuperadminMetricsResponse, error) {
	m, err := s.repo.GetGlobalMetrics(ctx)
	if err != nil {
		return nil, err
	}

	planCounts := make([]dto.PlanCountDTO, len(m.TenantsPorPlan))
	for i, pc := range m.TenantsPorPlan {
		planCounts[i] = dto.PlanCountDTO{PlanNombre: pc.PlanNombre, Count: pc.Count}
	}

	return &dto.SuperadminMetricsResponse{
		TotalTenants:    m.TotalTenants,
		TenantActivos:   m.TenantActivos,
		TotalVentas:     m.TotalVentas,
		VentasUltimoMes: m.VentasUltimoMes,
		TenantsPorPlan:  planCounts,
	}, nil
}

// ── JWT generation ────────────────────────────────────────────────────────────

func (s *tenantService) generateTokenPair(u *model.Usuario) (access, refresh string, err error) {
	access, err = s.generateToken(u, time.Duration(s.cfg.JWTExpirationHours)*time.Hour, "access")
	if err != nil {
		return "", "", err
	}
	refresh, err = s.generateToken(u, time.Duration(s.cfg.JWTRefreshHours)*time.Hour, "refresh")
	return access, refresh, err
}

func (s *tenantService) generateToken(u *model.Usuario, dur time.Duration, tokenType string) (string, error) {
	deviceID := ""
	if u.DeviceID != nil {
		deviceID = *u.DeviceID
	}
	var sucursalID string
	if u.SucursalID != nil {
		sucursalID = u.SucursalID.String()
	}
	claims := jwt.MapClaims{
		"jti":            uuid.New().String(),
		"user_id":        u.ID.String(),
		"username":       u.Username,
		"rol":            u.Rol,
		"punto_de_venta": u.PuntoDeVenta,
		"tid":            u.TenantID.String(),
		"did":            deviceID,
		"sid":            sucursalID, // sucursal_id
		"type":           tokenType,
		"exp":            time.Now().Add(dur).Unix(),
		"iat":            time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

// ── Mappers ───────────────────────────────────────────────────────────────────

func toTenantResponse(t *model.Tenant) dto.TenantResponse {
	resp := dto.TenantResponse{
		ID:          t.ID.String(),
		Slug:        t.Slug,
		Nombre:      t.Nombre,
		CUIT:        t.CUIT,
		TipoNegocio: t.TipoNegocio,
		Activo:      t.Activo,
		CreatedAt:   t.CreatedAt.Format(time.RFC3339),
	}
	if t.Plan != nil {
		pr := toPlanResponse(t.Plan)
		resp.Plan = &pr
	}
	return resp
}

func toPlanResponse(p *model.Plan) dto.PlanResponse {
	features := make(map[string]bool)
	if len(p.Features) > 0 && string(p.Features) != "null" {
		_ = json.Unmarshal(p.Features, &features)
	}
	return dto.PlanResponse{
		ID:            p.ID.String(),
		Nombre:        p.Nombre,
		MaxTerminales: p.MaxTerminales,
		MaxProductos:  p.MaxProductos,
		MaxSucursales: p.MaxSucursales,
		MaxUsuarios:   p.MaxUsuarios,
		PrecioMensual: p.PrecioMensual,
		Features:      features,
	}
}

func toSuperadminTenantItem(tw *repository.TenantWithMetrics) dto.SuperadminTenantListItem {
	item := dto.SuperadminTenantListItem{
		ID:             tw.ID.String(),
		Slug:           tw.Slug,
		Nombre:         tw.Nombre,
		CUIT:           tw.CUIT,
		Activo:         tw.Activo,
		TotalVentas:    tw.TotalVentas,
		TotalProductos: tw.TotalProductos,
		TotalUsuarios:  tw.TotalUsuarios,
		CreatedAt:      tw.CreatedAt.Format(time.RFC3339),
	}
	if tw.Plan != nil {
		pr := toPlanResponse(tw.Plan)
		item.Plan = &pr
	}
	if tw.UltimaVenta != nil {
		item.UltimaVenta = tw.UltimaVenta.Format(time.RFC3339)
	}
	return item
}
