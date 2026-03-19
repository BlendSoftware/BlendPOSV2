package middleware

import (
	"context"
	"net/http"

	"blendpos/internal/apierror"
	"blendpos/internal/tenantctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantMiddleware extracts the tenant_id from the validated JWT (claim "tid"),
// injects it into the request context, and sets it as a PostgreSQL session
// variable (app.tenant_id) so Row Level Security policies can filter rows.
//
// Must be placed AFTER JWTAuth in the middleware chain.
func TenantMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := c.MustGet(ClaimsKey).(*JWTClaims)
		if !ok || claims.TenantID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				apierror.New("tenant context missing"))
			return
		}

		tenantID, err := uuid.Parse(claims.TenantID)
		if err != nil || tenantID == uuid.Nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				apierror.New("invalid tenant_id in token"))
			return
		}

		// Inject into Go context for use in services and repositories.
		ctx := context.WithValue(c.Request.Context(), tenantctx.Key, tenantID)
		c.Request = c.Request.WithContext(ctx)

		// set_config('app.tenant_id', value, is_local=false) sets a session-level
		// parameter that activates the RLS policy current_tenant_id() for this request.
		// We use set_config() instead of SET because SET does not support parameterized values.
		if err := db.WithContext(ctx).Exec(
			"SELECT set_config('app.tenant_id', ?, false)", tenantID.String(),
		).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError,
				apierror.New("tenant context injection failed"))
			return
		}

		// Inject a request-scoped *gorm.DB that already has the tenant_id WHERE
		// clause baked in. This is a THIRD safety layer (on top of RLS + repo
		// scopedDB) so that ad-hoc queries or new repos get automatic isolation.
		scopedDB := db.WithContext(ctx).Where("tenant_id = ?", tenantID)
		ctx = context.WithValue(ctx, tenantctx.ScopedDBKey, scopedDB)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// TenantIDFromContext retrieves the tenant UUID injected by TenantMiddleware.
// Thin wrapper over tenantctx.FromContext kept for backward compatibility.
func TenantIDFromContext(ctx context.Context) (uuid.UUID, error) {
	return tenantctx.FromContext(ctx)
}
