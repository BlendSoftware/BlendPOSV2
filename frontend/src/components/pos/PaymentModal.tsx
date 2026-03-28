import { useState, useEffect, useMemo, useCallback } from 'react';
import {
    Modal, Stack, Text, Group, Button, Divider, Select, NumberInput,
    Badge, Box, Alert, TextInput, Collapse, Loader,
} from '@mantine/core';
import { useMediaQuery } from '@mantine/hooks';
import { CreditCard, Check, X, Wallet, AlertCircle, Mail, Receipt, UserCheck } from 'lucide-react';
import { usePOSUIStore } from '../../store/usePOSUIStore';
import { useCartStore } from '../../store/useCartStore';
import type { MetodoPago, PagoDetalle } from '../../store/useCartStore';
import { useSaleStore } from '../../store/useSaleStore';
import { useConfiguracionFiscal } from '../../hooks/useConfiguracionFiscal';
import { listarClientes, type ClienteResponse } from '../../services/api/clientes';
import styles from './PaymentModal.module.css';
import { formatCurrency } from '../../utils/format';

type TipoComprobante = 'auto' | 'ticket_interno' | 'factura_a' | 'factura_b' | 'factura_c';
type TipoDocumentoReceptor = 'dni' | 'cuit';

const DOCUMENTO_OPTIONS: Array<{ value: TipoDocumentoReceptor; label: string }> = [
    { value: 'dni', label: 'DNI' },
    { value: 'cuit', label: 'CUIT' },
];

