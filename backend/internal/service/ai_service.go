package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"blendpos/internal/dto"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

const (
	mistralAPIURL = "https://api.mistral.ai/v1/chat/completions"
	mistralModel  = "mistral-small-latest"
)

// AIService exposes AI-powered analytics for tenants.
type AIService interface {
	Chat(ctx context.Context, tenantID uuid.UUID, userMessage string) (string, error)
	GetMetricas(ctx context.Context, tenantID uuid.UUID) (*dto.AIMetricasResponse, error)
	IsConfigured() bool
}

type aiService struct {
	db     *gorm.DB
	apiKey string
}

// NewAIService creates a new AI service. Pass empty apiKey to disable AI features.
func NewAIService(db *gorm.DB, apiKey string) AIService {
	return &aiService{db: db, apiKey: apiKey}
}

func (s *aiService) IsConfigured() bool {
	return s.apiKey != ""
}

// ── Chat ────────────────────────────────────────────────────────────────────

func (s *aiService) Chat(ctx context.Context, tenantID uuid.UUID, userMessage string) (string, error) {
	if s.apiKey == "" {
		return "", fmt.Errorf("AI no configurada: falta la variable MISTRAL_API_KEY en el servidor")
	}

	// Build context with recent business metrics
	metricsCtx, err := s.buildMetricsContext(ctx, tenantID)
	if err != nil {
		log.Warn().Err(err).Str("tenant_id", tenantID.String()).Msg("ai: failed to build metrics context, proceeding without it")
		metricsCtx = "(no se pudieron obtener métricas del negocio)"
	}

	systemPrompt := fmt.Sprintf(`Sos un asistente de negocios experto para un punto de venta (POS) en Argentina.
Tu objetivo es ayudar al dueño/administrador del negocio a tomar mejores decisiones basándote en los datos de su negocio.

Reglas:
- Respondé SIEMPRE en español rioplatense (voseo, tuteo argentino).
- Sé conciso pero accionable. Nada de vaguedades.
- Si no tenés datos suficientes, decilo honestamente.
- Usá números concretos cuando los tengas.
- NO ejecutes SQL ni accedas a bases de datos. Solo usá la información que te paso como contexto.
- Formateá las respuestas con markdown para mejor legibilidad.

Contexto actual del negocio:
%s`, metricsCtx)

	return s.callMistral(ctx, systemPrompt, userMessage)
}

// ── GetMetricas ─────────────────────────────────────────────────────────────

func (s *aiService) GetMetricas(ctx context.Context, tenantID uuid.UUID) (*dto.AIMetricasResponse, error) {
	metrics := &dto.AIMetricasResponse{}

	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	startOfLastMonth := startOfMonth.AddDate(0, -1, 0)
	endOfLastMonth := startOfMonth.AddDate(0, 0, -1)

	// Ventas mes actual
	var ventasActual struct {
		Total    float64
		Cantidad int
	}
	s.db.WithContext(ctx).Raw(`
		SELECT COALESCE(SUM(total), 0) as total, COUNT(*) as cantidad
		FROM ventas
		WHERE tenant_id = ? AND estado != 'anulada' AND created_at >= ? AND created_at < ?`,
		tenantID, startOfMonth, now).Scan(&ventasActual)

	metrics.VentasMesActual = ventasActual.Total
	metrics.CantidadVentas = ventasActual.Cantidad
	if ventasActual.Cantidad > 0 {
		metrics.TicketPromedio = ventasActual.Total / float64(ventasActual.Cantidad)
	}

	// Ventas mes anterior
	var ventasAnterior struct {
		Total float64
	}
	s.db.WithContext(ctx).Raw(`
		SELECT COALESCE(SUM(total), 0) as total
		FROM ventas
		WHERE tenant_id = ? AND estado != 'anulada' AND created_at >= ? AND created_at <= ?`,
		tenantID, startOfLastMonth, endOfLastMonth).Scan(&ventasAnterior)

	metrics.VentasMesAnterior = ventasAnterior.Total
	if ventasAnterior.Total > 0 {
		metrics.VariacionPorcentaje = ((ventasActual.Total - ventasAnterior.Total) / ventasAnterior.Total) * 100
	}

	// Top 5 productos
	var topProds []dto.AIProductoMetric
	s.db.WithContext(ctx).Raw(`
		SELECT p.nombre, SUM(di.cantidad) as cantidad, SUM(di.subtotal) as total
		FROM detalle_ventas di
		JOIN ventas v ON v.id = di.venta_id
		JOIN productos p ON p.id = di.producto_id
		WHERE v.tenant_id = ? AND v.estado != 'anulada' AND v.created_at >= ?
		GROUP BY p.nombre
		ORDER BY total DESC
		LIMIT 5`, tenantID, startOfMonth).Scan(&topProds)
	metrics.TopProductos = topProds

	// Peores 5 productos (menos vendidos que tienen ventas)
	var worstProds []dto.AIProductoMetric
	s.db.WithContext(ctx).Raw(`
		SELECT p.nombre, SUM(di.cantidad) as cantidad, SUM(di.subtotal) as total
		FROM detalle_ventas di
		JOIN ventas v ON v.id = di.venta_id
		JOIN productos p ON p.id = di.producto_id
		WHERE v.tenant_id = ? AND v.estado != 'anulada' AND v.created_at >= ?
		GROUP BY p.nombre
		ORDER BY total ASC
		LIMIT 5`, tenantID, startOfMonth).Scan(&worstProds)
	metrics.PeoresProductos = worstProds

	// Horas pico
	var horasPico []dto.AIHoraPico
	s.db.WithContext(ctx).Raw(`
		SELECT EXTRACT(HOUR FROM created_at)::int as hora, COUNT(*) as cantidad
		FROM ventas
		WHERE tenant_id = ? AND estado != 'anulada' AND created_at >= ?
		GROUP BY hora
		ORDER BY cantidad DESC
		LIMIT 5`, tenantID, startOfMonth).Scan(&horasPico)
	metrics.HorasPico = horasPico

	// Alertas de stock (productos con stock <= stock_minimo)
	var alertasStock int64
	s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		FROM productos
		WHERE tenant_id = ? AND activo = true AND stock <= stock_minimo`,
		tenantID).Scan(&alertasStock)
	metrics.AlertasStock = int(alertasStock)

	// Generate AI analysis if configured
	if s.apiKey != "" {
		analysis, err := s.generateAnalysis(ctx, metrics)
		if err != nil {
			log.Warn().Err(err).Str("tenant_id", tenantID.String()).Msg("ai: failed to generate analysis")
			metrics.AnalisisIA = "No se pudo generar el análisis IA en este momento. Intentá de nuevo más tarde."
		} else {
			metrics.AnalisisIA = analysis
		}
	} else {
		metrics.AnalisisIA = "AI no configurada. Configurá la variable MISTRAL_API_KEY en el servidor."
	}

	return metrics, nil
}

// ── Internal helpers ────────────────────────────────────────────────────────

func (s *aiService) buildMetricsContext(ctx context.Context, tenantID uuid.UUID) (string, error) {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	var resumen struct {
		Total    float64
		Cantidad int
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT COALESCE(SUM(total), 0) as total, COUNT(*) as cantidad
		FROM ventas
		WHERE tenant_id = ? AND estado != 'anulada' AND created_at >= ?`,
		tenantID, startOfMonth).Scan(&resumen).Error; err != nil {
		return "", err
	}

	var topProds []struct {
		Nombre   string
		Cantidad int
		Total    float64
	}
	s.db.WithContext(ctx).Raw(`
		SELECT p.nombre, SUM(di.cantidad) as cantidad, SUM(di.subtotal) as total
		FROM detalle_ventas di
		JOIN ventas v ON v.id = di.venta_id
		JOIN productos p ON p.id = di.producto_id
		WHERE v.tenant_id = ? AND v.estado != 'anulada' AND v.created_at >= ?
		GROUP BY p.nombre
		ORDER BY total DESC
		LIMIT 5`, tenantID, startOfMonth).Scan(&topProds)

	var alertasStock int64
	s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM productos
		WHERE tenant_id = ? AND activo = true AND stock <= stock_minimo`,
		tenantID).Scan(&alertasStock)

	ticket := 0.0
	if resumen.Cantidad > 0 {
		ticket = resumen.Total / float64(resumen.Cantidad)
	}

	summary := fmt.Sprintf(`- Ventas este mes: $%.2f (%d transacciones)
