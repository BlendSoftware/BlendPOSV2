import { useState, useEffect, useCallback } from 'react';
import {
    Stack, Title, Text, Group, Button, Badge, Table,
    Modal, Alert, Paper, Skeleton, SegmentedControl,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { AlertTriangle, Trash2, Clock, XCircle, AlertOctagon } from 'lucide-react';
import {
    getAlertasVencimiento,
    eliminarLote,
    type AlertaVencimientoResponse,
} from '../../services/api/lotes';

// ── Helpers ──────────────────────────────────────────────────────────────────

const ESTADO_CONFIG = {
    vencido: { color: 'red', label: 'Vencido', icon: <XCircle size={14} /> },
    critico: { color: 'orange', label: 'Vence esta semana', icon: <AlertOctagon size={14} /> },
    proximo: { color: 'yellow', label: 'Próximo a vencer', icon: <Clock size={14} /> },
} as const;

function formatDate(dateStr: string): string {
    const [y, m, d] = dateStr.split('-');
    return `${d}/${m}/${y}`;
}

// ── Component ────────────────────────────────────────────────────────────────

export function VencimientosPage() {
    const [alertas, setAlertas] = useState<AlertaVencimientoResponse[]>([]);
    const [loading, setLoading] = useState(true);
    const [diasFiltro, setDiasFiltro] = useState('7');

    // Delete modal
    const [deleteTarget, setDeleteTarget] = useState<AlertaVencimientoResponse | null>(null);
    const [deleteOpen, setDeleteOpen] = useState(false);
    const [deleting, setDeleting] = useState(false);

    const fetchAlertas = useCallback(async () => {
        setLoading(true);
        try {
            const data = await getAlertasVencimiento(Number(diasFiltro));
            setAlertas(data ?? []);
        } catch {
            notifications.show({
                title: 'Error',
                message: 'No se pudieron cargar las alertas de vencimiento',
                color: 'red',
            });
        } finally {
            setLoading(false);
        }
    }, [diasFiltro]);

    useEffect(() => { fetchAlertas(); }, [fetchAlertas]);

    const vencidos = alertas.filter((a) => a.estado === 'vencido');
    const criticos = alertas.filter((a) => a.estado === 'critico');
    const proximos = alertas.filter((a) => a.estado === 'proximo');

    const handleDelete = async () => {
        if (!deleteTarget) return;
        setDeleting(true);
        try {
            await eliminarLote(deleteTarget.id);
            notifications.show({
                title: 'Lote dado de baja',
                message: `${deleteTarget.producto_nombre} — Lote ${deleteTarget.codigo_lote ?? 'sin código'}`,
                color: 'orange',
            });
            setDeleteOpen(false);
            await fetchAlertas();
        } catch (err) {
            notifications.show({
                title: 'Error',
                message: err instanceof Error ? err.message : 'No se pudo eliminar el lote',
                color: 'red',
            });
        } finally {
            setDeleting(false);
        }
    };

    const openDelete = (alerta: AlertaVencimientoResponse) => {
        setDeleteTarget(alerta);
        setDeleteOpen(true);
    };

    // ── Render helper for a section ──────────────────────────────────────────

    const renderSection = (
        titulo: string,
        items: AlertaVencimientoResponse[],
        estado: keyof typeof ESTADO_CONFIG,
    ) => {
        if (items.length === 0) return null;
        const cfg = ESTADO_CONFIG[estado];

        return (
            <Stack gap="sm" key={estado}>
                <Alert
                    color={cfg.color}
                    variant="light"
                    title={`${titulo} (${items.length})`}
                    icon={cfg.icon}
                >
                    {estado === 'vencido'
                        ? 'Estos productos ya vencieron. Retiralos de la venta inmediatamente.'
                        : estado === 'critico'
                            ? 'Estos productos vencen en los próximos 3 días.'
                            : `Estos productos vencen en los próximos ${diasFiltro} días.`}
                </Alert>
                <Paper radius="md" withBorder style={{ overflow: 'hidden' }}>
                    <Table highlightOnHover verticalSpacing="sm">
                        <Table.Thead>
                            <Table.Tr>
                                <Table.Th>Producto</Table.Th>
                                <Table.Th>Lote</Table.Th>
                                <Table.Th>Vencimiento</Table.Th>
                                <Table.Th>Días</Table.Th>
                                <Table.Th>Cantidad</Table.Th>
                                <Table.Th>Estado</Table.Th>
                                <Table.Th>Acción</Table.Th>
                            </Table.Tr>
                        </Table.Thead>
                        <Table.Tbody>
                            {items.map((a) => (
                                <Table.Tr key={a.id}>
                                    <Table.Td>
                                        <Text size="sm" fw={500}>{a.producto_nombre}</Text>
                                    </Table.Td>
                                    <Table.Td>
                                        <Text size="sm" c="dimmed" ff="monospace">
                                            {a.codigo_lote ?? '—'}
                                        </Text>
                                    </Table.Td>
                                    <Table.Td>
                                        <Text size="sm">{formatDate(a.fecha_vencimiento)}</Text>
                                    </Table.Td>
                                    <Table.Td>
                                        <Text
                                            size="sm"
                                            fw={700}
                                            c={a.dias_restantes < 0 ? 'red' : a.dias_restantes <= 3 ? 'orange' : 'yellow'}
                                        >
                                            {a.dias_restantes < 0
                                                ? `${Math.abs(a.dias_restantes)}d vencido`
                                                : a.dias_restantes === 0
                                                    ? 'Hoy'
                                                    : `${a.dias_restantes}d`}
                                        </Text>
                                    </Table.Td>
                                    <Table.Td>
                                        <Text size="sm" fw={600}>{a.cantidad} ud</Text>
                                    </Table.Td>
                                    <Table.Td>
                                        <Badge color={cfg.color} size="sm" variant="light">
                                            {cfg.label}
                                        </Badge>
                                    </Table.Td>
                                    <Table.Td>
                                        <Button
                                            size="xs"
                                            variant="light"
                                            color="red"
                                            leftSection={<Trash2 size={12} />}
                                            onClick={() => openDelete(a)}
                                        >
                                            Dar de baja
                                        </Button>
                                    </Table.Td>
                                </Table.Tr>
                            ))}
                        </Table.Tbody>
                    </Table>
                </Paper>
            </Stack>
        );
    };

    // ── Main render ──────────────────────────────────────────────────────────

    return (
        <Stack gap="xl">
            <Group justify="space-between" align="flex-end">
                <div>
                    <Title order={2} fw={800}>Vencimientos</Title>
                    <Text c="dimmed" size="sm">
                        Control de fechas de vencimiento — {alertas.length} alerta{alertas.length !== 1 ? 's' : ''}
                    </Text>
                </div>
                <SegmentedControl
                    value={diasFiltro}
                    onChange={setDiasFiltro}
                    data={[
                        { label: '7 días', value: '7' },
                        { label: '14 días', value: '14' },
                        { label: '30 días', value: '30' },
                    ]}
                />
            </Group>

            {loading ? (
                <Stack gap="sm">
                    {[1, 2, 3].map((i) => <Skeleton key={i} h={44} radius="sm" />)}
                </Stack>
            ) : alertas.length === 0 ? (
                <Alert color="teal" variant="light" title="Sin alertas" icon={<AlertTriangle size={16} />}>
                    No hay productos próximos a vencer en los próximos {diasFiltro} días.
                </Alert>
            ) : (
                <Stack gap="xl">
                    {renderSection('Vencidos', vencidos, 'vencido')}
                    {renderSection('Vencen esta semana', criticos, 'critico')}
                    {renderSection('Próximos a vencer', proximos, 'proximo')}
                </Stack>
            )}

            {/* Modal Confirmar Eliminación */}
            <Modal
                opened={deleteOpen}
                onClose={() => setDeleteOpen(false)}
                title={<Text fw={700} c="red">Dar de baja lote</Text>}
                size="sm"
                centered
            >
                {deleteTarget && (
                    <Stack gap="md">
                        <Alert color="red" variant="light">
                            ¿Dar de baja el lote de <strong>{deleteTarget.producto_nombre}</strong>?
                            <br />
                            Lote: <strong>{deleteTarget.codigo_lote ?? 'Sin código'}</strong>
                            <br />
                            Vencimiento: <strong>{formatDate(deleteTarget.fecha_vencimiento)}</strong>
                            <br />
                            Cantidad: <strong>{deleteTarget.cantidad} unidades</strong>
                        </Alert>
                        <Group justify="flex-end">
                            <Button variant="subtle" onClick={() => setDeleteOpen(false)}>Cancelar</Button>
                            <Button
                                color="red"
                                leftSection={<Trash2 size={14} />}
                                loading={deleting}
                                onClick={handleDelete}
                            >
                                Dar de baja
                            </Button>
                        </Group>
                    </Stack>
                )}
            </Modal>
        </Stack>
    );
}
