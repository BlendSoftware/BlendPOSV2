package dto

import "github.com/shopspring/decimal"

// ── Analytics / Reportes DTOs ────────────────────────────────────────────────

// VentasResumenResponse is the overall sales summary for a given period.
type VentasResumenResponse struct {
	TotalVentas    decimal.Decimal `json:"total_ventas"`
	CantidadVentas int64           `json:"cantidad_ventas"`
	TicketPromedio decimal.Decimal `json:"ticket_promedio"`
	PeriodoDesde   string          `json:"periodo_desde"`
	PeriodoHasta   string          `json:"periodo_hasta"`
}

// TopProductoResponse represents a single product in the top-selling ranking.
type TopProductoResponse struct {
	ProductoID      string          `json:"producto_id"`
	Nombre          string          `json:"nombre"`
	CantidadVendida int64           `json:"cantidad_vendida"`
	TotalRecaudado  decimal.Decimal `json:"total_recaudado"`
}

// VentasPorMedioPagoResponse breaks down sales by payment method.
type VentasPorMedioPagoResponse struct {
	MedioPago string          `json:"medio_pago"`
	Cantidad  int64           `json:"cantidad"`
	Total     decimal.Decimal `json:"total"`
}

// VentasPorPeriodoResponse groups sales by time bucket (day/week/month).
type VentasPorPeriodoResponse struct {
	Periodo  string          `json:"periodo"`
	Total    decimal.Decimal `json:"total"`
	Cantidad int64           `json:"cantidad"`
}

// ReporteCajeroResponse aggregates sales metrics per cashier for a given period.
type ReporteCajeroResponse struct {
	UsuarioID          string          `json:"usuario_id"`
	NombreCajero       string          `json:"nombre_cajero"`
	TotalVentas        decimal.Decimal `json:"total_ventas"`
	CantidadVentas     int64           `json:"cantidad_ventas"`
	TicketPromedio     decimal.Decimal `json:"ticket_promedio"`
	TotalDescuentos    decimal.Decimal `json:"total_descuentos"`
	CantidadAnulaciones int64          `json:"cantidad_anulaciones"`
	PeriodoDesde       string          `json:"periodo_desde"`
	PeriodoHasta       string          `json:"periodo_hasta"`
}

// ReporteTurnoResponse represents a single cash session (shift) with aggregated sales data.
type ReporteTurnoResponse struct {
	SesionID             string          `json:"sesion_id"`
	CajeroNombre         string          `json:"cajero_nombre"`
	FechaApertura        string          `json:"fecha_apertura"`
	FechaCierre          *string         `json:"fecha_cierre"`
	TotalVentas          decimal.Decimal `json:"total_ventas"`
	CantidadVentas       int64           `json:"cantidad_ventas"`
	Desvio               decimal.Decimal `json:"desvio"`
	DesvioClasificacion  string          `json:"desvio_clasificacion"`
}
