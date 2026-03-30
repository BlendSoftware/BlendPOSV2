// ─────────────────────────────────────────────────────────────────────────────
// Sucursal Store — Zustand con persistencia parcial.
//
// Selector global de sucursal para el panel de administración.
// Cuando sucursalId es null → vista consolidada ("Todas las sucursales").
// Solo se persisten sucursalId y sucursalNombre (no la lista completa).
//
// When switching sucursal, subscribers to `_switchCounter` are notified so
// pages can refetch their data (sales, caja, stock, etc.).
// ─────────────────────────────────────────────────────────────────────────────

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { listarSucursales, type SucursalResponse } from '../services/api/sucursales';

interface SucursalState {
    /** ID de la sucursal seleccionada. null = "Todas" (vista consolidada) */
    sucursalId: string | null;
    /** Nombre de la sucursal seleccionada. null cuando es "Todas" */
    sucursalNombre: string | null;
    /** Lista cacheada de sucursales del tenant */
    sucursales: SucursalResponse[];
    /**
     * Monotonically incrementing counter. Bumped on every sucursal switch.
     * Components that need to refetch when the branch changes should include
     * this in their useEffect dependency array.
     */
    _switchCounter: number;

    setSucursal: (id: string | null, nombre: string | null) => void;
    fetchSucursales: () => Promise<void>;
}

export const useSucursalStore = create<SucursalState>()(
    persist(
        (set, get) => ({
            sucursalId: null,
            sucursalNombre: null,
            sucursales: [],
            _switchCounter: 0,

            setSucursal: (id, nombre) => {
                const prev = get().sucursalId;
                set({ sucursalId: id, sucursalNombre: nombre });
                // Bump counter only when the value actually changed
                if (prev !== id) {
                    set((s) => ({ _switchCounter: s._switchCounter + 1 }));
                }
            },

            fetchSucursales: async () => {
                try {
                    const res = await listarSucursales();
                    const sucursales = res.data ?? [];
                    set({ sucursales });
                    // If only one sucursal exists and none is selected, auto-select it
                    const { sucursalId } = get();
                    if (!sucursalId && sucursales.length === 1) {
                        get().setSucursal(sucursales[0].id, sucursales[0].nombre);
                    }
                    // If the previously selected sucursal no longer exists, clear it
                    if (sucursalId && !sucursales.find((s) => s.id === sucursalId)) {
                        if (sucursales.length === 1) {
                            get().setSucursal(sucursales[0].id, sucursales[0].nombre);
                        } else {
                            get().setSucursal(null, null);
                        }
                    }
                } catch {
                    set({ sucursales: [] });
                }
            },
        }),
        {
            name: 'blendpos-sucursal',
            partialize: (state) => ({
                sucursalId: state.sucursalId,
                sucursalNombre: state.sucursalNombre,
            }),
        },
    ),
);
