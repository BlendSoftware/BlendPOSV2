// ─────────────────────────────────────────────────────────────────────────────
// Transferencias & Stock Sucursal API
// GET    /v1/transferencias              → TransferenciaListResponse
// POST   /v1/transferencias              → TransferenciaResponse
// POST   /v1/transferencias/:id/completar → TransferenciaResponse
// POST   /v1/transferencias/:id/rechazar  → TransferenciaResponse
// GET    /v1/stock-sucursal              → StockSucursalListResponse
// POST   /v1/stock-sucursal/ajustar      → StockSucursalResponse
// ─────────────────────────────────────────────────────────────────────────────

import { apiClient } from '../../api/client';

// ── Response Types ──────────────────────────────────────────────────────────

export type EstadoTransferencia = 'pendiente' | 'completada' | 'rechazada' | 'cancelada';

export interface TransferenciaItemResponse {
    id: string;
    producto_id: string;
    producto_nombre: string;
    cantidad: number;
}

export interface TransferenciaResponse {
    id: string;
    sucursal_origen_id: string;
    sucursal_origen_nombre: string;
    sucursal_destino_id: string;
    sucursal_destino_nombre: string;
    estado: EstadoTransferencia;
    notas?: string;
    items: TransferenciaItemResponse[];
    created_at: string;
    updated_at: string;
}

export interface TransferenciaListResponse {
    data: TransferenciaResponse[];
    total: number;
}

export interface StockSucursalResponse {
    id: string;
    sucursal_id: string;
    producto_id: string;
    producto_nombre: string;
    stock_actual: number;
    stock_minimo: number;
}

export interface StockSucursalListResponse {
    data: StockSucursalResponse[];
    total: number;
}

// ── Request Types ───────────────────────────────────────────────────────────

export interface TransferenciaItemRequest {
    producto_id: string;
    cantidad: number;
}

export interface CrearTransferenciaRequest {
    sucursal_origen_id: string;
    sucursal_destino_id: string;
    items: TransferenciaItemRequest[];
    notas?: string;
}

export interface AjustarStockSucursalRequest {
    sucursal_id: string;
    producto_id: string;
    delta: number;
    motivo: string;
}

// ── API Calls ───────────────────────────────────────────────────────────────

/** GET /v1/transferencias */
export async function listarTransferencias(estado?: EstadoTransferencia): Promise<TransferenciaListResponse> {
    return apiClient.get<TransferenciaListResponse>('/v1/transferencias', { estado });
}

/** POST /v1/transferencias */
export async function crearTransferencia(data: CrearTransferenciaRequest): Promise<TransferenciaResponse> {
    return apiClient.post<TransferenciaResponse>('/v1/transferencias', data);
}

/** POST /v1/transferencias/:id/completar */
export async function completarTransferencia(id: string): Promise<TransferenciaResponse> {
    return apiClient.post<TransferenciaResponse>(`/v1/transferencias/${id}/completar`, {});
}

/** POST /v1/transferencias/:id/rechazar */
export async function rechazarTransferencia(id: string): Promise<TransferenciaResponse> {
    return apiClient.post<TransferenciaResponse>(`/v1/transferencias/${id}/rechazar`, {});
}

/** GET /v1/stock-sucursal?sucursal_id= */
export async function listarStockSucursal(sucursalId: string): Promise<StockSucursalListResponse> {
    return apiClient.get<StockSucursalListResponse>('/v1/stock-sucursal', { sucursal_id: sucursalId });
}

/** POST /v1/stock-sucursal/ajustar */
export async function ajustarStockSucursal(data: AjustarStockSucursalRequest): Promise<StockSucursalResponse> {
    return apiClient.post<StockSucursalResponse>('/v1/stock-sucursal/ajustar', data);
}
