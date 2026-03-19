package dto

// ── AI Chat ─────────────────────────────────────────────────────────────────

// AIChatRequest is the body for POST /v1/ai/chat.
type AIChatRequest struct {
	Message string `json:"message" binding:"required,min=1,max=2000"`
}

// AIChatResponse is the response for POST /v1/ai/chat.
type AIChatResponse struct {
	Response string `json:"response"`
}

// ── AI Métricas ─────────────────────────────────────────────────────────────

// AIMetricasResponse wraps raw business metrics and the AI-generated analysis.
type AIMetricasResponse struct {
	VentasMesActual    float64            `json:"ventas_mes_actual"`
	VentasMesAnterior  float64            `json:"ventas_mes_anterior"`
	VariacionPorcentaje float64           `json:"variacion_porcentaje"`
	TicketPromedio     float64            `json:"ticket_promedio"`
	CantidadVentas     int               `json:"cantidad_ventas"`
	TopProductos       []AIProductoMetric `json:"top_productos"`
	PeoresProductos    []AIProductoMetric `json:"peores_productos"`
	HorasPico          []AIHoraPico       `json:"horas_pico"`
	AlertasStock       int                `json:"alertas_stock"`
	AnalisisIA         string             `json:"analisis_ia"`
}

// AIProductoMetric is a product entry in the AI metrics response.
type AIProductoMetric struct {
	Nombre   string  `json:"nombre"`
	Cantidad int     `json:"cantidad"`
	Total    float64 `json:"total"`
}

// AIHoraPico represents a busy hour with sale count.
type AIHoraPico struct {
	Hora     int `json:"hora"`
	Cantidad int `json:"cantidad"`
}

// AIStatusResponse tells the frontend if AI is configured.
type AIStatusResponse struct {
	Configured bool   `json:"configured"`
	Model      string `json:"model"`
}
