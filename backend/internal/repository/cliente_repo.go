package repository

import (
	"context"

	"blendpos/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClienteRepository interface {
	Create(ctx context.Context, c *model.Cliente) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Cliente, error)
	Update(ctx context.Context, c *model.Cliente) error
	List(ctx context.Context, search string, page, limit int) ([]model.Cliente, int64, error)
	GetDeudores(ctx context.Context) ([]model.Cliente, int64, error)
	GetMovimientos(ctx context.Context, clienteID uuid.UUID, page, limit int) ([]model.MovimientoCuenta, int64, error)
	// UpdateSaldoTx atomically updates saldo_deudor and inserts a movimiento in a single TX.
	UpdateSaldoTx(tx *gorm.DB, cliente *model.Cliente, mov *model.MovimientoCuenta) error
	DB() *gorm.DB
}

type clienteRepo struct{ db *gorm.DB }

func NewClienteRepository(db *gorm.DB) ClienteRepository { return &clienteRepo{db: db} }

func (r *clienteRepo) DB() *gorm.DB { return r.db }

func (r *clienteRepo) Create(ctx context.Context, c *model.Cliente) error {
	db, tid, err := scopedDBWithTenant(r.db, ctx)
	if err != nil {
		return err
	}
	c.TenantID = tid
	return db.Create(c).Error
}

func (r *clienteRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Cliente, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, err
	}
	var c model.Cliente
	err = db.First(&c, id).Error
	return &c, err
}

func (r *clienteRepo) Update(ctx context.Context, c *model.Cliente) error {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return err
	}
	return db.Save(c).Error
}

func (r *clienteRepo) List(ctx context.Context, search string, page, limit int) ([]model.Cliente, int64, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, 0, err
	}
	var clientes []model.Cliente
	var total int64

	q := db.Model(&model.Cliente{}).Where("activo = true")
	if search != "" {
		q = q.Where("nombre ILIKE ?", "%"+search+"%")
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err = q.Order("nombre ASC").Offset(offset).Limit(limit).Find(&clientes).Error
	return clientes, total, err
}

func (r *clienteRepo) GetDeudores(ctx context.Context) ([]model.Cliente, int64, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, 0, err
	}
	var clientes []model.Cliente
	var total int64

	q := db.Model(&model.Cliente{}).Where("saldo_deudor > 0").Where("activo = true")
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err = q.Order("saldo_deudor DESC").Find(&clientes).Error
	return clientes, total, err
}

func (r *clienteRepo) GetMovimientos(ctx context.Context, clienteID uuid.UUID, page, limit int) ([]model.MovimientoCuenta, int64, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, 0, err
	}
	var movs []model.MovimientoCuenta
	var total int64

	q := db.Model(&model.MovimientoCuenta{}).Where("cliente_id = ?", clienteID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err = q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&movs).Error
	return movs, total, err
}

func (r *clienteRepo) UpdateSaldoTx(tx *gorm.DB, cliente *model.Cliente, mov *model.MovimientoCuenta) error {
	if err := tx.Model(&model.Cliente{}).Where("id = ?", cliente.ID).
		Update("saldo_deudor", cliente.SaldoDeudor).Error; err != nil {
		return err
	}
	return tx.Create(mov).Error
}
