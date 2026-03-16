import { apiClient } from '../../api/client';

// ── Types ─────────────────────────────────────────────────────────────────────

export interface PlanResponse {
    id: string;
    nombre: string;
    max_terminales: number;
    max_productos: number;
    precio_mensual: string;
}

export interface TenantResponse {
    id: string;
    slug: string;
    nombre: string;
    cuit?: string;
    activo: boolean;
    plan?: PlanResponse;
    created_at: string;
}

export interface RegisterTenantRequest {
    nombre_negocio: string;
    slug: string;
    username: string;
    password: string;
    nombre: string;
    email?: string;
}

export interface RegisterTenantResponse {
    tenant: TenantResponse;
    access_token: string;
    refresh_token: string;
    token_type: string;
    expires_in: number;
}

export interface SuperadminTenantListItem {
    id: string;
    slug: string;
    nombre: string;
    cuit?: string;
    activo: boolean;
    plan?: PlanResponse;
    total_ventas: number;
    total_usuarios: number;
    created_at: string;
}

export interface SuperadminMetricsResponse {
    total_tenants: number;
    tenants_activos: number;
}

// ── Public ────────────────────────────────────────────────────────────────────

export async function registerTenant(req: RegisterTenantRequest): Promise<RegisterTenantResponse> {
    return apiClient.post<RegisterTenantResponse>('/v1/public/register', req);
}

export async function listarPlanes(): Promise<PlanResponse[]> {
    return apiClient.get<PlanResponse[]>('/v1/public/planes');
}

// ── Tenant self-service ───────────────────────────────────────────────────────

export async function obtenerTenantActual(): Promise<TenantResponse> {
    return apiClient.get<TenantResponse>('/v1/tenant/me');
}

export async function obtenerPlanActual(): Promise<PlanResponse> {
    return apiClient.get<PlanResponse>('/v1/tenant/plan');
}

// ── Superadmin ────────────────────────────────────────────────────────────────

export async function listarTenants(): Promise<SuperadminTenantListItem[]> {
    return apiClient.get<SuperadminTenantListItem[]>('/v1/superadmin/tenants');
}

export async function cambiarPlan(tenantId: string, planId: string): Promise<TenantResponse> {
    return apiClient.put<TenantResponse>(`/v1/superadmin/tenants/${tenantId}/plan`, { plan_id: planId });
}

export async function toggleTenantActivo(tenantId: string, activo: boolean): Promise<TenantResponse> {
    return apiClient.put<TenantResponse>(`/v1/superadmin/tenants/${tenantId}`, { activo });
}

export async function obtenerMetricasGlobales(): Promise<SuperadminMetricsResponse> {
    return apiClient.get<SuperadminMetricsResponse>('/v1/superadmin/metrics');
}
