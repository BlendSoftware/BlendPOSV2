package middleware

import (
	"context"

	"blendpos/internal/tenantctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SucursalMiddleware reads the optional X-Sucursal-Id header sent by the
// frontend's global branch selector and injects the parsed UUID into the
// request context under tenantctx.SucursalKey.
//
// When the header is absent or empty, the context value is not set, which
// means "all branches" (consolidated view).
//
// Must be placed AFTER JWTAuth + TenantMiddleware so that authentication
// and tenant isolation are already enforced.
func SucursalMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("X-Sucursal-Id")
		if raw == "" {
			c.Next()
			return
		}

		sucursalID, err := uuid.Parse(raw)
		if err != nil || sucursalID == uuid.Nil {
			// Invalid header — ignore silently (treat as consolidated view).
			c.Next()
			return
		}

		ctx := context.WithValue(c.Request.Context(), tenantctx.SucursalKey, sucursalID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// SucursalIDFromContext is a convenience wrapper around tenantctx.SucursalFromContext.
// Returns nil when no sucursal is selected (consolidated view).
func SucursalIDFromContext(ctx context.Context) *uuid.UUID {
	return tenantctx.SucursalFromContext(ctx)
}
