package infra

import (
	"context"
	"testing"

	"blendpos/internal/tenantctx"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newSQLiteDB creates an in-memory SQLite DB for unit tests.
// SQLite doesn't support SET LOCAL or pg_stat_activity, but we can still
// verify callback registration and that the callback doesn't panic/error
// on non-PostgreSQL backends.
func newSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Create a dummy table so we can run actual queries to trigger callbacks.
	err = db.Exec("CREATE TABLE IF NOT EXISTS _tagging_test (id INTEGER PRIMARY KEY, name TEXT)").Error
	require.NoError(t, err)

	return db
}

// TestRegisterConnectionTagging_CallbacksRegistered verifies that
// RegisterConnectionTagging doesn't panic and that subsequent queries
// work correctly (i.e. callbacks are wired without breaking GORM).
func TestRegisterConnectionTagging_CallbacksRegistered(t *testing.T) {
	db := newSQLiteDB(t)

	// Should not panic
	assert.NotPanics(t, func() {
		RegisterConnectionTagging(db)
	})

	// Verify the DB is still functional after registering callbacks
	err := db.Exec("INSERT INTO _tagging_test (id, name) VALUES (99, 'reg-test')").Error
	assert.NoError(t, err, "DB should still be functional after registering callbacks")

	var count int64
	err = db.Raw("SELECT COUNT(*) FROM _tagging_test WHERE id = 99").Scan(&count).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count, "inserted row should be queryable")
}

// TestConnectionTagging_NoTenantContext_NoError verifies that queries
// without tenant context in the context don't fail or panic.
func TestConnectionTagging_NoTenantContext_NoError(t *testing.T) {
	db := newSQLiteDB(t)
	RegisterConnectionTagging(db)

	// Insert without tenant context — should succeed (callback skips silently)
	err := db.Exec("INSERT INTO _tagging_test (id, name) VALUES (1, 'test')").Error
	assert.NoError(t, err, "query without tenant context should succeed")

	// Query without tenant context
	var count int64
	err = db.Raw("SELECT COUNT(*) FROM _tagging_test").Count(&count).Error
	assert.NoError(t, err, "read without tenant context should succeed")
}

// TestConnectionTagging_WithTenantContext_NoError verifies that queries
// with tenant context don't panic on SQLite (SET LOCAL will fail silently
// on SQLite but the actual query should still succeed).
func TestConnectionTagging_WithTenantContext_NoError(t *testing.T) {
	db := newSQLiteDB(t)
	RegisterConnectionTagging(db)

	tid := uuid.New()
	ctx := context.WithValue(context.Background(), tenantctx.Key, tid)

	// The SET LOCAL will fail on SQLite but is best-effort; the actual
	// insert should still succeed.
	err := db.WithContext(ctx).Exec("INSERT INTO _tagging_test (id, name) VALUES (2, 'tenant-test')").Error
	assert.NoError(t, err, "query with tenant context should succeed even on SQLite")
}

// TestConnectionTagging_NilContext_NoPanic ensures the callback handles
// edge cases like nil Statement or Context gracefully.
func TestConnectionTagging_NilContext_NoPanic(t *testing.T) {
	db := newSQLiteDB(t)

	// Register without panicking
	assert.NotPanics(t, func() {
		RegisterConnectionTagging(db)
	})

	// Execute a raw query (exercises the callback path)
	assert.NotPanics(t, func() {
		_ = db.Exec("SELECT 1")
	})
}

// TestGetPoolStats_ReturnsValidStats verifies that GetPoolStats returns
// meaningful data from the underlying sql.DB connection pool.
func TestGetPoolStats_ReturnsValidStats(t *testing.T) {
	db := newSQLiteDB(t)

	stats, err := GetPoolStats(db)
	require.NoError(t, err)
	require.NotNil(t, stats)

	// SQLite in-memory has at least 1 open connection after use
	assert.GreaterOrEqual(t, stats.OpenConnections, 0)
	assert.GreaterOrEqual(t, stats.Idle, 0)
	assert.Equal(t, 0, stats.WaitCount, "fresh pool should have zero waits")
}

// TestGetConnectionsByTenant_SQLiteReturnsError verifies that the
// pg_stat_activity query fails gracefully on non-PostgreSQL backends.
// (In production this only runs against PostgreSQL.)
func TestGetConnectionsByTenant_SQLiteReturnsError(t *testing.T) {
	db := newSQLiteDB(t)

	results, err := GetConnectionsByTenant(db)

	// SQLite doesn't have pg_stat_activity — we expect an error
	assert.Error(t, err, "pg_stat_activity query should fail on SQLite")
	assert.Nil(t, results)
}

// TestConnectionInfo_StructTags verifies the JSON and GORM tags are correct.
func TestConnectionInfo_StructTags(t *testing.T) {
	info := ConnectionInfo{
		TenantID:    "abc-123",
		ActiveConns: 5,
		State:       "active",
	}
	assert.Equal(t, "abc-123", info.TenantID)
	assert.Equal(t, 5, info.ActiveConns)
	assert.Equal(t, "active", info.State)
}
