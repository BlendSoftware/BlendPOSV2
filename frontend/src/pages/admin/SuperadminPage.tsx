import { useEffect, useState } from 'react';
import {
    Title, Text, Table, Badge, Button, Group, Select,
    Card, SimpleGrid, Stack, Alert, LoadingOverlay,
    Box, Modal,
} from '@mantine/core';
import { AlertCircle, Users, ShoppingBag, ToggleLeft, ToggleRight } from 'lucide-react';
import {
    listarTenants, cambiarPlan, toggleTenantActivo,
    obtenerMetricasGlobales, listarPlanes,
    type SuperadminTenantListItem, type PlanResponse, type SuperadminMetricsResponse,
} from '../../services/api/tenant';
import { notifications } from '@mantine/notifications';

export function SuperadminPage() {
    const [tenants, setTenants] = useState<SuperadminTenantListItem[]>([]);
    const [planes, setPlanes] = useState<PlanResponse[]>([]);
    const [metrics, setMetrics] = useState<SuperadminMetricsResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');

    // Plan change modal
    const [planModal, setPlanModal] = useState<{ tenantId: string; tenantNombre: string; currentPlanId: string } | null>(null);
    const [selectedPlan, setSelectedPlan] = useState<string | null>(null);
    const [saving, setSaving] = useState(false);

    const load = async () => {
        setLoading(true);
        setError('');
        try {
            const [t, p, m] = await Promise.all([listarTenants(), listarPlanes(), obtenerMetricasGlobales()]);
            setTenants(t);
            setPlanes(p);
            setMetrics(m);
        } catch {
            setError('Error al cargar datos. Verificá que tenés rol superadmin.');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => { void load(); }, []);

    const handleToggle = async (tenant: SuperadminTenantListItem) => {
        try {
            await toggleTenantActivo(tenant.id, !tenant.activo);
            notifications.show({
                message: `Tenant "${tenant.nombre}" ${!tenant.activo ? 'activado' : 'desactivado'}`,
                color: !tenant.activo ? 'teal' : 'orange',
            });
            void load();
        } catch {
            notifications.show({ message: 'Error al cambiar estado', color: 'red' });
        }
    };

    const handleChangePlan = async () => {
        if (!planModal || !selectedPlan) return;
        setSaving(true);
        try {
            await cambiarPlan(planModal.tenantId, selectedPlan);
            notifications.show({
                message: `Plan actualizado para "${planModal.tenantNombre}"`,
                color: 'teal',
            });
            setPlanModal(null);
            void load();
        } catch {
            notifications.show({ message: 'Error al cambiar plan', color: 'red' });
        } finally {
            setSaving(false);
        }
    };

    const planOptions = planes.map((p) => ({ value: p.id, label: p.nombre }));

    return (
        <Stack gap="xl">
            <Title order={2}>Panel Superadmin</Title>

            {error && (
                <Alert icon={<AlertCircle size={16} />} color="red" variant="light">
                    {error}
                </Alert>
            )}

            {/* Metrics */}
            {metrics && (
                <SimpleGrid cols={{ base: 1, sm: 2 }}>
                    <Card withBorder radius="md" p="lg">
                        <Group gap="sm">
                            <Users size={24} color="var(--mantine-color-blue-6)" />
                            <div>
                                <Text size="xs" c="dimmed" tt="uppercase" fw={700}>Total Tenants</Text>
                                <Text fw={700} size="xl">{metrics.total_tenants}</Text>
                            </div>
                        </Group>
                    </Card>
                    <Card withBorder radius="md" p="lg">
                        <Group gap="sm">
                            <ShoppingBag size={24} color="var(--mantine-color-teal-6)" />
                            <div>
                                <Text size="xs" c="dimmed" tt="uppercase" fw={700}>Tenants Activos</Text>
                                <Text fw={700} size="xl">{metrics.tenants_activos}</Text>
                            </div>
                        </Group>
                    </Card>
                </SimpleGrid>
            )}

            {/* Tenants table */}
            <Box pos="relative">
                <LoadingOverlay visible={loading} />
                <Table striped highlightOnHover withTableBorder>
                    <Table.Thead>
                        <Table.Tr>
                            <Table.Th>Negocio</Table.Th>
                            <Table.Th>Slug</Table.Th>
                            <Table.Th>Plan</Table.Th>
                            <Table.Th>Ventas</Table.Th>
                            <Table.Th>Usuarios</Table.Th>
                            <Table.Th>Estado</Table.Th>
                            <Table.Th>Acciones</Table.Th>
                        </Table.Tr>
                    </Table.Thead>
                    <Table.Tbody>
                        {tenants.map((t) => (
                            <Table.Tr key={t.id}>
                                <Table.Td>
                                    <Text fw={500}>{t.nombre}</Text>
                                    {t.cuit && <Text size="xs" c="dimmed">CUIT: {t.cuit}</Text>}
                                </Table.Td>
                                <Table.Td>
                                    <Text size="sm" ff="monospace">{t.slug}</Text>
                                </Table.Td>
                                <Table.Td>
                                    <Badge color="blue" variant="light">
                                        {t.plan?.nombre ?? 'Sin plan'}
                                    </Badge>
                                </Table.Td>
                                <Table.Td>{t.total_ventas.toLocaleString()}</Table.Td>
                                <Table.Td>{t.total_usuarios}</Table.Td>
                                <Table.Td>
                                    <Badge color={t.activo ? 'teal' : 'red'} variant="light">
                                        {t.activo ? 'Activo' : 'Inactivo'}
                                    </Badge>
                                </Table.Td>
                                <Table.Td>
                                    <Group gap="xs">
                                        <Button
                                            size="xs"
                                            variant="light"
                                            color="blue"
                                            onClick={() => {
                                                setSelectedPlan(t.plan?.id ?? null);
                                                setPlanModal({ tenantId: t.id, tenantNombre: t.nombre, currentPlanId: t.plan?.id ?? '' });
                                            }}
                                        >
                                            Plan
                                        </Button>
                                        <Button
                                            size="xs"
                                            variant="light"
                                            color={t.activo ? 'orange' : 'teal'}
                                            leftSection={t.activo ? <ToggleLeft size={12} /> : <ToggleRight size={12} />}
                                            onClick={() => handleToggle(t)}
                                        >
                                            {t.activo ? 'Desactivar' : 'Activar'}
                                        </Button>
                                    </Group>
                                </Table.Td>
                            </Table.Tr>
                        ))}
                        {!loading && tenants.length === 0 && (
                            <Table.Tr>
                                <Table.Td colSpan={7}>
                                    <Text c="dimmed" ta="center" py="md">No hay tenants registrados</Text>
                                </Table.Td>
                            </Table.Tr>
                        )}
                    </Table.Tbody>
                </Table>
            </Box>

            {/* Plan change modal */}
            <Modal
                opened={!!planModal}
                onClose={() => setPlanModal(null)}
                title={`Cambiar plan — ${planModal?.tenantNombre}`}
                centered
                size="sm"
            >
                <Stack gap="md">
                    <Select
                        label="Nuevo plan"
                        data={planOptions}
                        value={selectedPlan}
                        onChange={setSelectedPlan}
                    />
                    <Group justify="flex-end" gap="sm">
                        <Button variant="default" onClick={() => setPlanModal(null)}>Cancelar</Button>
                        <Button loading={saving} onClick={() => void handleChangePlan()} disabled={!selectedPlan}>
                            Guardar
                        </Button>
                    </Group>
                </Stack>
            </Modal>
        </Stack>
    );
}
