package repository

import (
	"context"

	"blendpos/internal/tenantctx"
	"blendpos/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MovimientoStockFilter defines filters for listing stock movements.
type MovimientoStockFilter struct {
	ProductoID *uuid.UUID
	Tipo       string
	Page       int
	Limit      int
}

type MovimientoStockRepository interface {
	Create(ctx context.Context, m *model.MovimientoStock) error
	// CreateTx creates a movimiento within an existing DB transaction.
	// The caller must set m.TenantID before calling this method.
	CreateTx(tx *gorm.DB, m *model.MovimientoStock) error
	List(ctx context.Context, filter MovimientoStockFilter) ([]model.MovimientoStock, int64, error)
}

type movimientoStockRepo struct{ db *gorm.DB }

func NewMovimientoStockRepository(db *gorm.DB) MovimientoStockRepository {
	return &movimientoStockRepo{db: db}
}

func (r *movimientoStockRepo) Create(ctx context.Context, m *model.MovimientoStock) error {
	tid, err := tenantctx.FromContext(ctx)
	if err != nil {
		return err
	}
	m.TenantID = tid
	return r.db.WithContext(ctx).Create(m).Error
}

// CreateTx creates a movimiento within an existing DB transaction.
// TenantID is extracted from the transaction's context automatically.
func (r *movimientoStockRepo) CreateTx(tx *gorm.DB, m *model.MovimientoStock) error {
	if m.TenantID == (uuid.UUID{}) && tx.Statement != nil && tx.Statement.Context != nil {
		if tid, err := tenantctx.FromContext(tx.Statement.Context); err == nil {
			m.TenantID = tid
		}
	}
	return tx.Create(m).Error
}

func (r *movimientoStockRepo) List(ctx context.Context, filter MovimientoStockFilter) ([]model.MovimientoStock, int64, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, 0, err
	}
	q := db.Model(&model.MovimientoStock{}).
		Preload("Producto")
	if filter.ProductoID != nil {
		q = q.Where("producto_id = ?", *filter.ProductoID)
	}
	if filter.Tipo != "" {
		q = q.Where("tipo = ?", filter.Tipo)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := filter.Page
	limit := filter.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 500 {
		limit = 100
	}
	offset := (page - 1) * limit

	var movimientos []model.MovimientoStock
	err = q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&movimientos).Error
	return movimientos, total, err
}
