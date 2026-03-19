package service

import (
	"context"
	"errors"
	"time"

	"blendpos/internal/dto"
	"blendpos/internal/model"
	"blendpos/internal/repository"
	"blendpos/internal/tenantctx"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var decimalHundred = decimal.NewFromInt(100)

// ── MercadoPago Client Interface ─────────────────────────────────────────────

// MPCreateSubscriptionRequest holds the data needed to create a subscription in MP.
type MPCreateSubscriptionRequest struct {
	PlanName      string
	PlanPriceCents int64 // price in ARS cents
	TenantID      string
	BackURL       string
}

// MPCreateSubscriptionResponse holds the response from MP after creating a subscription.
type MPCreateSubscriptionResponse struct {
	SubscriptionID string // MP-assigned subscription/preapproval ID
	CheckoutURL    string // URL to redirect the user for payment
}

// MPClient abstracts the MercadoPago API. The real implementation will be
// provided in a future task. Tests mock this interface.
type MPClient interface {
	CreateSubscription(ctx context.Context, req MPCreateSubscriptionRequest) (*MPCreateSubscriptionResponse, error)
}

// ── Billing Service ──────────────────────────────────────────────────────────

// BillingService manages subscription lifecycle and MercadoPago integration.
type BillingService interface {
	// Subscribe creates a new subscription for the current tenant.
	Subscribe(ctx context.Context, req dto.SubscribeRequest) (*dto.SubscribeResponse, error)

	// HandleWebhook processes a MercadoPago webhook event.
	HandleWebhook(ctx context.Context, event WebhookEvent) error

	// GetStatus returns the billing status of the current tenant.
	GetStatus(ctx context.Context) (*dto.BillingStatusResponse, error)
}

// WebhookEvent represents a parsed MercadoPago webhook payload.
type WebhookEvent struct {
	Type           string // e.g. "payment", "subscription_preapproval"
	Action         string // e.g. "payment.created", "updated"
	SubscriptionID string // MP subscription/preapproval ID
	Status         string // e.g. "authorized", "paused", "cancelled"
}

type billingService struct {
	subRepo    repository.SubscriptionRepository
	tenantRepo repository.TenantRepository
	mpClient   MPClient
}

// NewBillingService creates a new billing service with all dependencies injected.
func NewBillingService(
	subRepo repository.SubscriptionRepository,
	tenantRepo repository.TenantRepository,
	mpClient MPClient,
) BillingService {
	return &billingService{
		subRepo:    subRepo,
		tenantRepo: tenantRepo,
		mpClient:   mpClient,
	}
}

// ── Subscribe ────────────────────────────────────────────────────────────────

func (s *billingService) Subscribe(ctx context.Context, req dto.SubscribeRequest) (*dto.SubscribeResponse, error) {
	tid, err := tenantctx.FromContext(ctx)
	if err != nil {
		return nil, errors.New("tenant_id no encontrado en contexto")
	}

	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		return nil, errors.New("plan_id inválido")
	}

	// Verify plan exists
	plan, err := s.tenantRepo.FindPlanByID(ctx, planID)
	if err != nil {
		return nil, errors.New("plan no encontrado")
	}

	// Create subscription in DB with status "pending"
	sub := &model.Subscription{
		TenantID: tid,
		PlanID:   planID,
		Status:   "pending",
	}
	if err := s.subRepo.Create(ctx, sub); err != nil {
		log.Error().Err(err).Str("tenant_id", tid.String()).Msg("failed to create subscription")
		return nil, errors.New("error creando suscripción")
	}

	// Call MercadoPago to create the subscription/preapproval
	priceCents := plan.PrecioMensual.Mul(decimalHundred).IntPart()
	mpResp, err := s.mpClient.CreateSubscription(ctx, MPCreateSubscriptionRequest{
		PlanName:       plan.Nombre,
		PlanPriceCents: priceCents,
		TenantID:       tid.String(),
	})
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tid.String()).Msg("failed to create MP subscription")
		return nil, errors.New("error al crear suscripción en MercadoPago")
	}

	// Update subscription with MP IDs
	sub.MPSubscriptionID = &mpResp.SubscriptionID
	if err := s.subRepo.Update(ctx, sub); err != nil {
		log.Error().Err(err).Str("tenant_id", tid.String()).Msg("failed to update subscription with MP ID")
	}

	log.Info().
		Str("tenant_id", tid.String()).
		Str("subscription_id", sub.ID.String()).
		Str("plan", plan.Nombre).
		Msg("subscription created")

	return &dto.SubscribeResponse{
		SubscriptionID: sub.ID.String(),
		CheckoutURL:    mpResp.CheckoutURL,
		Status:         sub.Status,
	}, nil
}