- Ticket promedio: $%.2f
- Productos con stock bajo: %d`, resumen.Total, resumen.Cantidad, ticket, alertasStock)

	if len(topProds) > 0 {
		summary += "\n- Top productos:"
		for i, p := range topProds {
			summary += fmt.Sprintf("\n  %d. %s (%d unidades, $%.2f)", i+1, p.Nombre, p.Cantidad, p.Total)
		}
	}

	return summary, nil
}

func (s *aiService) generateAnalysis(ctx context.Context, m *dto.AIMetricasResponse) (string, error) {
	var topStr, worstStr, horasStr string
	for i, p := range m.TopProductos {
		topStr += fmt.Sprintf("%d. %s (%d unidades, $%.2f)\n", i+1, p.Nombre, p.Cantidad, p.Total)
	}
	for i, p := range m.PeoresProductos {
		worstStr += fmt.Sprintf("%d. %s (%d unidades, $%.2f)\n", i+1, p.Nombre, p.Cantidad, p.Total)
	}
	for _, h := range m.HorasPico {
		horasStr += fmt.Sprintf("- %02d:00 hs: %d ventas\n", h.Hora, h.Cantidad)
	}

	prompt := fmt.Sprintf(`Analizá estos datos de mi negocio (kiosco/punto de venta en Argentina) y dame 3-5 insights accionables en español rioplatense.
Sé concreto, usá los números, y sugerí acciones específicas.

Datos del mes actual:
- Ventas totales: $%.2f
- Cantidad de transacciones: %d
- Ticket promedio: $%.2f
- Variación vs mes anterior: %.1f%%

Top 5 productos más vendidos:
%s
5 productos menos vendidos (con ventas):
%s
Horas pico:
%s
Alertas de stock bajo: %d productos`, m.VentasMesActual, m.CantidadVentas, m.TicketPromedio,
		m.VariacionPorcentaje, topStr, worstStr, horasStr, m.AlertasStock)

	systemPrompt := `Sos un analista de negocios experto en retail argentino. Tu trabajo es dar insights accionables basados en datos reales. Respondé en español rioplatense, sé directo y usá formato markdown.`

	return s.callMistral(ctx, systemPrompt, prompt)
}

// ── Mistral API ─────────────────────────────────────────────────────────────

type mistralRequest struct {
	Model    string           `json:"model"`
	Messages []mistralMessage `json:"messages"`
}

type mistralMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type mistralResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (s *aiService) callMistral(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	reqBody := mistralRequest{
		Model: mistralModel,
		Messages: []mistralMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, mistralAPIURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("mistral API call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Error().
			Int("status", resp.StatusCode).
			Str("body", string(body)).
			Msg("mistral API error")
		return "", fmt.Errorf("mistral API returned status %d", resp.StatusCode)
	}

	var mistralResp mistralResponse
	if err := json.Unmarshal(body, &mistralResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(mistralResp.Choices) == 0 {
		return "", fmt.Errorf("mistral API returned no choices")
	}

	return mistralResp.Choices[0].Message.Content, nil
}
