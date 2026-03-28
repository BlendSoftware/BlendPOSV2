// Package tenantctx provides a zero-dependency context key and helper for
// propagating the tenant UUID through the request context.
//
// It is intentionally a leaf package (imports only stdlib + uuid + gorm) so that
// both middleware and repository can import it without creating an import cycle.
package tenantctx

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type contextKey string

// Key is the context key under which the tenant UUID is stored.
// Using an exported constant lets middleware set it and repository read it
// without either package importing the other.
const Key contextKey = "tenant_id"

// ScopedDBKey is the context key under which the request-scoped *gorm.DB
// (already filtered by tenant_id) is stored. Set by TenantMiddleware.
const ScopedDBKey contextKey = "scoped_db"

// SucursalKey is the context key under which the optional sucursal UUID is
// stored. Set by SucursalMiddleware from the X-Sucursal-Id header.
// nil/absent = consolidated view (all branches).
const SucursalKey contextKey = "sucursal_id"

// FromContext retrieves the tenant UUID injected by TenantMiddleware.
// Returns an error if the context was not enriched (e.g. unauthenticated path).
func FromContext(ctx context.Context) (uuid.UUID, error) {
	tid, ok := ctx.Value(Key).(uuid.UUID)
	if !ok || tid == uuid.Nil {
		return uuid.Nil, errors.New("tenant_id not found in context — is TenantMiddleware active?")
	}
	return tid, nil
}

// ScopedDBFromContext retrieves the request-scoped *gorm.DB that already has
// a WHERE tenant_id = ? clause applied. Returns an error if TenantMiddleware
// did not inject it (e.g. the request bypassed the middleware chain).
func ScopedDBFromContext(ctx context.Context) (*gorm.DB, error) {
	db, ok := ctx.Value(ScopedDBKey).(*gorm.DB)
	if !ok || db == nil {
		return nil, errors.New("scoped DB not in context — is TenantMiddleware active?")
	}
	return db, nil
}

// MustScopedDB is like ScopedDBFromContext but panics if the scoped DB is not
// present. Intended for use in tests and code paths where absence of tenant
// context is a programming error, not a runtime condition.
func MustScopedDB(ctx context.Context) *gorm.DB {
	db, err := ScopedDBFromContext(ctx)
	if err != nil {
		panic(err)
	}
	return db
}

// SucursalFromContext retrieves the optional sucursal UUID injected by
// SucursalMiddleware. Returns nil when not set (consolidated / all branches).
func SucursalFromContext(ctx context.Context) *uuid.UUID {
	sid, ok := ctx.Value(SucursalKey).(uuid.UUID)
	if !ok || sid == uuid.Nil {
		return nil
	}
	return &sid
}
