package repository

import (
	"context"

	"blendpos/internal/tenantctx"
	"blendpos/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProveedorRepository interface {
	Create(ctx context.Context, p *model.Proveedor) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Proveedor, error)
	List(ctx context.Context) ([]model.Proveedor, error)
	Update(ctx context.Context, p *model.Proveedor) error
	SoftDelete(ctx context.Context, id uuid.UUID) error

	// Price history — append-only (RF-26)
	CreateHistorialPrecio(ctx context.Context, h *model.HistorialPrecio) error
	ListHistorialPorProducto(ctx context.Context, productoID uuid.UUID) ([]model.HistorialPrecio, error)

	// Contacts
	ReplaceContactos(ctx context.Context, proveedorID uuid.UUID, contactos []model.ContactoProveedor) error

	// DB exposes the underlying *gorm.DB so services can open transactions.
	DB() *gorm.DB
}

type proveedorRepo struct{ db *gorm.DB }

func NewProveedorRepository(db *gorm.DB) ProveedorRepository { return &proveedorRepo{db: db} }

func (r *proveedorRepo) Create(ctx context.Context, p *model.Proveedor) error {
	tid, err := tenantctx.FromContext(ctx)
	if err != nil {
		return err
	}
	p.TenantID = tid
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *proveedorRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Proveedor, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, err
	}
	var p model.Proveedor
	err = db.Preload("Contactos").First(&p, id).Error
	return &p, err
}

func (r *proveedorRepo) List(ctx context.Context) ([]model.Proveedor, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, err
	}
	var proveedores []model.Proveedor
	err = db.Preload("Contactos").Where("activo = true").Order("razon_social ASC").Find(&proveedores).Error
	return proveedores, err
}

func (r *proveedorRepo) Update(ctx context.Context, p *model.Proveedor) error {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return err
	}
	return db.Save(p).Error
}

func (r *proveedorRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return err
	}
	return db.Model(&model.Proveedor{}).Where("id = ?", id).Update("activo", false).Error
}

func (r *proveedorRepo) CreateHistorialPrecio(ctx context.Context, h *model.HistorialPrecio) error {
	tid, err := tenantctx.FromContext(ctx)
	if err != nil {
		return err
	}
	h.TenantID = tid
	return r.db.WithContext(ctx).Create(h).Error
}

func (r *proveedorRepo) ListHistorialPorProducto(ctx context.Context, productoID uuid.UUID) ([]model.HistorialPrecio, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, err
	}
	var historial []model.HistorialPrecio
	err = db.
		Where("producto_id = ?", productoID).
		Order("created_at DESC").
		Limit(50).
		Find(&historial).Error
	return historial, err
}

func (r *proveedorRepo) DB() *gorm.DB { return r.db }

// ReplaceContactos deletes all existing contacts for the supplier and inserts the new ones.
// ContactoProveedor is scoped through proveedor_id — no direct tenant_id needed.
func (r *proveedorRepo) ReplaceContactos(ctx context.Context, proveedorID uuid.UUID, contactos []model.ContactoProveedor) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("proveedor_id = ?", proveedorID).Delete(&model.ContactoProveedor{}).Error; err != nil {
			return err
		}
		if len(contactos) > 0 {
			if err := tx.Create(&contactos).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
