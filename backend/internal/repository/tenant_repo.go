package repository

import (
	"context"

	"blendpos/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantRepository manages tenants and plans.
type TenantRepository interface {
	// Tenant CRUD
	CreateTenant(ctx context.Context, t *model.Tenant) error
	FindTenantByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error)
	FindTenantBySlug(ctx context.Context, slug string) (*model.Tenant, error)
	UpdateTenant(ctx context.Context, t *model.Tenant) error
	ListTenants(ctx context.Context) ([]model.Tenant, error)

	// Plan queries
	FindPlanByID(ctx context.Context, id uuid.UUID) (*model.Plan, error)
	ListPlans(ctx context.Context) ([]model.Plan, error)

	// Plan enforcement helpers (bypass RLS — these are internal checks)
	CountProductosByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error)
	CountUsuariosByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error)
	CountVentasByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error)

	DB() *gorm.DB
}

type tenantRepo struct{ db *gorm.DB }

func NewTenantRepository(db *gorm.DB) TenantRepository {
	return &tenantRepo{db: db}
}

func (r *tenantRepo) CreateTenant(ctx context.Context, t *model.Tenant) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *tenantRepo) FindTenantByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
	var t model.Tenant
	err := r.db.WithContext(ctx).Preload("Plan").First(&t, "id = ?", id).Error
	return &t, err
}

func (r *tenantRepo) FindTenantBySlug(ctx context.Context, slug string) (*model.Tenant, error) {
	var t model.Tenant
	err := r.db.WithContext(ctx).Preload("Plan").First(&t, "slug = ?", slug).Error
	return &t, err
}

func (r *tenantRepo) UpdateTenant(ctx context.Context, t *model.Tenant) error {
	return r.db.WithContext(ctx).Save(t).Error
}

func (r *tenantRepo) ListTenants(ctx context.Context) ([]model.Tenant, error) {
	var tenants []model.Tenant
	err := r.db.WithContext(ctx).Preload("Plan").Order("created_at DESC").Find(&tenants).Error
	return tenants, err
}

func (r *tenantRepo) FindPlanByID(ctx context.Context, id uuid.UUID) (*model.Plan, error) {
	var p model.Plan
	err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error
	return &p, err
}

func (r *tenantRepo) ListPlans(ctx context.Context) ([]model.Plan, error) {
	var plans []model.Plan
	err := r.db.WithContext(ctx).Where("activo = true").Order("precio_mensual ASC").Find(&plans).Error
	return plans, err
}

// CountProductosByTenant bypasses scopedDB intentionally — it counts
// globally for a given tenant_id to enforce plan limits.
func (r *tenantRepo) CountProductosByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Producto{}).
		Where("tenant_id = ? AND activo = true", tenantID).
		Count(&count).Error
	return count, err
}

func (r *tenantRepo) CountUsuariosByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Usuario{}).
		Where("tenant_id = ? AND activo = true", tenantID).
		Count(&count).Error
	return count, err
}

func (r *tenantRepo) CountVentasByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Venta{}).
		Where("tenant_id = ?", tenantID).
		Count(&count).Error
	return count, err
}

func (r *tenantRepo) DB() *gorm.DB { return r.db }
