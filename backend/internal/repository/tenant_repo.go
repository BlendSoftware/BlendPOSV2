package repository

import (
	"context"
	"time"

	"blendpos/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantListFilter contains filters for paginated tenant listing.
type TenantListFilter struct {
	Page     int
	PageSize int
	Search   string // match nombre or slug (ILIKE)
	Status   string // "active", "inactive", "all"
	PlanID   string // UUID string, empty = no filter
}

// TenantWithMetrics holds a tenant plus aggregated counts (populated by ListAllPaginated).
type TenantWithMetrics struct {
	model.Tenant
	TotalVentas    int64
	TotalProductos int64
	TotalUsuarios  int64
	UltimaVenta    *time.Time
}

// GlobalMetrics contains platform-wide aggregated counts.
type GlobalMetrics struct {
	TotalTenants    int64
	TenantActivos   int64
	TotalVentas     int64
	VentasUltimoMes int64
	TenantsPorPlan  []PlanCount
}

// PlanCount maps plan name to tenant count.
type PlanCount struct {
	PlanNombre string
	Count      int64
}

// TenantRepository manages tenants and plans.
type TenantRepository interface {
	// Tenant CRUD
	CreateTenant(ctx context.Context, t *model.Tenant) error
	FindTenantByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error)
	FindTenantBySlug(ctx context.Context, slug string) (*model.Tenant, error)
	UpdateTenant(ctx context.Context, t *model.Tenant) error
	ListTenants(ctx context.Context) ([]model.Tenant, error)

	// Superadmin: paginated listing with metrics
	ListAllPaginated(ctx context.Context, filter TenantListFilter) ([]TenantWithMetrics, int64, error)
	// Superadmin: tenant detail with metrics
	FindTenantWithMetrics(ctx context.Context, id uuid.UUID) (*TenantWithMetrics, error)
	// Superadmin: global platform metrics
	GetGlobalMetrics(ctx context.Context) (*GlobalMetrics, error)

	// Plan queries
	FindPlanByID(ctx context.Context, id uuid.UUID) (*model.Plan, error)
	FindPlanByNombre(ctx context.Context, nombre string) (*model.Plan, error)
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

func (r *tenantRepo) FindPlanByNombre(ctx context.Context, nombre string) (*model.Plan, error) {
	var p model.Plan
	err := r.db.WithContext(ctx).Where("nombre = ? AND activo = true", nombre).First(&p).Error
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

// ListAllPaginated returns tenants with metrics, applying server-side pagination and filters.
// Uses raw SQL subqueries for efficient COUNT aggregation across tables.
func (r *tenantRepo) ListAllPaginated(ctx context.Context, f TenantListFilter) ([]TenantWithMetrics, int64, error) {
	db := r.db.WithContext(ctx)

	// Base query: count total matching tenants (for pagination).
	countQ := db.Model(&model.Tenant{})
	dataQ := db.Model(&model.Tenant{}).Preload("Plan")

	// Apply filters to both count and data queries.
	if f.Search != "" {
		like := "%" + f.Search + "%"
		countQ = countQ.Where("(nombre ILIKE ? OR slug ILIKE ?)", like, like)
		dataQ = dataQ.Where("(nombre ILIKE ? OR slug ILIKE ?)", like, like)
	}
	switch f.Status {
	case "active":
		countQ = countQ.Where("activo = true")
		dataQ = dataQ.Where("activo = true")
	case "inactive":
		countQ = countQ.Where("activo = false")
		dataQ = dataQ.Where("activo = false")
	}
	if f.PlanID != "" {
		countQ = countQ.Where("plan_id = ?", f.PlanID)
		dataQ = dataQ.Where("plan_id = ?", f.PlanID)
	}

	var total int64
	if err := countQ.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Fetch tenant rows with pagination.
	var tenants []model.Tenant
	offset := (f.Page - 1) * f.PageSize
	if err := dataQ.Order("created_at DESC").Offset(offset).Limit(f.PageSize).Find(&tenants).Error; err != nil {
		return nil, 0, err
	}

	// Build metrics per tenant using efficient per-tenant COUNT subqueries.
	result := make([]TenantWithMetrics, len(tenants))
	for i, t := range tenants {
		tc := t
		result[i].Tenant = tc

		// Count ventas
		r.db.WithContext(ctx).Model(&model.Venta{}).
			Where("tenant_id = ?", tc.ID).Count(&result[i].TotalVentas)

		// Count productos (active)
		r.db.WithContext(ctx).Model(&model.Producto{}).
			Where("tenant_id = ? AND activo = true", tc.ID).Count(&result[i].TotalProductos)

		// Count usuarios (active)
		r.db.WithContext(ctx).Model(&model.Usuario{}).
			Where("tenant_id = ? AND activo = true", tc.ID).Count(&result[i].TotalUsuarios)

		// Latest venta
		var lastVenta time.Time
		row := r.db.WithContext(ctx).Model(&model.Venta{}).
			Select("MAX(created_at)").Where("tenant_id = ?", tc.ID).Row()
		if err := row.Scan(&lastVenta); err == nil && !lastVenta.IsZero() {
			result[i].UltimaVenta = &lastVenta
		}
	}

	return result, total, nil
}

// FindTenantWithMetrics returns a single tenant plus aggregated metrics.
func (r *tenantRepo) FindTenantWithMetrics(ctx context.Context, id uuid.UUID) (*TenantWithMetrics, error) {
	var t model.Tenant
	if err := r.db.WithContext(ctx).Preload("Plan").First(&t, "id = ?", id).Error; err != nil {
		return nil, err
	}

	result := &TenantWithMetrics{Tenant: t}

	r.db.WithContext(ctx).Model(&model.Venta{}).
		Where("tenant_id = ?", id).Count(&result.TotalVentas)
	r.db.WithContext(ctx).Model(&model.Producto{}).
		Where("tenant_id = ? AND activo = true", id).Count(&result.TotalProductos)
	r.db.WithContext(ctx).Model(&model.Usuario{}).
		Where("tenant_id = ? AND activo = true", id).Count(&result.TotalUsuarios)

	var lastVenta time.Time
	row := r.db.WithContext(ctx).Model(&model.Venta{}).
		Select("MAX(created_at)").Where("tenant_id = ?", id).Row()
	if err := row.Scan(&lastVenta); err == nil && !lastVenta.IsZero() {
		result.UltimaVenta = &lastVenta
	}

	return result, nil
}

// GetGlobalMetrics returns platform-wide aggregated metrics.
func (r *tenantRepo) GetGlobalMetrics(ctx context.Context) (*GlobalMetrics, error) {
	db := r.db.WithContext(ctx)
	m := &GlobalMetrics{}

	db.Model(&model.Tenant{}).Count(&m.TotalTenants)
	db.Model(&model.Tenant{}).Where("activo = true").Count(&m.TenantActivos)
	db.Model(&model.Venta{}).Count(&m.TotalVentas)

	// Ventas último mes
	oneMonthAgo := time.Now().AddDate(0, -1, 0)
	db.Model(&model.Venta{}).Where("created_at >= ?", oneMonthAgo).Count(&m.VentasUltimoMes)

	// Tenants por plan
	type planRow struct {
		PlanNombre string
		Count      int64
	}
	var rows []planRow
	db.Model(&model.Tenant{}).
		Select("COALESCE(plans.nombre, 'Sin plan') as plan_nombre, COUNT(*) as count").
		Joins("LEFT JOIN plans ON plans.id = tenants.plan_id").
		Group("plans.nombre").
		Order("count DESC").
		Scan(&rows)

	m.TenantsPorPlan = make([]PlanCount, len(rows))
	for i, r := range rows {
		m.TenantsPorPlan[i] = PlanCount{PlanNombre: r.PlanNombre, Count: r.Count}
	}

	return m, nil
}

func (r *tenantRepo) DB() *gorm.DB { return r.db }
