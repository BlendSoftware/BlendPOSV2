package middleware

import (
	"bytes"
	"context"
	"encoding/json"
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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newIDORTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// Create tables that mirror the production schema (id + tenant_id).
	for _, ddl := range []string{
		"CREATE TABLE productos (id TEXT, tenant_id TEXT, nombre TEXT)",
		"CREATE TABLE categorias (id TEXT, tenant_id TEXT, nombre TEXT)",
		"CREATE TABLE ventas (id TEXT, tenant_id TEXT, total TEXT)",
	} {
		require.NoError(t, db.Exec(ddl).Error)
	}
	// Register test tables in the whitelist.
	RegisterAllowedTable("productos")
	RegisterAllowedTable("categorias")
	RegisterAllowedTable("ventas")
	return db
}

// setupIDORRouter creates a Gin engine with simulated JWT + tenant context,
// then applies the given middleware before a dummy handler.
func setupIDORRouter(mw gin.HandlerFunc, tenantID uuid.UUID) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ClaimsKey, &JWTClaims{TenantID: tenantID.String(), Rol: "administrador"})
		ctx := context.WithValue(c.Request.Context(), tenantctx.Key, tenantID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.GET("/test/:id", mw, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	r.DELETE("/test/:id", mw, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

// setupIDORRouterNoTenant creates a Gin engine WITHOUT tenant context.
func setupIDORRouterNoTenant(mw gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ClaimsKey, &JWTClaims{Rol: "administrador"})
		c.Next()
	})
	r.GET("/test/:id", mw, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

// setupBodyRouter creates a Gin engine for POST with ValidateBodyReferences.
func setupBodyRouter(mw gin.HandlerFunc, tenantID uuid.UUID) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ClaimsKey, &JWTClaims{TenantID: tenantID.String(), Rol: "administrador"})
		ctx := context.WithValue(c.Request.Context(), tenantctx.Key, tenantID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.POST("/test", mw, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

// ---------------------------------------------------------------------------
// Tests: ValidateResourceOwnership
// ---------------------------------------------------------------------------

func TestIDORGuard_OwnResource_Passes(t *testing.T) {
	db := newIDORTestDB(t)
	tenantA := uuid.New()
	resourceID := uuid.New()

	db.Exec("INSERT INTO productos (id, tenant_id, nombre) VALUES (?, ?, ?)",
		resourceID.String(), tenantA.String(), "Coca Cola")

	r := setupIDORRouter(ValidateResourceOwnership(db, "productos", "id"), tenantA)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test/"+resourceID.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIDORGuard_OtherTenantResource_Returns404(t *testing.T) {
	db := newIDORTestDB(t)
	tenantA := uuid.New()
	tenantB := uuid.New()
	resourceID := uuid.New()

	// Resource belongs to tenant B.
	db.Exec("INSERT INTO productos (id, tenant_id, nombre) VALUES (?, ?, ?)",
		resourceID.String(), tenantB.String(), "Pepsi")

	// Request as tenant A → should get 404.
	r := setupIDORRouter(ValidateResourceOwnership(db, "productos", "id"), tenantA)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test/"+resourceID.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "resource not found", body["detail"])
}

func TestIDORGuard_NonexistentResource_Returns404(t *testing.T) {
	db := newIDORTestDB(t)
	tenantA := uuid.New()
	fakeID := uuid.New()

	r := setupIDORRouter(ValidateResourceOwnership(db, "productos", "id"), tenantA)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test/"+fakeID.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestIDORGuard_NoTenantContext_Returns401(t *testing.T) {
	db := newIDORTestDB(t)
	resourceID := uuid.New()

	r := setupIDORRouterNoTenant(ValidateResourceOwnership(db, "productos", "id"))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test/"+resourceID.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "tenant context required", body["detail"])
}

func TestIDORGuard_EmptyParam_SkipsValidation(t *testing.T) {
	db := newIDORTestDB(t)
	tenantA := uuid.New()

	// Register on a route that might have an empty param match.
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ClaimsKey, &JWTClaims{TenantID: tenantA.String(), Rol: "administrador"})
		ctx := context.WithValue(c.Request.Context(), tenantctx.Key, tenantA)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	// Use a paramName that doesn't exist in the route.
	r.GET("/test", ValidateResourceOwnership(db, "productos", "nonexistent"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIDORGuard_InvalidUUID_Returns404(t *testing.T) {
	db := newIDORTestDB(t)
	tenantA := uuid.New()

	r := setupIDORRouter(ValidateResourceOwnership(db, "productos", "id"), tenantA)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestIDORGuard_DELETE_OtherTenant_Returns404(t *testing.T) {
	db := newIDORTestDB(t)
	tenantA := uuid.New()
	tenantB := uuid.New()
	resourceID := uuid.New()

	db.Exec("INSERT INTO productos (id, tenant_id, nombre) VALUES (?, ?, ?)",
		resourceID.String(), tenantB.String(), "Fanta")

	r := setupIDORRouter(ValidateResourceOwnership(db, "productos", "id"), tenantA)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/test/"+resourceID.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// Tests: ValidateBodyReferences
// ---------------------------------------------------------------------------

func TestIDORGuard_BodyRef_OwnResource_Passes(t *testing.T) {
	db := newIDORTestDB(t)
	tenantA := uuid.New()
	catID := uuid.New()

	db.Exec("INSERT INTO categorias (id, tenant_id, nombre) VALUES (?, ?, ?)",
		catID.String(), tenantA.String(), "Bebidas")

	body, _ := json.Marshal(map[string]string{"categoria_id": catID.String()})

	r := setupBodyRouter(ValidateBodyReferences(db, map[string]string{
		"categoria_id": "categorias",
	}), tenantA)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIDORGuard_BodyRef_OtherTenant_Returns422(t *testing.T) {
	db := newIDORTestDB(t)
	tenantA := uuid.New()
	tenantB := uuid.New()
	catID := uuid.New()

	// Categoria belongs to tenant B.
	db.Exec("INSERT INTO categorias (id, tenant_id, nombre) VALUES (?, ?, ?)",
		catID.String(), tenantB.String(), "Golosinas")

	body, _ := json.Marshal(map[string]string{"categoria_id": catID.String()})

	r := setupBodyRouter(ValidateBodyReferences(db, map[string]string{
		"categoria_id": "categorias",
	}), tenantA)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var respBody map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody["detail"], "categoria_id")
}

func TestIDORGuard_BodyRef_MissingField_SkipsValidation(t *testing.T) {
	db := newIDORTestDB(t)
	tenantA := uuid.New()

	// Body without the tracked field.
	body, _ := json.Marshal(map[string]string{"nombre": "Test"})

	r := setupBodyRouter(ValidateBodyReferences(db, map[string]string{
		"categoria_id": "categorias",
	}), tenantA)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIDORGuard_BodyRef_NoTenantContext_Returns401(t *testing.T) {
	db := newIDORTestDB(t)
	catID := uuid.New()

	body, _ := json.Marshal(map[string]string{"categoria_id": catID.String()})

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ClaimsKey, &JWTClaims{Rol: "administrador"})
		c.Next()
	})
	r.POST("/test", ValidateBodyReferences(db, map[string]string{
		"categoria_id": "categorias",
	}), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIDORGuard_BodyRef_EmptyBody_SkipsValidation(t *testing.T) {
	db := newIDORTestDB(t)
	tenantA := uuid.New()

	r := setupBodyRouter(ValidateBodyReferences(db, map[string]string{
		"categoria_id": "categorias",
	}), tenantA)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	req.ContentLength = 0
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIDORGuard_BodyRef_InvalidUUID_Returns422(t *testing.T) {
	db := newIDORTestDB(t)
	tenantA := uuid.New()

	body, _ := json.Marshal(map[string]string{"categoria_id": "not-a-uuid"})

	r := setupBodyRouter(ValidateBodyReferences(db, map[string]string{
		"categoria_id": "categorias",
	}), tenantA)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ---------------------------------------------------------------------------
// Tests: Table name whitelist
// ---------------------------------------------------------------------------

func TestIDORGuard_UnregisteredTable_PanicsAtRegistration(t *testing.T) {
	db := newIDORTestDB(t)
	assert.Panics(t, func() {
		ValidateResourceOwnership(db, "tabla_que_no_existe_xyz", "id")
	})
}

func TestIDORGuard_BodyRef_UnregisteredTable_PanicsAtRegistration(t *testing.T) {
	db := newIDORTestDB(t)
	assert.Panics(t, func() {
		ValidateBodyReferences(db, map[string]string{
			"campo": "tabla_que_no_existe_xyz",
		})
	})
}
