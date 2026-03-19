package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"blendpos/internal/dto"
	"blendpos/internal/handler"
	"blendpos/internal/middleware"
	"blendpos/internal/model"
	"blendpos/internal/repository"
	"blendpos/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Stub Tenant Repo with Superadmin Methods ─────────────────────────────────

type stubSuperadminTenantRepo struct {
	stubTenantRepo
	// Per-tenant metric overrides for testing
	ventasCounts    map[string]int64
	productosCounts map[string]int64
	usuariosCounts  map[string]int64
	ultimaVenta     map[string]*time.Time
}

func newStubSuperadminTenantRepo() *stubSuperadminTenantRepo {
	base := newStubTenantRepo()
	return &stubSuperadminTenantRepo{
		stubTenantRepo:  *base,
		ventasCounts:    make(map[string]int64),
		productosCounts: make(map[string]int64),
		usuariosCounts:  make(map[string]int64),
		ultimaVenta:     make(map[string]*time.Time),
	}
}

func (r *stubSuperadminTenantRepo) CountVentasByTenant(_ context.Context, tenantID uuid.UUID) (int64, error) {
	return r.ventasCounts[tenantID.String()], nil
}

func (r *stubSuperadminTenantRepo) CountProductosByTenant(_ context.Context, tenantID uuid.UUID) (int64, error) {
	return r.productosCounts[tenantID.String()], nil
}

func (r *stubSuperadminTenantRepo) CountUsuariosByTenant(_ context.Context, tenantID uuid.UUID) (int64, error) {
	return r.usuariosCounts[tenantID.String()], nil
}

func (r *stubSuperadminTenantRepo) ListAllPaginated(_ context.Context, f repository.TenantListFilter) ([]repository.TenantWithMetrics, int64, error) {
	// Collect all tenants
	all := make([]model.Tenant, 0)
	for _, t := range r.tenants {
		all = append(all, *t)
	}

	// Apply filters
	filtered := make([]model.Tenant, 0)
	for _, t := range all {
		// Search filter
		if f.Search != "" {
			search := strings.ToLower(f.Search)
			if !strings.Contains(strings.ToLower(t.Nombre), search) &&
				!strings.Contains(strings.ToLower(t.Slug), search) {
				continue
			}
		}
		// Status filter
		switch f.Status {
		case "active":
			if !t.Activo {
				continue
			}
		case "inactive":
			if t.Activo {
				continue
			}
		}
		// Plan filter
		if f.PlanID != "" {
			if t.PlanID == nil || t.PlanID.String() != f.PlanID {
				continue
			}
		}
		filtered = append(filtered, t)
	}

	total := int64(len(filtered))

	// Paginate
	start := (f.Page - 1) * f.PageSize
	end := start + f.PageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}
	page := filtered[start:end]

	result := make([]repository.TenantWithMetrics, len(page))
	for i, t := range page {
		tc := t
		if tc.PlanID != nil {
			if p, ok := r.plans[tc.PlanID.String()]; ok {
				tc.Plan = p
			}
		}
		result[i] = repository.TenantWithMetrics{
			Tenant:         tc,
			TotalVentas:    r.ventasCounts[tc.ID.String()],
			TotalProductos: r.productosCounts[tc.ID.String()],
			TotalUsuarios:  r.usuariosCounts[tc.ID.String()],
			UltimaVenta:    r.ultimaVenta[tc.ID.String()],
		}
	}

	return result, total, nil
}

func (r *stubSuperadminTenantRepo) FindTenantWithMetrics(_ context.Context, id uuid.UUID) (*repository.TenantWithMetrics, error) {
	for _, t := range r.tenants {
		if t.ID == id {
			if t.PlanID != nil {
				if p, ok := r.plans[t.PlanID.String()]; ok {
					t.Plan = p
				}
			}
			return &repository.TenantWithMetrics{
				Tenant:         *t,
				TotalVentas:    r.ventasCounts[t.ID.String()],
				TotalProductos: r.productosCounts[t.ID.String()],
				TotalUsuarios:  r.usuariosCounts[t.ID.String()],
				UltimaVenta:    r.ultimaVenta[t.ID.String()],
			}, nil
		}
	}
	return nil, fmt.Errorf("tenant not found")
}

