package repository

import (
	"context"

	"blendpos/internal/tenantctx"
	"blendpos/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CategoriaRepository defines CRUD operations for Categoria.
type CategoriaRepository interface {
	Crear(ctx context.Context, c *model.Categoria) error
	Listar(ctx context.Context) ([]model.Categoria, error)
	ObtenerPorID(ctx context.Context, id uuid.UUID) (*model.Categoria, error)
	ObtenerPorNombre(ctx context.Context, nombre string) (*model.Categoria, error)
	Actualizar(ctx context.Context, c *model.Categoria) error
	Desactivar(ctx context.Context, id uuid.UUID) error
}

type categoriaRepository struct{ db *gorm.DB }

func NewCategoriaRepository(db *gorm.DB) CategoriaRepository {
	return &categoriaRepository{db: db}
}

func (r *categoriaRepository) Crear(ctx context.Context, c *model.Categoria) error {
	tid, err := tenantctx.FromContext(ctx)
	if err != nil {
		return err
	}
	c.TenantID = tid
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *categoriaRepository) Listar(ctx context.Context) ([]model.Categoria, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, err
	}
	var list []model.Categoria
	err = db.Order("nombre asc").Find(&list).Error
	return list, err
}

func (r *categoriaRepository) ObtenerPorID(ctx context.Context, id uuid.UUID) (*model.Categoria, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, err
	}
	var c model.Categoria
	err = db.First(&c, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *categoriaRepository) ObtenerPorNombre(ctx context.Context, nombre string) (*model.Categoria, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, err
	}
	var c model.Categoria
	err = db.Where("lower(nombre) = lower(?)", nombre).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *categoriaRepository) Actualizar(ctx context.Context, c *model.Categoria) error {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return err
	}
	return db.Save(c).Error
}

func (r *categoriaRepository) Desactivar(ctx context.Context, id uuid.UUID) error {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return err
	}
	return db.Model(&model.Categoria{}).Where("id = ?", id).Update("activo", false).Error
}
