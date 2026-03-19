package repository

import (
	"context"

	"blendpos/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SucursalRepository interface {
	Create(ctx context.Context, s *model.Sucursal) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Sucursal, error)
	Update(ctx context.Context, s *model.Sucursal) error
	List(ctx context.Context, incluirInactivas bool) ([]model.Sucursal, int64, error)
}

type sucursalRepo struct{ db *gorm.DB }

func NewSucursalRepository(db *gorm.DB) SucursalRepository {
	return &sucursalRepo{db: db}
}

func (r *sucursalRepo) Create(ctx context.Context, s *model.Sucursal) error {
	db, tid, err := scopedDBWithTenant(r.db, ctx)
	if err != nil {
		return err
	}
	s.TenantID = tid
	return db.Create(s).Error
}

func (r *sucursalRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Sucursal, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, err
	}
	var s model.Sucursal
	err = db.First(&s, id).Error
	return &s, err
}

func (r *sucursalRepo) Update(ctx context.Context, s *model.Sucursal) error {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return err
	}
	return db.Save(s).Error
}

func (r *sucursalRepo) List(ctx context.Context, incluirInactivas bool) ([]model.Sucursal, int64, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, 0, err
	}
	var sucursales []model.Sucursal
	var total int64

	q := db.Model(&model.Sucursal{})
	if !incluirInactivas {
		q = q.Where("activa = true")
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err = q.Order("nombre ASC").Find(&sucursales).Error
	return sucursales, total, err
}
