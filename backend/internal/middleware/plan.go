package middleware

import (
	"context"
	"net/http"
	"strconv"

	"blendpos/internal/apierror"
	"blendpos/internal/repository"
	"blendpos/internal/tenantctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EnforcePlanLimitProductos blocks product creation when the tenant's plan
// cap (max_productos > 0) has been reached.
// Applied to: POST /v1/productos
func EnforcePlanLimitProductos(tenantRepo repository.TenantRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tid, err := tenantctx.FromContext(c.Request.Context())
		if err != nil {
			c.Next()
			return
		}
		tenant, err := tenantRepo.FindTenantByID(c.Request.Context(), tid)
		if err != nil || tenant.Plan == nil {
			c.Next() // allow if plan can't be loaded
			return
		}
		limit := tenant.Plan.MaxProductos
		if limit == 0 {
			c.Next() // 0 = unlimited
			return
		}
		count, err := tenantRepo.CountProductosByTenant(c.Request.Context(), tid)
		if err != nil {
			c.Next()
			return
		}
		if count >= int64(limit) {
			c.AbortWithStatusJSON(http.StatusForbidden, apierror.New(
				"Límite de productos alcanzado. Tu plan "+tenant.Plan.Nombre+
					" permite hasta "+strconv.Itoa(limit)+" productos activos. "+
					"Actualizá tu plan para agregar más.",
			))
			return
		}
		c.Next()
	}
}

// EnforcePlanLimitTerminales blocks caja sessions when the tenant's plan
// cap (max_terminales) has been reached.
// Applied to: POST /v1/caja/abrir
func EnforcePlanLimitTerminales(tenantRepo repository.TenantRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tid, err := tenantctx.FromContext(c.Request.Context())
		if err != nil {
			c.Next()
			return
		}
		tenant, err := tenantRepo.FindTenantByID(c.Request.Context(), tid)
		if err != nil || tenant.Plan == nil {
			c.Next()
			return
		}
		limit := tenant.Plan.MaxTerminales
		if limit <= 0 {
			c.Next()
			return
		}
		count, err := countOpenSessions(c.Request.Context(), tenantRepo.DB(), tid)
		if err != nil {
			c.Next()
			return
		}
		if count >= int64(limit) {
			c.AbortWithStatusJSON(http.StatusForbidden, apierror.New(
				"Límite de terminales alcanzado. Tu plan "+tenant.Plan.Nombre+
					" permite hasta "+strconv.Itoa(limit)+" cajas abiertas simultáneamente. "+
					"Cerrá una caja antes de abrir otra, o actualizá tu plan.",
			))
			return
		}
		c.Next()
	}
}

// RequireSuperAdmin aborts requests whose JWT role is not "superadmin".
func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := c.MustGet(ClaimsKey).(*JWTClaims)
		if !ok || claims.Rol != "superadmin" {
			c.AbortWithStatusJSON(http.StatusForbidden, apierror.New("acceso restringido a superadmin"))
			return
		}
		c.Next()
	}
}

func countOpenSessions(ctx context.Context, db *gorm.DB, tenantID uuid.UUID) (int64, error) {
	var count int64
	err := db.WithContext(ctx).
		Table("sesion_cajas").
		Where("tenant_id = ? AND estado = 'abierta'", tenantID).
		Count(&count).Error
	return count, err
}