export function PaymentModal() {
    const isMobile = useMediaQuery('(max-width: 48em)');
    const isOpen = usePOSUIStore((s) => s.isPaymentModalOpen);
    const closePaymentModal = usePOSUIStore((s) => s.closePaymentModal);
    const tipoComprobanteSeleccionado = usePOSUIStore((s) => s.tipoComprobante);
    const total = useCartStore((s) => s.total);
    const descuentoGlobal = useCartStore((s) => s.descuentoGlobal);
    const totalConDescuento = useCartStore((s) => s.totalConDescuento);
    const cart = useCartStore((s) => s.cart);
    const confirmSale = useSaleStore((s) => s.confirmSale);

    const { config } = useConfiguracionFiscal();

    const [metodoPago, setMetodoPago] = useState<MetodoPago>('efectivo');
    const [montoRecibido, setMontoRecibido] = useState<number | string>('');
    const [mixtoDebito, setMixtoDebito] = useState<number | string>('');
    const [mixtoCredito, setMixtoCredito] = useState<number | string>('');
    const [mixtoQr, setMixtoQr] = useState<number | string>('');
    const [mixtoTransferencia, setMixtoTransferencia] = useState<number | string>('');
    const [clienteEmail, setClienteEmail] = useState('');
    const [tipoComprobante, setTipoComprobante] = useState<TipoComprobante>('auto');
    const [tipoDocumentoReceptor, setTipoDocumentoReceptor] = useState<TipoDocumentoReceptor>('dni');
    const [documentoReceptor, setDocumentoReceptor] = useState('');
    const [nombreReceptor, setNombreReceptor] = useState('');
    const [domicilioReceptor, setDomicilioReceptor] = useState('');

    // Fiado state
    const [fiadoClientes, setFiadoClientes] = useState<ClienteResponse[]>([]);
    const [fiadoClienteId, setFiadoClienteId] = useState<string | null>(null);
    const [fiadoSearch, setFiadoSearch] = useState('');
    const [fiadoLoading, setFiadoLoading] = useState(false);

    // Map ComprobanteModal selection to PaymentModal format
    useEffect(() => {
        if (isOpen) {
            const mapping: Record<string, 'ticket_interno' | 'factura_a' | 'factura_b' | 'factura_c'> = {
                'ticket': 'ticket_interno',
                'factura_a': 'factura_a',
                'factura_b': 'factura_b',
                'factura_c': 'factura_c',
            };
            const mappedTipo = mapping[tipoComprobanteSeleccionado];
            if (mappedTipo) {
                setTipoComprobante(mappedTipo);
            }
        }
    }, [isOpen, tipoComprobanteSeleccionado]);

    // Determine allowed invoice types based on fiscal condition
    const opcionesComprobante = useMemo(() => {
        const baseOptions: Array<{ value: TipoComprobante; label: string }> = [
            { value: 'auto', label: 'Automático' },
            { value: 'ticket_interno', label: 'Ticket' },
            { value: 'factura_c', label: 'Factura C' },
        ];

        // If no config or Monotributista, only allow auto, ticket, and factura_c
        if (!config || config.condicion_fiscal === 'Monotributo') {
            return baseOptions;
        }

        // If Responsable Inscripto, allow all types
        if (config.condicion_fiscal === 'Responsable Inscripto') {
            return [
                ...baseOptions,
                { value: 'factura_b', label: 'Factura B' },
                { value: 'factura_a', label: 'Factura A' },
            ];
        }

        return baseOptions;
    }, [config]);

    // Fiado: fetch clients when fiado is selected
    const fetchFiadoClientes = useCallback(async (search?: string) => {
        setFiadoLoading(true);
        try {
            const resp = await listarClientes({ search: search || undefined, limit: 20 });
            setFiadoClientes(resp.data);
        } catch {
            setFiadoClientes([]);
        } finally {
            setFiadoLoading(false);
        }
    }, []);

    useEffect(() => {
        if (metodoPago === 'fiado' && isOpen) {
            fetchFiadoClientes(fiadoSearch);
        }
    }, [metodoPago, isOpen, fiadoSearch, fetchFiadoClientes]);

    const selectedFiadoCliente = useMemo(
        () => fiadoClientes.find((c) => c.id === fiadoClienteId) ?? null,
        [fiadoClientes, fiadoClienteId],
    );

    const isEmailValid = clienteEmail === '' || /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(clienteEmail);
    const requiresFiscalBuyerData = tipoComprobante === 'factura_a' || tipoComprobante === 'factura_b' || tipoComprobante === 'factura_c';
    const resolvedDocType: TipoDocumentoReceptor = tipoComprobante === 'factura_a' ? 'cuit' : tipoDocumentoReceptor;
    const normalizedDocumento = documentoReceptor.replace(/\D/g, '');
    const isDocumentoValid = !requiresFiscalBuyerData || (
        resolvedDocType === 'cuit'
            ? /^\d{11}$/.test(normalizedDocumento)
            : /^\d{7,8}$/.test(normalizedDocumento)
    );
    const isNombreValid = !requiresFiscalBuyerData || nombreReceptor.trim().length >= 3;
    const isDomicilioValid = !requiresFiscalBuyerData || domicilioReceptor.trim().length >= 5;

    const toNumber = (val: number | string): number =>
        (typeof val === 'string' ? parseFloat(val) || 0 : val) || 0;

    const finalTotal = descuentoGlobal > 0 ? totalConDescuento : total;
    const itemCount = cart.reduce((sum, item) => sum + item.cantidad, 0);

    const fiadoCreditoDisponible = selectedFiadoCliente
        ? (selectedFiadoCliente.limite_credito > 0
            ? selectedFiadoCliente.credito_disponible
            : Infinity)
        : 0;
    const fiadoExceedsCredit = metodoPago === 'fiado' && selectedFiadoCliente !== null && finalTotal > fiadoCreditoDisponible;

    const isRecibidoVacio = (metodoPago === 'efectivo' || metodoPago === 'mixto') && montoRecibido === '';

    const numericRecibido = toNumber(montoRecibido);

    const nonCashTotal = metodoPago === 'mixto'
        ? (toNumber(mixtoDebito) + toNumber(mixtoCredito) + toNumber(mixtoQr) + toNumber(mixtoTransferencia))
        : 0;

    const cashDue = metodoPago === 'mixto' ? (finalTotal - nonCashTotal) : 0;

    const efectivoRecibido = metodoPago === 'efectivo'
        ? (isRecibidoVacio ? finalTotal : numericRecibido)
        : metodoPago === 'mixto'
            ? (cashDue > 0 ? (isRecibidoVacio ? cashDue : numericRecibido) : 0)
            : 0;

    const vuelto = metodoPago === 'efectivo'
        ? (efectivoRecibido - finalTotal)
        : metodoPago === 'mixto'
            ? (cashDue > 0 ? (efectivoRecibido - cashDue) : 0)
            : null;

    const canConfirm = (() => {
        // Validar datos fiscales del receptor si es factura A/B/C
        if (requiresFiscalBuyerData && (!isDocumentoValid || !isNombreValid || !isDomicilioValid)) {
            return false;
        }

        // Fiado: require client selected and credit not exceeded
        if (metodoPago === 'fiado') {
            return !!fiadoClienteId && !fiadoExceedsCredit;
        }

        if (metodoPago === 'efectivo') return efectivoRecibido >= finalTotal;
        if (metodoPago !== 'mixto') return true;

        // No permitir que los pagos no-efectivo superen el total.
        if (cashDue < 0) return false;
        if (cashDue === 0) return nonCashTotal === finalTotal;
        return efectivoRecibido >= cashDue;
    })();

    // Reset form when modal opens
    useEffect(() => {
        if (isOpen) {
            setMetodoPago('efectivo');
            setMontoRecibido('');
            setMixtoDebito('');
            setMixtoCredito('');
            setMixtoQr('');
            setNombreReceptor('');
            setMixtoTransferencia('');
            setFiadoClienteId(null);
            setFiadoSearch('');
            setFiadoClientes([]);
            setClienteEmail('');
            setTipoComprobante('auto');
            setTipoDocumentoReceptor('dni');
            setDocumentoReceptor('');
            setDomicilioReceptor('');
        }
    }, [isOpen]);

    // Guard against double-submit (rapid clicks / Enter + click)
    const [isSubmitting, setIsSubmitting] = useState(false);

    // Reset submitting guard when modal opens
    useEffect(() => {
        if (isOpen) setIsSubmitting(false);
    }, [isOpen]);

    const handleConfirmPayment = useCallback(() => {
        if (!canConfirm || isSubmitting) return;
        setIsSubmitting(true);

        let pagos: PagoDetalle[] | undefined;
        let vueltoCalc: number | undefined;
        let efectivoRecibidoToSave: number | undefined;

        if (metodoPago === 'fiado') {
            pagos = [{ metodo: 'fiado' as Exclude<MetodoPago, 'mixto'>, monto: finalTotal }];
        } else if (metodoPago === 'efectivo') {
            efectivoRecibidoToSave = efectivoRecibido;
            vueltoCalc = efectivoRecibido - finalTotal;
        } else if (metodoPago === 'mixto') {
            const deb = toNumber(mixtoDebito);
            const cre = toNumber(mixtoCredito);
            const qr = toNumber(mixtoQr);
            const trans = toNumber(mixtoTransferencia);
            const detalles: PagoDetalle[] = [];

            if (deb > 0) detalles.push({ metodo: 'debito', monto: deb });
            if (cre > 0) detalles.push({ metodo: 'credito', monto: cre });
            if (qr > 0) detalles.push({ metodo: 'qr', monto: qr });
            if (trans > 0) detalles.push({ metodo: 'transferencia', monto: trans });

            if (cashDue > 0) detalles.push({ metodo: 'efectivo', monto: cashDue });

            pagos = detalles;
            if (cashDue > 0) {
                efectivoRecibidoToSave = efectivoRecibido;
                vueltoCalc = efectivoRecibido - cashDue;
            }
        }

        try {
            const record = confirmSale({
                metodoPago,
                pagos,
                efectivoRecibido: efectivoRecibidoToSave,
                vuelto: vueltoCalc,
                clienteEmail: clienteEmail.trim() || undefined,
                receptorNombre: requiresFiscalBuyerData ? nombreReceptor.trim() : undefined,
                tipoComprobante: tipoComprobante === 'auto' ? undefined : tipoComprobante,
                cuitReceptor: requiresFiscalBuyerData && resolvedDocType === 'cuit' ? normalizedDocumento : undefined,
                tipoDocReceptor: requiresFiscalBuyerData
                    ? (resolvedDocType === 'cuit' ? 80 : 96)
                    : undefined,
                nroDocReceptor: requiresFiscalBuyerData ? normalizedDocumento : undefined,
                receptorDomicilio: requiresFiscalBuyerData ? domicilioReceptor.trim() : undefined,
                clienteId: metodoPago === 'fiado' ? fiadoClienteId ?? undefined : undefined,
                clienteNombre: metodoPago === 'fiado' ? selectedFiadoCliente?.nombre : undefined,
            });
            closePaymentModal();

            // Open the post-sale modal for print option
            const { openPostSaleModal } = usePOSUIStore.getState();
            openPostSaleModal(record);
        } finally {
            setIsSubmitting(false);
        }
    }, [
        canConfirm, isSubmitting, metodoPago, finalTotal, efectivoRecibido,
        mixtoDebito, mixtoCredito, mixtoQr, mixtoTransferencia, cashDue,
        clienteEmail, requiresFiscalBuyerData, nombreReceptor, tipoComprobante,
        resolvedDocType, normalizedDocumento, domicilioReceptor,
        fiadoClienteId, selectedFiadoCliente, confirmSale, closePaymentModal,
    ]);

    // Confirm with Enter key when modal is open and not on efectivo input
    useEffect(() => {
        if (!isOpen) return;
        const handler = (e: KeyboardEvent) => {
            if (e.key === 'Enter' && canConfirm && metodoPago !== 'efectivo') {
                e.preventDefault();
                e.stopPropagation();
                handleConfirmPayment();
            }
        };
        window.addEventListener('keydown', handler);
        return () => window.removeEventListener('keydown', handler);
        // All state consumed by handleConfirmPayment must be in deps to avoid stale closures
    }, [isOpen, canConfirm, metodoPago, handleConfirmPayment]);

    return (
        <Modal
            opened={isOpen}
            onClose={closePaymentModal}
            title={
                <Group gap="xs">
                    <CreditCard size={22} />
                    <Text size="lg" fw={700}>
                        Cobrar Venta
                    </Text>
                </Group>
            }
            size="md"
            centered
            fullScreen={isMobile ?? false}
        >
            <Stack gap="lg" className={styles.content}>
                {/* Resumen */}
                <div className={styles.summary}>
                    <Group justify="space-between">
                        <Text size="sm" c="dimmed">Artículos</Text>
                        <Text size="sm" fw={600}>{itemCount}</Text>
                    </Group>

                    {descuentoGlobal > 0 && (
                        <>
                            <Group justify="space-between" mt={4}>
                                <Text size="sm" c="dimmed">Subtotal</Text>
                                <Text size="sm" td="line-through" c="dimmed" ff="monospace">
                                    {formatCurrency(total)}
                                </Text>
                            </Group>
                            <Group justify="space-between" mt={4}>
                                <Group gap="xs">
                                    <Text size="sm" c="dimmed">Descuento</Text>
                                    <Badge size="xs" color="orange" variant="light">
                                        -{descuentoGlobal}%
                                    </Badge>
                                </Group>
                                <Text size="sm" fw={500} c="orange.4" ff="monospace">
                                    - {formatCurrency(total - totalConDescuento)}
                                </Text>
                            </Group>
                        </>
                    )}

                    <Divider my="xs" />
                    <Group justify="space-between">
                        <Text size="lg" fw={700}>TOTAL</Text>
                        <Text size="xl" fw={800} className={styles.modalTotal} ff="monospace">
                            {formatCurrency(finalTotal)}
                        </Text>
                    </Group>
                </div>

                {/* Tipo de comprobante */}
                <Box className={styles.comprobanteBox}>
                    <Stack gap="sm">
                        <Group gap="xs">
                            <Receipt size={18} />
                            <Text size="sm" fw={600}>Tipo de comprobante</Text>
                        </Group>
                        {config && config.condicion_fiscal === 'Monotributo' && (
                            <Alert icon={<AlertCircle size={14} />} color="orange" variant="light" p="xs">
                                <Text size="xs">
                                    ⚠️ Como Monotributista, solo podés emitir Tickets internos o Facturas C
                                </Text>
                            </Alert>
                        )}
                        <Select
                            label="Comprobante"
                            placeholder="Elegi el comprobante"
                            value={tipoComprobante}
                            onChange={(value) => setTipoComprobante((value as TipoComprobante | null) ?? 'auto')}
                            data={opcionesComprobante}
                            size="sm"
                            comboboxProps={{ position: 'bottom', middlewares: { flip: true, shift: true } }}
                        />
                        <Alert icon={<AlertCircle size={14} />} color="blue" variant="light" p="xs">
                            <Text size="xs">
                                {tipoComprobante === 'auto' 
                                    ? 'Se determinará según tu condición fiscal. Monotributo/Exento -> Factura C. Responsable Inscripto -> Factura B o A.'
                                    : tipoComprobante === 'ticket_interno'
                                    ? 'Comprobante no fiscal. Solo para uso interno.'
                                    : tipoComprobante === 'factura_c'
                                    ? 'Factura C para consumidor final o monotributo. Requiere documento y domicilio del comprador.'
                                    : tipoComprobante === 'factura_b'
                                    ? 'Factura B con IVA incluido. Requiere documento y domicilio del comprador.'
                                    : 'Factura A para Responsable Inscripto. Requiere CUIT y domicilio del comprador.'}
                            </Text>
                        </Alert>
                    </Stack>
                </Box>

                <Collapse in={requiresFiscalBuyerData}>
                    <Stack gap="sm">
                        <Alert icon={<AlertCircle size={14} />} color="cyan" variant="light" p="xs">
                            <Text size="xs" fw={600}>
                                📋 Datos obligatorios del comprador según ARCA (ex-AFIP)
                            </Text>
                        </Alert>

                        {tipoComprobante !== 'factura_a' && (
                            <Select
                                label="Tipo de documento"
                                value={tipoDocumentoReceptor}
                                onChange={(value) => setTipoDocumentoReceptor((value as TipoDocumentoReceptor | null) ?? 'dni')}
                                data={DOCUMENTO_OPTIONS}
                                size="sm"
                            />
                        )}

                        <TextInput
                            label={resolvedDocType === 'cuit' ? 'CUIT del comprador' : 'DNI del comprador'}
                            description={resolvedDocType === 'cuit'
                                ? '11 dígitos sin guiones'
                                : '7 u 8 dígitos sin puntos'}
                            placeholder={resolvedDocType === 'cuit' ? '20123456789' : '30123456'}
                            value={documentoReceptor}
                            onChange={(e) => setDocumentoReceptor(e.currentTarget.value.replace(/\D/g, '').slice(0, resolvedDocType === 'cuit' ? 11 : 8))}
                            error={documentoReceptor && !isDocumentoValid
                                ? resolvedDocType === 'cuit'
                                    ? 'El CUIT debe tener 11 dígitos'
                                    : 'El DNI debe tener 7 u 8 dígitos'
                                : undefined}
                            size="sm"
                            required
                        />

                        <TextInput
                            label={resolvedDocType === 'cuit' ? 'Razón Social' : 'Nombre completo del comprador'}
                            description="Nombre/apellido o razón social como aparece en ARCA"
                            placeholder={resolvedDocType === 'cuit' ? 'EMPRESA SA' : 'Juan Pérez'}
                            value={nombreReceptor}
                            onChange={(e) => setNombreReceptor(e.currentTarget.value)}
                            error={nombreReceptor && !isNombreValid ? 'Ingresá un nombre válido (mínimo 3 caracteres)' : undefined}
                            size="sm"
                            required
                        />

                        <TextInput
                            label="Domicilio del comprador"
                            description="Domicilio fiscal completo"
                            placeholder="Av. Siempre Viva 742, Springfield"
                            value={domicilioReceptor}
                            onChange={(e) => setDomicilioReceptor(e.currentTarget.value)}
                            error={domicilioReceptor && !isDomicilioValid ? 'Ingresá un domicilio válido (mínimo 5 caracteres)' : undefined}
                            size="sm"
                            required
                        />
                    </Stack>
                </Collapse>

                {/* Método de pago */}
                <Select
                    label="Método de pago"
                    placeholder="Seleccioná un método"
                    value={metodoPago}
                    onChange={(val) => setMetodoPago((val as MetodoPago) ?? 'efectivo')}
                    data={[
                        { value: 'efectivo', label: '💵 Efectivo' },
                        { value: 'debito', label: '💳 Tarjeta de Débito' },
                        { value: 'credito', label: '💳 Tarjeta de Crédito' },
                        { value: 'qr', label: '📱 QR' },
                        { value: 'transferencia', label: '🔁 Transferencia' },
                        { value: 'fiado', label: '📒 Fiado (Cuenta Corriente)' },
                        { value: 'mixto', label: '🧾 Mixto' },
                    ]}
                    data-pos-focusable
                    size="md"
                />

                {/* Efectivo: monto recibido + vuelto */}
                {metodoPago === 'efectivo' && (
                    <Stack gap="sm">
                        <NumberInput
                            label="Monto recibido"
                            placeholder={formatCurrency(finalTotal)}
                            value={montoRecibido}
                            onChange={setMontoRecibido}
                            min={0}
                            prefix="$ "
                            thousandSeparator="."
                            decimalSeparator=","
                            decimalScale={2}
                            size="md"
                            leftSection={<Wallet size={16} />}
                            data-pos-focusable
                            autoFocus
                            onKeyDown={(e) => {
                                if (e.key === 'Enter' && canConfirm) { e.stopPropagation(); handleConfirmPayment(); }
                                if (e.key === 'Escape') { e.stopPropagation(); closePaymentModal(); }
                            }}
                        />

                        <Group gap={6}>
                            {[1000, 2000, 5000, 10000, 20000].map((m) => (
                                <Button key={m} variant="light" size="compact-xs" onClick={() => setMontoRecibido(m)}>
                                    {formatCurrency(m)}
                                </Button>
                            ))}
                            <Button variant="light" size="compact-xs" color="teal" onClick={() => setMontoRecibido(finalTotal)}>
                                Exacto
                            </Button>
                        </Group>

                        {vuelto !== null && vuelto >= 0 && (
                            <Box className={styles.vueltoBox}>
                                <Group justify="space-between">
                                    <Text size="md" fw={700}>Vuelto</Text>
                                    <Text size="xl" fw={800} c="teal.4" ff="monospace">
                                        {formatCurrency(vuelto)}
                                    </Text>
                                </Group>
                            </Box>
                        )}

                        {vuelto !== null && vuelto < 0 && (
                            <Alert
                                icon={<AlertCircle size={16} />}
                                color="red"
                                variant="light"
                            >
                                El monto recibido es insuficiente. Faltan{' '}
                                <strong>{formatCurrency(Math.abs(vuelto))}</strong>.
                            </Alert>
                        )}
                    </Stack>
                )}

                {/* Fiado: client selection */}
                {metodoPago === 'fiado' && (
                    <Stack gap="sm">
                        <Select
                            label="Cliente"
                            placeholder="Buscar cliente por nombre..."
                            searchable
                            nothingFoundMessage={fiadoLoading ? 'Buscando...' : 'Sin resultados'}
                            data={fiadoClientes.map((c) => ({
                                value: c.id,
                                label: `${c.nombre}${c.saldo_deudor > 0 ? ` (Debe: ${formatCurrency(c.saldo_deudor)})` : ''}`,
                            }))}
                            value={fiadoClienteId}
                            onChange={setFiadoClienteId}
                            onSearchChange={setFiadoSearch}
                            leftSection={<UserCheck size={16} />}
                            rightSection={fiadoLoading ? <Loader size={16} /> : undefined}
                            size="md"
                            comboboxProps={{ position: 'bottom', middlewares: { flip: true, shift: true } }}
                        />

                        {selectedFiadoCliente && (
                            <Box style={{
                                padding: 'var(--mantine-spacing-sm)',
                                borderRadius: 'var(--mantine-radius-md)',
                                border: '1px solid var(--mantine-color-default-border)',
                                background: 'var(--mantine-color-default-hover)',
                            }}>
                                <Group justify="space-between" mb={4}>
                                    <Text size="sm" fw={600}>{selectedFiadoCliente.nombre}</Text>
                                    {selectedFiadoCliente.saldo_deudor > 0 && (
                                        <Badge color="red" variant="light" size="sm">
                                            Debe: {formatCurrency(selectedFiadoCliente.saldo_deudor)}
                                        </Badge>
                                    )}
                                </Group>
                                <Group gap="xl">
                                    <div>
                                        <Text size="xs" c="dimmed">Límite de crédito</Text>
                                        <Text size="sm" fw={500} ff="monospace">
                                            {selectedFiadoCliente.limite_credito > 0
                                                ? formatCurrency(selectedFiadoCliente.limite_credito)
                                                : 'Sin límite'}
                                        </Text>
                                    </div>
                                    <div>
                                        <Text size="xs" c="dimmed">Disponible</Text>
                                        <Text size="sm" fw={500} ff="monospace" c="blue">
                                            {selectedFiadoCliente.limite_credito > 0
                                                ? formatCurrency(selectedFiadoCliente.credito_disponible)
                                                : 'Ilimitado'}
                                        </Text>
                                    </div>
                                </Group>
                            </Box>
                        )}

                        {fiadoExceedsCredit && (
                            <Alert icon={<AlertCircle size={16} />} color="red" variant="light">
                                El monto de la venta ({formatCurrency(finalTotal)}) excede el credito disponible
                                ({selectedFiadoCliente!.limite_credito > 0
                                    ? formatCurrency(selectedFiadoCliente!.credito_disponible)
                                    : 'sin limite'}).
                            </Alert>
                        )}

                        {!fiadoClienteId && (
                            <Alert icon={<AlertCircle size={16} />} color="yellow" variant="light">
                                Selecciona un cliente para registrar la venta como fiado.
                            </Alert>
                        )}
                    </Stack>
                )}

                {metodoPago === 'mixto' && (
                    <Stack gap="sm">
                        <Group grow>
                            <NumberInput
                                label="Débito"
                                placeholder="$ 0"
                                value={mixtoDebito}
                                onChange={setMixtoDebito}
                                min={0}
                                prefix="$ "
                                thousandSeparator="."
                                decimalSeparator=","
                                decimalScale={2}
                                size="md"
                                data-pos-focusable
                                autoFocus
                            />
                            <NumberInput
                                label="Crédito"
                                placeholder="$ 0"
                                value={mixtoCredito}
                                onChange={setMixtoCredito}
                                min={0}
                                prefix="$ "
                                thousandSeparator="."
                                decimalSeparator=","
                                decimalScale={2}
                                size="md"
                                data-pos-focusable
                            />
                        </Group>

                        <Group grow>
                            <NumberInput
                                label="QR"
                                placeholder="$ 0"
                                value={mixtoQr}
                                onChange={setMixtoQr}
                                min={0}
                                prefix="$ "
                                thousandSeparator="."
                                decimalSeparator=","
                                decimalScale={2}
                                size="md"
                                data-pos-focusable
                            />
                            <NumberInput
                                label="Transferencia"
                                placeholder="$ 0"
                                value={mixtoTransferencia}
                                onChange={setMixtoTransferencia}
                                min={0}
                                prefix="$ "
                                thousandSeparator="."
                                decimalSeparator=","
                                decimalScale={2}
                                size="md"
                                data-pos-focusable
                            />
                        </Group>

                        {cashDue > 0 && (
                            <NumberInput
                                label="Efectivo recibido"
                                placeholder={formatCurrency(cashDue)}
                                value={montoRecibido}
                                onChange={setMontoRecibido}
                                min={0}
                                prefix="$ "
                                thousandSeparator="."
                                decimalSeparator=","
                                decimalScale={2}
                                size="md"
                                leftSection={<Wallet size={16} />}
                                data-pos-focusable
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter' && canConfirm) { e.stopPropagation(); handleConfirmPayment(); }
                                    if (e.key === 'Escape') { e.stopPropagation(); closePaymentModal(); }
                                }}
                            />
                        )}

                        {cashDue < 0 && (
                            <Alert icon={<AlertCircle size={16} />} color="red" variant="light">
                                Los pagos con tarjeta/QR superan el total.
                            </Alert>
                        )}

                        {cashDue > 0 && vuelto !== null && vuelto >= 0 && (
                            <Box className={styles.vueltoBox}>
                                <Group justify="space-between">
                                    <Text size="md" fw={700}>Vuelto</Text>
                                    <Text size="xl" fw={800} c="teal.4" ff="monospace">
                                        {formatCurrency(vuelto)}
                                    </Text>
                                </Group>
                            </Box>
                        )}

                        {cashDue > 0 && vuelto !== null && vuelto < 0 && (
                            <Alert icon={<AlertCircle size={16} />} color="red" variant="light">
                                El efectivo es insuficiente. Faltan{' '}
                                <strong>{formatCurrency(Math.abs(vuelto))}</strong>.
                            </Alert>
                        )}
                    </Stack>
                )}

                {/* Email opcional para recibo digital (RF-21) */}
                <Collapse in>
                    <TextInput
                        label="Email del cliente (opcional)"
                        description="Si se indica, se enviará el comprobante por email."
                        placeholder="cliente@ejemplo.com"
                        value={clienteEmail}
                        onChange={(e) => setClienteEmail(e.currentTarget.value)}
                        leftSection={<Mail size={16} />}
                        error={!isEmailValid ? 'Email inválido' : undefined}
                        size="sm"
                    />
                </Collapse>

                <Group grow mt="xs">
                    <Button
                        variant="outline"
                        color="gray"
                        size="lg"
                        leftSection={<X size={18} />}
                        onClick={closePaymentModal}
                    >
                        Cancelar
                    </Button>
                    <Button
                        color="green"
                        size="lg"
                        leftSection={<Check size={18} />}
                        onClick={handleConfirmPayment}
                        disabled={!canConfirm || !isEmailValid || !isDocumentoValid || !isDomicilioValid}
                    >
                        Confirmar Pago
                    </Button>
                </Group>
            </Stack>
        </Modal>
    );
}
