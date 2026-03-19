// ─────────────────────────────────────────────────────────────────────────────
// Sucursales API — gestión de sucursales.
// GET    /v1/sucursales           → SucursalListResponse
// POST   /v1/sucursales           → SucursalResponse
// PUT    /v1/sucursales/:id       → SucursalResponse
// DELETE /v1/sucursales/:id       → 204
// ─────────────────────────────────────────────────────────────────────────────

import { apiClient } from '../../api/client';

// ── Response Types ────────────────────────────────────────────────────────────

export interface SucursalResponse {
    id: string;
    nombre: string;
    direccion?: string;
    telefono?: string;
    activa: boolean;
    created_at: string;
    updated_at: string;
}

export interface SucursalListResponse {
    data: SucursalResponse[];
    total: number;
}

// ── Request Types ─────────────────────────────────────────────────────────────

export interface CrearSucursalRequest {
    nombre: string;
    direccion?: string;
    telefono?: string;
}

export interface UpdateSucursalRequest {
    nombre?: string;
    direccion?: string;
    telefono?: string;
    activa?: boolean;
}

// ── API Calls ─────────────────────────────────────────────────────────────────

/** GET /v1/sucursales  (administrador) */
export async function listarSucursales(): Promise<SucursalListResponse> {
    return apiClient.get<SucursalListResponse>('/v1/sucursales');
}

/** POST /v1/sucursales  (administrador) */
export async function crearSucursal(data: CrearSucursalRequest): Promise<SucursalResponse> {
    return apiClient.post<SucursalResponse>('/v1/sucursales', data);
}

/** PUT /v1/sucursales/:id  (administrador) */
export async function actualizarSucursal(id: string, data: UpdateSucursalRequest): Promise<SucursalResponse> {
    return apiClient.put<SucursalResponse>(`/v1/sucursales/${id}`, data);
}

/** DELETE /v1/sucursales/:id  (administrador) */
export async function eliminarSucursal(id: string): Promise<void> {
    return apiClient.delete<void>(`/v1/sucursales/${id}`);
}
