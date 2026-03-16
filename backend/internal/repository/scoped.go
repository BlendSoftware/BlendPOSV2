package repository

import (
	"context"

	"blendpos/internal/tenantctx"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// scopedDB builds a *gorm.DB scoped to the tenant in the context.
// It adds an explicit WHERE tenant_id = ? clause as a double-safety layer
// on top of PostgreSQL RLS, making query plans predictable and avoiding
// full table scans when RLS is bypassed (e.g. in tests).
//
// Usage inside any repository method:
//
//	db, err := scopedDB(r.db, ctx)
//	if err != nil { return err }
//	return db.Where(...).Find(&result).Error
func scopedDB(db *gorm.DB, ctx context.Context) (*gorm.DB, error) {
	tid, err := tenantctx.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	return db.WithContext(ctx).Where("tenant_id = ?", tid), nil
}

// scopedDBWithTenant is the same as scopedDB but also returns the tenant UUID
// for callers that need to stamp new records (Create calls).
func scopedDBWithTenant(db *gorm.DB, ctx context.Context) (*gorm.DB, uuid.UUID, error) {
	tid, err := tenantctx.FromContext(ctx)
	if err != nil {
		return nil, uuid.Nil, err
	}
	return db.WithContext(ctx).Where("tenant_id = ?", tid), tid, nil
}
