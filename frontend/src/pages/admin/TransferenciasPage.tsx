import { useState, useEffect, useCallback, useMemo } from 'react';
import {
    Stack, Title, Text, Group, Button, Table, Paper, Badge, Modal,
    Select, NumberInput, Textarea, Skeleton, ActionIcon, Tooltip, Alert,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { notifications } from '@mantine/notifications';
import { Plus, Check, X, ArrowLeftRight, Trash2, AlertTriangle } from 'lucide-react';
import {
    listarTransferencias, crearTransferencia, completarTransferencia, rechazarTransferencia,
    type TransferenciaResponse, type EstadoTransferencia, type TransferenciaItemRequest,
} from '../../services/api/transferencias';
import { listarSucursales, type SucursalResponse } from '../../services/api/sucursales';
import { listarProductos, type ProductoResponse } from '../../services/api/products';

// ── Helpers ─────────────────────────────────────────────────────────────────

const ESTADO_COLOR: Record<EstadoTransferencia, string> = {
    pendiente: 'yellow',
    completada: 'green',
    rechazada: 'red',
    cancelada: 'gray',
};

const ESTADO_OPTIONS = [
    { value: '', label: 'Todas' },
    { value: 'pendiente', label: 'Pendientes' },
    { value: 'completada', label: 'Completadas' },
    { value: 'rechazada', label: 'Rechazadas' },
    { value: 'cancelada', label: 'Canceladas' },
];

interface ItemRow {
    producto_id: string;
    cantidad: number;
}

// ── Component ───────────────────────────────────────────────────────────────

export function TransferenciasPage() {
    // ── State ───────────────────────────────────────────────────────────────
    const [transferencias, setTransferencias] = useState<TransferenciaResponse[]>([]);
    const [sucursales, setSucursales] = useState<SucursalResponse[]>([]);
    const [productos, setProductos] = useState<ProductoResponse[]>([]);
    const [loading, setLoading] = useState(true);
    const [filtroEstado, setFiltroEstado] = useState<string>('');
    const [modalOpen, setModalOpen] = useState(false);
    const [confirmTarget, setConfirmTarget] = useState<{ id: string; action: 'completar' | 'rechazar' } | null>(null);

    // ── Data fetching ───────────────────────────────────────────────────────
    const fetchData = useCallback(async () => {
        setLoading(true);
        try {
            const [transResp, sucResp, prodResp] = await Promise.allSettled([
                listarTransferencias(filtroEstado ? filtroEstado as EstadoTransferencia : undefined),
                listarSucursales(),
                listarProductos({ limit: 500 }),
            ]);
            if (transResp.status === 'fulfilled') setTransferencias(transResp.value.data);
            if (sucResp.status === 'fulfilled') setSucursales(sucResp.value.data);
            if (prodResp.status === 'fulfilled') setProductos(prodResp.value.data);
        } catch {
            notifications.show({ title: 'Error', message: 'No se pudieron cargar las transferencias', color: 'red' });
        } finally {
            setLoading(false);
        }
    }, [filtroEstado]);

    useEffect(() => { fetchData(); }, [fetchData]);

    // ── Form ────────────────────────────────────────────────────────────────
    const [items, setItems] = useState<ItemRow[]>([{ producto_id: '', cantidad: 1 }]);

    const form = useForm({
        initialValues: {
            sucursal_origen_id: '',
            sucursal_destino_id: '',
            notas: '',
        },
        validate: {
            sucursal_origen_id: (v) => (v ? null : 'Seleccioná sucursal origen'),
            sucursal_destino_id: (v, vals) =>
                !v ? 'Seleccioná sucursal destino'
                    : v === vals.sucursal_origen_id ? 'Destino debe ser diferente al origen'
                    : null,
        },
    });

    const sucursalesSelect = useMemo(
        () => sucursales.filter((s) => s.activa).map((s) => ({ value: s.id, label: s.nombre })),
        [sucursales],
    );

    const productosSelect = useMemo(
        () => productos.filter((p) => p.activo).map((p) => ({ value: p.id, label: `${p.nombre} (stock: ${p.stock_actual})` })),
        [productos],
    );

    const destinoOptions = useMemo(
        () => sucursalesSelect.filter((s) => s.value !== form.values.sucursal_origen_id),
        [sucursalesSelect, form.values.sucursal_origen_id],
    );

    const openCreate = () => {
        form.reset();
        setItems([{ producto_id: '', cantidad: 1 }]);
        setModalOpen(true);
    };

    const addItem = () => setItems((prev) => [...prev, { producto_id: '', cantidad: 1 }]);

    const removeItem = (idx: number) => setItems((prev) => prev.filter((_, i) => i !== idx));

    const updateItem = (idx: number, field: keyof ItemRow, value: string | number) => {
        setItems((prev) => prev.map((item, i) => i === idx ? { ...item, [field]: value } : item));
    };

    const handleSubmit = form.onSubmit(async (values) => {
        const validItems = items.filter((it) => it.producto_id && it.cantidad >= 1);
        if (validItems.length === 0) {
            notifications.show({ title: 'Error', message: 'Agregá al menos un producto', color: 'red' });
            return;
        }

        try {
            await crearTransferencia({
                sucursal_origen_id: values.sucursal_origen_id,
                sucursal_destino_id: values.sucursal_destino_id,
                items: validItems as TransferenciaItemRequest[],
                notas: values.notas.trim() || undefined,
            });
            notifications.show({ title: 'Transferencia creada', message: 'La transferencia fue registrada', color: 'teal' });
            setModalOpen(false);
            await fetchData();
        } catch (err) {
            notifications.show({
                title: 'Error',
                message: err instanceof Error ? err.message : 'Error al crear transferencia',
                color: 'red',
            });
        }
    });

    // ── Actions ─────────────────────────────────────────────────────────────
    const handleAction = async () => {
        if (!confirmTarget) return;
        try {
            if (confirmTarget.action === 'completar') {
                await completarTransferencia(confirmTarget.id);
                notifications.show({ title: 'Transferencia completada', message: 'El stock fue transferido', color: 'green' });
            } else {
                await rechazarTransferencia(confirmTarget.id);
                notifications.show({ title: 'Transferencia rechazada', message: 'La transferencia fue rechazada', color: 'orange' });
            }
            setConfirmTarget(null);
            await fetchData();
        } catch (err) {
            notifications.show({
                title: 'Error',
                message: err instanceof Error ? err.message : 'Error al procesar',
                color: 'red',
            });
        }
    };

    // ── Render ──────────────────────────────────────────────────────────────
    return (
        <Stack gap="xl">
            <Group justify="space-between">
                <div>
                    <Title order={2} fw={800}>Transferencias</Title>
                    <Text c="dimmed" size="sm">
                        Movimiento de stock entre sucursales
                    </Text>
                </div>
                <Button leftSection={<Plus size={16} />} onClick={openCreate}>Nueva Transferencia</Button>
            </Group>

            <Select
                placeholder="Filtrar por estado"
                data={ESTADO_OPTIONS}
                value={filtroEstado}
                onChange={(v) => setFiltroEstado(v ?? '')}
                clearable
                style={{ maxWidth: 240 }}
            />

            <Paper radius="md" withBorder style={{ overflow: 'hidden' }}>
                <Table highlightOnHover verticalSpacing="sm">
                    <Table.Thead>
                        <Table.Tr>
                            <Table.Th>Fecha</Table.Th>
                            <Table.Th>Origen</Table.Th>
                            <Table.Th>Destino</Table.Th>
                            <Table.Th>Items</Table.Th>
                            <Table.Th>Estado</Table.Th>
                            <Table.Th>Acciones</Table.Th>
                        </Table.Tr>
                    </Table.Thead>
                    <Table.Tbody>
                        {loading ? (
                            Array.from({ length: 4 }).map((_, i) => (
                                <Table.Tr key={i}>
                                    {[1, 2, 3, 4, 5, 6].map((j) => (
                                        <Table.Td key={j}><Skeleton h={20} radius="sm" /></Table.Td>
                                    ))}
                                </Table.Tr>
                            ))
                        ) : transferencias.length === 0 ? (
                            <Table.Tr>
                                <Table.Td colSpan={6}>
                                    <Text size="sm" c="dimmed" ta="center" py="lg">
                                        No hay transferencias registradas
                                    </Text>
                                </Table.Td>
                            </Table.Tr>
                        ) : transferencias.map((t) => (
                            <Table.Tr key={t.id}>
                                <Table.Td>
                                    <Text size="xs">
                                        {new Date(t.created_at).toLocaleString('es-AR', { dateStyle: 'short', timeStyle: 'short' })}
                                    </Text>
                                </Table.Td>
                                <Table.Td><Text size="sm" fw={500}>{t.sucursal_origen_nombre}</Text></Table.Td>
                                <Table.Td><Text size="sm" fw={500}>{t.sucursal_destino_nombre}</Text></Table.Td>
                                <Table.Td>
                                    <Badge variant="outline" color="gray" size="sm">
                                        {t.items?.length ?? 0} producto{(t.items?.length ?? 0) !== 1 ? 's' : ''}
                                    </Badge>
                                </Table.Td>
                                <Table.Td>
                                    <Badge color={ESTADO_COLOR[t.estado]} size="sm" variant="light">
                                        {t.estado}
                                    </Badge>
                                </Table.Td>
                                <Table.Td>
                                    {t.estado === 'pendiente' ? (
                                        <Group gap={4}>
                                            <Tooltip label="Completar" withArrow>
                                                <ActionIcon
                                                    variant="subtle"
                                                    color="green"
                                                    onClick={() => setConfirmTarget({ id: t.id, action: 'completar' })}
                                                >
                                                    <Check size={15} />
                                                </ActionIcon>
                                            </Tooltip>
                                            <Tooltip label="Rechazar" withArrow>
                                                <ActionIcon
                                                    variant="subtle"
                                                    color="red"
                                                    onClick={() => setConfirmTarget({ id: t.id, action: 'rechazar' })}
                                                >
                                                    <X size={15} />
                                                </ActionIcon>
                                            </Tooltip>
                                        </Group>
                                    ) : (
                                        <Text size="xs" c="dimmed">—</Text>
                                    )}
                                </Table.Td>
                            </Table.Tr>
                        ))}
                    </Table.Tbody>
                </Table>
            </Paper>

            {/* ── Create Modal ────────────────────────────────────────────── */}
            <Modal
                opened={modalOpen}
                onClose={() => setModalOpen(false)}
                title={<Group gap="xs"><ArrowLeftRight size={16} /><Text fw={700}>Nueva Transferencia</Text></Group>}
                size="lg"
                centered
            >
                <form onSubmit={handleSubmit}>
                    <Stack gap="md">
                        <Group grow>
                            <Select
                                label="Sucursal origen"
                                placeholder="Seleccioná origen..."
                                data={sucursalesSelect}
                                searchable
                                {...form.getInputProps('sucursal_origen_id')}
                            />
                            <Select
                                label="Sucursal destino"
                                placeholder="Seleccioná destino..."
                                data={destinoOptions}
                                searchable
                                {...form.getInputProps('sucursal_destino_id')}
                            />
                        </Group>

                        <div>
                            <Group justify="space-between" mb="xs">
                                <Text size="sm" fw={600}>Productos</Text>
                                <Button size="xs" variant="light" leftSection={<Plus size={12} />} onClick={addItem}>
                                    Agregar producto
                                </Button>
                            </Group>
                            <Stack gap="xs">
                                {items.map((item, idx) => (
                                    <Group key={idx} gap="xs" align="flex-end">
                                        <Select
                                            label={idx === 0 ? 'Producto' : undefined}
                                            placeholder="Seleccioná producto..."
                                            data={productosSelect}
                                            searchable
                                            value={item.producto_id}
                                            onChange={(v) => updateItem(idx, 'producto_id', v ?? '')}
                                            style={{ flex: 1 }}
                                        />
                                        <NumberInput
                                            label={idx === 0 ? 'Cantidad' : undefined}
                                            min={1}
                                            value={item.cantidad}
                                            onChange={(v) => updateItem(idx, 'cantidad', typeof v === 'number' ? v : 1)}
                                            style={{ width: 100 }}
                                        />
                                        {items.length > 1 && (
                                            <ActionIcon variant="subtle" color="red" onClick={() => removeItem(idx)}>
                                                <Trash2 size={14} />
                                            </ActionIcon>
                                        )}
                                    </Group>
                                ))}
                            </Stack>
                        </div>

                        <Textarea
                            label="Notas (opcional)"
                            placeholder="Observaciones sobre la transferencia..."
                            {...form.getInputProps('notas')}
                        />

                        <Group justify="flex-end" mt="sm">
                            <Button variant="subtle" onClick={() => setModalOpen(false)}>Cancelar</Button>
                            <Button type="submit" leftSection={<ArrowLeftRight size={14} />}>Crear Transferencia</Button>
                        </Group>
                    </Stack>
                </form>
            </Modal>

            {/* ── Confirm Action Modal ────────────────────────────────────── */}
            <Modal
                opened={!!confirmTarget}
                onClose={() => setConfirmTarget(null)}
                title={
                    <Text fw={700} c={confirmTarget?.action === 'completar' ? 'green' : 'red'}>
                        {confirmTarget?.action === 'completar' ? 'Completar Transferencia' : 'Rechazar Transferencia'}
                    </Text>
                }
                size="sm"
                centered
            >
                <Stack gap="md">
                    <Alert
                        color={confirmTarget?.action === 'completar' ? 'green' : 'red'}
                        variant="light"
                        icon={<AlertTriangle size={14} />}
                    >
                        {confirmTarget?.action === 'completar'
                            ? 'Se va a completar la transferencia y mover el stock entre sucursales. Esta acción no se puede deshacer.'
                            : 'Se va a rechazar la transferencia. El stock no será modificado.'}
                    </Alert>
                    <Group justify="flex-end">
                        <Button variant="subtle" onClick={() => setConfirmTarget(null)}>Cancelar</Button>
                        <Button
                            color={confirmTarget?.action === 'completar' ? 'green' : 'red'}
                            onClick={handleAction}
                        >
                            {confirmTarget?.action === 'completar' ? 'Completar' : 'Rechazar'}
                        </Button>
                    </Group>
                </Stack>
            </Modal>
        </Stack>
    );
}