func (r *stubSuperadminTenantRepo) GetGlobalMetrics(_ context.Context) (*repository.GlobalMetrics, error) {
	var total, activos, totalVentas int64
	planCounts := make(map[string]int64)

	for _, t := range r.tenants {
		total++
		if t.Activo {
			activos++
		}
		totalVentas += r.ventasCounts[t.ID.String()]
		planName := "Sin plan"
		if t.PlanID != nil {
			if p, ok := r.plans[t.PlanID.String()]; ok {
				planName = p.Nombre
			}
		}
		planCounts[planName]++
	}

	pcs := make([]repository.PlanCount, 0, len(planCounts))
	for name, count := range planCounts {
		pcs = append(pcs, repository.PlanCount{PlanNombre: name, Count: count})
	}

	return &repository.GlobalMetrics{
		TotalTenants:    total,
		TenantActivos:   activos,
		TotalVentas:     totalVentas,
		VentasUltimoMes: totalVentas, // simplified for testing
		TenantsPorPlan:  pcs,
	}, nil
}

// ── Test Helpers ─────────────────────────────────────────────────────────────

func seedTenant(repo *stubSuperadminTenantRepo, nombre, slug string, activo bool, planID *uuid.UUID) *model.Tenant {
	t := &model.Tenant{
		ID:        uuid.New(),
		Slug:      slug,
		Nombre:    nombre,
		Activo:    activo,
		PlanID:    planID,
		CreatedAt: time.Now(),
	}
	repo.tenants[slug] = t
	return t
}

func addProPlan(repo *stubSuperadminTenantRepo) *model.Plan {
	proID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	pro := &model.Plan{
		ID:            proID,
		Nombre:        "Pro",
		MaxTerminales: 5,
		MaxProductos:  0, // unlimited
		PrecioMensual: decimal.NewFromInt(2999),
		Activo:        true,
	}
	repo.plans[proID.String()] = pro
	return pro
}

func newSuperadminTestService() (service.TenantService, *stubSuperadminTenantRepo) {
	repo := newStubSuperadminTenantRepo()
	usuarioRepo := newStubRepo()
	cfg := newTenantTestCfg()
	svc := service.NewTenantService(repo, usuarioRepo, cfg, nil)
	return svc, repo
}

// superadminRouter creates a test Gin engine wired with superadmin endpoints + JWT + RequireSuperAdmin.
func superadminRouter(svc service.TenantService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewTenantsHandler(svc)

	// Protected with JWT + RequireSuperAdmin
	v1 := r.Group("/v1", middleware.JWTAuth(testSecret, nil))
	sa := v1.Group("/superadmin", middleware.RequireSuperAdmin())
	{
		sa.GET("/tenants", h.ListarTodos)
		sa.GET("/tenants/:id", h.ObtenerTenantDetalle)
		sa.PUT("/tenants/:id", h.ToggleActivo)
		sa.PUT("/tenants/:id/plan", h.CambiarPlan)
		sa.GET("/metrics", h.ObtenerMetricas)
	}
	return r
}

func doSuperadminRequest(router *gin.Engine, method, path, token string, body ...string) *httptest.ResponseRecorder {
	var req *http.Request
	if len(body) > 0 {
		req, _ = http.NewRequest(method, path, strings.NewReader(body[0]))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, path, nil)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ── Tests: Listar tenants con paginación ─────────────────────────────────────

func TestSuperadmin_ListarTenants_Paginated(t *testing.T) {
	svc, repo := newSuperadminTestService()
	starterID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	// Seed 5 tenants
	for i := 0; i < 5; i++ {
		seedTenant(repo, fmt.Sprintf("Kiosco %d", i), fmt.Sprintf("kiosco-%d", i), true, &starterID)
	}

	req := dto.TenantListRequest{Page: 1, PageSize: 2}
	resp, err := svc.ListarTodos(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, int64(5), resp.Total)
	assert.Equal(t, 2, len(resp.Tenants))
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 2, resp.PageSize)
	assert.Equal(t, 3, resp.TotalPages) // ceil(5/2) = 3
}

