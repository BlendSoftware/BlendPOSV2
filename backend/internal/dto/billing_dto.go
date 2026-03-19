package dto

// ── Request DTOs ──────────────────────────────────────────────────────────────

// SubscribeRequest is the payload for POST /v1/billing/subscribe.
type SubscribeRequest struct {
	PlanID string `json:"plan_id" validate:"required,uuid"`
}

// ── Response DTOs ─────────────────────────────────────────────────────────────

// SubscribeResponse is returned after initiating a subscription.
type SubscribeResponse struct {
	SubscriptionID string `json:"subscription_id"`
	CheckoutURL    string `json:"checkout_url"`
	Status         string `json:"status"`
}

// BillingStatusResponse is returned by GET /v1/billing/status.
type BillingStatusResponse struct {
	HasSubscription bool    `json:"has_subscription"`
	Status          string  `json:"status"`
	PlanNombre      string  `json:"plan_nombre"`
	PeriodEnd       *string `json:"period_end,omitempty"`
}
