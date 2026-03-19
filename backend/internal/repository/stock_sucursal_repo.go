package repository

import (
	"context"

	"blendpos/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// StockSucursalRepository defines the data access contract for per-branch stock.
type StockSucursalRepository interface {
	GetStock(ctx context.Context, productoID, sucursalID uuid.UUID) (*model.StockSucursal, error)
	GetOrCreateStock(ctx context.Context, productoID, sucursalID uuid.UUID) (*model.StockSucursal, error)
	AjustarStockSucursal(ctx context.Context, productoID, sucursalID uuid.UUID, delta int) error
	AjustarStockSucursalTx(tx *gorm.DB, productoID, sucursalID uuid.UUID, delta int) error
	ListBySucursal(ctx context.Context, sucursalID uuid.UUID) ([]model.StockSucursal, int64, error)
	GetAlertasBySucursal(ctx context.Context, sucursalID uuid.UUID) ([]model.StockSucursal, error)
	DB() *gorm.DB
}

type stockSucursalRepo struct{ db *gorm.DB }

func NewStockSucursalRepository(db *gorm.DB) StockSucursalRepository {
	return &stockSucursalRepo{db: db}
}

func (r *stockSucursalRepo) DB() *gorm.DB { return r.db }

func (r *stockSucursalRepo) GetStock(ctx context.Context, productoID, sucursalID uuid.UUID) (*model.StockSucursal, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, err
	}
	var ss model.StockSucursal
	err = db.Where("producto_id = ? AND sucursal_id = ?", productoID, sucursalID).First(&ss).Error
	return &ss, err
}

func (r *stockSucursalRepo) GetOrCreateStock(ctx context.Context, productoID, sucursalID uuid.UUID) (*model.StockSucursal, error) {
	db, tid, err := scopedDBWithTenant(r.db, ctx)
	if err != nil {
		return nil, err
	}
	ss := model.StockSucursal{
		TenantID:   tid,
		ProductoID: productoID,
		SucursalID: sucursalID,
	}
	// Upsert: insert if not exists, do nothing on conflict — then fetch
	err = db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "producto_id"}, {Name: "sucursal_id"}},
		DoNothing: true,
	}).Create(&ss).Error
	if err != nil {
		return nil, err
	}
	// Re-fetch to get the actual record (may have existed already)
	return r.GetStock(ctx, productoID, sucursalID)
}

// AjustarStockSucursal atomically increments/decrements stock_actual.
func (r *stockSucursalRepo) AjustarStockSucursal(ctx context.Context, productoID, sucursalID uuid.UUID, delta int) error {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return err
	}
	return db.Model(&model.StockSucursal{}).
		Where("producto_id = ? AND sucursal_id = ?", productoID, sucursalID).
		Update("stock_actual", gorm.Expr("stock_actual + ?", delta)).Error
}

// AjustarStockSucursalTx atomically increments/decrements stock_actual within an existing transaction.
func (r *stockSucursalRepo) AjustarStockSucursalTx(tx *gorm.DB, productoID, sucursalID uuid.UUID, delta int) error {
	return tx.Model(&model.StockSucursal{}).
		Where("producto_id = ? AND sucursal_id = ?", productoID, sucursalID).
		Update("stock_actual", gorm.Expr("stock_actual + ?", delta)).Error
}

func (r *stockSucursalRepo) ListBySucursal(ctx context.Context, sucursalID uuid.UUID) ([]model.StockSucursal, int64, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, 0, err
	}
	var items []model.StockSucursal
	var total int64

	q := db.Model(&model.StockSucursal{}).Where("sucursal_id = ?", sucursalID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err = q.Preload("Producto").Preload("Sucursal").Order("updated_at DESC").Find(&items).Error
	return items, total, err
}

func (r *stockSucursalRepo) GetAlertasBySucursal(ctx context.Context, sucursalID uuid.UUID) ([]model.StockSucursal, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, err
	}
	var items []model.StockSucursal
	err = db.Where("sucursal_id = ? AND stock_actual <= stock_minimo", sucursalID).
		Preload("Producto").
		Find(&items).Error
	return items, err
}