func TestSuperadmin_ListarTenants_Page2(t *testing.T) {
	svc, repo := newSuperadminTestService()
	starterID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	for i := 0; i < 5; i++ {
		seedTenant(repo, fmt.Sprintf("Kiosco %d", i), fmt.Sprintf("kiosco-%d", i), true, &starterID)
	}

	req := dto.TenantListRequest{Page: 3, PageSize: 2}
	resp, err := svc.ListarTodos(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, 1, len(resp.Tenants)) // last page has 1 item
}

// ── Tests: Filtrar tenants por status ────────────────────────────────────────

func TestSuperadmin_ListarTenants_FilterActive(t *testing.T) {
	svc, repo := newSuperadminTestService()
	starterID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	seedTenant(repo, "Activo 1", "activo-1", true, &starterID)
	seedTenant(repo, "Activo 2", "activo-2", true, &starterID)
	seedTenant(repo, "Inactivo 1", "inactivo-1", false, &starterID)

	req := dto.TenantListRequest{Status: "active"}
	resp, err := svc.ListarTodos(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	for _, item := range resp.Tenants {
		assert.True(t, item.Activo)
	}
}

func TestSuperadmin_ListarTenants_FilterInactive(t *testing.T) {
	svc, repo := newSuperadminTestService()
	starterID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	seedTenant(repo, "Activo 1", "activo-1", true, &starterID)
	seedTenant(repo, "Inactivo 1", "inactivo-1", false, &starterID)

	req := dto.TenantListRequest{Status: "inactive"}
	resp, err := svc.ListarTodos(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Total)
	assert.False(t, resp.Tenants[0].Activo)
}

// ── Tests: Buscar tenants por nombre/slug ────────────────────────────────────

func TestSuperadmin_ListarTenants_SearchByNombre(t *testing.T) {
	svc, repo := newSuperadminTestService()
	starterID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	seedTenant(repo, "Kiosco del Centro", "kiosco-centro", true, &starterID)
	seedTenant(repo, "Almacen Norte", "almacen-norte", true, &starterID)
	seedTenant(repo, "Mi Kiosco", "mi-kiosco", true, &starterID)

	req := dto.TenantListRequest{Search: "kiosco"}
	resp, err := svc.ListarTodos(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
}

func TestSuperadmin_ListarTenants_SearchBySlug(t *testing.T) {
	svc, repo := newSuperadminTestService()
	starterID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	seedTenant(repo, "Test A", "almacen-norte", true, &starterID)
	seedTenant(repo, "Test B", "kiosco-sur", true, &starterID)

	req := dto.TenantListRequest{Search: "almacen"}
	resp, err := svc.ListarTodos(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Total)
	assert.Equal(t, "almacen-norte", resp.Tenants[0].Slug)
}

// ── Tests: Cambiar plan de tenant ────────────────────────────────────────────

func TestSuperadmin_CambiarPlan(t *testing.T) {
	svc, repo := newSuperadminTestService()
	starterID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	pro := addProPlan(repo)

	tenant := seedTenant(repo, "Plan Test", "plan-test", true, &starterID)

	resp, err := svc.CambiarPlan(context.Background(), tenant.ID, pro.ID)

	require.NoError(t, err)
	require.NotNil(t, resp.Plan)
	assert.Equal(t, "Pro", resp.Plan.Nombre)
}

func TestSuperadmin_CambiarPlan_TenantNotFound(t *testing.T) {
	svc, _ := newSuperadminTestService()

	_, err := svc.CambiarPlan(context.Background(), uuid.New(), uuid.New())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no encontrado")
}

// ── Tests: Activar/desactivar tenant ─────────────────────────────────────────

func TestSuperadmin_ToggleActivo_Desactivar(t *testing.T) {
	svc, repo := newSuperadminTestService()
	starterID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	tenant := seedTenant(repo, "Toggle Test", "toggle-test", true, &starterID)

	resp, err := svc.ToggleActivo(context.Background(), tenant.ID, false)

	require.NoError(t, err)
	assert.False(t, resp.Activo)
}

func TestSuperadmin_ToggleActivo_Activar(t *testing.T) {
	svc, repo := newSuperadminTestService()
	starterID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	tenant := seedTenant(repo, "Toggle Test 2", "toggle-test-2", false, &starterID)

	resp, err := svc.ToggleActivo(context.Background(), tenant.ID, true)

	require.NoError(t, err)
	assert.True(t, resp.Activo)
}

// ── Tests: Métricas globales ─────────────────────────────────────────────────

func TestSuperadmin_MetricasGlobales(t *testing.T) {
	svc, repo := newSuperadminTestService()
	starterID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	t1 := seedTenant(repo, "Active 1", "active-1", true, &starterID)
	t2 := seedTenant(repo, "Active 2", "active-2", true, &starterID)
	seedTenant(repo, "Inactive", "inactive", false, &starterID)

	repo.ventasCounts[t1.ID.String()] = 100
	repo.ventasCounts[t2.ID.String()] = 50

	resp, err := svc.ObtenerMetricas(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(3), resp.TotalTenants)
	assert.Equal(t, int64(2), resp.TenantActivos)
	assert.Equal(t, int64(150), resp.TotalVentas)
	assert.NotEmpty(t, resp.TenantsPorPlan)

	// All tenants are on Starter plan
	found := false
	for _, pc := range resp.TenantsPorPlan {
		if pc.PlanNombre == "Starter" {
			assert.Equal(t, int64(3), pc.Count)
			found = true
		}
	}
	assert.True(t, found, "should have Starter plan count")
}

// ── Tests: Superadmin ve datos de todos los tenants (RLS bypass) ─────────────

func TestSuperadmin_ListarTodos_AllTenants(t *testing.T) {
	svc, repo := newSuperadminTestService()
	starterID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	t1 := seedTenant(repo, "Tenant A", "tenant-a", true, &starterID)
	t2 := seedTenant(repo, "Tenant B", "tenant-b", true, &starterID)

	repo.ventasCounts[t1.ID.String()] = 42
	repo.productosCounts[t1.ID.String()] = 10
	repo.ventasCounts[t2.ID.String()] = 88
	repo.productosCounts[t2.ID.String()] = 20

	resp, err := svc.ListarTodos(context.Background(), dto.TenantListRequest{})

	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)

	// Verify metrics are populated for each tenant
	for _, item := range resp.Tenants {
		if item.Slug == "tenant-a" {
			assert.Equal(t, int64(42), item.TotalVentas)
			assert.Equal(t, int64(10), item.TotalProductos)
		} else if item.Slug == "tenant-b" {
			assert.Equal(t, int64(88), item.TotalVentas)
			assert.Equal(t, int64(20), item.TotalProductos)
		}
	}
}

// ── Tests: Non-superadmin recibe 403 (handler level) ─────────────────────────

func TestSuperadmin_NonSuperadmin_Returns403(t *testing.T) {
	svc, _ := newSuperadminTestService()
	router := superadminRouter(svc)

	// Sign a token with "administrador" role (not superadmin)
	token := signToken(t, uuid.New().String(), "administrador", time.Hour)

	w := doSuperadminRequest(router, http.MethodGet, "/v1/superadmin/tenants", token)
	assert.Equal(t, http.StatusForbidden, w.Code)

	w = doSuperadminRequest(router, http.MethodGet, "/v1/superadmin/metrics", token)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSuperadmin_Superadmin_Returns200(t *testing.T) {
	svc, _ := newSuperadminTestService()
	router := superadminRouter(svc)

	// Sign a token with "superadmin" role
	token := signToken(t, uuid.New().String(), "superadmin", time.Hour)

	w := doSuperadminRequest(router, http.MethodGet, "/v1/superadmin/tenants", token)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.TenantListResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(0), resp.Total)
}

// ── Tests: Handler-level pagination params ───────────────────────────────────

func TestSuperadmin_Handler_PaginationParams(t *testing.T) {
	svc, repo := newSuperadminTestService()
	starterID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	for i := 0; i < 5; i++ {
		seedTenant(repo, fmt.Sprintf("Kiosco %d", i), fmt.Sprintf("kiosco-%d", i), true, &starterID)
	}

	router := superadminRouter(svc)
	token := signToken(t, uuid.New().String(), "superadmin", time.Hour)

	w := doSuperadminRequest(router, http.MethodGet, "/v1/superadmin/tenants?page=1&page_size=2&status=active", token)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.TenantListResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(5), resp.Total)
	assert.Equal(t, 2, len(resp.Tenants))
	assert.Equal(t, 3, resp.TotalPages)
}

