package tenantctx

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

// ── Tests: ScopedDBFromContext ──────────────────────────────────────────────

func TestScopedDBFromContext_ReturnsDB_WhenPresent(t *testing.T) {
	db := newTestDB(t)
	tenantID := uuid.New()

	scopedDB := db.Where("tenant_id = ?", tenantID)
	ctx := context.WithValue(context.Background(), ScopedDBKey, scopedDB)

	got, err := ScopedDBFromContext(ctx)
	require.NoError(t, err)
	assert.NotNil(t, got)
}

func TestScopedDBFromContext_ReturnsError_WhenAbsent(t *testing.T) {
	ctx := context.Background()

	got, err := ScopedDBFromContext(ctx)
	assert.Nil(t, got)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scoped DB not in context")
}

func TestScopedDBFromContext_ReturnsError_WhenNilDB(t *testing.T) {
	ctx := context.WithValue(context.Background(), ScopedDBKey, (*gorm.DB)(nil))

	got, err := ScopedDBFromContext(ctx)
	assert.Nil(t, got)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scoped DB not in context")
}

func TestScopedDBFromContext_ReturnsError_WhenWrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), ScopedDBKey, "not a db")

	got, err := ScopedDBFromContext(ctx)
	assert.Nil(t, got)
	assert.Error(t, err)
}

// ── Tests: MustScopedDB ────────────────────────────────────────────────────

func TestMustScopedDB_ReturnsDB_WhenPresent(t *testing.T) {
	db := newTestDB(t)
	tenantID := uuid.New()

	scopedDB := db.Where("tenant_id = ?", tenantID)
	ctx := context.WithValue(context.Background(), ScopedDBKey, scopedDB)

	assert.NotPanics(t, func() {
		got := MustScopedDB(ctx)
		assert.NotNil(t, got)
	})
}

func TestMustScopedDB_Panics_WhenAbsent(t *testing.T) {
	ctx := context.Background()

	assert.Panics(t, func() {
		MustScopedDB(ctx)
	})
}

// ── Tests: FromContext (existing, for completeness) ────────────────────────

func TestFromContext_ReturnsUUID_WhenPresent(t *testing.T) {
	tid := uuid.New()
	ctx := context.WithValue(context.Background(), Key, tid)

	got, err := FromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, tid, got)
}

func TestFromContext_ReturnsError_WhenAbsent(t *testing.T) {
	ctx := context.Background()

	_, err := FromContext(ctx)
	assert.Error(t, err)
}

// ── Tests: Scoped DB has correct tenant isolation ──────────────────────────

func TestScopedDBFromContext_DifferentTenantsGetDifferentDBs(t *testing.T) {
	db := newTestDB(t)
	tenantA := uuid.New()
	tenantB := uuid.New()

	scopedA := db.Where("tenant_id = ?", tenantA)
	scopedB := db.Where("tenant_id = ?", tenantB)

	ctxA := context.WithValue(context.Background(), ScopedDBKey, scopedA)
	ctxB := context.WithValue(context.Background(), ScopedDBKey, scopedB)

	gotA, err := ScopedDBFromContext(ctxA)
	require.NoError(t, err)

	gotB, err := ScopedDBFromContext(ctxB)
	require.NoError(t, err)

	// They should be different gorm.DB instances (different sessions)
	assert.NotSame(t, gotA, gotB)
}
