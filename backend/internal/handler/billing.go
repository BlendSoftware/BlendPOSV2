package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"blendpos/internal/apierror"
	"blendpos/internal/dto"
	"blendpos/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// BillingHandler exposes billing/subscription endpoints.
type BillingHandler struct {
	svc service.BillingService
}

func NewBillingHandler(svc service.BillingService) *BillingHandler {
	return &BillingHandler{svc: svc}
}

// Subscribe godoc
// @Summary Crear suscripción (genera link de pago MercadoPago)
// @Tags    billing
// @Accept  json
// @Produce json
// @Param   body body dto.SubscribeRequest true "Plan a suscribir"
// @Success 201 {object} dto.SubscribeResponse
// @Router  /v1/billing/subscribe [post]
func (h *BillingHandler) Subscribe(c *gin.Context) {
	var req dto.SubscribeRequest
	if !bindAndValidate(c, &req) {
		return
	}

	resp, err := h.svc.Subscribe(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusCreated, resp)
}

// Webhook godoc
// @Summary Webhook de MercadoPago (público, sin JWT)
// @Tags    billing
// @Accept  json
// @Produce json
// @Success 200
// @Router  /v1/billing/webhook [post]
func (h *BillingHandler) Webhook(c *gin.Context) {
	// TODO: Verify X-Signature header from MercadoPago.
	// For now we log a warning and process anyway.
	signature := c.GetHeader("X-Signature")
	if signature == "" {
		log.Warn().Msg("billing webhook: missing X-Signature header")
		// In production, this should return 401. For now, allow for development.
		// Uncomment the following lines when MP signature verification is implemented:
		// c.JSON(http.StatusUnauthorized, apierror.New("firma inválida"))
		// return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("no se pudo leer el body"))
		return
	}
	defer c.Request.Body.Close()

	// Parse the generic webhook payload.
	// MercadoPago sends different structures depending on the event type.
	// We extract the fields we need.
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("JSON inválido"))
		return
	}

	event := service.WebhookEvent{}

	if t, ok := raw["type"].(string); ok {
		event.Type = t
	}
	if a, ok := raw["action"].(string); ok {
		event.Action = a
	}

	// Extract subscription ID and status from the nested data object.
	if data, ok := raw["data"].(map[string]interface{}); ok {
		if id, ok := data["id"].(string); ok {
			event.SubscriptionID = id
		}
		if st, ok := data["status"].(string); ok {
			event.Status = st
		}
	}

	if event.SubscriptionID == "" {
		// Try alternate field names for compatibility
		if id, ok := raw["subscription_id"].(string); ok {
			event.SubscriptionID = id
		}
		if st, ok := raw["status"].(string); ok {
			event.Status = st
		}
	}

	log.Info().
		Str("type", event.Type).
		Str("action", event.Action).
		Str("subscription_id", event.SubscriptionID).
		Str("status", event.Status).
		Msg("billing webhook received")

	if err := h.svc.HandleWebhook(c.Request.Context(), event); err != nil {
		log.Error().Err(err).Msg("billing webhook processing failed")
		c.JSON(http.StatusInternalServerError, apierror.New("error procesando webhook"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetStatus godoc
// @Summary Estado de suscripción del tenant actual
// @Tags    billing
// @Produce json
// @Success 200 {object} dto.BillingStatusResponse
// @Router  /v1/billing/status [get]
func (h *BillingHandler) GetStatus(c *gin.Context) {
	resp, err := h.svc.GetStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}
