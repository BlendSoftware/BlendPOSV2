package handler

import (
	"net/http"

	"blendpos/internal/apierror"
	"blendpos/internal/dto"
	"blendpos/internal/middleware"
	"blendpos/internal/service"

	"github.com/gin-gonic/gin"
)

// AIHandler exposes AI-powered analytics endpoints.
type AIHandler struct {
	svc service.AIService
}

// NewAIHandler creates a handler wired to the AI service.
func NewAIHandler(svc service.AIService) *AIHandler {
	return &AIHandler{svc: svc}
}

// Chat handles POST /v1/ai/chat
// @Summary      Chat con asistente IA
// @Description  Envía un mensaje al asistente IA y recibe una respuesta basada en los datos del negocio.
// @Tags         ai
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body dto.AIChatRequest true "Mensaje del usuario"
// @Success      200 {object} dto.AIChatResponse
// @Failure      400 {object} apierror.APIError
// @Failure      401 {object} apierror.APIError
// @Failure      500 {object} apierror.APIError
// @Router       /v1/ai/chat [post]
func (h *AIHandler) Chat(c *gin.Context) {
	tenantID, err := middleware.TenantIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, apierror.New("tenant context missing"))
		return
	}

	var req dto.AIChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("mensaje requerido (máximo 2000 caracteres)"))
		return
	}

	response, err := h.svc.Chat(c.Request.Context(), tenantID, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.New(err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.AIChatResponse{Response: response})
}

// GetMetricas handles GET /v1/ai/metricas
// @Summary      Métricas con análisis IA
// @Description  Retorna métricas pre-calculadas del negocio junto con un análisis generado por IA.
// @Tags         ai
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} dto.AIMetricasResponse
// @Failure      401 {object} apierror.APIError
// @Failure      500 {object} apierror.APIError
// @Router       /v1/ai/metricas [get]
func (h *AIHandler) GetMetricas(c *gin.Context) {
	tenantID, err := middleware.TenantIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, apierror.New("tenant context missing"))
		return
	}

	resp, err := h.svc.GetMetricas(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.New(err.Error()))
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetStatus handles GET /v1/ai/status
// @Summary      Estado de configuración IA
// @Description  Retorna si la IA está configurada y qué modelo se usa.
// @Tags         ai
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} dto.AIStatusResponse
// @Router       /v1/ai/status [get]
func (h *AIHandler) GetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, dto.AIStatusResponse{
		Configured: h.svc.IsConfigured(),
		Model:      "mistral-small-latest",
	})
}
