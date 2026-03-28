import { useState, useCallback, useMemo, useEffect } from 'react';
import {
    Stack, Title, Text, Group, Button, TextInput, Modal, NumberInput,
    Table, Paper, Badge, ActionIcon, Tooltip, Skeleton, Tabs,
    Divider, Pagination,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { notifications } from '@mantine/notifications';
import {
    Plus, Edit, Search, X, DollarSign,
    ChevronRight, Users, AlertTriangle,
} from 'lucide-react';
import {
    listarClientes, crearCliente, actualizarCliente,
    obtenerCliente, registrarPago, listarMovimientos, listarDeudores,
    type ClienteResponse, type MovimientoCuentaResponse, type DeudorResponse,
} from '../../services/api/clientes';
import { formatCurrency } from '../../utils/format';

function formatDate(isoDate: string): string {
    return new Date(isoDate).toLocaleDateString('es-AR', {
        day: '2-digit',
        month: '2-digit',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
    });
}

// ── Component ─────────────────────────────────────────────────────────────────

export function ClientesPage() {
    // ── State ─────────────────────────────────────────────────────────────────
    const [clientes, setClientes] = useState<ClienteResponse[]>([]);
    const [deudores, setDeudores] = useState<DeudorResponse[]>([]);
    const [loading, setLoading] = useState(true);
    const [busqueda, setBusqueda] = useState('');
    const [activeTab, setActiveTab] = useState<string>('lista');

    // Create/Edit modal
    const [modalOpen, setModalOpen] = useState(false);
    const [editTarget, setEditTarget] = useState<ClienteResponse | null>(null);

    // Detail view
    const [selectedCliente, setSelectedCliente] = useState<ClienteResponse | null>(null);
    const [movimientos, setMovimientos] = useState<MovimientoCuentaResponse[]>([]);
    const [movTotal, setMovTotal] = useState(0);
    const [movPage, setMovPage] = useState(1);
    const [movLoading, setMovLoading] = useState(false);

    // Pago modal
    const [pagoModalOpen, setPagoModalOpen] = useState(false);
    const [pagoMonto, setPagoMonto] = useState<number | string>('');
    const [pagoDescripcion, setPagoDescripcion] = useState('');
    const [pagoLoading, setPagoLoading] = useState(false);

    // ── Data fetching ─────────────────────────────────────────────────────────
    const fetchClientes = useCallback(async () => {
        setLoading(true);
        try {
            const resp = await listarClientes({ limit: 100 });
            setClientes(resp.data);
        } catch {
            notifications.show({ title: 'Error', message: 'No se pudieron cargar los clientes', color: 'red' });
        } finally {
            setLoading(false);
        }
    }, []);

    const fetchDeudores = useCallback(async () => {
        try {
            const resp = await listarDeudores();
            setDeudores(resp.data);
        } catch {
            // silent
        }
    }, []);

    useEffect(() => { fetchClientes(); }, [fetchClientes]);
    useEffect(() => { if (activeTab === 'deudores') fetchDeudores(); }, [activeTab, fetchDeudores]);

    const fetchMovimientos = useCallback(async (clienteId: string, page: number) => {
        setMovLoading(true);
        try {
            const resp = await listarMovimientos(clienteId, { page, limit: 20 });
            setMovimientos(resp.data);
            setMovTotal(resp.total);
            setMovPage(page);
        } catch {
            notifications.show({ title: 'Error', message: 'No se pudieron cargar movimientos', color: 'red' });
        } finally {
            setMovLoading(false);
        }
    }, []);

    // ── Form ──────────────────────────────────────────────────────────────────
    const form = useForm({
        initialValues: {
            nombre: '',
            telefono: '',
            email: '',
            dni: '',
            limite_credito: 0,
            notas: '',
        },
        validate: {
            nombre: (v) => (v.trim().length >= 2 ? null : 'Nombre requerido (min 2 caracteres)'),
            email: (v) => (!v || /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v) ? null : 'Email inválido'),
            limite_credito: (v) => (v >= 0 ? null : 'El límite debe ser >= 0'),
        },
    });

    const openCreate = () => {
        setEditTarget(null);
        form.reset();
        setModalOpen(true);
    };

    const openEdit = (c: ClienteResponse) => {
        setEditTarget(c);
        form.setValues({
            nombre: c.nombre,
            telefono: c.telefono ?? '',
            email: c.email ?? '',
            dni: c.dni ?? '',
            limite_credito: c.limite_credito,
            notas: c.notas ?? '',
        });
        setModalOpen(true);
    };

    const handleSubmit = form.onSubmit(async (values) => {
        const payload = {
            nombre: values.nombre.trim(),
            telefono: values.telefono.trim() || undefined,
            email: values.email.trim() || undefined,
            dni: values.dni.trim() || undefined,
            limite_credito: values.limite_credito,
            notas: values.notas.trim() || undefined,
        };

        try {
            if (editTarget) {
                await actualizarCliente(editTarget.id, payload);
                notifications.show({ title: 'Cliente actualizado', message: values.nombre, color: 'blue' });
            } else {
                await crearCliente(payload);
                notifications.show({ title: 'Cliente creado', message: values.nombre, color: 'teal' });
            }
            setModalOpen(false);
            await fetchClientes();
            // Refresh detail if viewing the edited client
            if (editTarget && selectedCliente?.id === editTarget.id) {
                const updated = await obtenerCliente(editTarget.id);
                setSelectedCliente(updated);
            }
        } catch (err) {
            notifications.show({
                title: 'Error',
                message: err instanceof Error ? err.message : 'Error al guardar cliente',
                color: 'red',
            });
        }
    });

    // ── Client detail ─────────────────────────────────────────────────────────
    const openDetail = async (id: string) => {
        try {
            const c = await obtenerCliente(id);
            setSelectedCliente(c);
            setMovPage(1);
            await fetchMovimientos(id, 1);
        } catch {
            notifications.show({ title: 'Error', message: 'No se pudo cargar el cliente', color: 'red' });
        }
    };

    const closeDetail = () => {
        setSelectedCliente(null);
        setMovimientos([]);
    };

    // ── Pago ──────────────────────────────────────────────────────────────────
    const openPagoModal = () => {
        setPagoMonto('');
        setPagoDescripcion('');
        setPagoModalOpen(true);
    };

    const handlePago = async () => {
        if (!selectedCliente) return;
        const monto = typeof pagoMonto === 'string' ? parseFloat(pagoMonto) : pagoMonto;
        if (!monto || monto <= 0) {
            notifications.show({ title: 'Error', message: 'Ingresá un monto válido', color: 'red' });
            return;
        }
        setPagoLoading(true);
        try {
            await registrarPago(selectedCliente.id, {
                monto,
                descripcion: pagoDescripcion.trim() || undefined,
            });
            notifications.show({
                title: 'Pago registrado',
                message: `Se registro un pago de ${formatCurrency(monto)} para ${selectedCliente.nombre}`,
                color: 'teal',
            });
            setPagoModalOpen(false);
            // Refresh detail
            const updated = await obtenerCliente(selectedCliente.id);
            setSelectedCliente(updated);
            await fetchMovimientos(selectedCliente.id, 1);
            await fetchClientes();
            if (activeTab === 'deudores') await fetchDeudores();
        } catch (err) {
            notifications.show({
                title: 'Error al registrar pago',
                message: err instanceof Error ? err.message : 'Error',
                color: 'red',
            });
        } finally {
            setPagoLoading(false);
        }
    };

    // ── Filtered list ─────────────────────────────────────────────────────────
    const filtered = useMemo(() => {
        if (!busqueda.trim()) return clientes;
        const q = busqueda.toLowerCase();
        return clientes.filter(
            (c) => c.nombre.toLowerCase().includes(q) || (c.dni && c.dni.includes(q)),
        );
    }, [clientes, busqueda]);

    // ── Detail view ───────────────────────────────────────────────────────────
    if (selectedCliente) {
        return (
            <Stack gap="lg">
                <Group justify="space-between">
                    <Group gap="sm">
                        <Button variant="subtle" size="sm" onClick={closeDetail}>
                            &larr; Volver
                        </Button>
                        <Title order={3} fw={800}>{selectedCliente.nombre}</Title>
                        {selectedCliente.saldo_deudor > 0 && (
                            <Badge color="red" variant="light" size="lg">
                                Debe: {formatCurrency(selectedCliente.saldo_deudor)}
                            </Badge>
                        )}
                    </Group>
                    <Group gap="sm">
                        <Button
                            variant="outline"
                            size="sm"
                            leftSection={<Edit size={14} />}
                            onClick={() => openEdit(selectedCliente)}
                        >
                            Editar
                        </Button>
                        <Button
                            color="teal"
                            size="sm"
                            leftSection={<DollarSign size={14} />}
                            onClick={openPagoModal}
                            disabled={selectedCliente.saldo_deudor <= 0}
                        >
                            Registrar Pago
                        </Button>
                    </Group>
                </Group>

                {/* Info cards */}
                <Group grow>
                    <Paper p="md" radius="md" withBorder>
                        <Text size="xs" c="dimmed" fw={600}>Saldo Deudor</Text>
                        <Text size="xl" fw={800} c={selectedCliente.saldo_deudor > 0 ? 'red' : 'teal'} ff="monospace">
                            {formatCurrency(selectedCliente.saldo_deudor)}
                        </Text>
                    </Paper>
                    <Paper p="md" radius="md" withBorder>
                        <Text size="xs" c="dimmed" fw={600}>Límite de Crédito</Text>
                        <Text size="xl" fw={800} ff="monospace">
                            {selectedCliente.limite_credito > 0 ? formatCurrency(selectedCliente.limite_credito) : 'Sin límite'}
                        </Text>
                    </Paper>
                    <Paper p="md" radius="md" withBorder>
                        <Text size="xs" c="dimmed" fw={600}>Crédito Disponible</Text>
                        <Text size="xl" fw={800} c="blue" ff="monospace">
                            {selectedCliente.limite_credito > 0 ? formatCurrency(selectedCliente.credito_disponible) : 'Ilimitado'}
                        </Text>
                    </Paper>
                </Group>

                {/* Contact info */}
                <Paper p="md" radius="md" withBorder>
                    <Group gap="xl">
                        {selectedCliente.telefono && (
                            <div>
                                <Text size="xs" c="dimmed">Telefono</Text>
                                <Text size="sm" fw={500}>{selectedCliente.telefono}</Text>
                            </div>
                        )}
                        {selectedCliente.email && (
                            <div>
                                <Text size="xs" c="dimmed">Email</Text>
                                <Text size="sm" fw={500}>{selectedCliente.email}</Text>
                            </div>
                        )}
                        {selectedCliente.dni && (
                            <div>
                                <Text size="xs" c="dimmed">DNI</Text>
                                <Text size="sm" fw={500}>{selectedCliente.dni}</Text>
                            </div>
                        )}
                    </Group>
                    {selectedCliente.notas && (
                        <Text size="sm" c="dimmed" mt="sm">
                            Notas: {selectedCliente.notas}
                        </Text>
                    )}
                </Paper>

                {/* Movimientos table */}
                <Divider label="Movimientos de Cuenta Corriente" labelPosition="left" />
                <Paper radius="md" withBorder style={{ overflow: 'hidden' }}>
                    <Table verticalSpacing="sm">
                        <Table.Thead>
                            <Table.Tr>
                                <Table.Th>Fecha</Table.Th>
                                <Table.Th>Tipo</Table.Th>
                                <Table.Th>Monto</Table.Th>
                                <Table.Th>Saldo Posterior</Table.Th>
                                <Table.Th>Descripción</Table.Th>
                            </Table.Tr>
                        </Table.Thead>
                        <Table.Tbody>
                            {movLoading ? (
                                Array.from({ length: 5 }).map((_, i) => (
                                    <Table.Tr key={i}>
                                        {Array.from({ length: 5 }).map((__, j) => (
                                            <Table.Td key={j}><Skeleton height={20} radius="sm" /></Table.Td>
                                        ))}
                                    </Table.Tr>
                                ))
                            ) : movimientos.length === 0 ? (
                                <Table.Tr>
                                    <Table.Td colSpan={5}>
                                        <Text size="sm" c="dimmed" ta="center" py="lg">
                                            Sin movimientos registrados
                                        </Text>
                                    </Table.Td>
                                </Table.Tr>
                            ) : movimientos.map((mov) => (
                                <Table.Tr key={mov.id}>
                                    <Table.Td>
                                        <Text size="xs" c="dimmed">{formatDate(mov.created_at)}</Text>
                                    </Table.Td>
                                    <Table.Td>
                                        <Badge
                                            color={mov.tipo === 'cargo' ? 'red' : mov.tipo === 'pago' ? 'teal' : 'yellow'}
                                            variant="light"
                                            size="sm"
                                        >
                                            {mov.tipo === 'cargo' ? 'Cargo' : mov.tipo === 'pago' ? 'Pago' : 'Ajuste'}
                                        </Badge>
                                    </Table.Td>
                                    <Table.Td>
                                        <Text
                                            size="sm"
                                            fw={600}
                                            ff="monospace"
                                            c={mov.tipo === 'cargo' ? 'red' : 'teal'}
                                        >
                                            {mov.tipo === 'cargo' ? '+' : '-'}{formatCurrency(mov.monto)}
                                        </Text>
                                    </Table.Td>
                                    <Table.Td>
                                        <Text size="sm" ff="monospace">{formatCurrency(mov.saldo_posterior)}</Text>
                                    </Table.Td>
                                    <Table.Td>
                                        <Text size="xs" c="dimmed">{mov.descripcion ?? '-'}</Text>
                                    </Table.Td>
                                </Table.Tr>
                            ))}
                        </Table.Tbody>
                    </Table>
                </Paper>
                {movTotal > 20 && (
                    <Group justify="center">
                        <Pagination
                            total={Math.ceil(movTotal / 20)}
                            value={movPage}
                            onChange={(p) => fetchMovimientos(selectedCliente.id, p)}
                        />
                    </Group>
                )}

                {/* Pago Modal */}
                <Modal
                    opened={pagoModalOpen}
                    onClose={() => setPagoModalOpen(false)}
                    title={<Text fw={700}>Registrar Pago — {selectedCliente.nombre}</Text>}
                    size="sm"
                    centered
                >
                    <Stack gap="md">
                        <Text size="sm" c="dimmed">
                            Saldo actual: <strong>{formatCurrency(selectedCliente.saldo_deudor)}</strong>
                        </Text>
                        <NumberInput
                            label="Monto del pago"
                            placeholder="0.00"
                            value={pagoMonto}
                            onChange={setPagoMonto}
                            min={0.01}
                            max={selectedCliente.saldo_deudor}
                            prefix="$ "
                            decimalScale={2}
                            decimalSeparator=","
                            thousandSeparator="."
                            size="md"
                            autoFocus
                        />
                        <TextInput
                            label="Descripción (opcional)"
                            placeholder="Ej: Pago parcial en efectivo"
                            value={pagoDescripcion}
                            onChange={(e) => setPagoDescripcion(e.currentTarget.value)}
                        />
                        <Group justify="flex-end" mt="sm">
                            <Button variant="subtle" onClick={() => setPagoModalOpen(false)}>Cancelar</Button>
                            <Button
                                color="teal"
                                loading={pagoLoading}
                                onClick={handlePago}
                                leftSection={<DollarSign size={16} />}
                            >
                                Registrar Pago
                            </Button>
                        </Group>
                    </Stack>
                </Modal>
            </Stack>
        );
    }

    // ── Main list view ────────────────────────────────────────────────────────
    return (
        <Stack gap="xl">
            <Group justify="space-between">
                <div>
                    <Title order={2} fw={800}>Clientes — Fiado</Title>
                    <Text c="dimmed" size="sm">Gestión de cuentas corrientes</Text>
                </div>
                <Button leftSection={<Plus size={16} />} onClick={openCreate}>Nuevo cliente</Button>
            </Group>

            <TextInput
                placeholder="Buscar por nombre o DNI..."
                leftSection={<Search size={14} />}
                value={busqueda}
                onChange={(e) => setBusqueda(e.currentTarget.value)}
                style={{ maxWidth: 360 }}
                rightSection={busqueda ? (
                    <ActionIcon size="sm" variant="subtle" onClick={() => setBusqueda('')}>
                        <X size={12} />
                    </ActionIcon>
                ) : null}
            />

            <Tabs value={activeTab} onChange={(v) => setActiveTab(v ?? 'lista')}>
                <Tabs.List>
                    <Tabs.Tab value="lista" leftSection={<Users size={14} />}>
                        Todos ({clientes.length})
                    </Tabs.Tab>
                    <Tabs.Tab value="deudores" leftSection={<AlertTriangle size={14} />}>
                        Deudores ({deudores.length})
                    </Tabs.Tab>
                </Tabs.List>

                {/* ── Tab: Lista ────────────────────────────────────────── */}
                <Tabs.Panel value="lista" pt="lg">
                    <Paper radius="md" withBorder style={{ overflow: 'hidden' }}>
                        <Table verticalSpacing="sm">
                            <Table.Thead>
                                <Table.Tr>
                                    <Table.Th>Nombre</Table.Th>
                                    <Table.Th>Teléfono</Table.Th>
                                    <Table.Th>DNI</Table.Th>
                                    <Table.Th>Saldo Deudor</Table.Th>
                                    <Table.Th>Límite Crédito</Table.Th>
                                    <Table.Th>Acciones</Table.Th>
                                </Table.Tr>
                            </Table.Thead>
                            <Table.Tbody>
                                {loading ? (
                                    Array.from({ length: 5 }).map((_, i) => (
                                        <Table.Tr key={i}>
                                            {Array.from({ length: 6 }).map((__, j) => (
                                                <Table.Td key={j}><Skeleton height={20} radius="sm" /></Table.Td>
                                            ))}
                                        </Table.Tr>
                                    ))
                                ) : filtered.length === 0 ? (
                                    <Table.Tr>
                                        <Table.Td colSpan={6}>
                                            <Text size="sm" c="dimmed" ta="center" py="lg">
                                                {busqueda ? 'Sin resultados para la busqueda' : 'No hay clientes registrados'}
                                            </Text>
                                        </Table.Td>
                                    </Table.Tr>
                                ) : filtered.map((c) => (
                                    <Table.Tr
                                        key={c.id}
                                        style={{ cursor: 'pointer' }}
                                        onClick={() => openDetail(c.id)}
                                    >
                                        <Table.Td>
                                            <Group gap="xs">
                                                <Text size="sm" fw={500}>{c.nombre}</Text>
                                                <ChevronRight size={14} color="var(--mantine-color-dimmed)" />
                                            </Group>
                                        </Table.Td>
                                        <Table.Td>
                                            <Text size="xs" c="dimmed">{c.telefono ?? '-'}</Text>
                                        </Table.Td>
                                        <Table.Td>
                                            <Text size="xs" ff="monospace" c="dimmed">{c.dni ?? '-'}</Text>
                                        </Table.Td>
                                        <Table.Td>
                                            {c.saldo_deudor > 0 ? (
                                                <Badge color="red" variant="light" size="sm">
                                                    {formatCurrency(c.saldo_deudor)}
                                                </Badge>
                                            ) : (
                                                <Text size="xs" c="dimmed">{formatCurrency(0)}</Text>
                                            )}
                                        </Table.Td>
                                        <Table.Td>
                                            <Text size="xs" ff="monospace">
                                                {c.limite_credito > 0 ? formatCurrency(c.limite_credito) : 'Sin límite'}
                                            </Text>
                                        </Table.Td>
                                        <Table.Td>
                                            <Group gap="xs">
                                                <Tooltip label="Editar" withArrow>
                                                    <ActionIcon
                                                        variant="subtle"
                                                        color="blue"
                                                        onClick={(e) => { e.stopPropagation(); openEdit(c); }}
                                                    >
                                                        <Edit size={15} />
                                                    </ActionIcon>
                                                </Tooltip>
                                            </Group>
                                        </Table.Td>
                                    </Table.Tr>
                                ))}
                            </Table.Tbody>
                        </Table>
                    </Paper>
                </Tabs.Panel>

                {/* ── Tab: Deudores ─────────────────────────────────────── */}
                <Tabs.Panel value="deudores" pt="lg">
                    <Paper radius="md" withBorder style={{ overflow: 'hidden' }}>
                        <Table verticalSpacing="sm">
                            <Table.Thead>
                                <Table.Tr>
                                    <Table.Th>Nombre</Table.Th>
                                    <Table.Th>Teléfono</Table.Th>
                                    <Table.Th>Saldo Deudor</Table.Th>
                                    <Table.Th>Límite Crédito</Table.Th>
                                    <Table.Th>Acciones</Table.Th>
                                </Table.Tr>
                            </Table.Thead>
                            <Table.Tbody>
                                {deudores.length === 0 ? (
                                    <Table.Tr>
                                        <Table.Td colSpan={5}>
                                            <Text size="sm" c="dimmed" ta="center" py="lg">
                                                No hay clientes con deuda
                                            </Text>
                                        </Table.Td>
                                    </Table.Tr>
                                ) : deudores.map((d) => (
                                    <Table.Tr
                                        key={d.id}
                                        style={{ cursor: 'pointer' }}
                                        onClick={() => openDetail(d.id)}
                                    >
                                        <Table.Td>
                                            <Group gap="xs">
                                                <Text size="sm" fw={500}>{d.nombre}</Text>
                                                <ChevronRight size={14} color="var(--mantine-color-dimmed)" />
                                            </Group>
                                        </Table.Td>
                                        <Table.Td>
                                            <Text size="xs" c="dimmed">{d.telefono ?? '-'}</Text>
                                        </Table.Td>
                                        <Table.Td>
                                            <Badge color="red" variant="light" size="lg">
                                                {formatCurrency(d.saldo_deudor)}
                                            </Badge>
                                        </Table.Td>
                                        <Table.Td>
                                            <Text size="xs" ff="monospace">
                                                {d.limite_credito > 0 ? formatCurrency(d.limite_credito) : 'Sin límite'}
                                            </Text>
                                        </Table.Td>
                                        <Table.Td>
                                            <Tooltip label="Registrar pago" withArrow>
                                                <ActionIcon
                                                    variant="subtle"
                                                    color="teal"
                                                    onClick={(e) => { e.stopPropagation(); openDetail(d.id); }}
                                                >
                                                    <DollarSign size={15} />
                                                </ActionIcon>
                                            </Tooltip>
                                        </Table.Td>
                                    </Table.Tr>
                                ))}
                            </Table.Tbody>
                        </Table>
                    </Paper>
                </Tabs.Panel>
            </Tabs>

            {/* ── Create/Edit Modal ────────────────────────────────────────── */}
            <Modal
                opened={modalOpen}
                onClose={() => setModalOpen(false)}
                title={<Text fw={700}>{editTarget ? 'Editar cliente' : 'Nuevo cliente'}</Text>}
                size="md"
                centered
            >
                <form onSubmit={handleSubmit}>
                    <Stack gap="md">
                        <TextInput
                            label="Nombre"
                            placeholder="Juan Perez"
                            required
                            {...form.getInputProps('nombre')}
                        />
                        <Group grow>
                            <TextInput
                                label="Teléfono"
                                placeholder="11-1234-5678"
                                {...form.getInputProps('telefono')}
                            />
                            <TextInput
                                label="DNI"
                                placeholder="30123456"
                                {...form.getInputProps('dni')}
                            />
                        </Group>
                        <TextInput
                            label="Email"
                            placeholder="cliente@ejemplo.com"
                            {...form.getInputProps('email')}
                        />
                        <NumberInput
                            label="Límite de crédito"
                            description="Dejar en 0 para sin limite"
                            placeholder="0.00"
                            min={0}
                            prefix="$ "
                            decimalScale={2}
                            decimalSeparator=","
                            thousandSeparator="."
                            {...form.getInputProps('limite_credito')}
                        />
                        <TextInput
                            label="Notas (opcional)"
                            placeholder="Observaciones sobre el cliente..."
                            {...form.getInputProps('notas')}
                        />
                        <Group justify="flex-end" mt="sm">
                            <Button variant="subtle" onClick={() => setModalOpen(false)}>Cancelar</Button>
                            <Button type="submit">{editTarget ? 'Guardar' : 'Crear'}</Button>
                        </Group>
                    </Stack>
                </form>
            </Modal>
        </Stack>
    );
}
