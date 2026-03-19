package middleware

import (
	"fmt"
	"net/http"
	"time"

	"blendpos/internal/repository"
	"blendpos/internal/tenantctx"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// Per-tenant rate limit defaults (requests per minute).
const (
	defaultTenantRateLimit = 200  // free / default plan
	paidTenantRateLimit    = 1000 // pro / paid plans
	tenantRateLimitWindow  = 60   // seconds (1 minute bucket)
	tenantRateLimitTTL     = 120  // seconds (key expiry — extra buffer)
)

// RateLimitPerTenant enforces a per-tenant request rate limit using Redis
// INCR + EXPIRE on a per-minute bucket.
//
// The limit is derived from the tenant's plan:
//   - Plans with PrecioMensual > 0 (paid): 1000 req/min
//   - Free / no plan: 200 req/min
//
// Must be placed AFTER TenantMiddleware (needs tenant_id in context).
// Must be placed AFTER JWTAuth (transitive via TenantMiddleware).
//
// On limit exceeded: 429 with JSON body + standard rate limit headers.
// Fail-open: if Redis or plan lookup fails, the request is allowed through.
func RateLimitPerTenant(tenantRepo repository.TenantRepository, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		tid, err := tenantctx.FromContext(c.Request.Context())
		if err != nil {
			// No tenant in context — skip (fail-open).
			c.Next()
			return
		}

		// Determine limit from cached plan (same pattern as plan.go).
		limit := defaultTenantRateLimit
		plan, err := fetchPlan(c.Request.Context(), tenantRepo, rdb, tid)
		if err != nil {
			log.Warn().Err(err).Str("tenant_id", tid.String()).
				Msg("per-tenant rate limit: could not load plan, using default limit")
		} else if plan != nil && plan.PrecioMensual.IsPositive() {
			limit = paidTenantRateLimit
		}

		// When Redis is nil (e.g. tests), use in-memory fallback.
		if rdb == nil {
			bucket := time.Now().Unix() / tenantRateLimitWindow
			memKey := fmt.Sprintf("rl:tenant:%s:%d", tid.String(), bucket)
			count := memIncr(memKey, time.Duration(tenantRateLimitWindow)*time.Second)
			remaining := limit - count
			if remaining < 0 {
				remaining = 0
			}
			resetAt := (bucket + 1) * tenantRateLimitWindow
			retryAfter := resetAt - time.Now().Unix()
			if retryAfter < 1 {
				retryAfter = 1
			}

			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt))

			if count > limit {
				c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error":       "rate limit exceeded",
					"retry_after": retryAfter,
				})
				return
			}
			c.Next()
			return
		}

		ctx := c.Request.Context()
		bucket := time.Now().Unix() / tenantRateLimitWindow
		key := fmt.Sprintf("rl:tenant:%s:%d", tid.String(), bucket)

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			// Redis unavailable — fail-open with in-memory fallback.
			log.Error().Err(err).Str("tenant_id", tid.String()).
				Msg("per-tenant rate limit: Redis unavailable, falling back to in-memory")
			memKey := fmt.Sprintf("rl:tenant:%s:%d", tid.String(), bucket)
			memCount := memIncr(memKey, time.Duration(tenantRateLimitWindow)*time.Second)
			// Use conservative limit (half) when Redis is down.
			conservativeLimit := limit / 2
			if conservativeLimit < 1 {
				conservativeLimit = 1
			}
			if memCount > conservativeLimit {
				resetAt := (bucket + 1) * tenantRateLimitWindow
				retryAfter := resetAt - time.Now().Unix()
				if retryAfter < 1 {
					retryAfter = 1
				}
				c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", conservativeLimit))
				c.Header("X-RateLimit-Remaining", "0")
				c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt))
				c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error":       "rate limit exceeded",
					"retry_after": retryAfter,
				})
				return
			}
			c.Next()
			return
		}

		// Set TTL only on first increment.
		if count == 1 {
			rdb.Expire(ctx, key, time.Duration(tenantRateLimitTTL)*time.Second) //nolint:errcheck
		}

		remaining := int64(limit) - count
		if remaining < 0 {
			remaining = 0
		}
		resetAt := (bucket + 1) * tenantRateLimitWindow
		retryAfter := resetAt - time.Now().Unix()
		if retryAfter < 1 {
			retryAfter = 1
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt))

		if count > int64(limit) {
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": retryAfter,
			})
			return
		}
		c.Next()
	}
}
