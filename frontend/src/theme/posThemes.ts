// ── POS Theme System — Type definitions + preset themes ─────────────────────

export interface PosThemeColors {
    background: string;
    surface: string;
    surfaceHover: string;
    primary: string;
    primaryHover: string;
    text: string;
    textSecondary: string;
    border: string;
    success: string;
    danger: string;
    warning: string;
}

export interface PosThemeFont {
    family: string;
    heading: string;
    mono: string;
}

export type PosThemeStyle = 'sharp' | 'rounded' | 'soft';

export interface PosTheme {
    id: string;
    name: string;
    description: string;
    colors: PosThemeColors;
    font: PosThemeFont;
    borderRadius: string;
    style: PosThemeStyle;
}

// ── Preset themes ───────────────────────────────────────────────────────────

export const POS_THEME_PRESETS: PosTheme[] = [
    {
        id: 'clasico',
        name: 'Clasico',
        description: 'El tema original de BlendPOS. Fondo oscuro con acentos azul electrico.',
        colors: {
            background: '#020617',
            surface: '#0d1526',
            surfaceHover: '#131d33',
            primary: '#2563eb',
            primaryHover: '#1d4ed8',
            text: '#f8fafc',
            textSecondary: 'rgba(148,163,184,0.85)',
            border: 'rgba(255,255,255,0.09)',
            success: '#22c55e',
            danger: '#ef4444',
            warning: '#f59e0b',
        },
        font: {
            family: "'Manrope', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
            heading: "'Manrope', sans-serif",
            mono: "'Manrope', monospace",
        },
        borderRadius: '10px',
        style: 'rounded',
    },
    {
        id: 'moderno-claro',
        name: 'Moderno Claro',
        description: 'Interfaz luminosa con acentos esmeralda. Ideal para locales bien iluminados.',
        colors: {
            background: '#fafaf9',
            surface: '#ffffff',
            surfaceHover: '#f5f5f4',
            primary: '#059669',
            primaryHover: '#047857',
            text: '#1c1917',
            textSecondary: 'rgba(87,83,78,0.75)',
            border: 'rgba(5,150,105,0.15)',
            success: '#16a34a',
            danger: '#dc2626',
            warning: '#d97706',
        },
        font: {
            family: "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
            heading: "'Inter', sans-serif",
            mono: "'JetBrains Mono', 'Fira Code', monospace",
        },
        borderRadius: '12px',
        style: 'rounded',
    },
    {
        id: 'neon',
        name: 'Neon',
        description: 'Estetica cyberpunk con acentos neon. Para los que quieren algo distinto.',
        colors: {
            background: '#0a0a0a',
            surface: '#141414',
            surfaceHover: '#1f1f1f',
            primary: '#00ff88',
            primaryHover: '#00cc6e',
            text: '#e0ffe0',
            textSecondary: 'rgba(160,255,200,0.6)',
            border: 'rgba(0,255,136,0.12)',
            success: '#00ff88',
            danger: '#ff3366',
            warning: '#ffaa00',
        },
        font: {
            family: "'JetBrains Mono', 'Fira Code', 'Courier New', monospace",
            heading: "'JetBrains Mono', monospace",
            mono: "'JetBrains Mono', monospace",
        },
        borderRadius: '4px',
        style: 'sharp',
    },
    {
        id: 'minimalista',
        name: 'Minimalista',
        description: 'Blanco puro, bordes sutiles, tipografia del sistema. Menos es mas.',
        colors: {
            background: '#ffffff',
            surface: '#fafafa',
            surfaceHover: '#f5f5f5',
            primary: '#18181b',
            primaryHover: '#27272a',
            text: '#09090b',
            textSecondary: 'rgba(113,113,122,0.8)',
            border: 'rgba(0,0,0,0.08)',
            success: '#16a34a',
            danger: '#dc2626',
            warning: '#ca8a04',
        },
        font: {
            family: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
            heading: "-apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
            mono: "ui-monospace, 'SF Mono', monospace",
        },
        borderRadius: '6px',
        style: 'sharp',
    },
    {
        id: 'calido',
        name: 'Calido',
        description: 'Tonos ambar y marron oscuro. Sensacion acogedora y calida.',
        colors: {
            background: '#1c1210',
            surface: '#271c18',
            surfaceHover: '#332520',
            primary: '#f97316',
            primaryHover: '#ea580c',
            text: '#fef3c7',
            textSecondary: 'rgba(253,230,138,0.65)',
            border: 'rgba(249,115,22,0.15)',
            success: '#22c55e',
            danger: '#ef4444',
            warning: '#fbbf24',
        },
        font: {
            family: "'Nunito', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
            heading: "'Nunito', sans-serif",
            mono: "'Fira Code', monospace",
        },
        borderRadius: '14px',
        style: 'soft',
    },
    {
        id: 'profesional',
        name: 'Profesional',
        description: 'Gris pizarra con acentos indigo. Bordes definidos, aspecto corporativo.',
        colors: {
            background: '#1e293b',
            surface: '#273449',
            surfaceHover: '#314159',
            primary: '#6366f1',
            primaryHover: '#4f46e5',
            text: '#f1f5f9',
            textSecondary: 'rgba(203,213,225,0.7)',
            border: 'rgba(99,102,241,0.18)',
            success: '#34d399',
            danger: '#f87171',
            warning: '#fbbf24',
        },
        font: {
            family: "'IBM Plex Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
            heading: "'IBM Plex Sans', sans-serif",
            mono: "'IBM Plex Mono', monospace",
        },
        borderRadius: '4px',
        style: 'sharp',
    },
];

