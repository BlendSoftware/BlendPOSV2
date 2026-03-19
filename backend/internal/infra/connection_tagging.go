// Package infra — connection_tagging.go
//
// F2-5: Connection pool with tenant tagging.
//
// Registers a GORM callback that sets PostgreSQL's application_name to
// "blendpos:tenant:{uuid}" via SET LOCAL, making every active connection
// attributable to a specific tenant in pg_stat_activity.
//
// SET LOCAL scopes the change to the current transaction, so connection pool
// connections are not permanently polluted — once the tx ends (or the
// statement finishes for non-tx queries), the value reverts.
package infra

import (
	"fmt"

	"blendpos/internal/tenantctx"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// RegisterConnectionTagging installs GORM callbacks that tag the PostgreSQL
// connection with the tenant ID from the request context. The tag is visible
// in pg_stat_activity.application_name and is invaluable for debugging
// connection leaks and per-tenant monitoring.
//
// Registers on Query, Create, Update, Delete, and Row (raw) callbacks so that
// ALL database operations are tagged, not just reads.
func RegisterConnectionTagging(db *gorm.DB) {
	tagFn := func(db *gorm.DB) {
		if db.Statement == nil || db.Statement.Context == nil {
			return
		}
		tid, err := tenantctx.FromContext(db.Statement.Context)
		if err != nil {
			return // no tenant context — skip silently (e.g. health checks, migrations)
		}
		// SET LOCAL scopes to current transaction. For non-tx queries GORM wraps
		// them in an implicit transaction, so this still works correctly.
		//
		// SET doesn't support prepared statement parameters ($1), so we use
		// fmt.Sprintf with a sanitized value (UUID only, no user input).
		tx := db.Session(&gorm.Session{NewDB: true})
		setSQL := fmt.Sprintf("SET LOCAL application_name = 'blendpos:tenant:%s'", tid.String())
		if err := tx.Exec(setSQL).Error; err != nil {
			// Log but don't fail the actual query — tagging is best-effort.
			log.Debug().Err(err).Str("tenant_id", tid.String()).
				Msg("connection_tagging: failed to set application_name")
		}
	}

	// Register before each operation type so all paths are covered.
	_ = db.Callback().Query().Before("gorm:query").Register("tenant:tag_connection_query", tagFn)
	_ = db.Callback().Create().Before("gorm:create").Register("tenant:tag_connection_create", tagFn)
	_ = db.Callback().Update().Before("gorm:update").Register("tenant:tag_connection_update", tagFn)
	_ = db.Callback().Delete().Before("gorm:delete").Register("tenant:tag_connection_delete", tagFn)
	_ = db.Callback().Row().Before("gorm:row").Register("tenant:tag_connection_row", tagFn)

	log.Info().Msg("connection_tagging: GORM callbacks registered for tenant tagging")
}

// ConnectionInfo represents per-tenant connection statistics from pg_stat_activity.
type ConnectionInfo struct {
	TenantID    string `json:"tenant_id" gorm:"column:tenant_id"`
	ActiveConns int    `json:"active_connections" gorm:"column:active_connections"`
	State       string `json:"state" gorm:"column:state"`
}

// GetConnectionsByTenant queries pg_stat_activity and returns connection counts
// grouped by tenant and connection state. Only connections tagged with the
// "blendpos:tenant:" prefix are included.
//
// This is intended for superadmin/debugging endpoints — not for hot paths.
func GetConnectionsByTenant(db *gorm.DB) ([]ConnectionInfo, error) {
	var results []ConnectionInfo

	// Extract tenant UUID from the application_name pattern "blendpos:tenant:{uuid}"
	query := `
		SELECT
			REPLACE(application_name, 'blendpos:tenant:', '') AS tenant_id,
			COALESCE(state, 'unknown') AS state,
			COUNT(*) AS active_connections
		FROM pg_stat_activity
		WHERE application_name LIKE 'blendpos:tenant:%'
		GROUP BY application_name, state
		ORDER BY active_connections DESC
	`

	if err := db.Raw(query).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("connection_tagging: failed to query pg_stat_activity: %w", err)
	}

	return results, nil
}

// PoolStats holds connection pool metrics from the underlying sql.DB.
type PoolStats struct {
	MaxOpenConnections int `json:"max_open_connections"`
	OpenConnections    int `json:"open_connections"`
	InUse              int `json:"in_use"`
	Idle               int `json:"idle"`
	WaitCount          int `json:"wait_count"`
	WaitDurationMs     int `json:"wait_duration_ms"`
}

// GetPoolStats returns current connection pool metrics from the underlying
// database/sql pool. Lightweight — just reads atomic counters, no DB queries.
func GetPoolStats(db *gorm.DB) (*PoolStats, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("connection_tagging: failed to get sql.DB: %w", err)
	}

	stats := sqlDB.Stats()
	return &PoolStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          int(stats.WaitCount),
		WaitDurationMs:     int(stats.WaitDuration.Milliseconds()),
	}, nil
}
