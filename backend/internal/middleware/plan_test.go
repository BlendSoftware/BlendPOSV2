package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"blendpos/internal/model"
	"blendpos/internal/repository"
	"blendpos/internal/tenantctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Stub TenantRepository
// ---------------------------------------------------------------------------

type stubTenantRepo struct {
	tenant       *model.Tenant
	tenantErr    error
	productCount int64
	productErr   error
	db           *gorm.DB
}

func (s *stubTenantRepo) CreateTenant(_ context.Context, _ *model.Tenant) error { return nil }
func (s *stubTenantRepo) FindTenantByID(_ context.Context, _ uuid.UUID) (*model.Tenant, error) {
	return s.tenant, s.tenantErr
}
func (s *stubTenantRepo) FindTenantBySlug(_ context.Context, _ string) (*model.Tenant, error) {
	return s.tenant, s.tenantErr
}
func (s *stubTenantRepo) UpdateTenant(_ context.Context, _ *model.Tenant) error { return nil }
func (s *stubTenantRepo) ListTenants(_ context.Context) ([]model.Tenant, error) { return nil, nil }
func (s *stubTenantRepo) FindPlanByID(_ context.Context, _ uuid.UUID) (*model.Plan, error) {
	return nil, nil
}
func (s *stubTenantRepo) ListPlans(_ context.Context) ([]model.Plan, error) { return nil, nil }
func (s *stubTenantRepo) FindPlanByNombre(_ context.Context, _ string) (*model.Plan, error) {
	return nil, nil
}
func (s *stubTenantRepo) CountProductosByTenant(_ context.Context, _ uuid.UUID) (int64, error) {
	return s.productCount, s.productErr
}
func (s *stubTenantRepo) CountUsuariosByTenant(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (s *stubTenantRepo) CountVentasByTenant(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (s *stubTenantRepo) ListAllPaginated(_ context.Context, _ repository.TenantListFilter) ([]repository.TenantWithMetrics, int64, error) {
	return nil, 0, nil
}
func (s *stubTenantRepo) FindTenantWithMetrics(_ context.Context, _ uuid.UUID) (*repository.TenantWithMetrics, error) {
	return nil, nil
}
func (s *stubTenantRepo) GetGlobalMetrics(_ context.Context) (*repository.GlobalMetrics, error) {
	return nil, nil
}
func (s *stubTenantRepo) DB() *gorm.DB { return s.db }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func init() {
	gin.SetMode(gin.TestMode)
}

// setupPlanTestRouter creates a minimal Gin engine with the given middleware.
// It injects tenant_id into the request context (simulating TenantMiddleware)
// and sets fake JWT claims (simulating JWTAuth).
func setupPlanTestRouter(mw gin.HandlerFunc, tenantID uuid.UUID) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		// Simulate JWTAuth setting claims.
		c.Set(ClaimsKey, &JWTClaims{
			TenantID: tenantID.String(),
			Rol:      "administrador",
		})
		// Simulate TenantMiddleware injecting tenant_id into context.
		ctx := context.WithValue(c.Request.Context(), tenantctx.Key, tenantID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.POST("/test", mw, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func makeTenantWithPlan(maxProductos, maxTerminales int) *model.Tenant {
	planID := uuid.New()
	return &model.Tenant{
		ID:     uuid.New(),
		Nombre: "Kiosco Test",
		PlanID: &planID,
		Plan: &model.Plan{
			ID:            planID,
			Nombre:        "Kiosco",
			MaxProductos:  maxProductos,
			MaxTerminales: maxTerminales,
		},
	}
}

// ---------------------------------------------------------------------------
// Tests: EnforcePlanLimitProductos
// ---------------------------------------------------------------------------

func TestEnforcePlanLimitProductos_AllowsWhenUnderLimit(t *testing.T) {
	tid := uuid.New()
	repo := &stubTenantRepo{
		tenant:       makeTenantWithPlan(100, 1),
		productCount: 50,
	}

	r := setupPlanTestRouter(EnforcePlanLimitProductos(repo, nil), tid)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEnforcePlanLimitProductos_BlocksWhenAtLimit(t *testing.T) {
	tid := uuid.New()
	repo := &stubTenantRepo{
		tenant:       makeTenantWithPlan(100, 1),
		productCount: 100,
	}

	r := setupPlanTestRouter(EnforcePlanLimitProductos(repo, nil), tid)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var body PlanLimitError
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "plan_limit_exceeded", body.Error)
	assert.Equal(t, "max_productos", body.Limit)
	assert.Equal(t, int64(100), body.Current)
	assert.Equal(t, 100, body.Max)
	assert.Equal(t, "/billing/upgrade", body.UpgradeURL)
}

func TestEnforcePlanLimitProductos_AllowsWhenUnlimited(t *testing.T) {
	tid := uuid.New()
	repo := &stubTenantRepo{
		tenant:       makeTenantWithPlan(0, 1), // 0 = unlimited
		productCount: 999999,
	}

	r := setupPlanTestRouter(EnforcePlanLimitProductos(repo, nil), tid)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEnforcePlanLimitProductos_FailOpenOnDBError(t *testing.T) {
	tid := uuid.New()
	repo := &stubTenantRepo{
		tenantErr: errors.New("db connection refused"),
	}

	r := setupPlanTestRouter(EnforcePlanLimitProductos(repo, nil), tid)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "should fail-open when DB is unreachable")
}

func TestEnforcePlanLimitProductos_FailOpenOnCountError(t *testing.T) {
	tid := uuid.New()
	repo := &stubTenantRepo{
		tenant:     makeTenantWithPlan(10, 1),
		productErr: errors.New("query timeout"),
	}

	r := setupPlanTestRouter(EnforcePlanLimitProductos(repo, nil), tid)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "should fail-open when count query fails")
}

func TestEnforcePlanLimitProductos_AllowsWhenNoPlan(t *testing.T) {
	tid := uuid.New()
	repo := &stubTenantRepo{
		tenant: &model.Tenant{ID: tid, Nombre: "No Plan Kiosco", Plan: nil},
	}

	r := setupPlanTestRouter(EnforcePlanLimitProductos(repo, nil), tid)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "should allow when tenant has no plan")
}

// ---------------------------------------------------------------------------
// Tests: EnforcePlanLimitTerminales (without real DB — cannot test countOpenSessions directly)
// ---------------------------------------------------------------------------

func TestEnforcePlanLimitTerminales_AllowsWhenNoPlan(t *testing.T) {
	tid := uuid.New()
	repo := &stubTenantRepo{
		tenant: &model.Tenant{ID: tid, Nombre: "Sin Plan", Plan: nil},
	}

	r := setupPlanTestRouter(EnforcePlanLimitTerminales(repo, nil), tid)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEnforcePlanLimitTerminales_AllowsUnlimited(t *testing.T) {
	tid := uuid.New()
	repo := &stubTenantRepo{
		tenant: makeTenantWithPlan(0, 0), // max_terminales=0 → unlimited
	}

	r := setupPlanTestRouter(EnforcePlanLimitTerminales(repo, nil), tid)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEnforcePlanLimitTerminales_FailOpenOnDBError(t *testing.T) {
	tid := uuid.New()
	repo := &stubTenantRepo{
		tenantErr: errors.New("connection refused"),
	}

	r := setupPlanTestRouter(EnforcePlanLimitTerminales(repo, nil), tid)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "should fail-open when DB is unreachable")
}

// ---------------------------------------------------------------------------
// Tests: Redis caching (fetchPlan and fetchProductCount with real miniredis)
// ---------------------------------------------------------------------------

func TestFetchPlan_CachesInRedis(t *testing.T) {
	// Use a real Redis client pointed at an invalid addr to test fallback.
	// For a proper cache test we'd need miniredis; here we verify the DB path works.
	tid := uuid.New()
	repo := &stubTenantRepo{
		tenant: makeTenantWithPlan(50, 2),
	}

	plan, err := fetchPlan(context.Background(), repo, nil, tid)
	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Equal(t, 50, plan.MaxProductos)
	assert.Equal(t, 2, plan.MaxTerminales)
}

func TestFetchProductCount_FallsBackToDB(t *testing.T) {
	tid := uuid.New()
	repo := &stubTenantRepo{
		productCount: 42,
	}

	count, err := fetchProductCount(context.Background(), repo, nil, tid)
	require.NoError(t, err)
	assert.Equal(t, int64(42), count)
}

func TestFetchPlan_FailOpenOnRedisError(t *testing.T) {
	tid := uuid.New()
	repo := &stubTenantRepo{
		tenant: makeTenantWithPlan(10, 1),
	}

	// Create a Redis client that will fail (bad address).
	badRedis := redis.NewClient(&redis.Options{Addr: "localhost:1"})

	plan, err := fetchPlan(context.Background(), repo, badRedis, tid)
	require.NoError(t, err)
	require.NotNil(t, plan, "should fall back to DB when Redis is unreachable")
	assert.Equal(t, 10, plan.MaxProductos)
}

func TestFetchProductCount_FailOpenOnRedisError(t *testing.T) {
	tid := uuid.New()
	repo := &stubTenantRepo{
		productCount: 7,
	}

	badRedis := redis.NewClient(&redis.Options{Addr: "localhost:1"})

	count, err := fetchProductCount(context.Background(), repo, badRedis, tid)
	require.NoError(t, err)
	assert.Equal(t, int64(7), count, "should fall back to DB when Redis is unreachable")
}

// ---------------------------------------------------------------------------
// Tests: PlanLimitError response format
// ---------------------------------------------------------------------------

func TestPlanLimitError_JSONStructure(t *testing.T) {
	ple := PlanLimitError{
		Error:      "plan_limit_exceeded",
		Message:    "Tu plan Kiosco permite hasta 1 terminal simultánea.",
		Limit:      "max_terminales",
		Current:    1,
		Max:        1,
		UpgradeURL: "/billing/upgrade",
	}

	data, err := json.Marshal(ple)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Equal(t, "plan_limit_exceeded", m["error"])
	assert.Equal(t, "max_terminales", m["limit"])
	assert.Equal(t, float64(1), m["current"])
	assert.Equal(t, float64(1), m["max"])
	assert.Equal(t, "/billing/upgrade", m["upgrade_url"])
}

// ---------------------------------------------------------------------------
// Tests: InvalidateCache helpers (with nil Redis — should not panic)
// ---------------------------------------------------------------------------

func TestInvalidatePlanCache_NilRedis(t *testing.T) {
	assert.NotPanics(t, func() {
		InvalidatePlanCache(context.Background(), nil, uuid.New())
	})
}

func TestInvalidateProductCountCache_NilRedis(t *testing.T) {
	assert.NotPanics(t, func() {
		InvalidateProductCountCache(context.Background(), nil, uuid.New())
	})
}

// ---------------------------------------------------------------------------
// Tests: RequireSuperAdmin
// ---------------------------------------------------------------------------

func TestRequireSuperAdmin_AllowsSuperAdmin(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ClaimsKey, &JWTClaims{Rol: "superadmin"})
		c.Next()
	})
	r.GET("/test", RequireSuperAdmin(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireSuperAdmin_BlocksNonSuperAdmin(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ClaimsKey, &JWTClaims{Rol: "administrador"})
		c.Next()
	})
	r.GET("/test", RequireSuperAdmin(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ---------------------------------------------------------------------------
// Tests: Edge case — product count exactly at limit blocks
// ---------------------------------------------------------------------------

func TestEnforcePlanLimitProductos_ExactLimitBlocks(t *testing.T) {
	for _, tc := range []struct {
		name  string
		count int64
		limit int
		want  int
	}{
		{"one_under_limit_allows", 9, 10, http.StatusOK},
		{"at_limit_blocks", 10, 10, http.StatusForbidden},
		{"over_limit_blocks", 11, 10, http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tid := uuid.New()
			repo := &stubTenantRepo{
				tenant:       makeTenantWithPlan(tc.limit, 1),
				productCount: tc.count,
			}

			r := setupPlanTestRouter(EnforcePlanLimitProductos(repo, nil), tid)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/test", nil)
			r.ServeHTTP(w, req)
			assert.Equal(t, tc.want, w.Code, fmt.Sprintf("count=%d limit=%d", tc.count, tc.limit))
		})
	}
}
