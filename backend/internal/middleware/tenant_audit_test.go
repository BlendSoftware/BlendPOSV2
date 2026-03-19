package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"

	"blendpos/internal/tenantctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newAuditSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	db.Exec("CREATE TABLE items (id TEXT, tenant_id TEXT, name TEXT)")
	return db
}

type auditItem struct {
	ID       string `gorm:"column:id"`
	TenantID string `gorm:"column:tenant_id"`
	Name     string `gorm:"column:name"`
}

// setupAuditRouter creates a Gin engine with JWT claims injection,
// tenant scoping simulation, and the TenantAuditMiddleware.
func setupAuditRouter(db *gorm.DB, tenantID uuid.UUID, handler gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	// Inject JWT claims
	r.Use(func(c *gin.Context) {
		c.Set(ClaimsKey, &JWTClaims{
			TenantID: tenantID.String(),
			UserID:   uuid.New().String(),
			Rol:      "administrador",
		})
		c.Next()
	})
	// Simulate TenantMiddleware (tenant in context)
	r.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), tenantctx.Key, tenantID)
		scopedDB := db.WithContext(ctx).Where("tenant_id = ?", tenantID)
		ctx = context.WithValue(ctx, tenantctx.ScopedDBKey, scopedDB)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	// Audit middleware under test
	r.Use(TenantAuditMiddleware())
	r.GET("/test", handler)
	return r
}

// ---------------------------------------------------------------------------
// Tests: GORM callback — checkDestTenantID
// ---------------------------------------------------------------------------

func TestCheckDestTenantID_NoViolation_SameTenant(t *testing.T) {
	tid := uuid.New()
	items := []auditItem{
		{ID: "1", TenantID: tid.String(), Name: "ok1"},
		{ID: "2", TenantID: tid.String(), Name: "ok2"},
	}
	assert.Equal(t, 0, checkDestTenantID(&items, tid))
}

func TestCheckDestTenantID_DetectsViolation_DifferentTenant(t *testing.T) {
	tidA := uuid.New()
	tidB := uuid.New()
	items := []auditItem{
		{ID: "1", TenantID: tidA.String(), Name: "ok"},
		{ID: "2", TenantID: tidB.String(), Name: "VIOLATION"},
	}
	assert.Equal(t, 1, checkDestTenantID(&items, tidA))
}

func TestCheckDestTenantID_SingleStruct_NoViolation(t *testing.T) {
	tid := uuid.New()
	item := auditItem{ID: "1", TenantID: tid.String(), Name: "ok"}
	assert.Equal(t, 0, checkDestTenantID(&item, tid))
}

func TestCheckDestTenantID_SingleStruct_Violation(t *testing.T) {
	tidA := uuid.New()
	tidB := uuid.New()
	item := auditItem{ID: "1", TenantID: tidB.String(), Name: "bad"}
	assert.Equal(t, 1, checkDestTenantID(&item, tidA))
}

func TestCheckDestTenantID_NoTenantIDField_Skips(t *testing.T) {
	type noTenantModel struct {
		ID   string
		Name string
	}
	item := noTenantModel{ID: "1", Name: "ok"}
	assert.Equal(t, 0, checkDestTenantID(&item, uuid.New()))
}

func TestCheckDestTenantID_EmptySlice_NoViolation(t *testing.T) {
	var items []auditItem
	assert.Equal(t, 0, checkDestTenantID(&items, uuid.New()))
}

func TestCheckDestTenantID_NilDest_NoViolation(t *testing.T) {
	assert.Equal(t, 0, checkDestTenantID(nil, uuid.New()))
}

func TestCheckDestTenantID_ZeroTenantID_Skips(t *testing.T) {
	tid := uuid.New()
	// Empty TenantID (zero value) should not flag as violation
	item := auditItem{ID: "1", TenantID: "", Name: "new"}
	assert.Equal(t, 0, checkDestTenantID(&item, tid))
}

// Test with uuid.UUID field type (not string)
func TestCheckDestTenantID_UUIDFieldType(t *testing.T) {
	type uuidModel struct {
		ID       string
		TenantID uuid.UUID
	}

	tidA := uuid.New()
	tidB := uuid.New()

	// Same tenant — no violation
	assert.Equal(t, 0, checkDestTenantID(&uuidModel{ID: "1", TenantID: tidA}, tidA))
	// Different tenant — violation
	assert.Equal(t, 1, checkDestTenantID(&uuidModel{ID: "1", TenantID: tidB}, tidA))
	// Zero UUID — skip
	assert.Equal(t, 0, checkDestTenantID(&uuidModel{ID: "1", TenantID: uuid.Nil}, tidA))
}

// ---------------------------------------------------------------------------
// Tests: mismatch function
// ---------------------------------------------------------------------------

func TestMismatch_NoTenantIDField(t *testing.T) {
	type plain struct {
		ID string
	}
	v := makeReflectValue(plain{ID: "1"})
	assert.False(t, mismatch(v, uuid.New()))
}

func TestMismatch_UnsupportedFieldType(t *testing.T) {
	type weird struct {
		TenantID int
	}
	v := makeReflectValue(weird{TenantID: 42})
	assert.False(t, mismatch(v, uuid.New()))
}

// ---------------------------------------------------------------------------
// Tests: TenantAuditMiddleware HTTP behavior
// ---------------------------------------------------------------------------

