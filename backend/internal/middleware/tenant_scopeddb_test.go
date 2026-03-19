package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"blendpos/internal/tenantctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// Create a dummy table to test scoping.
	db.Exec("CREATE TABLE items (id TEXT, tenant_id TEXT, name TEXT)")
	return db
}

// simulateTenantMiddlewareScoping reproduces the scoped DB injection logic
// from TenantMiddleware without requiring PostgreSQL set_config().
// This lets us test the context injection + DB scoping in isolation.
func simulateTenantMiddlewareScoping(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := c.MustGet(ClaimsKey).(*JWTClaims)
		if !ok || claims.TenantID == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		tenantID, err := uuid.Parse(claims.TenantID)
		if err != nil || tenantID == uuid.Nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Same logic as TenantMiddleware:
		ctx := context.WithValue(c.Request.Context(), tenantctx.Key, tenantID)

		// The scoped DB injection (what F2-1 adds):
		scopedDB := db.WithContext(ctx).Where("tenant_id = ?", tenantID)
		ctx = context.WithValue(ctx, tenantctx.ScopedDBKey, scopedDB)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// ---------------------------------------------------------------------------
// Tests: Scoped DB injection via TenantMiddleware
// ---------------------------------------------------------------------------

func TestTenantMiddleware_InjectsScopedDB(t *testing.T) {
	db := newSQLiteDB(t)
	tenantID := uuid.New()

	var capturedDB *gorm.DB
	var capturedErr error

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ClaimsKey, &JWTClaims{TenantID: tenantID.String(), Rol: "administrador"})
		c.Next()
	})
	r.Use(simulateTenantMiddlewareScoping(db))
	r.GET("/test", func(c *gin.Context) {
		capturedDB, capturedErr = tenantctx.ScopedDBFromContext(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, capturedErr)
	assert.NotNil(t, capturedDB)
}

func TestTenantMiddleware_ScopedDB_FiltersCorrectTenant(t *testing.T) {
	db := newSQLiteDB(t)

	tenantA := uuid.New()
	tenantB := uuid.New()

	// Insert rows for two tenants.
	db.Exec("INSERT INTO items (id, tenant_id, name) VALUES (?, ?, ?)", "1", tenantA.String(), "Item A")
	db.Exec("INSERT INTO items (id, tenant_id, name) VALUES (?, ?, ?)", "2", tenantB.String(), "Item B")
	db.Exec("INSERT INTO items (id, tenant_id, name) VALUES (?, ?, ?)", "3", tenantA.String(), "Item A2")

	type Item struct {
		ID       string `gorm:"column:id"`
		TenantID string `gorm:"column:tenant_id"`
		Name     string `gorm:"column:name"`
	}

	var itemsA []Item

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ClaimsKey, &JWTClaims{TenantID: tenantA.String(), Rol: "administrador"})
		c.Next()
	})
	r.Use(simulateTenantMiddlewareScoping(db))
	r.GET("/test", func(c *gin.Context) {
		scopedDB, err := tenantctx.ScopedDBFromContext(c.Request.Context())
		require.NoError(t, err)

		// Query using the scoped DB — should only return tenant A's items.
		scopedDB.Table("items").Find(&itemsA)
		c.JSON(http.StatusOK, gin.H{"count": len(itemsA)})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, itemsA, 2, "should only return tenant A's items")
	for _, item := range itemsA {
		assert.Equal(t, tenantA.String(), item.TenantID)
	}
}

func TestTenantMiddleware_ScopedDB_IsolationBetweenRequests(t *testing.T) {
	db := newSQLiteDB(t)

	tenantA := uuid.New()
	tenantB := uuid.New()

	db.Exec("INSERT INTO items (id, tenant_id, name) VALUES (?, ?, ?)", "1", tenantA.String(), "A only")
	db.Exec("INSERT INTO items (id, tenant_id, name) VALUES (?, ?, ?)", "2", tenantB.String(), "B only")

	type Item struct {
		ID       string `gorm:"column:id"`
		TenantID string `gorm:"column:tenant_id"`
		Name     string `gorm:"column:name"`
	}

	// Test that two concurrent requests for different tenants get isolated results.
	for _, tc := range []struct {
		name     string
		tid      uuid.UUID
		expected string
	}{
		{"tenant_A_sees_A_only", tenantA, "A only"},
		{"tenant_B_sees_B_only", tenantB, "B only"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var items []Item

			r := gin.New()
			r.Use(func(c *gin.Context) {
				c.Set(ClaimsKey, &JWTClaims{TenantID: tc.tid.String(), Rol: "administrador"})
				c.Next()
			})
			r.Use(simulateTenantMiddlewareScoping(db))
			r.GET("/test", func(c *gin.Context) {
				scopedDB, err := tenantctx.ScopedDBFromContext(c.Request.Context())
				require.NoError(t, err)
				scopedDB.Table("items").Find(&items)
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			r.ServeHTTP(w, req)

			require.Len(t, items, 1)
			assert.Equal(t, tc.expected, items[0].Name)
			assert.Equal(t, tc.tid.String(), items[0].TenantID)
		})
	}
}

func TestScopedDBFromContext_ReturnsError_WithoutMiddleware(t *testing.T) {
	var capturedErr error

	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		_, capturedErr = tenantctx.ScopedDBFromContext(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Error(t, capturedErr)
	assert.Contains(t, capturedErr.Error(), "scoped DB not in context")
}

func TestMustScopedDB_Panics_WithoutMiddleware(t *testing.T) {
	assert.Panics(t, func() {
		tenantctx.MustScopedDB(context.Background())
	})
}
