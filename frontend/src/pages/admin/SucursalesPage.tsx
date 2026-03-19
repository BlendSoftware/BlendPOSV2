import { useState, useMemo, useEffect, useCallback } from 'react';
import {
    Stack, Title, Text, Group, Button, TextInput, Textarea, Table,
    Paper, Badge, ActionIcon, Tooltip, Modal, Skeleton,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { notifications } from '@mantine/notifications';
import { Plus, Edit, Search, X, Power, PowerOff } from 'lucide-react';
import {
    listarSucursales, crearSucursal, actualizarSucursal,
    type SucursalResponse,
} from '../../services/api/sucursales';

// ── Component ─────────────────────────────────────────────────────────────────

export function SucursalesPage() {
    // ── State ─────────────────────────────────────────────────────────────────
    const [sucursales, setSucursales] = useState<SucursalResponse[]>([]);
    const [loading, setLoading] = useState(true);
    const [busqueda, setBusqueda] = useState('');
    const [modalOpen, setModalOpen] = useState(false);
    const [editTarget, setEditTarget] = useState<SucursalResponse | null>(null);

    // ── Data fetching ─────────────────────────────────────────────────────────
    const fetchSucursales = useCallback(async () => {
        setLoading(true);
        try {
            const resp = await listarSucursales();
            setSucursales(resp.data);
        } catch {
            notifications.show({ title: 'Error', message: 'No se pudieron cargar las sucursales', color: 'red' });
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => { fetchSucursales(); }, [fetchSucursales]);

    // ── Form ──────────────────────────────────────────────────────────────────
    const form = useForm({
        initialValues: {
            nombre: '',
            direccion: '',
            telefono: '',
        },
        validate: {
            nombre: (v) => (v.trim().length >= 2 ? null : 'Nombre requerido (min 2 caracteres)'),
        },
    });

    const openCreate = () => {
        setEditTarget(null);
        form.reset();
        setModalOpen(true);
    };

    const openEdit = (s: SucursalResponse) => {
        setEditTarget(s);
        form.setValues({
            nombre: s.nombre,
            direccion: s.direccion ?? '',
            telefono: s.telefono ?? '',
        });
        setModalOpen(true);
    };

    const handleSubmit = form.onSubmit(async (values) => {
        const payload = {
            nombre: values.nombre.trim(),
            direccion: values.direccion.trim() || undefined,
            telefono: values.telefono.trim() || undefined,
        };

        try {
            if (editTarget) {
                await actualizarSucursal(editTarget.id, payload);
                notifications.show({ title: 'Sucursal actualizada', message: values.nombre, color: 'blue' });
            } else {
                await crearSucursal(payload);
                notifications.show({ title: 'Sucursal creada', message: values.nombre, color: 'teal' });
            }
            setModalOpen(false);
            await fetchSucursales();
        } catch (err) {
            notifications.show({
                title: 'Error',
                message: err instanceof Error ? err.message : 'Error al guardar sucursal',
                color: 'red',
            });
        }
    });

    // ── Toggle activa/inactiva ────────────────────────────────────────────────
    const toggleActiva = async (s: SucursalResponse) => {
        try {
            await actualizarSucursal(s.id, { activa: !s.activa });
            setSucursales((prev) =>
                prev.map((x) => x.id === s.id ? { ...x, activa: !x.activa } : x),
            );
            notifications.show({
                title: s.activa ? 'Sucursal desactivada' : 'Sucursal activada',
                message: s.nombre,
                color: s.activa ? 'gray' : 'teal',
            });
        } catch (err) {
            notifications.show({
                title: 'Error',
                message: err instanceof Error ? err.message : 'Error al cambiar estado',
                color: 'red',
            });
        }
    };

    // ── Filtered list ─────────────────────────────────────────────────────────
    const filtered = useMemo(() => {
        if (!busqueda.trim()) return sucursales;
        const q = busqueda.toLowerCase();
        return sucursales.filter((s) => s.nombre.toLowerCase().includes(q));
    }, [sucursales, busqueda]);

    // ── Render ────────────────────────────────────────────────────────────────
    return (
        <Stack gap="xl">
            <Group justify="space-between">
                <div>
                    <Title order={2} fw={800}>Sucursales</Title>
                    <Text c="dimmed" size="sm">
                        {sucursales.filter((s) => s.activa).length} activas · {sucursales.length} total
                    </Text>
                </div>
                <Button leftSection={<Plus size={16} />} onClick={openCreate}>Nueva sucursal</Button>
            </Group>

            <TextInput
                placeholder="Buscar por nombre..."
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

            <Paper radius="md" withBorder style={{ overflow: 'hidden' }}>
                <Table highlightOnHover verticalSpacing="sm">
                    <Table.Thead>
                        <Table.Tr>
                            <Table.Th>Nombre</Table.Th>
                            <Table.Th>Dirección</Table.Th>
                            <Table.Th>Teléfono</Table.Th>
                            <Table.Th>Estado</Table.Th>
                            <Table.Th>Acciones</Table.Th>
                        </Table.Tr>
                    </Table.Thead>
                    <Table.Tbody>
                        {loading ? (
                            Array.from({ length: 4 }).map((_, i) => (
                                <Table.Tr key={i}>
                                    {[1, 2, 3, 4, 5].map((j) => (
                                        <Table.Td key={j}><Skeleton h={20} radius="sm" /></Table.Td>
                                    ))}
                                </Table.Tr>
                            ))
                        ) : filtered.length === 0 ? (
                            <Table.Tr>
                                <Table.Td colSpan={5}>
                                    <Text size="sm" c="dimmed" ta="center" py="lg">
                                        {busqueda ? 'Sin resultados para la búsqueda' : 'No hay sucursales registradas'}
                                    </Text>
                                </Table.Td>
                            </Table.Tr>
                        ) : filtered.map((s) => (
                            <Table.Tr key={s.id} style={{ opacity: s.activa ? 1 : 0.5 }}>
                                <Table.Td><Text size="sm" fw={500}>{s.nombre}</Text></Table.Td>
                                <Table.Td><Text size="sm" c="dimmed">{s.direccion ?? '—'}</Text></Table.Td>
                                <Table.Td><Text size="sm" c="dimmed">{s.telefono ?? '—'}</Text></Table.Td>
                                <Table.Td>
                                    <Badge color={s.activa ? 'teal' : 'gray'} size="sm" variant="light">
                                        {s.activa ? 'Activa' : 'Inactiva'}
                                    </Badge>
                                </Table.Td>
                                <Table.Td>
                                    <Group gap={4}>
                                        <Tooltip label="Editar" withArrow>
                                            <ActionIcon variant="subtle" color="blue" onClick={() => openEdit(s)}>
                                                <Edit size={15} />
                                            </ActionIcon>
                                        </Tooltip>
                                        <Tooltip label={s.activa ? 'Desactivar' : 'Activar'} withArrow>
                                            <ActionIcon
                                                variant="subtle"
                                                color={s.activa ? 'gray' : 'teal'}
                                                onClick={() => toggleActiva(s)}
                                            >
                                                {s.activa ? <PowerOff size={15} /> : <Power size={15} />}
                                            </ActionIcon>
                                        </Tooltip>
                                    </Group>
                                </Table.Td>
                            </Table.Tr>
                        ))}
                    </Table.Tbody>
                </Table>
            </Paper>

            {/* ── Create/Edit Modal ────────────────────────────────────────── */}
            <Modal
                opened={modalOpen}
                onClose={() => setModalOpen(false)}
                title={<Text fw={700}>{editTarget ? 'Editar sucursal' : 'Nueva sucursal'}</Text>}
                size="sm"
                centered
            >
                <form onSubmit={handleSubmit}>
                    <Stack gap="md">
                        <TextInput
                            label="Nombre"
                            placeholder="Sucursal Centro"
                            required
                            {...form.getInputProps('nombre')}
                        />
                        <Textarea
                            label="Dirección"
                            placeholder="Av. Corrientes 1234, CABA"
                            {...form.getInputProps('direccion')}
                        />
                        <TextInput
                            label="Teléfono"
                            placeholder="11-1234-5678"
                            {...form.getInputProps('telefono')}
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
