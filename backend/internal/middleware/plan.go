package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"blendpos/internal/model"
	"blendpos/internal/repository"
	"blendpos/internal/tenantctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Redis key templates and TTLs for plan enforcement.
const (
	planCacheKey        = "t:%s:plan"         // JSON-serialised Plan
	planCacheTTL        = 5 * time.Minute
	productCountKey     = "t:%s:product_count" // cached COUNT(*)
	productCountTTL     = 1 * time.Minute
	activeDevicesKey    = "t:%s:active_devices" // Redis SET of device IDs
	activeDevicesTTL    = 24 * time.Hour
)

// PlanLimitError is the structured 403 body returned when a plan cap is reached.
type PlanLimitError struct {
	Error      string `json:"error"`
	Message    string `json:"message"`
	Limit      string `json:"limit"`
	Current    int64  `json:"current"`
	Max        int    `json:"max"`
	UpgradeURL string `json:"upgrade_url"`
}

// ---------------------------------------------------------------------------
// Public middleware constructors
// ---------------------------------------------------------------------------

// EnforcePlanLimitProductos blocks product creation when the tenant's plan
// cap (max_productos > 0) has been reached.
// Applied to: POST /v1/productos
//
// When rdb is non-nil the plan and product count are cached in Redis to avoid
// hitting PostgreSQL on every request. If Redis is unavailable the middleware
// falls back to a direct DB query (fail-open on total infrastructure failure).
func EnforcePlanLimitProductos(tenantRepo repository.TenantRepository, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		tid, err := tenantctx.FromContext(c.Request.Context())
		if err != nil {
			c.Next() // fail-open: no tenant context
			return
		}

		plan, err := fetchPlan(c.Request.Context(), tenantRepo, rdb, tid)
		if err != nil || plan == nil {
			// Cannot determine plan — allow the operation (fail-open).
			if err != nil {
				log.Warn().Err(err).Str("tenant_id", tid.String()).
					Msg("plan enforcement: could not load plan, allowing request")
			}
			c.Next()
			return
		}

		limit := plan.MaxProductos
		if limit == 0 {
			c.Next() // 0 = unlimited
			return
		}

		count, err := fetchProductCount(c.Request.Context(), tenantRepo, rdb, tid)
		if err != nil {
			log.Warn().Err(err).Str("tenant_id", tid.String()).
				Msg("plan enforcement: could not count products, allowing request")
			c.Next() // fail-open
			return
		}

		if count >= int64(limit) {
			c.AbortWithStatusJSON(http.StatusForbidden, PlanLimitError{
				Error:      "plan_limit_exceeded",
				Message:    fmt.Sprintf("Tu plan %s permite hasta %d productos activos. Actualizá tu plan para agregar más.", plan.Nombre, limit),
				Limit:      "max_productos",
				Current:    count,
				Max:        limit,
				UpgradeURL: "/billing/upgrade",
			})
			return
		}
		c.Next()
	}
}

// EnforcePlanLimitTerminales blocks caja sessions when the tenant's plan
// cap (max_terminales) has been reached.
// Applied to: POST /v1/caja/abrir
//
// Terminal count is derived from open sesion_cajas rows (same source of truth
// as before). Redis caching is used only for the plan lookup.
func EnforcePlanLimitTerminales(tenantRepo repository.TenantRepository, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		tid, err := tenantctx.FromContext(c.Request.Context())
		if err != nil {
			c.Next()
			return
		}

		plan, err := fetchPlan(c.Request.Context(), tenantRepo, rdb, tid)
		if err != nil || plan == nil {
			if err != nil {
				log.Warn().Err(err).Str("tenant_id", tid.String()).
					Msg("plan enforcement: could not load plan, allowing request")
			}
			c.Next()
			return
		}

		limit := plan.MaxTerminales
		if limit <= 0 {
			c.Next() // 0 or negative = unlimited
			return
		}

		count, err := countOpenSessions(c.Request.Context(), tenantRepo.DB(), tid)
		if err != nil {
			log.Warn().Err(err).Str("tenant_id", tid.String()).
				Msg("plan enforcement: could not count sessions, allowing request")
			c.Next()
			return
		}

		if count >= int64(limit) {
			c.AbortWithStatusJSON(http.StatusForbidden, PlanLimitError{
				Error:      "plan_limit_exceeded",
				Message:    fmt.Sprintf("Tu plan %s permite hasta %d cajas abiertas simultáneamente. Cerrá una caja antes de abrir otra, o actualizá tu plan.", plan.Nombre, limit),
				Limit:      "max_terminales",
				Current:    count,
				Max:        limit,
				UpgradeURL: "/billing/upgrade",
			})
			return
		}
		c.Next()
	}
}