func TestTenantAuditMiddleware_NormalRequest_NoViolations(t *testing.T) {
	db := newAuditSQLiteDB(t)
	RegisterTenantAuditCallback(db)

	tenantID := uuid.New()
	db.Exec("INSERT INTO items (id, tenant_id, name) VALUES (?, ?, ?)", "1", tenantID.String(), "mine")

	var capturedViolations int64
	r := setupAuditRouter(db, tenantID, func(c *gin.Context) {
		scopedDB, _ := tenantctx.ScopedDBFromContext(c.Request.Context())
		var items []auditItem
		scopedDB.Table("items").Find(&items)
		capturedViolations = ViolationCount(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"count": len(items)})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, int64(0), capturedViolations)
}

func TestTenantAuditMiddleware_WithoutTenantContext_NoAudit(t *testing.T) {
	// Public endpoint without tenant context — middleware should not panic.
	r := gin.New()
	r.Use(TenantAuditMiddleware())
	r.GET("/public", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/public", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantAuditMiddleware_LogsTenantInfo(t *testing.T) {
	// Verifies that the middleware doesn't panic and completes with tenant info.
	db := newAuditSQLiteDB(t)
	tenantID := uuid.New()

	r := setupAuditRouter(db, tenantID, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestViolationCount_ResetsBetweenRequests(t *testing.T) {
	_ = newAuditSQLiteDB(t)
	_ = uuid.New()

	// Two requests should have independent violation counters.
	var count1, count2 int64

	r := gin.New()
	r.Use(TenantAuditMiddleware())
	requestNum := 0
	r.GET("/test", func(c *gin.Context) {
		requestNum++
		if requestNum == 1 {
			count1 = ViolationCount(c.Request.Context())
		} else {
			count2 = ViolationCount(c.Request.Context())
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Request 1
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w1, req1)

	// Request 2
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w2, req2)

	assert.Equal(t, int64(0), count1)
	assert.Equal(t, int64(0), count2)
}

func TestViolationCount_NoContext_ReturnsZero(t *testing.T) {
	assert.Equal(t, int64(0), ViolationCount(context.Background()))
}

// ---------------------------------------------------------------------------
// Tests: GORM callback integration with violation counter
// ---------------------------------------------------------------------------

func TestGORMCallback_IncrementsViolationCounter(t *testing.T) {
	db := newAuditSQLiteDB(t)
	RegisterTenantAuditCallback(db)

	tenantA := uuid.New()
	tenantB := uuid.New()

	db.Exec("INSERT INTO items (id, tenant_id, name) VALUES (?, ?, ?)", "1", tenantA.String(), "A's item")
	db.Exec("INSERT INTO items (id, tenant_id, name) VALUES (?, ?, ?)", "2", tenantB.String(), "B's item")

	// Create a context with tenant A and a violation counter.
	var counter atomic.Int64
	ctx := context.WithValue(context.Background(), tenantctx.Key, tenantA)
	ctx = context.WithValue(ctx, violationCtxKey, &counter)

	// Query ALL items (unscoped) — tenant B's item should trigger a violation.
	var items []auditItem
	db.WithContext(ctx).Table("items").Find(&items)

	assert.Equal(t, 2, len(items), "should return all items (unscoped query)")
	assert.Equal(t, int64(1), counter.Load(), "should detect 1 violation (tenant B's row)")
}

func TestGORMCallback_NoViolation_WhenAllRowsMatchTenant(t *testing.T) {
	db := newAuditSQLiteDB(t)
	RegisterTenantAuditCallback(db)

	tenantA := uuid.New()
	db.Exec("INSERT INTO items (id, tenant_id, name) VALUES (?, ?, ?)", "1", tenantA.String(), "ok1")
	db.Exec("INSERT INTO items (id, tenant_id, name) VALUES (?, ?, ?)", "2", tenantA.String(), "ok2")

	var counter atomic.Int64
	ctx := context.WithValue(context.Background(), tenantctx.Key, tenantA)
	ctx = context.WithValue(ctx, violationCtxKey, &counter)

	var items []auditItem
	db.WithContext(ctx).Table("items").Find(&items)

	assert.Equal(t, 2, len(items))
	assert.Equal(t, int64(0), counter.Load())
}

func TestGORMCallback_SkipsPublicQueries(t *testing.T) {
	db := newAuditSQLiteDB(t)
	RegisterTenantAuditCallback(db)

	db.Exec("INSERT INTO items (id, tenant_id, name) VALUES (?, ?, ?)", "1", uuid.New().String(), "any")

	// No tenant in context — should not panic or flag anything.
	var items []auditItem
	db.WithContext(context.Background()).Table("items").Find(&items)

	assert.Equal(t, 1, len(items))
}

func TestGORMCallback_MultipleViolationsInSlice(t *testing.T) {
	db := newAuditSQLiteDB(t)
	RegisterTenantAuditCallback(db)

	tenantA := uuid.New()
	tenantB := uuid.New()
	tenantC := uuid.New()

	db.Exec("INSERT INTO items (id, tenant_id, name) VALUES (?, ?, ?)", "1", tenantA.String(), "ok")
	db.Exec("INSERT INTO items (id, tenant_id, name) VALUES (?, ?, ?)", "2", tenantB.String(), "bad1")
	db.Exec("INSERT INTO items (id, tenant_id, name) VALUES (?, ?, ?)", "3", tenantC.String(), "bad2")

	var counter atomic.Int64
	ctx := context.WithValue(context.Background(), tenantctx.Key, tenantA)
	ctx = context.WithValue(ctx, violationCtxKey, &counter)

	var items []auditItem
	db.WithContext(ctx).Table("items").Find(&items)

	assert.Equal(t, 3, len(items))
	assert.Equal(t, int64(2), counter.Load(), "should detect 2 violations")
}

// ---------------------------------------------------------------------------
// reflect helper
// ---------------------------------------------------------------------------

func makeReflectValue(v interface{}) reflect.Value {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	return rv
}