// ── Tests: Tenant detail endpoint ────────────────────────────────────────────

func TestSuperadmin_TenantDetalle(t *testing.T) {
	svc, repo := newSuperadminTestService()
	starterID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	tenant := seedTenant(repo, "Detail Test", "detail-test", true, &starterID)
	repo.ventasCounts[tenant.ID.String()] = 77
	repo.productosCounts[tenant.ID.String()] = 33

	router := superadminRouter(svc)
	token := signToken(t, uuid.New().String(), "superadmin", time.Hour)

	w := doSuperadminRequest(router, http.MethodGet, "/v1/superadmin/tenants/"+tenant.ID.String(), token)
	assert.Equal(t, http.StatusOK, w.Code)

	var item dto.SuperadminTenantListItem
	err := json.Unmarshal(w.Body.Bytes(), &item)
	require.NoError(t, err)
	assert.Equal(t, "Detail Test", item.Nombre)
	assert.Equal(t, int64(77), item.TotalVentas)
	assert.Equal(t, int64(33), item.TotalProductos)
}

func TestSuperadmin_TenantDetalle_NotFound(t *testing.T) {
	svc, _ := newSuperadminTestService()
	router := superadminRouter(svc)
	token := signToken(t, uuid.New().String(), "superadmin", time.Hour)

	w := doSuperadminRequest(router, http.MethodGet, "/v1/superadmin/tenants/"+uuid.New().String(), token)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── Tests: Cambiar plan via handler ──────────────────────────────────────────

func TestSuperadmin_Handler_CambiarPlan(t *testing.T) {
	svc, repo := newSuperadminTestService()
	starterID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	pro := addProPlan(repo)
	tenant := seedTenant(repo, "Plan Handler", "plan-handler", true, &starterID)

	router := superadminRouter(svc)
	token := signToken(t, uuid.New().String(), "superadmin", time.Hour)

	body := fmt.Sprintf(`{"plan_id":"%s"}`, pro.ID.String())
	w := doSuperadminRequest(router, http.MethodPut, "/v1/superadmin/tenants/"+tenant.ID.String()+"/plan", token, body)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── Tests: Toggle activo via handler ─────────────────────────────────────────

func TestSuperadmin_Handler_ToggleActivo(t *testing.T) {
	svc, repo := newSuperadminTestService()
	starterID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	tenant := seedTenant(repo, "Toggle Handler", "toggle-handler", true, &starterID)

	router := superadminRouter(svc)
	token := signToken(t, uuid.New().String(), "superadmin", time.Hour)

	w := doSuperadminRequest(router, http.MethodPut, "/v1/superadmin/tenants/"+tenant.ID.String(), token, `{"activo":false}`)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.TenantResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Activo)
}

// ── Tests: Metrics via handler ───────────────────────────────────────────────

func TestSuperadmin_Handler_Metrics(t *testing.T) {
	svc, repo := newSuperadminTestService()
	starterID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	t1 := seedTenant(repo, "M1", "m1", true, &starterID)
	repo.ventasCounts[t1.ID.String()] = 200

	router := superadminRouter(svc)
	token := signToken(t, uuid.New().String(), "superadmin", time.Hour)

	w := doSuperadminRequest(router, http.MethodGet, "/v1/superadmin/metrics", token)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.SuperadminMetricsResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.TotalTenants)
	assert.Equal(t, int64(1), resp.TenantActivos)
	assert.Equal(t, int64(200), resp.TotalVentas)
	assert.NotNil(t, resp.TenantsPorPlan)
}

// ── Tests: Default pagination ────────────────────────────────────────────────

func TestSuperadmin_ListarTenants_DefaultPagination(t *testing.T) {
	svc, _ := newSuperadminTestService()

	// No params — should use defaults (page=1, page_size=20)
	resp, err := svc.ListarTodos(context.Background(), dto.TenantListRequest{})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 20, resp.PageSize)
}
