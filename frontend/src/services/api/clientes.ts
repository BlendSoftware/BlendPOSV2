// ─────────────────────────────────────────────────────────────────────────────
// Clientes API — Fiado / Cuenta Corriente
// POST /v1/clientes, GET /v1/clientes, GET /v1/clientes/:id,
// PUT /v1/clientes/:id, POST /v1/clientes/:id/pago,
// GET /v1/clientes/:id/movimientos, GET /v1/clientes/deudores
// ─────────────────────────────────────────────────────────────────────────────

import { apiClient } from '../../api/client';

// ── Response Types ────────────────────────────────────────────────────────────

export interface ClienteResponse {
    id: string;
    nombre: string;
    telefono?: string;
    email?: string;
    dni?: string;
    limite_credito: number;
    saldo_deudor: number;
    credito_disponible: number;
    activo: boolean;
    notas?: string;
    created_at: string;
    updated_at: string;
}

export interface ClienteListResponse {
    data: ClienteResponse[];
    total: number;
}

export interface MovimientoCuentaResponse {
    id: string;
    cliente_id: string;
    tipo: 'cargo' | 'pago' | 'ajuste';
    monto: number;
    saldo_posterior: number;
    referencia_id?: string;
    referencia_tipo?: string;
    descripcion?: string;
    created_at: string;
}

export interface MovimientosListResponse {
    data: MovimientoCuentaResponse[];
    total: number;
    page: number;
    limit: number;
}

export interface DeudorResponse {
    id: string;
    nombre: string;
    telefono?: string;
    saldo_deudor: number;
    limite_credito: number;
}

export interface ListDeudoresResponse {
    data: DeudorResponse[];
    total: number;
}

// ── Request Types ─────────────────────────────────────────────────────────────

export interface CrearClienteRequest {
    nombre: string;
    telefono?: string;
    email?: string;
    dni?: string;
    limite_credito: number;
    notas?: string;
}

export interface UpdateClienteRequest {
    nombre?: string;
    telefono?: string;
    email?: string;
    dni?: string;
    limite_credito?: number;
    activo?: boolean;
    notas?: string;
}

export interface RegistrarPagoRequest {
    monto: number;
    descripcion?: string;
}

// ── API Calls ─────────────────────────────────────────────────────────────────

/**
 * POST /v1/clientes  (supervisor, administrador)
 * Crea un nuevo cliente con cuenta corriente.
 */
export async function crearCliente(data: CrearClienteRequest): Promise<ClienteResponse> {
    return apiClient.post<ClienteResponse>('/v1/clientes', data);
}

/**
 * GET /v1/clientes  (cajero, supervisor, administrador)
 * Lista clientes activos con busqueda opcional por nombre.
 */
export async function listarClientes(params?: {
    search?: string;
    page?: number;
    limit?: number;
}): Promise<ClienteListResponse> {
    return apiClient.get<ClienteListResponse>('/v1/clientes', {
        search: params?.search,
        page: params?.page ?? 1,
        limit: params?.limit ?? 50,
    });
}

/**
 * GET /v1/clientes/:id  (cajero, supervisor, administrador)
 * Detalle del cliente con saldo actual.
 */
export async function obtenerCliente(id: string): Promise<ClienteResponse> {
    return apiClient.get<ClienteResponse>(`/v1/clientes/${id}`);
}

/**
 * PUT /v1/clientes/:id  (supervisor, administrador)
 * Actualiza datos del cliente.
 */
export async function actualizarCliente(id: string, data: UpdateClienteRequest): Promise<ClienteResponse> {
    return apiClient.put<ClienteResponse>(`/v1/clientes/${id}`, data);
}

/**
 * POST /v1/clientes/:id/pago  (supervisor, administrador)
 * Registra un pago que reduce el saldo deudor.
 */
export async function registrarPago(id: string, data: RegistrarPagoRequest): Promise<MovimientoCuentaResponse> {
    return apiClient.post<MovimientoCuentaResponse>(`/v1/clientes/${id}/pago`, data);
}

/**
 * GET /v1/clientes/:id/movimientos  (cajero, supervisor, administrador)
 * Historial paginado de movimientos de cuenta corriente.
 */
export async function listarMovimientos(
    id: string,
    params?: { page?: number; limit?: number },
): Promise<MovimientosListResponse> {
    return apiClient.get<MovimientosListResponse>(`/v1/clientes/${id}/movimientos`, {
        page: params?.page ?? 1,
        limit: params?.limit ?? 50,
    });
}

/**
 * GET /v1/clientes/deudores  (cajero, supervisor, administrador)
 * Lista todos los clientes con saldo > 0, ordenados por deuda descendente.
 */
export async function listarDeudores(): Promise<ListDeudoresResponse> {
    return apiClient.get<ListDeudoresResponse>('/v1/clientes/deudores');
}
