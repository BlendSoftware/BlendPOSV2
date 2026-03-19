// ─────────────────────────────────────────────────────────────────────────────
// Sucursal Store — Zustand con persistencia parcial.
//
// Selector global de sucursal para el panel de administración.
// Cuando sucursalId es null → vista consolidada ("Todas las sucursales").
// Solo se persisten sucursalId y sucursalNombre (no la lista completa).
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

    setSucursal: (id: string | null, nombre: string | null) => void;
    fetchSucursales: () => Promise<void>;
}

export const useSucursalStore = create<SucursalState>()(
    persist(
        (set) => ({
            sucursalId: null,
            sucursalNombre: null,
            sucursales: [],

            setSucursal: (id, nombre) => set({ sucursalId: id, sucursalNombre: nombre }),

            fetchSucursales: async () => {
                try {
                    const res = await listarSucursales();
                    set({ sucursales: res.data ?? [] });
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
