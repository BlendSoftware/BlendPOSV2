package service

import (
	"context"
	"errors"
	"time"

	"blendpos/internal/config"
	"blendpos/internal/dto"
	"blendpos/internal/model"
	"blendpos/internal/repository"
	"blendpos/internal/tenantctx"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// starterPlanID is the fixed UUID of the Starter plan seeded in migration 000027.
const starterPlanID = "00000000-0000-0000-0000-000000000002"

// TenantService manages tenant lifecycle and self-service operations.
type TenantService interface {
	// Registrar creates a new tenant + admin user. Returns a ready-to-use JWT.
	Registrar(ctx context.Context, req dto.RegisterTenantRequest) (*dto.RegisterTenantResponse, error)

	// ObtenerActual returns the tenant info for the caller's JWT.
	ObtenerActual(ctx context.Context) (*dto.TenantResponse, error)

	// ActualizarActual updates mutable fields of the caller's tenant.
	ActualizarActual(ctx context.Context, req dto.ActualizarTenantRequest) (*dto.TenantResponse, error)

	// ListarTodos returns all tenants (superadmin use only).
	ListarTodos(ctx context.Context) ([]dto.SuperadminTenantListItem, error)

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
	repo     repository.TenantRepository
	usuRepo  repository.UsuarioRepository
	cfg      *config.Config
	rdb      *redis.Client
}

func NewTenantService(
	repo repository.TenantRepository,
	usuRepo repository.UsuarioRepository,
	cfg *config.Config,
	rdb *redis.Client,
) TenantService {
	return &tenantService{repo: repo, usuRepo: usuRepo, cfg: cfg, rdb: rdb}
}

// ── Registration ──────────────────────────────────────────────────────────────

func (s *tenantService) Registrar(ctx context.Context, req dto.RegisterTenantRequest) (*dto.RegisterTenantResponse, error) {
	// Check slug uniqueness
	if existing, err := s.repo.FindTenantBySlug(ctx, req.Slug); err == nil && existing.ID != uuid.Nil {
		return nil, errors.New("el slug ya está en uso")
	}

	planID := uuid.MustParse(starterPlanID)
	tenant := &model.Tenant{
		Slug:   req.Slug,
		Nombre: req.NombreNegocio,
		PlanID: &planID,
		Activo: true,
	}
	if err := s.repo.CreateTenant(ctx, tenant); err != nil {
		return nil, err
	}

	// Create admin user inside the new tenant context.
	// We inject the new tenant_id into ctx so Create() stamps it correctly.
	tenantCtx := context.WithValue(ctx, tenantctx.Key, tenant.ID)

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, err
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
	if err := s.usuRepo.Create(tenantCtx, admin); err != nil {
		return nil, err
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

	return &dto.RegisterTenantResponse{
		Tenant:       toTenantResponse(tenant),
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

func (s *tenantService) ListarTodos(ctx context.Context) ([]dto.SuperadminTenantListItem, error) {
	tenants, err := s.repo.ListTenants(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]dto.SuperadminTenantListItem, len(tenants))
	for i, t := range tenants {
		tc := t
		ventas, _ := s.repo.CountVentasByTenant(ctx, tc.ID)
		usuarios, _ := s.repo.CountUsuariosByTenant(ctx, tc.ID)
		items[i] = dto.SuperadminTenantListItem{
			ID:            tc.ID.String(),
			Slug:          tc.Slug,
			Nombre:        tc.Nombre,
			CUIT:          tc.CUIT,
			Activo:        tc.Activo,
			TotalVentas:   ventas,
			TotalUsuarios: usuarios,
			CreatedAt:     tc.CreatedAt.Format(time.RFC3339),
		}
		if tc.Plan != nil {
			pr := toPlanResponse(tc.Plan)
			items[i].Plan = &pr
		}
	}
	return items, nil
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
	tenants, err := s.repo.ListTenants(ctx)
	if err != nil {
		return nil, err
	}
	var activos int64
	for _, t := range tenants {
		if t.Activo {
			activos++
		}
	}
	return &dto.SuperadminMetricsResponse{
		TotalTenants:  int64(len(tenants)),
		TenantActivos: activos,
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
	claims := jwt.MapClaims{
		"jti":            uuid.New().String(),
		"user_id":        u.ID.String(),
		"username":       u.Username,
		"rol":            u.Rol,
		"punto_de_venta": u.PuntoDeVenta,
		"tid":            u.TenantID.String(),
		"did":            deviceID,
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
		ID:        t.ID.String(),
		Slug:      t.Slug,
		Nombre:    t.Nombre,
		CUIT:      t.CUIT,
		Activo:    t.Activo,
		CreatedAt: t.CreatedAt.Format(time.RFC3339),
	}
	if t.Plan != nil {
		pr := toPlanResponse(t.Plan)
		resp.Plan = &pr
	}
	return resp
}

func toPlanResponse(p *model.Plan) dto.PlanResponse {
	return dto.PlanResponse{
		ID:            p.ID.String(),
		Nombre:        p.Nombre,
		MaxTerminales: p.MaxTerminales,
		MaxProductos:  p.MaxProductos,
		PrecioMensual: p.PrecioMensual,
	}
}