// ── Webhook ──────────────────────────────────────────────────────────────────

func (s *billingService) HandleWebhook(ctx context.Context, event WebhookEvent) error {
	if event.SubscriptionID == "" {
		return errors.New("subscription_id vacío en el evento")
	}

	sub, err := s.subRepo.FindByMPSubscriptionID(ctx, event.SubscriptionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn().Str("mp_subscription_id", event.SubscriptionID).Msg("webhook: subscription not found, ignoring")
			return nil // Idempotent — don't fail on unknown subscriptions
		}
		return err
	}

	oldStatus := sub.Status

	switch event.Status {
	case "authorized", "active":
		sub.Status = "active"
		now := time.Now()
		sub.CurrentPeriodStart = &now
		endDate := now.AddDate(0, 1, 0) // +1 month
		sub.CurrentPeriodEnd = &endDate

		// Upgrade tenant plan
		if err := s.upgradeTenantPlan(ctx, sub.TenantID, sub.PlanID); err != nil {
			log.Error().Err(err).
				Str("tenant_id", sub.TenantID.String()).
				Str("plan_id", sub.PlanID.String()).
				Msg("webhook: failed to upgrade tenant plan")
			return err
		}

	case "paused", "pending":
		sub.Status = "paused"

	case "cancelled":
		sub.Status = "cancelled"

	default:
		log.Warn().
			Str("mp_subscription_id", event.SubscriptionID).
			Str("status", event.Status).
			Msg("webhook: unrecognized status, ignoring")
		return nil
	}

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return err
	}

	log.Info().
		Str("mp_subscription_id", event.SubscriptionID).
		Str("tenant_id", sub.TenantID.String()).
		Str("old_status", oldStatus).
		Str("new_status", sub.Status).
		Msg("webhook: subscription updated")

	return nil
}

// upgradeTenantPlan assigns the given plan to the tenant.
func (s *billingService) upgradeTenantPlan(ctx context.Context, tenantID, planID uuid.UUID) error {
	tenant, err := s.tenantRepo.FindTenantByID(ctx, tenantID)
	if err != nil {
		return err
	}
	tenant.PlanID = &planID
	return s.tenantRepo.UpdateTenant(ctx, tenant)
}

// ── Status ───────────────────────────────────────────────────────────────────

func (s *billingService) GetStatus(ctx context.Context) (*dto.BillingStatusResponse, error) {
	tid, err := tenantctx.FromContext(ctx)
	if err != nil {
		return nil, errors.New("tenant_id no encontrado en contexto")
	}

	sub, err := s.subRepo.FindActiveByTenantID(ctx, tid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &dto.BillingStatusResponse{
				HasSubscription: false,
			}, nil
		}
		return nil, err
	}

	resp := &dto.BillingStatusResponse{
		HasSubscription: true,
		Status:          sub.Status,
	}

	if sub.Plan != nil {
		resp.PlanNombre = sub.Plan.Nombre
	}

	if sub.CurrentPeriodEnd != nil {
		formatted := sub.CurrentPeriodEnd.Format(time.RFC3339)
		resp.PeriodEnd = &formatted
	}

	return resp, nil
}
