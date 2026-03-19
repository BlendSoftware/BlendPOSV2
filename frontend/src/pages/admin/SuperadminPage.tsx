import { useCallback, useEffect, useState } from 'react';
import {
    Title, Text, Table, Badge, Button, Group, Select,
    Card, SimpleGrid, Stack, Alert, LoadingOverlay,
    Box, Modal, TextInput, Pagination, Flex,
} from '@mantine/core';
import { useDebouncedValue } from '@mantine/hooks';
import {
    AlertCircle, Users, ShoppingBag, ToggleLeft, ToggleRight,
    Search, ShoppingCart, TrendingUp, Package,
} from 'lucide-react';
import {
    listarTenants, cambiarPlan, toggleTenantActivo,
    obtenerMetricasGlobales, listarPlanes, obtenerTenantDetalle,
    type SuperadminTenantListItem, type PlanResponse,
    type SuperadminMetricsResponse, type TenantListResponse,
} from '../../services/api/tenant';
import { notifications } from '@mantine/notifications';

const PAGE_SIZE = 15;

export function SuperadminPage() {
    const [tenantsResp, setTenantsResp] = useState<TenantListResponse | null>(null);
    const [planes, setPlanes] = useState<PlanResponse[]>([]);
    const [metrics, setMetrics] = useState<SuperadminMetricsResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');

    // Filters
    const [search, setSearch] = useState('');
    const [debouncedSearch] = useDebouncedValue(search, 300);
    const [statusFilter, setStatusFilter] = useState<string | null>('all');
    const [planFilter, setPlanFilter] = useState<string | null>(null);
    const [page, setPage] = useState(1);

    // Plan change modal
    const [planModal, setPlanModal] = useState<{ tenantId: string; tenantNombre: string; currentPlanId: string } | null>(null);
    const [selectedPlan, setSelectedPlan] = useState<string | null>(null);
    const [saving, setSaving] = useState(false);

    // Detail modal
    const [detailModal, setDetailModal] = useState<SuperadminTenantListItem | null>(null);
    const [detailLoading, setDetailLoading] = useState(false);

    const loadTenants = useCallback(async () => {
        setLoading(true);
        setError('');
        try {
            const resp = await listarTenants({
                page,
                page_size: PAGE_SIZE,
                search: debouncedSearch || undefined,
                status: statusFilter || 'all',
                plan_id: planFilter || undefined,
            });
            setTenantsResp(resp);
        } catch {
            setError('Error al cargar tenants. Verifica que tenes rol superadmin.');
        } finally {
            setLoading(false);
        }
    }, [page, debouncedSearch, statusFilter, planFilter]);

    const loadMetrics = useCallback(async () => {
        try {
            const m = await obtenerMetricasGlobales();
            setMetrics(m);
        } catch {
            // Metrics are non-critical
        }
    }, []);

    const loadPlanes = useCallback(async () => {
        try {
            const p = await listarPlanes();
            setPlanes(p);
        } catch {
            // Plans are non-critical
        }
    }, []);

    // Initial load of planes and metrics
    useEffect(() => {
        void loadPlanes();
        void loadMetrics();
    }, [loadPlanes, loadMetrics]);

    // Load tenants when filters/page change
    useEffect(() => {
        void loadTenants();
    }, [loadTenants]);

    // Reset page when filters change
    useEffect(() => {
        setPage(1);
    }, [debouncedSearch, statusFilter, planFilter]);

    const handleToggle = async (tenant: SuperadminTenantListItem) => {
        try {
            await toggleTenantActivo(tenant.id, !tenant.activo);
            notifications.show({
                message: `Tenant "${tenant.nombre}" ${!tenant.activo ? 'activado' : 'desactivado'}`,
                color: !tenant.activo ? 'teal' : 'orange',
            });
            void loadTenants();
            void loadMetrics();
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
            void loadTenants();
            void loadMetrics();
        } catch {
            notifications.show({ message: 'Error al cambiar plan', color: 'red' });
        } finally {
            setSaving(false);
        }
    };

    const handleRowClick = async (tenantId: string) => {
        setDetailLoading(true);
        try {
            const detail = await obtenerTenantDetalle(tenantId);
            setDetailModal(detail);
        } catch {
            notifications.show({ message: 'Error al cargar detalle del tenant', color: 'red' });
        } finally {
            setDetailLoading(false);
        }
    };

    const planOptions = planes.map((p) => ({ value: p.id, label: p.nombre }));
    const tenants = tenantsResp?.tenants ?? [];

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
                <SimpleGrid cols={{ base: 1, sm: 2, md: 4 }}>
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
                    <Card withBorder radius="md" p="lg">
                        <Group gap="sm">
                            <ShoppingCart size={24} color="var(--mantine-color-violet-6)" />
                            <div>
                                <Text size="xs" c="dimmed" tt="uppercase" fw={700}>Ventas Totales</Text>
                                <Text fw={700} size="xl">{metrics.total_ventas.toLocaleString()}</Text>
                            </div>
                        </Group>
                    </Card>
                    <Card withBorder radius="md" p="lg">
                        <Group gap="sm">
                            <TrendingUp size={24} color="var(--mantine-color-orange-6)" />
                            <div>
                                <Text size="xs" c="dimmed" tt="uppercase" fw={700}>Ventas (30 dias)</Text>
                                <Text fw={700} size="xl">{metrics.ventas_ultimo_mes.toLocaleString()}</Text>
                            </div>
                        </Group>
                    </Card>
                </SimpleGrid>
            )}

            {/* Tenants por plan */}
            {metrics && metrics.tenants_por_plan && metrics.tenants_por_plan.length > 0 && (
                <Card withBorder radius="md" p="md">
                    <Text size="sm" fw={600} mb="xs">Distribucion por Plan</Text>
                    <Group gap="md">
                        {metrics.tenants_por_plan.map((pc) => (
                            <Badge key={pc.plan_nombre} size="lg" variant="light" color="blue">
                                {pc.plan_nombre}: {pc.count}
                            </Badge>
                        ))}
                    </Group>
                </Card>
            )}

            {/* Filters */}
            <Flex gap="md" wrap="wrap" align="flex-end">
                <TextInput
                    placeholder="Buscar por nombre o slug..."
                    leftSection={<Search size={16} />}
                    value={search}
                    onChange={(e) => setSearch(e.currentTarget.value)}
                    style={{ flex: 1, minWidth: 200 }}
                />
                <Select
                    label="Estado"
                    data={[
                        { value: 'all', label: 'Todos' },
                        { value: 'active', label: 'Activos' },
                        { value: 'inactive', label: 'Inactivos' },
                    ]}
                    value={statusFilter}
                    onChange={setStatusFilter}
                    w={140}
                    clearable={false}
                />
                <Select
                    label="Plan"
                    data={[{ value: '', label: 'Todos los planes' }, ...planOptions]}
                    value={planFilter}
                    onChange={setPlanFilter}
                    w={180}
                    clearable
                />
            </Flex>

            {/* Tenants table */}
            <Box pos="relative">
                <LoadingOverlay visible={loading || detailLoading} />
                <Table striped highlightOnHover withTableBorder>
                    <Table.Thead>
                        <Table.Tr>
                            <Table.Th>Negocio</Table.Th>
                            <Table.Th>Slug</Table.Th>
                            <Table.Th>Plan</Table.Th>
                            <Table.Th>Ventas</Table.Th>
                            <Table.Th>Productos</Table.Th>
                            <Table.Th>Usuarios</Table.Th>
                            <Table.Th>Estado</Table.Th>
                            <Table.Th>Acciones</Table.Th>
                        </Table.Tr>
                    </Table.Thead>
                    <Table.Tbody>
                        {tenants.map((t) => (
                            <Table.Tr
                                key={t.id}
                                style={{ cursor: 'pointer' }}
                                onClick={() => void handleRowClick(t.id)}
                            >
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
                                <Table.Td>{t.total_productos.toLocaleString()}</Table.Td>
                                <Table.Td>{t.total_usuarios}</Table.Td>
                                <Table.Td>
                                    <Badge color={t.activo ? 'teal' : 'red'} variant="light">
                                        {t.activo ? 'Activo' : 'Inactivo'}
                                    </Badge>
                                </Table.Td>
                                <Table.Td>
                                    <Group gap="xs" onClick={(e) => e.stopPropagation()}>
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
                                <Table.Td colSpan={8}>
                                    <Text c="dimmed" ta="center" py="md">No hay tenants registrados</Text>
                                </Table.Td>
                            </Table.Tr>
                        )}
                    </Table.Tbody>
                </Table>
            </Box>

            {/* Pagination */}
            {tenantsResp && tenantsResp.total_pages > 1 && (
                <Flex justify="center">
                    <Pagination
                        total={tenantsResp.total_pages}
                        value={page}
                        onChange={setPage}
                    />
                </Flex>
            )}

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

            {/* Detail modal */}
            <Modal
                opened={!!detailModal}
                onClose={() => setDetailModal(null)}
                title={`Detalle — ${detailModal?.nombre}`}
                centered
                size="lg"
            >
                {detailModal && (
                    <Stack gap="md">
                        <SimpleGrid cols={2}>
                            <div>
                                <Text size="xs" c="dimmed">Slug</Text>
                                <Text ff="monospace">{detailModal.slug}</Text>
                            </div>
                            <div>
                                <Text size="xs" c="dimmed">CUIT</Text>
                                <Text>{detailModal.cuit ?? 'No configurado'}</Text>
                            </div>
                            <div>
                                <Text size="xs" c="dimmed">Plan</Text>
                                <Badge color="blue" variant="light">
                                    {detailModal.plan?.nombre ?? 'Sin plan'}
                                </Badge>
                            </div>
                            <div>
                                <Text size="xs" c="dimmed">Estado</Text>
                                <Badge color={detailModal.activo ? 'teal' : 'red'} variant="light">
                                    {detailModal.activo ? 'Activo' : 'Inactivo'}
                                </Badge>
                            </div>
                            <div>
                                <Text size="xs" c="dimmed">Fecha de Creacion</Text>
                                <Text>{new Date(detailModal.created_at).toLocaleDateString('es-AR')}</Text>
                            </div>
                            {detailModal.ultima_venta && (
                                <div>
                                    <Text size="xs" c="dimmed">Ultima Venta</Text>
                                    <Text>{new Date(detailModal.ultima_venta).toLocaleDateString('es-AR')}</Text>
                                </div>
                            )}
                        </SimpleGrid>

                        <Text size="sm" fw={600} mt="sm">Metricas</Text>
                        <SimpleGrid cols={3}>
                            <Card withBorder radius="md" p="sm">
                                <Group gap="xs">
                                    <ShoppingCart size={18} color="var(--mantine-color-violet-6)" />
                                    <div>
                                        <Text size="xs" c="dimmed">Ventas</Text>
                                        <Text fw={700}>{detailModal.total_ventas.toLocaleString()}</Text>
                                    </div>
                                </Group>
                            </Card>
                            <Card withBorder radius="md" p="sm">
                                <Group gap="xs">
                                    <Package size={18} color="var(--mantine-color-blue-6)" />
                                    <div>
                                        <Text size="xs" c="dimmed">Productos</Text>
                                        <Text fw={700}>{detailModal.total_productos.toLocaleString()}</Text>
                                    </div>
                                </Group>
                            </Card>
                            <Card withBorder radius="md" p="sm">
                                <Group gap="xs">
                                    <Users size={18} color="var(--mantine-color-teal-6)" />
                                    <div>
                                        <Text size="xs" c="dimmed">Usuarios</Text>
                                        <Text fw={700}>{detailModal.total_usuarios}</Text>
                                    </div>
                                </Group>
                            </Card>
                        </SimpleGrid>

                        {detailModal.plan && (
                            <>
                                <Text size="sm" fw={600} mt="sm">Limites del Plan</Text>
                                <SimpleGrid cols={2}>
                                    <div>
                                        <Text size="xs" c="dimmed">Max Terminales</Text>
                                        <Text>{detailModal.plan.max_terminales || 'Ilimitado'}</Text>
                                    </div>
                                    <div>
                                        <Text size="xs" c="dimmed">Max Productos</Text>
                                        <Text>{detailModal.plan.max_productos || 'Ilimitado'}</Text>
                                    </div>
                                </SimpleGrid>
                            </>
                        )}
                    </Stack>
                )}
            </Modal>
        </Stack>
    );
}