// ── Google Fonts URLs per theme ─────────────────────────────────────────────

const GOOGLE_FONTS_MAP: Record<string, string> = {
    clasico: 'https://fonts.googleapis.com/css2?family=Manrope:wght@400;500;600;700;800&display=swap',
    'moderno-claro': 'https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&display=swap',
    neon: 'https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700;800&display=swap',
    // minimalista uses system font — no Google Font needed
    calido: 'https://fonts.googleapis.com/css2?family=Nunito:wght@400;500;600;700;800&display=swap',
    profesional: 'https://fonts.googleapis.com/css2?family=IBM+Plex+Sans:wght@400;500;600;700&family=IBM+Plex+Mono:wght@400;500;600;700&display=swap',
};

/**
 * Inject or update the Google Fonts <link> for the given theme.
 * No-op if the theme uses system fonts.
 */
export function injectThemeFont(themeId: string): void {
    const LINK_ID = 'pos-theme-font';
    const url = GOOGLE_FONTS_MAP[themeId];

    // Remove existing link if any
    const existing = document.getElementById(LINK_ID);

    if (!url) {
        existing?.remove();
        return;
    }

    if (existing instanceof HTMLLinkElement && existing.href === url) {
        return; // already loaded
    }

    const link = document.createElement('link');
    link.id = LINK_ID;
    link.rel = 'stylesheet';
    link.href = url;
    existing?.remove();
    document.head.appendChild(link);
}

/**
 * Apply a PosTheme's values as CSS custom properties on the POS wrapper element.
 * If no element is passed, applies to document.documentElement (`:root`).
 */
export function applyThemeCSSVariables(theme: PosTheme, el?: HTMLElement | null): void {
    const target = el ?? document.documentElement;
    const { colors, font, borderRadius } = theme;

    // Derive an RGB value from the primary hex for rgba() usage
    const primaryRgb = hexToRgb(colors.primary);

    target.style.setProperty('--pos-bg', colors.background);
    target.style.setProperty('--pos-surface', colors.surface);
    target.style.setProperty('--pos-surface-2', colors.surfaceHover);
    target.style.setProperty('--pos-primary', colors.primary);
    target.style.setProperty('--pos-primary-hover', colors.primaryHover);
    target.style.setProperty('--pos-primary-rgb', primaryRgb);
    target.style.setProperty('--pos-text', colors.text);
    target.style.setProperty('--pos-text-sub', colors.textSecondary);
    target.style.setProperty('--pos-border', colors.border);
    target.style.setProperty('--pos-border-hi', `rgba(${primaryRgb}, 0.28)`);
    target.style.setProperty('--pos-grid-color', `rgba(${primaryRgb}, 0.03)`);
    target.style.setProperty('--pos-success', colors.success);
    target.style.setProperty('--pos-danger', colors.danger);
    target.style.setProperty('--pos-warning', colors.warning);
    target.style.setProperty('--pos-font', font.family);
    target.style.setProperty('--pos-font-heading', font.heading);
    target.style.setProperty('--pos-font-mono', font.mono);
    target.style.setProperty('--pos-radius', borderRadius);
    // Also set the workspace background for the header/footer
    // Derive workspace as a slightly lighter variant of background
    target.style.setProperty('--pos-workspace', colors.surface);
}

function hexToRgb(hex: string): string {
    const clean = hex.replace('#', '');
    const r = parseInt(clean.substring(0, 2), 16);
    const g = parseInt(clean.substring(2, 4), 16);
    const b = parseInt(clean.substring(4, 6), 16);
    return `${r}, ${g}, ${b}`;
}
