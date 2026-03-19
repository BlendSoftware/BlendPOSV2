import type { LocalSale } from '../../offline/db';
import { apiClient } from '../../api/client';

/**
 * Result returned by the backend for each individual sale in a sync-batch.
 */
export interface SyncSaleResult {
    id?: string;
    numero_ticket?: number;
    estado: string;
    conflicto_stock?: boolean;
    /** Echoes the offline_id sent in the request so the client can correlate
     *  results by ID rather than by array position (P2-005). */
    offline_id?: string;
}

/**
 * Transforma una LocalSale (formato frontend) a RegistrarVentaRequest (formato backend).
 * Asegura que items, pagos y campos clave estén en el schema correcto.
 */
function toRegistrarVentaRequest(sale: LocalSale): Record<string, unknown> {
    // Transformar CartItem[] → ItemVentaRequest[]
    // El descuento de cada ítem combina: descuento manual/promo (por ítem) + descuento global del carrito.
    // Se aplican en cascada: subtotal = lineTotal * (1 - perItemPct) * (1 - globalPct)
    // El monto de descuento total = lineTotal - subtotal
    const globalPct = (sale.descuentoGlobal ?? 0) / 100;
    const items = sale.items.map((item) => {
        // Weight items have a composite ID: {productId}__w_{random}
        const isWeightItem = item.id.includes('__w_');
        const realProductId = isWeightItem ? item.id.split('__w_')[0] : item.id;
        const peso = (item as unknown as { peso?: number }).peso;
        const lineTotal = peso != null ? item.precio * peso : item.precio * item.cantidad;
        const perItemPct = Math.max(item.descuento, (item as unknown as { promoDescuento?: number }).promoDescuento ?? 0) / 100;
        const effectiveSubtotal = lineTotal * (1 - perItemPct) * (1 - globalPct);
        const discountAmount = lineTotal - effectiveSubtotal;

        const mapped: Record<string, unknown> = {
            producto_id: realProductId,
            cantidad: item.cantidad,
            descuento: discountAmount > 0 ? +discountAmount.toFixed(2) : 0,
        };
        if (peso != null) {
            mapped.peso = +peso.toFixed(3);
        }
        return mapped;
    });

    // Construir pagos: usar sale.pagos si existe, sino construir desde metodoPago + total
    let pagos: { metodo: string; monto: number }[];
    if (sale.pagos && sale.pagos.length > 0) {
        pagos = sale.pagos.map((p) => ({ metodo: p.metodo, monto: p.monto }));
    } else {
        // Fallback para ventas legacy sin pagos array
        const monto = sale.totalConDescuento ?? sale.total;
        pagos = [{ metodo: sale.metodoPago === 'mixto' ? 'efectivo' : sale.metodoPago, monto }];
    }

    const payload: Record<string, unknown> = {
        sesion_caja_id: sale.sesionCajaId ?? '',
        items,
        pagos,
        offline_id: sale.id,
    };

    // Include cliente_id for fiado (credit) sales
    if (sale.clienteId) {
        payload.cliente_id = sale.clienteId;
    }

    // Include customer email if present (RF-21)
    if (sale.clienteEmail && sale.clienteEmail.trim() !== '') {
        payload.cliente_email = sale.clienteEmail.trim();
    }

    // Include fiscal comprobante fields if present
    const tipoComp = sale.tipoComprobante ?? 'ticket_interno';
    payload.tipo_comprobante = tipoComp;
    if (sale.receptorNombre && sale.receptorNombre.trim() !== '') {
        payload.receptor_nombre = sale.receptorNombre.trim();
    }
    if (sale.receptorDomicilio && sale.receptorDomicilio.trim() !== '') {
        payload.receptor_domicilio = sale.receptorDomicilio.trim();
    }
    if (sale.tipoDocReceptor && sale.nroDocReceptor) {
        payload.tipo_doc_receptor = sale.tipoDocReceptor;
        payload.nro_doc_receptor = sale.nroDocReceptor;
    } else if (tipoComp === 'factura_a' && sale.cuitReceptor) {
        payload.tipo_doc_receptor = 80; // CUIT
        payload.nro_doc_receptor = sale.cuitReceptor;
    } else if (tipoComp !== 'ticket_interno') {
        payload.tipo_doc_receptor = 99; // ConsumidorFinal
        payload.nro_doc_receptor = '0';
    }

    return payload;
}

/**
 * Sends a batch of local sales to the backend for sync.
 * Returns per-sale results so the caller can determine which sales
 * were accepted and which were rejected.
 */
export async function syncSalesBatch(sales: LocalSale[]): Promise<SyncSaleResult[]> {
    const ventas = sales.map(toRegistrarVentaRequest);
    const resp = await apiClient.post<SyncSaleResult[]>('/v1/ventas/sync-batch', { ventas });
    return resp;
}
