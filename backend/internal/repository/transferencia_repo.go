package repository

import (
	"context"

	"blendpos/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TransferenciaRepository defines the data access contract for stock transfers.
type TransferenciaRepository interface {
	Create(ctx context.Context, t *model.TransferenciaStock) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.TransferenciaStock, error)
	List(ctx context.Context, estado string) ([]model.TransferenciaStock, int64, error)
	UpdateEstado(ctx context.Context, id uuid.UUID, estado string) error
	UpdateEstadoTx(tx *gorm.DB, id uuid.UUID, estado string, completadoPor *uuid.UUID) error
	DB() *gorm.DB
}

type transferenciaRepo struct{ db *gorm.DB }

func NewTransferenciaRepository(db *gorm.DB) TransferenciaRepository {
	return &transferenciaRepo{db: db}
}

func (r *transferenciaRepo) DB() *gorm.DB { return r.db }

func (r *transferenciaRepo) Create(ctx context.Context, t *model.TransferenciaStock) error {
	db, tid, err := scopedDBWithTenant(r.db, ctx)
	if err != nil {
		return err
	}
	t.TenantID = tid
	return db.Create(t).Error
}

func (r *transferenciaRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.TransferenciaStock, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, err
	}
	var t model.TransferenciaStock
	err = db.
		Preload("Items").
		Preload("Items.Producto").
		Preload("SucursalOrigen").
		Preload("SucursalDestino").
		Preload("Creador").
		First(&t, id).Error
	return &t, err
}

func (r *transferenciaRepo) List(ctx context.Context, estado string) ([]model.TransferenciaStock, int64, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, 0, err
	}
	var items []model.TransferenciaStock
	var total int64

	q := db.Model(&model.TransferenciaStock{})
	if estado != "" {
		q = q.Where("estado = ?", estado)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err = q.
		Preload("Items").
		Preload("Items.Producto").
		Preload("SucursalOrigen").
		Preload("SucursalDestino").
		Preload("Creador").
		Order("created_at DESC").
		Find(&items).Error
	return items, total, err
}

func (r *transferenciaRepo) UpdateEstado(ctx context.Context, id uuid.UUID, estado string) error {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return err
	}
	return db.Model(&model.TransferenciaStock{}).Where("id = ?", id).Update("estado", estado).Error
}

func (r *transferenciaRepo) UpdateEstadoTx(tx *gorm.DB, id uuid.UUID, estado string, completadoPor *uuid.UUID) error {
	updates := map[string]interface{}{
		"estado": estado,
	}
	if completadoPor != nil {
		updates["completado_por"] = *completadoPor
		updates["completed_at"] = gorm.Expr("NOW()")
	}
	return tx.Model(&model.TransferenciaStock{}).Where("id = ?", id).Updates(updates).Error
}
