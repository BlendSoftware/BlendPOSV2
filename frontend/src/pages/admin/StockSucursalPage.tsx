import { useState, useEffect, useCallback, useMemo } from 'react';
import {
    Stack, Title, Text, Group, Table, Paper, Badge, Select,
    Modal, NumberInput, Textarea, Button, Skeleton, Alert,
    ActionIcon, Tooltip,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { notifications } from '@mantine/notifications';
import { Pencil, Warehouse } from 'lucide-react';
import {
    listarStockSucursal, ajustarStockSucursal,
    type StockSucursalResponse,
} from '../../services/api/transferencias';
import { listarSucursales, type SucursalResponse } from '../../services/api/sucursales';

// ── Component ───────────────────────────────────────────────────────────────

export function StockSucursalPage() {
    // ── State ───────────────────────────────────────────────────────────────
    const [sucursales, setSucursales] = useState<SucursalResponse[]>([]);
    const [sucursalId, setSucursalId] = useState<string>('');
    const [stock, setStock] = useState<StockSucursalResponse[]>([]);
    const [loading, setLoading] = useState(false);
    const [loadingSucursales, setLoadingSucursales] = useState(true);
    const [ajusteTarget, setAjusteTarget] = useState<StockSucursalResponse | null>(null);

    // ── Fetch sucursales on mount ───────────────────────────────────────────
    useEffect(() => {
        (async () => {
            try {
                const resp = await listarSucursales();
                setSucursales(resp.data);
            } catch {
                notifications.show({ title: 'Error', message: 'No se pudieron cargar las sucursales', color: 'red' });
            } finally {
                setLoadingSucursales(false);
            }
        })();
    }, []);

    // ── Fetch stock when sucursal changes ───────────────────────────────────
    const fetchStock = useCallback(async () => {
        if (!sucursalId) { setStock([]); return; }
        setLoading(true);
        try {
            const resp = await listarStockSucursal(sucursalId);
            setStock(resp.data);
        } catch {
            notifications.show({ title: 'Error', message: 'No se pudo cargar el stock', color: 'red' });
        } finally {
            setLoading(false);
        }
    }, [sucursalId]);

    useEffect(() => { fetchStock(); }, [fetchStock]);

    // ── Sucursales select data ──────────────────────────────────────────────
    const sucursalesSelect = useMemo(
        () => sucursales.filter((s) => s.activa).map((s) => ({ value: s.id, label: s.nombre })),
        [sucursales],
    );

    // ── Ajuste form ─────────────────────────────────────────────────────────
    const ajusteForm = useForm({
        initialValues: {
            delta: 0,
            motivo: '',
        },
        validate: {
            delta: (v) => (v !== 0 ? null : 'El ajuste no puede ser 0'),
            motivo: (v) => (v.trim().length >= 3 ? null : 'Motivo requerido (min 3 caracteres)'),
        },
    });

    const openAjuste = (item: StockSucursalResponse) => {
        setAjusteTarget(item);
        ajusteForm.reset();
    };

    const handleAjuste = ajusteForm.onSubmit(async (values) => {
        if (!ajusteTarget) return;
        try {
            await ajustarStockSucursal({
                sucursal_id: sucursalId,
                producto_id: ajusteTarget.producto_id,
                delta: values.delta,
                motivo: values.motivo.trim(),
            });
            notifications.show({
                title: 'Stock ajustado',
                message: `${ajusteTarget.producto_nombre}: ${values.delta > 0 ? '+' : ''}${values.delta}`,
                color: 'teal',
            });
            setAjusteTarget(null);
            await fetchStock();
        } catch (err) {
            notifications.show({
                title: 'Error',
                message: err instanceof Error ? err.message : 'Error al ajustar stock',
                color: 'red',
            });
        }
    });

    // ── Render ──────────────────────────────────────────────────────────────
    return (
        <Stack gap="xl">
            <div>
                <Title order={2} fw={800}>Stock por Sucursal</Title>
                <Text c="dimmed" size="sm">Consultá y ajustá el stock de cada sucursal</Text>
            </div>

            <Select
                label="Sucursal"
                placeholder={loadingSucursales ? 'Cargando...' : 'Seleccioná una sucursal'}
                data={sucursalesSelect}
                value={sucursalId}
                onChange={(v) => setSucursalId(v ?? '')}
                searchable
                disabled={loadingSucursales}
                leftSection={<Warehouse size={14} />}
                style={{ maxWidth: 360 }}
            />

            {!sucursalId ? (
                <Alert variant="light" color="blue">
                    Seleccioná una sucursal para ver el stock
                </Alert>
            ) : (
                <Paper radius="md" withBorder style={{ overflow: 'hidden' }}>
                    <Table highlightOnHover verticalSpacing="sm">
                        <Table.Thead>
                            <Table.Tr>
                                <Table.Th>Producto</Table.Th>
                                <Table.Th>Stock Actual</Table.Th>
                                <Table.Th>Stock Mínimo</Table.Th>
                                <Table.Th>Estado</Table.Th>
                                <Table.Th>Acciones</Table.Th>
                            </Table.Tr>
                        </Table.Thead>
                        <Table.Tbody>
                            {loading ? (
                                Array.from({ length: 5 }).map((_, i) => (
                                    <Table.Tr key={i}>
                                        {[1, 2, 3, 4, 5].map((j) => (
                                            <Table.Td key={j}><Skeleton h={20} radius="sm" /></Table.Td>
                                        ))}
                                    </Table.Tr>
                                ))
                            ) : stock.length === 0 ? (
                                <Table.Tr>
                                    <Table.Td colSpan={5}>
                                        <Text size="sm" c="dimmed" ta="center" py="lg">
                                            No hay productos con stock en esta sucursal
                                        </Text>
                                    </Table.Td>
                                </Table.Tr>
                            ) : stock.map((item) => {
                                const esCritico = item.stock_actual <= item.stock_minimo;
                                return (
                                    <Table.Tr key={item.id}>
                                        <Table.Td><Text size="sm" fw={500}>{item.producto_nombre}</Text></Table.Td>
                                        <Table.Td>
                                            <Text size="sm" fw={700} c={esCritico ? 'red' : undefined}>
                                                {item.stock_actual}
                                            </Text>
                                        </Table.Td>
                                        <Table.Td><Text size="sm" c="dimmed">{item.stock_minimo}</Text></Table.Td>
                                        <Table.Td>
                                            <Badge
                                                color={esCritico ? 'red' : 'green'}
                                                size="sm"
                                                variant="light"
                                            >
                                                {esCritico ? 'Crítico' : 'Normal'}
                                            </Badge>
                                        </Table.Td>
                                        <Table.Td>
                                            <Tooltip label="Ajustar stock" withArrow>
                                                <ActionIcon
                                                    variant="subtle"
                                                    color="blue"
                                                    onClick={() => openAjuste(item)}
                                                >
                                                    <Pencil size={15} />
                                                </ActionIcon>
                                            </Tooltip>
                                        </Table.Td>
                                    </Table.Tr>
                                );
                            })}
                        </Table.Tbody>
                    </Table>
                </Paper>
            )}

            {/* ── Ajuste Modal ────────────────────────────────────────────── */}
            <Modal
                opened={!!ajusteTarget}
                onClose={() => setAjusteTarget(null)}
                title={<Text fw={700}>Ajustar Stock</Text>}
                size="sm"
                centered
            >
                {ajusteTarget && (
                    <form onSubmit={handleAjuste}>
                        <Stack gap="md">
                            <div>
                                <Text size="sm" c="dimmed">Producto</Text>
                                <Text size="sm" fw={600}>{ajusteTarget.producto_nombre}</Text>
                            </div>
                            <div>
                                <Text size="sm" c="dimmed">Stock actual</Text>
                                <Text size="sm" fw={600}>{ajusteTarget.stock_actual}</Text>
                            </div>
                            <NumberInput
                                label="Ajuste (+ o -)"
                                description="Valores positivos suman, negativos restan"
                                placeholder="Ej: 5 o -3"
                                {...ajusteForm.getInputProps('delta')}
                            />
                            <Textarea
                                label="Motivo"
                                placeholder="Ej: recuento de inventario"
                                required
                                {...ajusteForm.getInputProps('motivo')}
                            />
                            <Group justify="flex-end">
                                <Button variant="subtle" onClick={() => setAjusteTarget(null)}>Cancelar</Button>
                                <Button type="submit">Guardar ajuste</Button>
                            </Group>
                        </Stack>
                    </form>
                )}
            </Modal>
        </Stack>
    );
}
