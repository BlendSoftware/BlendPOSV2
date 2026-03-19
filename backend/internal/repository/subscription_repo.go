package repository

import (
	"context"

	"blendpos/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SubscriptionRepository manages subscription persistence.
type SubscriptionRepository interface {
	Create(ctx context.Context, s *model.Subscription) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Subscription, error)
	FindActiveByTenantID(ctx context.Context, tenantID uuid.UUID) (*model.Subscription, error)
	FindByMPSubscriptionID(ctx context.Context, mpSubID string) (*model.Subscription, error)
	Update(ctx context.Context, s *model.Subscription) error
}

type subscriptionRepo struct{ db *gorm.DB }

func NewSubscriptionRepository(db *gorm.DB) SubscriptionRepository {
	return &subscriptionRepo{db: db}
}

func (r *subscriptionRepo) Create(ctx context.Context, s *model.Subscription) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *subscriptionRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Subscription, error) {
	var s model.Subscription
	err := r.db.WithContext(ctx).Preload("Plan").Preload("Tenant").First(&s, "id = ?", id).Error
	return &s, err
}

func (r *subscriptionRepo) FindActiveByTenantID(ctx context.Context, tenantID uuid.UUID) (*model.Subscription, error) {
	var s model.Subscription
	err := r.db.WithContext(ctx).
		Preload("Plan").
		Where("tenant_id = ? AND status IN ('active', 'pending')", tenantID).
		Order("created_at DESC").
		First(&s).Error
	return &s, err
}

func (r *subscriptionRepo) FindByMPSubscriptionID(ctx context.Context, mpSubID string) (*model.Subscription, error) {
	var s model.Subscription
	err := r.db.WithContext(ctx).
		Preload("Plan").
		Preload("Tenant").
		Where("mp_subscription_id = ?", mpSubID).
		First(&s).Error
	return &s, err
}

func (r *subscriptionRepo) Update(ctx context.Context, s *model.Subscription) error {
	return r.db.WithContext(ctx).Save(s).Error
}
