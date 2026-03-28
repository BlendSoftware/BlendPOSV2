// ─────────────────────────────────────────────────────────────────────────────
// Plan Configuration — Subscription tiers & feature gating
// Shared constants that mirror the backend plan seed (migration 000040).
// ─────────────────────────────────────────────────────────────────────────────

/** Feature keys that can be toggled per plan. */
export type Feature =
    | 'multi_sucursal'
    | 'transferencias'
    | 'stock_sucursal'
    | 'usuarios_extra'
    | 'vencimientos'
    | 'proveedores'
    | 'compras'
    | 'facturacion_afip'
    | 'facturacion_ri'
    | 'reportes_avanzados'
    | 'ai_assistant'
    | 'apariencia'
    | 'clientes_management'
    | 'api_access';

export type PlanId = 'basico' | 'profesional' | 'enterprise';

export interface PlanLimits {
    sucursales: number;  // 0 = unlimited
    terminales: number;  // 0 = unlimited
    usuarios: number;    // 0 = unlimited
    productos: number;   // 0 = unlimited
}

export interface PlanConfig {
    id: PlanId;
    name: string;
    features: Feature[];
    limits: PlanLimits;
    color: string;       // Badge color
}

// ── Plan definitions ─────────────────────────────────────────────────────────

export const PLAN_BASICO: PlanConfig = {
    id: 'basico',
    name: 'Basico',
    features: [],
    limits: { sucursales: 1, terminales: 1, usuarios: 1, productos: 500 },
    color: 'gray',
};

export const PLAN_PROFESIONAL: PlanConfig = {
    id: 'profesional',
    name: 'Profesional',
    features: [
        'multi_sucursal', 'transferencias', 'stock_sucursal',
        'usuarios_extra', 'vencimientos', 'proveedores', 'compras',
        'facturacion_afip', 'reportes_avanzados', 'apariencia',
        'clientes_management',
    ],
    limits: { sucursales: 3, terminales: 5, usuarios: 10, productos: 5000 },
    color: 'blue',
};

export const PLAN_ENTERPRISE: PlanConfig = {
    id: 'enterprise',
    name: 'Enterprise',
    features: [
        'multi_sucursal', 'transferencias', 'stock_sucursal',
        'usuarios_extra', 'vencimientos', 'proveedores', 'compras',
        'facturacion_afip', 'facturacion_ri', 'reportes_avanzados',
        'ai_assistant', 'apariencia', 'clientes_management', 'api_access',
    ],
    limits: { sucursales: 0, terminales: 0, usuarios: 0, productos: 0 },
    color: 'yellow',
};

export const ALL_PLANS: PlanConfig[] = [PLAN_BASICO, PLAN_PROFESIONAL, PLAN_ENTERPRISE];

/**
 * Maps a backend plan name (e.g. "Basico", "Profesional", "Enterprise") to
 * one of the known PlanId values. Falls back to 'basico' for unknown names.
 */
export function resolvePlanId(backendNombre: string): PlanId {
    const lower = backendNombre.toLowerCase().normalize('NFD').replace(/[\u0300-\u036f]/g, '');
    if (lower.startsWith('enterprise')) return 'enterprise';
    if (lower.startsWith('profesional') || lower.startsWith('pro')) return 'profesional';
    return 'basico';
}

/**
 * Returns the PlanConfig for a given plan id.
 */
export function getPlanConfig(id: PlanId): PlanConfig {
    return ALL_PLANS.find((p) => p.id === id) ?? PLAN_BASICO;
}

/**
 * Given a feature, returns the minimum plan that includes it.
 */
export function getMinimumPlanForFeature(feature: Feature): PlanConfig {
    if (PLAN_BASICO.features.includes(feature)) return PLAN_BASICO;
    if (PLAN_PROFESIONAL.features.includes(feature)) return PLAN_PROFESIONAL;
    return PLAN_ENTERPRISE;
}

/**
 * Maps a nav path to the Feature required to access it, or null if always available.
 */
export const NAV_FEATURE_MAP: Record<string, Feature> = {
    '/admin/transferencias': 'transferencias',
    '/admin/stock-sucursal': 'stock_sucursal',
    '/admin/vencimientos': 'vencimientos',
    '/admin/proveedores': 'proveedores',
    '/admin/compras': 'compras',
    '/admin/facturacion': 'facturacion_afip',
    '/admin/ai': 'ai_assistant',
    '/admin/clientes': 'clientes_management',
    '/admin/apariencia': 'apariencia',
};
