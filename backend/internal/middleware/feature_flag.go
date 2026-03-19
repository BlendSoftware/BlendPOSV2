package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"

	"blendpos/internal/repository"
	"blendpos/internal/tenantctx"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// FeatureFlagError is the structured 403 body returned when a plan feature is
// not enabled for the tenant's current plan.
type FeatureFlagError struct {
	Error      string `json:"error"`
	Feature    string `json:"feature"`
	UpgradeURL string `json:"upgrade_url"`
}

// RequireFeature blocks the request if the tenant's plan does not include the
// specified boolean feature flag in its JSONB `features` column.
//
// It reuses the same fetchPlan() helper and Redis caching pattern from plan.go
// so that the plan lookup is cached and fail-open on infrastructure errors.
func RequireFeature(feature string, tenantRepo repository.TenantRepository, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		tid, err := tenantctx.FromContext(c.Request.Context())
		if err != nil {
			c.Next() // fail-open: no tenant context
			return
		}

		plan, err := fetchPlan(c.Request.Context(), tenantRepo, rdb, tid)
		if err != nil || plan == nil {
			if err != nil {
				log.Warn().Err(err).Str("tenant_id", tid.String()).
					Str("feature", feature).
					Msg("feature flag: could not load plan, allowing request")
			}
			c.Next() // fail-open
			return
		}

		// Parse features JSONB into map[string]bool.
		features, parseErr := parseFeatures(plan.Features)
		if parseErr != nil {
			log.Warn().Err(parseErr).Str("tenant_id", tid.String()).
				Str("feature", feature).
				Msg("feature flag: could not parse features JSON, allowing request")
			c.Next() // fail-open on corrupt data
			return
		}

		enabled, exists := features[feature]
		if !exists || !enabled {
			c.AbortWithStatusJSON(http.StatusForbidden, FeatureFlagError{
				Error:      fmt.Sprintf("Función no disponible en tu plan actual"),
				Feature:    feature,
				UpgradeURL: "/planes",
			})
			return
		}

		c.Next()
	}
}

// parseFeatures deserialises the plan's Features JSONB into a simple
// map[string]bool. Returns an empty map (not nil) if the JSON is null or empty.
func parseFeatures(raw json.RawMessage) (map[string]bool, error) {
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "{}" {
		return make(map[string]bool), nil
	}
	var features map[string]bool
	if err := json.Unmarshal(raw, &features); err != nil {
		return nil, fmt.Errorf("unmarshal features: %w", err)
	}
	return features, nil
}
