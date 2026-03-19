package model

import (
	"time"

	"github.com/google/uuid"
)

// Subscription represents a billing subscription linking a tenant to a plan
// via MercadoPago (or any future payment gateway).
type Subscription struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID           uuid.UUID  `gorm:"type:uuid;not null"`
	PlanID             uuid.UUID  `gorm:"type:uuid;not null"`
	MPSubscriptionID   *string    `gorm:"column:mp_subscription_id;type:varchar(255)"`
	MPPayerID          *string    `gorm:"column:mp_payer_id;type:varchar(255)"`
	Status             string     `gorm:"type:varchar(50);not null;default:'pending'"` // pending, active, paused, cancelled
	CurrentPeriodStart *time.Time `gorm:"type:timestamptz"`
	CurrentPeriodEnd   *time.Time `gorm:"type:timestamptz"`
	CreatedAt          time.Time
	UpdatedAt          time.Time

	// Associations
	Tenant *Tenant `gorm:"foreignKey:TenantID"`
	Plan   *Plan   `gorm:"foreignKey:PlanID"`
}

func (Subscription) TableName() string { return "subscriptions" }
