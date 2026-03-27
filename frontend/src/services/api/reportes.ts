// ─────────────────────────────────────────────────────────────────────────────
// Reportes API — GET /v1/reportes/resumen, top-productos, medios-pago, ventas-periodo
// All endpoints accept an optional sucursal_id query param for branch filtering.
// ─────────────────────────────────────────────────────────────────────────────

import { apiClient } from '../../api/client';

type ReportQueryParams = Record<string, string | number | boolean | undefined | null>;

// ── Response Types ────────────────────────────────────────────────────────────

export interface ResumenResponse {
    total_ventas: number;
    cantidad_ventas: number;
    ticket_promedio: number;
}

export interface TopProductoResponse {
    nombre: string;
    cantidad_vendida: number;
    total_recaudado: number;
}

export interface MedioPagoResponse {
    medio_pago: string;
    cantidad: number;
    total: number;
}

export type Agrupacion = 'dia' | 'semana' | 'mes';

export interface VentaPeriodoResponse {
    periodo: string;
    total: number;
    cantidad: number;
}

export interface CajeroResponse {
    usuario_id: string;
    nombre_cajero: string;
    total_ventas: number;
    cantidad_ventas: number;
    ticket_promedio: number;
    total_descuentos: number;
    cantidad_anulaciones: number;
    periodo_desde: string;
    periodo_hasta: string;
}

export interface TurnoResponse {
    sesion_id: string;
    cajero_nombre: string;
    fecha_apertura: string;
    fecha_cierre: string | null;
    total_ventas: number;
    cantidad_ventas: number;
    desvio: number;
    desvio_clasificacion: string;
}

// ── Helpers ──────────────────────────────────────────────────────────────────

/** Build query params, omitting undefined/null values. */
function buildParams(base: ReportQueryParams, sucursalId?: string): ReportQueryParams {
    if (sucursalId) {
        return { ...base, sucursal_id: sucursalId };
    }
    return base;
}

// ── API Calls ─────────────────────────────────────────────────────────────────

/**
 * GET /v1/reportes/resumen
 * Resumen general: total vendido, cantidad de ventas, ticket promedio.
 */
export async function getResumen(desde: string, hasta: string, sucursalId?: string): Promise<ResumenResponse> {
    return apiClient.get<ResumenResponse>('/v1/reportes/resumen', buildParams({ desde, hasta }, sucursalId));
}

/**
 * GET /v1/reportes/top-productos
 * Ranking de productos más vendidos por recaudación.
 */
export async function getTopProductos(
    desde: string,
    hasta: string,
    limit = 10,
    sucursalId?: string,
): Promise<TopProductoResponse[]> {
    return apiClient.get<TopProductoResponse[]>('/v1/reportes/top-productos', buildParams({
        desde,
        hasta,
        limit,
    }, sucursalId));
}

/**
 * GET /v1/reportes/medios-pago
 * Desglose de ventas por método de pago.
 */
export async function getMediosPago(desde: string, hasta: string, sucursalId?: string): Promise<MedioPagoResponse[]> {
    return apiClient.get<MedioPagoResponse[]>('/v1/reportes/medios-pago', buildParams({ desde, hasta }, sucursalId));
}

/**
 * GET /v1/reportes/ventas-periodo
 * Ventas agrupadas por día, semana o mes.
 */
export async function getVentasPorPeriodo(
    desde: string,
    hasta: string,
    agrupacion: Agrupacion = 'dia',
    sucursalId?: string,
): Promise<VentaPeriodoResponse[]> {
    return apiClient.get<VentaPeriodoResponse[]>('/v1/reportes/ventas-periodo', buildParams({
        desde,
        hasta,
        agrupacion,
    }, sucursalId));
}

/**
 * GET /v1/reportes/cajeros
 * Ventas agrupadas por cajero con métricas.
 */
export async function getCajeros(desde: string, hasta: string, sucursalId?: string): Promise<CajeroResponse[]> {
    return apiClient.get<CajeroResponse[]>('/v1/reportes/cajeros', buildParams({ desde, hasta }, sucursalId));
}

/**
 * GET /v1/reportes/turnos
 * Sesiones de caja con totales y desvío.
 */
export async function getTurnos(desde: string, hasta: string, sucursalId?: string): Promise<TurnoResponse[]> {
    return apiClient.get<TurnoResponse[]>('/v1/reportes/turnos', buildParams({ desde, hasta }, sucursalId));
}
