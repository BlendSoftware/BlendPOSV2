import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import {
    POS_THEME_PRESETS,
    applyThemeCSSVariables,
    injectThemeFont,
} from '../theme/posThemes';
import type { PosTheme } from '../theme/posThemes';

// ── State interface ─────────────────────────────────────────────────────────

interface PosThemeState {
    activeThemeId: string;
    activeTheme: PosTheme;
    setTheme: (themeId: string) => void;
}

// ── Default theme ───────────────────────────────────────────────────────────

const DEFAULT_THEME = POS_THEME_PRESETS[0]; // Clasico

// ── Store ───────────────────────────────────────────────────────────────────

export const usePosThemeStore = create<PosThemeState>()(
    persist(
        (set) => ({
            activeThemeId: DEFAULT_THEME.id,
            activeTheme: DEFAULT_THEME,

            setTheme: (themeId: string) => {
                const theme = POS_THEME_PRESETS.find((t) => t.id === themeId);
                if (!theme) return;

                // Apply CSS variables and font immediately
                applyThemeCSSVariables(theme);
                injectThemeFont(theme.id);

                set({ activeThemeId: theme.id, activeTheme: theme });
            },
        }),
        {
            name: 'pos-theme',
            // Only persist the theme ID, not the whole object
            partialize: (state) => ({ activeThemeId: state.activeThemeId }),
            onRehydrate: () => {
                return (state) => {
                    if (!state) return;
                    const theme = POS_THEME_PRESETS.find((t) => t.id === state.activeThemeId) ?? DEFAULT_THEME;
                    state.activeTheme = theme;
                    state.activeThemeId = theme.id;
                    // Apply on rehydrate
                    applyThemeCSSVariables(theme);
                    injectThemeFont(theme.id);
                };
            },
        },
    ),
);

// ── Eager apply: set CSS variables immediately on module load ───────────
// This prevents a flash of unstyled content before zustand rehydrates from
// localStorage. The onRehydrate callback will override with the persisted
// theme once async hydration completes.
applyThemeCSSVariables(DEFAULT_THEME);
injectThemeFont(DEFAULT_THEME.id);
