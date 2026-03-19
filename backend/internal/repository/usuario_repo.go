package repository

import (
	"context"

	"blendpos/internal/tenantctx"
	"blendpos/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UsuarioRepository interface {
	Create(ctx context.Context, u *model.Usuario) error
	// FindByUsername is intentionally non-scoped — called at login before a tenant JWT exists.
	FindByUsername(ctx context.Context, username string) (*model.Usuario, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.Usuario, error)
	// FindByIDUnscoped is non-scoped — for auth operations (change-password, refresh) where tenant middleware may not be present.
	FindByIDUnscoped(ctx context.Context, id uuid.UUID) (*model.Usuario, error)
	List(ctx context.Context) ([]model.Usuario, error)
	ListAll(ctx context.Context) ([]model.Usuario, error)
	Update(ctx context.Context, u *model.Usuario) error
	UpdateUnscoped(ctx context.Context, u *model.Usuario) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Reactivar(ctx context.Context, id uuid.UUID) error
}

type usuarioRepo struct{ db *gorm.DB }

func NewUsuarioRepository(db *gorm.DB) UsuarioRepository { return &usuarioRepo{db: db} }

func (r *usuarioRepo) Create(ctx context.Context, u *model.Usuario) error {
	tid, err := tenantctx.FromContext(ctx)
	if err != nil {
		return err
	}
	u.TenantID = tid
	return r.db.WithContext(ctx).Create(u).Error
}

// FindByUsername is intentionally non-scoped — called at login before a tenant JWT exists.
func (r *usuarioRepo) FindByUsername(ctx context.Context, username string) (*model.Usuario, error) {
	var u model.Usuario
	// Accept login by username OR email (case-insensitive email match)
	err := r.db.WithContext(ctx).
		Where("(username = ? OR LOWER(email::text) = LOWER(?)) AND activo = true", username, username).
		First(&u).Error
	return &u, err
}

func (r *usuarioRepo) FindByIDUnscoped(ctx context.Context, id uuid.UUID) (*model.Usuario, error) {
	var u model.Usuario
	err := r.db.WithContext(ctx).First(&u, id).Error
	return &u, err
}

func (r *usuarioRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Usuario, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, err
	}
	var u model.Usuario
	err = db.First(&u, id).Error
	return &u, err
}

func (r *usuarioRepo) List(ctx context.Context) ([]model.Usuario, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, err
	}
	var users []model.Usuario
	err = db.Where("activo = true").Find(&users).Error
	return users, err
}

func (r *usuarioRepo) ListAll(ctx context.Context) ([]model.Usuario, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, err
	}
	var users []model.Usuario
	err = db.Find(&users).Error
	return users, err
}

func (r *usuarioRepo) Update(ctx context.Context, u *model.Usuario) error {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return err
	}
	return db.Save(u).Error
}

func (r *usuarioRepo) UpdateUnscoped(ctx context.Context, u *model.Usuario) error {
	return r.db.WithContext(ctx).Save(u).Error
}

func (r *usuarioRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return err
	}
	return db.Model(&model.Usuario{}).Where("id = ?", id).Update("activo", false).Error
}

func (r *usuarioRepo) Reactivar(ctx context.Context, id uuid.UUID) error {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return err
	}
	return db.Model(&model.Usuario{}).Where("id = ?", id).Update("activo", true).Error
}