// EnforcePlanLimitSucursales blocks sucursal creation when the tenant's plan
// cap (max_sucursales > 0) has been reached.
// Applied to: POST /v1/sucursales
func EnforcePlanLimitSucursales(tenantRepo repository.TenantRepository, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		tid, err := tenantctx.FromContext(c.Request.Context())
		if err != nil {
			c.Next()
			return
		}

		plan, err := fetchPlan(c.Request.Context(), tenantRepo, rdb, tid)
		if err != nil || plan == nil {
			if err != nil {
				log.Warn().Err(err).Str("tenant_id", tid.String()).
					Msg("plan enforcement: could not load plan, allowing request")
			}
			c.Next()
			return
		}

		limit := plan.MaxSucursales
		if limit == 0 {
			c.Next() // 0 = unlimited
			return
		}

		count, err := countActiveSucursales(c.Request.Context(), tenantRepo.DB(), tid)
		if err != nil {
			log.Warn().Err(err).Str("tenant_id", tid.String()).
				Msg("plan enforcement: could not count sucursales, allowing request")
			c.Next()
			return
		}

		if count >= int64(limit) {
			c.AbortWithStatusJSON(http.StatusForbidden, PlanLimitError{
				Error:      "plan_limit_exceeded",
				Message:    fmt.Sprintf("Tu plan %s permite hasta %d sucursal(es). Actualizá tu plan para agregar más.", plan.Nombre, limit),
				Limit:      "max_sucursales",
				Current:    count,
				Max:        limit,
				UpgradeURL: "/billing/upgrade",
			})
			return
		}
		c.Next()
	}
}

// EnforcePlanLimitUsuarios blocks user creation when the tenant's plan
// cap (max_usuarios > 0) has been reached.
// Applied to: POST /v1/usuarios
func EnforcePlanLimitUsuarios(tenantRepo repository.TenantRepository, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		tid, err := tenantctx.FromContext(c.Request.Context())
		if err != nil {
			c.Next()
			return
		}

		plan, err := fetchPlan(c.Request.Context(), tenantRepo, rdb, tid)
		if err != nil || plan == nil {
			if err != nil {
				log.Warn().Err(err).Str("tenant_id", tid.String()).
					Msg("plan enforcement: could not load plan, allowing request")
			}
			c.Next()
			return
		}

		limit := plan.MaxUsuarios
		if limit == 0 {
			c.Next() // 0 = unlimited
			return
		}

		count, err := tenantRepo.CountUsuariosByTenant(c.Request.Context(), tid)
		if err != nil {
			log.Warn().Err(err).Str("tenant_id", tid.String()).
				Msg("plan enforcement: could not count usuarios, allowing request")
			c.Next()
			return
		}

		if count >= int64(limit) {
			c.AbortWithStatusJSON(http.StatusForbidden, PlanLimitError{
				Error:      "plan_limit_exceeded",
				Message:    fmt.Sprintf("Tu plan %s permite hasta %d usuario(s). Actualizá tu plan para agregar más.", plan.Nombre, limit),
				Limit:      "max_usuarios",
				Current:    count,
				Max:        limit,
				UpgradeURL: "/billing/upgrade",
			})
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
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"detail": "acceso restringido a superadmin"})
			return
		}
		c.Next()
	}
}

// InvalidatePlanCache removes the cached plan for a tenant. Call this when
// the tenant's plan changes (e.g. upgrade/downgrade).
func InvalidatePlanCache(ctx context.Context, rdb *redis.Client, tenantID uuid.UUID) {
	if rdb == nil {
		return
	}
	key := fmt.Sprintf(planCacheKey, tenantID.String())
	if err := rdb.Del(ctx, key).Err(); err != nil {
		log.Warn().Err(err).Str("tenant_id", tenantID.String()).
			Msg("failed to invalidate plan cache")
	}
}

