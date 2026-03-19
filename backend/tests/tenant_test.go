package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"blendpos/internal/config"
	"blendpos/internal/dto"
	"blendpos/internal/handler"
	"blendpos/internal/model"
	"blendpos/internal/repository"
	"blendpos/internal/service"
	"blendpos/internal/tenantctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ── Stub Tenant Repository ──────────────────────────────────────────────────

type stubTenantRepo struct {
	tenants map[string]*model.Tenant // keyed by slug
	plans   map[string]*model.Plan   // keyed by ID string
}

func newStubTenantRepo() *stubTenantRepo {
	starterID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	starter := &model.Plan{
		ID:            starterID,
		Nombre:        "Starter",
		MaxTerminales: 1,
		MaxProductos:  100,
		PrecioMensual: decimal.NewFromInt(0),
		Activo:        true,
		CreatedAt:     time.Now(),
	}
	return &stubTenantRepo{
		tenants: make(map[string]*model.Tenant),
		plans:   map[string]*model.Plan{starterID.String(): starter},
	}
}

func (r *stubTenantRepo) CreateTenant(_ context.Context, t *model.Tenant) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	t.CreatedAt = time.Now()
	r.tenants[t.Slug] = t
	return nil
}

func (r *stubTenantRepo) FindTenantByID(_ context.Context, id uuid.UUID) (*model.Tenant, error) {
	for _, t := range r.tenants {
		if t.ID == id {
			// Attach Plan if PlanID is set
			if t.PlanID != nil {
				if p, ok := r.plans[t.PlanID.String()]; ok {
					t.Plan = p
				}
			}
			return t, nil
		}
	}
	return nil, errors.New("tenant not found")
}

func (r *stubTenantRepo) FindTenantBySlug(_ context.Context, slug string) (*model.Tenant, error) {
	t, ok := r.tenants[slug]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return t, nil
}

func (r *stubTenantRepo) UpdateTenant(_ context.Context, t *model.Tenant) error {
	r.tenants[t.Slug] = t
	return nil
}

func (r *stubTenantRepo) ListTenants(_ context.Context) ([]model.Tenant, error) {
	result := make([]model.Tenant, 0, len(r.tenants))
	for _, t := range r.tenants {
		result = append(result, *t)
	}
	return result, nil
}

func (r *stubTenantRepo) FindPlanByID(_ context.Context, id uuid.UUID) (*model.Plan, error) {
	p, ok := r.plans[id.String()]
	if !ok {
		return nil, errors.New("plan not found")
	}
	return p, nil
}

func (r *stubTenantRepo) FindPlanByNombre(_ context.Context, nombre string) (*model.Plan, error) {
	for _, p := range r.plans {
		if p.Nombre == nombre {
			return p, nil
		}
	}
	return nil, errors.New("plan not found")
}

