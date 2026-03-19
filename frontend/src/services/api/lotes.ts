// ─────────────────────────────────────────────────────────────────────────────
// Lotes API — product lot/batch management + expiry alerts.
// POST   /v1/lotes                    → LoteResponse
// GET    /v1/lotes?producto_id=       → LoteResponse[]
// DELETE /v1/lotes/:id                → void
// GET    /v1/vencimientos/alertas     → AlertaVencimientoResponse[]
// ─────────────────────────────────────────────────────────────────────────────

import { apiClient } from '../../api/client';

// ── Response Types ────────────────────────────────────────────────────────────

export interface LoteResponse {
    id: string;
    producto_id: string;
    producto_nombre: string;
    codigo_lote: string | null;
    fecha_vencimiento: string; // YYYY-MM-DD
    cantidad: number;
    created_at: string;
}

export interface AlertaVencimientoResponse {
    id: string;
    producto_id: string;
    producto_nombre: string;
    codigo_lote: string | null;
    fecha_vencimiento: string; // YYYY-MM-DD
    dias_restantes: number;
    cantidad: number;
    estado: 'vencido' | 'critico' | 'proximo';
}

// ── Request Types ─────────────────────────────────────────────────────────────

export interface CrearLoteRequest {
    producto_id: string;
    codigo_lote?: string;
    fecha_vencimiento: string; // YYYY-MM-DD
    cantidad: number;
}

// ── API Calls ─────────────────────────────────────────────────────────────────

/**
 * POST /v1/lotes  (administrador, supervisor)
 * Crea un nuevo lote para un producto que controla vencimiento.
 */
export async function crearLote(data: CrearLoteRequest): Promise<LoteResponse> {
    return apiClient.post<LoteResponse>('/v1/lotes', data);
}

/**
 * GET /v1/lotes?producto_id=  (administrador, supervisor)
 * Lista los lotes de un producto ordenados por fecha de vencimiento.
 */
export async function listarLotes(productoId: string): Promise<LoteResponse[]> {
    return apiClient.get<LoteResponse[]>('/v1/lotes', { producto_id: productoId });
}

/**
 * DELETE /v1/lotes/:id  (administrador, supervisor)
 * Elimina/da de baja un lote (producto vencido o retirado).
 */
export async function eliminarLote(id: string): Promise<void> {
    return apiClient.delete<void>(`/v1/lotes/${id}`);
}

/**
 * GET /v1/vencimientos/alertas?dias=7  (administrador, supervisor)
 * Retorna lotes que vencen dentro de los próximos N días.
 */
export async function getAlertasVencimiento(dias = 7): Promise<AlertaVencimientoResponse[]> {
    return apiClient.get<AlertaVencimientoResponse[]>('/v1/vencimientos/alertas', { dias });
}