// InvalidateProductCountCache removes the cached product count for a tenant.
// Call after creating or deleting products.
func InvalidateProductCountCache(ctx context.Context, rdb *redis.Client, tenantID uuid.UUID) {
	if rdb == nil {
		return
	}
	key := fmt.Sprintf(productCountKey, tenantID.String())
	if err := rdb.Del(ctx, key).Err(); err != nil {
		log.Warn().Err(err).Str("tenant_id", tenantID.String()).
			Msg("failed to invalidate product count cache")
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// fetchPlan returns the tenant's plan, trying Redis cache first.
// On any Redis error it falls back transparently to the DB.
func fetchPlan(ctx context.Context, tenantRepo repository.TenantRepository, rdb *redis.Client, tenantID uuid.UUID) (*model.Plan, error) {
	// Try Redis cache.
	if rdb != nil {
		key := fmt.Sprintf(planCacheKey, tenantID.String())
		data, err := rdb.Get(ctx, key).Bytes()
		if err == nil {
			var plan model.Plan
			if jsonErr := json.Unmarshal(data, &plan); jsonErr == nil {
				return &plan, nil
			}
			// Corrupt cache entry — fall through to DB.
			log.Warn().Str("tenant_id", tenantID.String()).
				Msg("plan cache: corrupt JSON, falling back to DB")
		} else if err != redis.Nil {
			// Redis error (not a cache miss) — log and fall through.
			log.Warn().Err(err).Str("tenant_id", tenantID.String()).
				Msg("plan cache: Redis read failed, falling back to DB")
		}
	}

	// DB fallback.
	tenant, err := tenantRepo.FindTenantByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("fetch tenant: %w", err)
	}
	if tenant.Plan == nil {
		return nil, nil // no plan assigned
	}

	// Write-through to cache (best-effort).
	if rdb != nil {
		if data, jsonErr := json.Marshal(tenant.Plan); jsonErr == nil {
			key := fmt.Sprintf(planCacheKey, tenantID.String())
			if setErr := rdb.Set(ctx, key, data, planCacheTTL).Err(); setErr != nil {
				log.Warn().Err(setErr).Str("tenant_id", tenantID.String()).
					Msg("plan cache: Redis write failed")
			}
		}
	}

	return tenant.Plan, nil
}

// fetchProductCount returns the number of active products for the tenant,
// trying a Redis cache before falling back to the DB query.
func fetchProductCount(ctx context.Context, tenantRepo repository.TenantRepository, rdb *redis.Client, tenantID uuid.UUID) (int64, error) {
	if rdb != nil {
		key := fmt.Sprintf(productCountKey, tenantID.String())
		val, err := rdb.Get(ctx, key).Int64()
		if err == nil {
			return val, nil
		}
		if err != redis.Nil {
			log.Warn().Err(err).Str("tenant_id", tenantID.String()).
				Msg("product count cache: Redis read failed, falling back to DB")
		}
	}

	count, err := tenantRepo.CountProductosByTenant(ctx, tenantID)
	if err != nil {
		return 0, err
	}

	// Write-through (best-effort).
	if rdb != nil {
		key := fmt.Sprintf(productCountKey, tenantID.String())
		if setErr := rdb.Set(ctx, key, count, productCountTTL).Err(); setErr != nil {
			log.Warn().Err(setErr).Str("tenant_id", tenantID.String()).
				Msg("product count cache: Redis write failed")
		}
	}

	return count, nil
}

// countOpenSessions counts sesion_cajas with estado='abierta' for the tenant.
func countOpenSessions(ctx context.Context, db *gorm.DB, tenantID uuid.UUID) (int64, error) {
	var count int64
	err := db.WithContext(ctx).
		Table("sesion_cajas").
		Where("tenant_id = ? AND estado = 'abierta'", tenantID).
		Count(&count).Error
	return count, err
}

// countActiveSucursales counts active sucursales for the tenant.
func countActiveSucursales(ctx context.Context, db *gorm.DB, tenantID uuid.UUID) (int64, error) {
	var count int64
	err := db.WithContext(ctx).
		Table("sucursales").
		Where("tenant_id = ? AND activa = true", tenantID).
		Count(&count).Error
	return count, err
}
