// ─────────────────────────────────────────────────────────────────────────────
// AI API — POST /v1/ai/chat, GET /v1/ai/metricas, GET /v1/ai/status
// ─────────────────────────────────────────────────────────────────────────────

import { apiClient } from '../../api/client';

// ── Response Types ────────────────────────────────────────────────────────────

export interface AIChatResponse {
    response: string;
}

export interface AIProductoMetric {
    nombre: string;
    cantidad: number;
    total: number;
}

export interface AIHoraPico {
    hora: number;
    cantidad: number;
}

export interface AIMetricasResponse {
    ventas_mes_actual: number;
    ventas_mes_anterior: number;
    variacion_porcentaje: number;
    ticket_promedio: number;
    cantidad_ventas: number;
    top_productos: AIProductoMetric[];
    peores_productos: AIProductoMetric[];
    horas_pico: AIHoraPico[];
    alertas_stock: number;
    analisis_ia: string;
}

export interface AIStatusResponse {
    configured: boolean;
    model: string;
}

// ── API Calls ─────────────────────────────────────────────────────────────────

/**
 * POST /v1/ai/chat
 * Send a message to the AI assistant.
 */
export async function chatAI(message: string): Promise<AIChatResponse> {
    return apiClient.post<AIChatResponse>('/v1/ai/chat', { message });
}

/**
 * GET /v1/ai/metricas
 * Get pre-built business metrics with AI analysis.
 */
export async function getMetricasAI(): Promise<AIMetricasResponse> {
    return apiClient.get<AIMetricasResponse>('/v1/ai/metricas');
}

/**
 * GET /v1/ai/status
 * Check if AI is configured on the server.
 */
export async function getAIStatus(): Promise<AIStatusResponse> {
    return apiClient.get<AIStatusResponse>('/v1/ai/status');
}
