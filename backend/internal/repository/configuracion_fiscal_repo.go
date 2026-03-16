package repository

import (
	"context"

	"blendpos/internal/tenantctx"
	"blendpos/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ConfiguracionFiscalRepository interface {
	Get(ctx context.Context) (*model.ConfiguracionFiscal, error)
	Upsert(ctx context.Context, config *model.ConfiguracionFiscal) error
}

type configuracionFiscalRepository struct {
	db *gorm.DB
}

func NewConfiguracionFiscalRepository(db *gorm.DB) ConfiguracionFiscalRepository {
	return &configuracionFiscalRepository{db}
}

func (r *configuracionFiscalRepository) Get(ctx context.Context) (*model.ConfiguracionFiscal, error) {
	db, err := scopedDB(r.db, ctx)
	if err != nil {
		return nil, err
	}
	var cfg model.ConfiguracionFiscal
	if err := db.First(&cfg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Not found is acceptable (first time setup)
		}
		return nil, err
	}
	return &cfg, nil
}

func (r *configuracionFiscalRepository) Upsert(ctx context.Context, config *model.ConfiguracionFiscal) error {
	tid, err := tenantctx.FromContext(ctx)
	if err != nil {
		return err
	}
	config.TenantID = tid

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.ConfiguracionFiscal
		err := tx.Where("tenant_id = ?", tid).First(&existing).Error

		if err == gorm.ErrRecordNotFound {
			// New record — generate a proper UUID
			config.ID = uuid.New()
			return tx.Create(config).Error
		} else if err != nil {
			return err
		}

		// Update — preserve the existing ID and creation time
		config.ID = existing.ID
		config.CreatedAt = existing.CreatedAt
		return tx.Save(config).Error
	})
}