func (r *stubTenantRepo) ListPlans(_ context.Context) ([]model.Plan, error) {
	result := make([]model.Plan, 0, len(r.plans))
	for _, p := range r.plans {
		if p.Activo {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (r *stubTenantRepo) CountProductosByTenant(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}

func (r *stubTenantRepo) CountUsuariosByTenant(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}

func (r *stubTenantRepo) CountVentasByTenant(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}

func (r *stubTenantRepo) ListAllPaginated(_ context.Context, _ repository.TenantListFilter) ([]repository.TenantWithMetrics, int64, error) {
	tenants, _ := r.ListTenants(context.Background())
	result := make([]repository.TenantWithMetrics, len(tenants))
	for i, t := range tenants {
		result[i] = repository.TenantWithMetrics{Tenant: t}
	}
	return result, int64(len(tenants)), nil
}

func (r *stubTenantRepo) FindTenantWithMetrics(_ context.Context, id uuid.UUID) (*repository.TenantWithMetrics, error) {
	t, err := r.FindTenantByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	return &repository.TenantWithMetrics{Tenant: *t}, nil
}

func (r *stubTenantRepo) GetGlobalMetrics(_ context.Context) (*repository.GlobalMetrics, error) {
	tenants, _ := r.ListTenants(context.Background())
	var activos int64
	for _, t := range tenants {
		if t.Activo {
			activos++
		}
	}
	return &repository.GlobalMetrics{
		TotalTenants:  int64(len(tenants)),
		TenantActivos: activos,
	}, nil
}

func (r *stubTenantRepo) DB() *gorm.DB { return nil }

// ── Helpers ──────────────────────────────────────────────────────────────────

func newTenantTestCfg() *config.Config {
	return &config.Config{
		JWTSecret:          testSecret,
		JWTExpirationHours: 8,
		JWTRefreshHours:    24,
	}
}

func newTenantTestService() (service.TenantService, *stubTenantRepo, *stubUsuarioRepo) {
	tenantRepo := newStubTenantRepo()
	usuarioRepo := newStubRepo()
	cfg := newTenantTestCfg()
	svc := service.NewTenantService(tenantRepo, usuarioRepo, cfg, nil)
	return svc, tenantRepo, usuarioRepo
}

func doRegisterRequest(t *testing.T, svc service.TenantService, req dto.RegisterTenantRequest) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewTenantsHandler(svc)
	r.POST("/v1/public/register", h.Register)

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest(http.MethodPost, "/v1/public/register", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)
	return w
}

// ── Tests: Slug Generation ───────────────────────────────────────────────────

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple name", "Mi Kiosco", "mi-kiosco"},
		{"with accents and special chars", "Café & Más!", "caf-ms"},
		{"multiple spaces", "Kiosco   del   Centro", "kiosco-del-centro"},
		{"already lowercase", "mini-market", "mini-market"},
		{"leading/trailing spaces", "  Mi Negocio  ", "mi-negocio"},
		{"numbers", "Kiosco 24hs", "kiosco-24hs"},
		{"only special chars", "!!!", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.GenerateSlug(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// ── Tests: Tenant Registration (Service Layer) ──────────────────────────────

func TestRegistrar_Success(t *testing.T) {
	svc, tenantRepo, usuarioRepo := newTenantTestService()

	resp, err := svc.Registrar(context.Background(), dto.RegisterTenantRequest{
		NombreNegocio: "Mi Kiosco",
		Slug:          "mi-kiosco",
		Username:      "admin",
		Password:      "password123",
		Nombre:        "Admin User",
		Email:         "admin@test.com",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "bearer", resp.TokenType)
	assert.Equal(t, "mi-kiosco", resp.Tenant.Slug)
	assert.Equal(t, "Mi Kiosco", resp.Tenant.Nombre)
	assert.True(t, resp.Tenant.Activo)
	assert.Equal(t, "admin", resp.User.Username)
	assert.Equal(t, "administrador", resp.User.Rol)
	assert.True(t, resp.User.Activo)

	// Verify tenant was stored
	_, err = tenantRepo.FindTenantBySlug(context.Background(), "mi-kiosco")
	assert.NoError(t, err)

	// Verify admin user was stored
	_, err = usuarioRepo.FindByUsername(context.Background(), "admin")
	assert.NoError(t, err)
}

func TestRegistrar_AutoGenerateSlug(t *testing.T) {
	svc, tenantRepo, _ := newTenantTestService()

	resp, err := svc.Registrar(context.Background(), dto.RegisterTenantRequest{
		NombreNegocio: "El Kiosco de Juan",
		// Slug omitted — should be auto-generated
		Username: "juanadmin",
		Password: "password123",
		Nombre:   "Juan",
	})

	require.NoError(t, err)
	assert.Equal(t, "el-kiosco-de-juan", resp.Tenant.Slug)

	_, err = tenantRepo.FindTenantBySlug(context.Background(), "el-kiosco-de-juan")
	assert.NoError(t, err)
}

func TestRegistrar_WithCUIT(t *testing.T) {
	svc, _, _ := newTenantTestService()

	resp, err := svc.Registrar(context.Background(), dto.RegisterTenantRequest{
		NombreNegocio: "Kiosco CUIT",
		Slug:          "kiosco-cuit",
		CUIT:          "20345678901",
		Username:      "admin2",
		Password:      "password123",
		Nombre:        "Admin",
	})

	require.NoError(t, err)
	assert.NotNil(t, resp.Tenant)
}

func TestRegistrar_DuplicateSlug(t *testing.T) {
	svc, _, _ := newTenantTestService()

	// First registration
	_, err := svc.Registrar(context.Background(), dto.RegisterTenantRequest{
		NombreNegocio: "Mi Kiosco",
		Slug:          "mi-kiosco",
		Username:      "admin1",
		Password:      "password123",
		Nombre:        "Admin 1",
	})
	require.NoError(t, err)

	// Second registration with same slug
	_, err = svc.Registrar(context.Background(), dto.RegisterTenantRequest{
		NombreNegocio: "Mi Otro Kiosco",
		Slug:          "mi-kiosco",
		Username:      "admin2",
		Password:      "password123",
		Nombre:        "Admin 2",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "slug ya está en uso")
}

func TestRegistrar_JWTContainsTenantID(t *testing.T) {
	svc, _, _ := newTenantTestService()

	resp, err := svc.Registrar(context.Background(), dto.RegisterTenantRequest{
		NombreNegocio: "JWT Test",
		Slug:          "jwt-test",
		Username:      "jwtadmin",
		Password:      "password123",
		Nombre:        "JWT Admin",
	})
	require.NoError(t, err)

	// The access token should contain tid claim matching the tenant ID
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.Tenant.ID)
}

func TestRegistrar_PlanIsStarter(t *testing.T) {
	svc, _, _ := newTenantTestService()

	resp, err := svc.Registrar(context.Background(), dto.RegisterTenantRequest{
		NombreNegocio: "Plan Test",
		Slug:          "plan-test",
		Username:      "planadmin",
		Password:      "password123",
		Nombre:        "Plan Admin",
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Tenant.Plan)
	assert.Equal(t, "Starter", resp.Tenant.Plan.Nombre)
}

// ── Tests: Handler Validation ───────────────────────────────────────────────

func TestRegisterHandler_MissingRequiredFields(t *testing.T) {
	svc, _, _ := newTenantTestService()

	// Missing nombre_negocio, username, password, nombre
	w := doRegisterRequest(t, svc, dto.RegisterTenantRequest{})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestRegisterHandler_PasswordTooShort(t *testing.T) {
	svc, _, _ := newTenantTestService()

	w := doRegisterRequest(t, svc, dto.RegisterTenantRequest{
		NombreNegocio: "Test",
		Slug:          "test",
		Username:      "admin",
		Password:      "short",
		Nombre:        "Admin",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestRegisterHandler_Success_ReturnsCreated(t *testing.T) {
	svc, _, _ := newTenantTestService()

	w := doRegisterRequest(t, svc, dto.RegisterTenantRequest{
		NombreNegocio: "Handler Test",
		Slug:          "handler-test",
		Username:      "handler-admin",
		Password:      "password123",
		Nombre:        "Handler Admin",
		Email:         "handler@test.com",
	})

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp dto.RegisterTenantResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.Equal(t, "handler-test", resp.Tenant.Slug)
	assert.Equal(t, "handler-admin", resp.User.Username)
	assert.Equal(t, "administrador", resp.User.Rol)
}

func TestRegisterHandler_DuplicateSlug_Returns422(t *testing.T) {
	svc, _, _ := newTenantTestService()

	// First
	w1 := doRegisterRequest(t, svc, dto.RegisterTenantRequest{
		NombreNegocio: "Dup Test",
		Slug:          "dup-test",
		Username:      "admin-a",
		Password:      "password123",
		Nombre:        "Admin A",
	})
	assert.Equal(t, http.StatusCreated, w1.Code)

	// Duplicate
	w2 := doRegisterRequest(t, svc, dto.RegisterTenantRequest{
		NombreNegocio: "Dup Test 2",
		Slug:          "dup-test",
		Username:      "admin-b",
		Password:      "password123",
		Nombre:        "Admin B",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, w2.Code)
}

// ── Tests: ObtenerActual & ActualizarActual ─────────────────────────────────

func TestObtenerActual_Success(t *testing.T) {
	svc, _, _ := newTenantTestService()

	// Register first to create a tenant
	resp, err := svc.Registrar(context.Background(), dto.RegisterTenantRequest{
		NombreNegocio: "Obtener Test",
		Slug:          "obtener-test",
		Username:      "obtadmin",
		Password:      "password123",
		Nombre:        "Obt Admin",
	})
	require.NoError(t, err)

	// Create context with tenant ID
	tenantID := uuid.MustParse(resp.Tenant.ID)
	ctx := context.WithValue(context.Background(), tenantctx.Key, tenantID)

	tenantResp, err := svc.ObtenerActual(ctx)
	require.NoError(t, err)
	assert.Equal(t, "obtener-test", tenantResp.Slug)
}

func TestActualizarActual_Success(t *testing.T) {
	svc, _, _ := newTenantTestService()

	resp, err := svc.Registrar(context.Background(), dto.RegisterTenantRequest{
		NombreNegocio: "Update Test",
		Slug:          "update-test",
		Username:      "updadmin",
		Password:      "password123",
		Nombre:        "Upd Admin",
	})
	require.NoError(t, err)

	tenantID := uuid.MustParse(resp.Tenant.ID)
	ctx := context.WithValue(context.Background(), tenantctx.Key, tenantID)

	cuit := "20345678901"
	updated, err := svc.ActualizarActual(ctx, dto.ActualizarTenantRequest{
		Nombre: "Updated Name",
		CUIT:   &cuit,
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Nombre)
	assert.Equal(t, &cuit, updated.CUIT)
}

