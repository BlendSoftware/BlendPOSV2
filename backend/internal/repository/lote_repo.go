package repository

import (
	"context"
	"errors"
	"math"
	"time"

	"blendpos/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LoteRepository defines the data access contract for product lots/batches.
type LoteRepository interface {
	Create(ctx context.Context, lote *model.LoteProducto) error
	ListByProducto(ctx context.Context, productoID uuid.UUID) ([]model.LoteProducto, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetAlertasVencimiento(ctx context.Context, diasAnticipacion int) ([]LoteAlerta, error)
}

// LoteAlerta is the repository-level result for expiry alerts.
type LoteAlerta struct {
	model.LoteProducto
	DiasRestantes int
	Estado        string // "vencido" | "critico" | "proximo"
}

type loteRepo struct{ db *gorm.DB }

func NewLoteRepository(db *gorm.DB) LoteRepository { return &loteRepo{db: db} }

func (r *loteRepo) Create(ctx context.Context, lote *model.LoteProducto) error {
	db, tid, err := scopedDBWithTenant(r.db, ctx)
	if err != nil {
		return err
	}
	lote.TenantID = tid
	return db.Create(lote).Error
}

func (r *loteRepo) ListByProducto(ctx context.Context, productoID uuid.UUID) ([]model.LoteProducto, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, err
	}
	var lotes []model.LoteProducto
	err = db.Where("producto_id = ?", productoID).
		Preload("Producto").
		Order("fecha_vencimiento ASC").
		Find(&lotes).Error
	return lotes, err
}

func (r *loteRepo) Delete(ctx context.Context, id uuid.UUID) error {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return err
	}
	result := db.Delete(&model.LoteProducto{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("lote no encontrado")
	}
	return nil
}

// GetAlertasVencimiento returns lots expiring within diasAnticipacion days.
// Classification:
//   - vencido:  fecha_vencimiento < today
//   - critico:  fecha_vencimiento <= today + 3 days
//   - proximo:  fecha_vencimiento <= today + diasAnticipacion days
func (r *loteRepo) GetAlertasVencimiento(ctx context.Context, diasAnticipacion int) ([]LoteAlerta, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, err
	}

	if diasAnticipacion <= 0 {
		diasAnticipacion = 7
	}

	limite := time.Now().AddDate(0, 0, diasAnticipacion)

	var lotes []model.LoteProducto
	err = db.Where("fecha_vencimiento <= ? AND cantidad > 0", limite).
		Preload("Producto").
		Order("fecha_vencimiento ASC").
		Find(&lotes).Error
	if err != nil {
		return nil, err
	}

	now := time.Now().Truncate(24 * time.Hour)
	alertas := make([]LoteAlerta, 0, len(lotes))
	for _, lote := range lotes {
		dias := int(math.Ceil(lote.FechaVencimiento.Sub(now).Hours() / 24))
		estado := "proximo"
		if dias < 0 {
			estado = "vencido"
		} else if dias <= 3 {
			estado = "critico"
		}
		alertas = append(alertas, LoteAlerta{
			LoteProducto:  lote,
			DiasRestantes: dias,
			Estado:        estado,
		})
	}

	return alertas, nil
}
