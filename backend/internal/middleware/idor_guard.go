// Package middleware — idor_guard.go implements anti-IDOR (Insecure Direct
// Object Reference) middleware that validates resource ownership before the
// request reaches the handler.
//
// Design decisions:
//   - Returns 404 (not 403) to avoid revealing resource existence to other tenants.
//   - tableName is validated against an allow-list to prevent SQL injection.
//   - Uses parameterized queries for all user-supplied values (resourceID, tenantID).
//   - Body reference validation reads the body, validates, then restores it.
package middleware

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"blendpos/internal/apierror"
	"blendpos/internal/tenantctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// allowedTables is the whitelist of table names that can be used with
// ValidateResourceOwnership and ValidateBodyReferences. Any table name not in
// this set will cause a panic at startup (fail-fast, not at runtime).
var allowedTables = map[string]bool{
	"productos":               true,
	"ventas":                  true,
	"categorias":              true,
	"proveedores":             true,
	"usuarios":                true,
	"compras":                 true,
	"promociones":             true,
	"sesiones_caja":           true,
	"comprobantes":            true,
	"inventario_vinculos":     true,
	"inventario_movimientos":  true,
	"historial_precios":       true,
	"configuraciones_fiscales": true,
	"lotes_producto":          true,
	"clientes":                true,
	"movimientos_cuenta":      true,
}

// RegisterAllowedTable adds a table name to the whitelist at init time.
// Useful for tests or extensions that add new tenant-scoped tables.
func RegisterAllowedTable(name string) {
	allowedTables[name] = true
}

// validateTableName checks that tableName is in the allow-list.
// Panics at middleware registration time (startup), NOT at request time.
func validateTableName(tableName string) {
	if !allowedTables[tableName] {
		panic(fmt.Sprintf("idor_guard: table %q not in allowed tables whitelist — register it with RegisterAllowedTable or add to allowedTables", tableName))
	}
}

// ValidateResourceOwnership verifies that a resource identified by a path
// parameter belongs to the tenant extracted from the JWT context.
//
// Usage:
//
//	router.GET("/productos/:id", ValidateResourceOwnership(db, "productos", "id"), handler.GetProducto)
//	router.PUT("/ventas/:id", ValidateResourceOwnership(db, "ventas", "id"), handler.UpdateVenta)
func ValidateResourceOwnership(db *gorm.DB, tableName string, paramName string) gin.HandlerFunc {
	// Validate table name at registration time (fail-fast).
	validateTableName(tableName)

	return func(c *gin.Context) {
		resourceID := c.Param(paramName)
		if resourceID == "" {
			c.Next()
			return
		}

		// Validate that resourceID is a valid UUID to avoid garbage queries.
		if _, err := uuid.Parse(resourceID); err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, apierror.New("resource not found"))
			return
		}

		tenantID, err := tenantctx.FromContext(c.Request.Context())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apierror.New("tenant context required"))
			return
		}

		// Query uses parameterized values for id and tenant_id.
		// tableName is safe because it was validated against the whitelist above.
		var exists int64
		query := fmt.Sprintf("SELECT COUNT(1) FROM %s WHERE id = ? AND tenant_id = ? LIMIT 1", tableName)
		if err := db.Raw(query, resourceID, tenantID).Scan(&exists).Error; err != nil {
			log.Error().Err(err).
				Str("table", tableName).
				Str("resource_id", resourceID).
				Str("tenant_id", tenantID.String()).
				Msg("idor_guard: ownership check query failed")
			// Fail-closed: deny access on DB error.
			c.AbortWithStatusJSON(http.StatusInternalServerError, apierror.New("internal error"))
			return
		}

		if exists == 0 {
			log.Warn().
				Str("tenant_id", tenantID.String()).
				Str("resource_id", resourceID).
				Str("table", tableName).
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Msg("IDOR attempt: resource not found for tenant")

			c.AbortWithStatusJSON(http.StatusNotFound, apierror.New("resource not found"))
			return
		}

		c.Next()
	}
}

// ValidateBodyReferences verifies that UUID fields in the JSON request body
// reference resources that belong to the current tenant. The fieldMap maps
// JSON field names to database table names.
//
// Usage:
//
//	router.POST("/ventas", ValidateBodyReferences(db, map[string]string{
//	    "producto_id":  "productos",
//	    "categoria_id": "categorias",
//	}), handler.CreateVenta)
//
// Notes:
//   - Only top-level fields are checked (no nested object traversal).
//   - Fields that are missing or empty in the body are skipped.
//   - The body is read and then restored so the handler can re-read it.
func ValidateBodyReferences(db *gorm.DB, fieldMap map[string]string) gin.HandlerFunc {
	// Validate all table names at registration time.
	for field, table := range fieldMap {
		_ = field
		validateTableName(table)
	}

	return func(c *gin.Context) {
		// Only apply to methods that carry a body.
		if c.Request.Body == nil || c.Request.ContentLength == 0 {
			c.Next()
			return
		}

		tenantID, err := tenantctx.FromContext(c.Request.Context())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apierror.New("tenant context required"))
			return
		}

		// Read body.
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, apierror.New("unable to read request body"))
			return
		}
		// Restore body for downstream handlers.
		c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

		var bodyMap map[string]json.RawMessage
		if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
			// Not a JSON object — let the handler deal with it.
			c.Next()
			return
		}

		for field, table := range fieldMap {
			raw, ok := bodyMap[field]
			if !ok {
				continue
			}

			var refID string
			if err := json.Unmarshal(raw, &refID); err != nil {
				continue // Not a string — skip.
			}
			if refID == "" {
				continue
			}

			// Validate UUID format.
			if _, err := uuid.Parse(refID); err != nil {
				c.AbortWithStatusJSON(http.StatusUnprocessableEntity,
					apierror.New(fmt.Sprintf("invalid UUID in field %q", field)))
				return
			}

			var exists int64
			query := fmt.Sprintf("SELECT COUNT(1) FROM %s WHERE id = ? AND tenant_id = ? LIMIT 1", table)
			if err := db.Raw(query, refID, tenantID).Scan(&exists).Error; err != nil {
				log.Error().Err(err).
					Str("table", table).
					Str("field", field).
					Str("ref_id", refID).
					Str("tenant_id", tenantID.String()).
					Msg("idor_guard: body reference check query failed")
				c.AbortWithStatusJSON(http.StatusInternalServerError, apierror.New("internal error"))
				return
			}

			if exists == 0 {
				log.Warn().
					Str("tenant_id", tenantID.String()).
					Str("field", field).
					Str("ref_id", refID).
					Str("table", table).
					Str("method", c.Request.Method).
					Str("path", c.Request.URL.Path).
					Msg("IDOR attempt: body reference not found for tenant")

				c.AbortWithStatusJSON(http.StatusUnprocessableEntity,
					apierror.New(fmt.Sprintf("referenced resource in %q not found", field)))
				return
			}
		}

		c.Next()
	}
}
