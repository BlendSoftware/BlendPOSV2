package infra

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newTestDB creates an in-memory SQLite database that acts as a stand-in for
// *gorm.DB in unit tests. No real PostgreSQL needed.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return db
}

// TestReadReplica_FallbackWhenEmpty verifies that when the replica URL is empty,
// NewDatabaseReadReplica returns the primary DB instance (same pointer).
func TestReadReplica_FallbackWhenEmpty(t *testing.T) {
	primary := newTestDB(t)

	got := NewDatabaseReadReplica(primary, "")

	assert.Same(t, primary, got, "should return primary DB when replica URL is empty")
}

// TestReadReplica_FallbackOnBadURL verifies that an invalid/unreachable replica
// URL causes a graceful fallback to the primary DB instead of an error.
func TestReadReplica_FallbackOnBadURL(t *testing.T) {
	primary := newTestDB(t)

	// Use a bogus DSN that will fail to connect
	got := NewDatabaseReadReplica(primary, "host=192.0.2.1 port=1 user=x password=x dbname=x connect_timeout=1")

	assert.Same(t, primary, got, "should fall back to primary DB when replica connection fails")
}

// TestReadReplica_ReturnedDBIsValid ensures the returned *gorm.DB (whether
// primary or replica) is usable — i.e., its underlying sql.DB is pingable.
func TestReadReplica_ReturnedDBIsValid(t *testing.T) {
	primary := newTestDB(t)

	dbRead := NewDatabaseReadReplica(primary, "")

	sqlDB, err := dbRead.DB()
	require.NoError(t, err)
	assert.NoError(t, sqlDB.Ping(), "returned DB should be pingable")
}
