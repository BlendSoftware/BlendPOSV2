import { apiClient } from '../../api/client';

// ── Types ─────────────────────────────────────────────────────────────────────

export interface PlanResponse {
    id: string;
    nombre: string;
    max_terminales: number;
    max_productos: number;
    max_sucursales: number;
    max_usuarios: number;
    precio_mensual: string;
    features: Record<string, boolean>;
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
    tipo_negocio?: string;
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
    total_productos: number;
    total_usuarios: number;
    ultima_venta?: string;
    created_at: string;
}

export interface TenantListResponse {
    tenants: SuperadminTenantListItem[];
    total: number;
    page: number;
    page_size: number;
    total_pages: number;
}

export interface PlanCountDTO {
    plan_nombre: string;
    count: number;
}

export interface SuperadminMetricsResponse {
    total_tenants: number;
    tenants_activos: number;
    total_ventas: number;
    ventas_ultimo_mes: number;
    tenants_por_plan: PlanCountDTO[];
}

export interface TenantListParams {
    page?: number;
    page_size?: number;
    search?: string;
    status?: string;
    plan_id?: string;
}

// ── Preset types ─────────────────────────────────────────────────────────────

export interface PresetCategoryResponse {
    nombre: string;
    product_count: number;
}

export interface PresetResponse {
    tipo_negocio: string;
    label: string;
    total_categorias: number;
    total_productos: number;
    categorias: PresetCategoryResponse[];
}

// ── Public ────────────────────────────────────────────────────────────────────

export async function registerTenant(req: RegisterTenantRequest): Promise<RegisterTenantResponse> {
    return apiClient.post<RegisterTenantResponse>('/v1/public/register', req);
}

export async function listarPlanes(): Promise<PlanResponse[]> {
    return apiClient.get<PlanResponse[]>('/v1/public/planes');
}

export async function listarPresets(): Promise<PresetResponse[]> {
    return apiClient.get<PresetResponse[]>('/v1/public/presets');
}

export async function obtenerPreset(tipo: string): Promise<PresetResponse> {
    return apiClient.get<PresetResponse>(`/v1/public/presets/${tipo}`);
}

// ── Tenant self-service ───────────────────────────────────────────────────────

export async function obtenerTenantActual(): Promise<TenantResponse> {
    return apiClient.get<TenantResponse>('/v1/tenant/me');
}

export async function obtenerPlanActual(): Promise<PlanResponse> {
    return apiClient.get<PlanResponse>('/v1/tenant/plan');
}

// ── Superadmin ────────────────────────────────────────────────────────────────

export async function listarTenants(params?: TenantListParams): Promise<TenantListResponse> {
    const query = new URLSearchParams();
    if (params?.page) query.set('page', String(params.page));
    if (params?.page_size) query.set('page_size', String(params.page_size));
    if (params?.search) query.set('search', params.search);
    if (params?.status) query.set('status', params.status);
    if (params?.plan_id) query.set('plan_id', params.plan_id);
    const qs = query.toString();
    return apiClient.get<TenantListResponse>(`/v1/superadmin/tenants${qs ? '?' + qs : ''}`);
}

export async function obtenerTenantDetalle(tenantId: string): Promise<SuperadminTenantListItem> {
    return apiClient.get<SuperadminTenantListItem>(`/v1/superadmin/tenants/${tenantId}`);
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
