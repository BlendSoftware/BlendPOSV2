import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    Center, Paper, Title, Text, Button, Stack, Group,
    Stepper, ThemeIcon, Box, List, TextInput, NumberInput,
    Select, Alert, Badge,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import {
    CheckCircle, ShoppingCart, Users, Receipt, Zap,
    AlertCircle, Upload, Package,
} from 'lucide-react';
import { updateConfiguracionFiscal } from '../services/api/configuracion_fiscal';

// ── Step metadata (informational steps) ─────────────────────────────────────

interface InfoStep {
    kind: 'info';
    label: string;
    icon: React.ReactNode;
    title: string;
    description: string;
    bullets: string[];
}

interface FiscalStep {
    kind: 'fiscal';
    label: string;
    icon: React.ReactNode;
    title: string;
    description: string;
}

interface CatalogStep {
    kind: 'catalog';
    label: string;
    icon: React.ReactNode;
    title: string;
    description: string;
    bullets: string[];
}

type OnboardingStep = InfoStep | FiscalStep | CatalogStep;

const STEPS: OnboardingStep[] = [
    {
        kind: 'info',
        label: 'Bienvenido',
        icon: <Zap size={20} />,
        title: 'Bienvenido a BlendPOS!',
        description: 'Tu cuenta fue creada con el plan Starter gratuito. En pocos pasos vas a estar listo para vender.',
        bullets: [
            '2 terminales de caja incluidas',
            'Facturación AFIP habilitada',
            'Catálogo ilimitado de productos',
            'Modo offline automático',
        ],
    },
    {
        kind: 'fiscal',
        label: 'Facturación',
        icon: <Receipt size={20} />,
        title: 'Configurá la facturación AFIP',
        description: 'Para emitir comprobantes electrónicos, cargá los datos fiscales de tu negocio. Si no facturás aún, podés saltear este paso.',
    },
    {
        kind: 'catalog',
        label: 'Productos',
        icon: <ShoppingCart size={20} />,
        title: 'Tus productos de ejemplo están listos',
        description: 'Cargamos categorías y productos de ejemplo según tu tipo de negocio. Podés editarlos, agregar más o importar desde CSV.',
        bullets: [
            'Ir a Admin -> Productos para ver los productos cargados',
            'Editalos con tus precios y stock reales',
            'Importación CSV: Admin -> Productos -> Importar CSV',
            'Las categorías ayudan a organizar el catálogo en el POS',
        ],
    },
    {
        kind: 'info',
        label: 'Usuarios',
        icon: <Users size={20} />,
        title: 'Invita a tu equipo',
        description: 'Podés crear cajeros y supervisores para que usen el POS sin acceder al panel de administración.',
        bullets: [
            'Ir a Admin -> Usuarios -> Nuevo usuario',
            'Roles: Cajero (solo POS), Supervisor (POS + reportes), Administrador (todo)',
            'Cada cajero puede tener su propio punto de venta',
        ],
    },
];

// ── Fiscal config sub-form ──────────────────────────────────────────────────

interface FiscalFormValues {
    cuit_emisor: string;
    razon_social: string;
    condicion_fiscal: string;
    punto_de_venta: number;
}

function FiscalConfigForm({ onSaved, onSkip }: { onSaved: () => void; onSkip: () => void }) {
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [saved, setSaved] = useState(false);

    const form = useForm<FiscalFormValues>({
        initialValues: {
            cuit_emisor: '',
            razon_social: '',
            condicion_fiscal: 'Monotributo',
            punto_de_venta: 1,
        },
        validate: {
            cuit_emisor: (v) =>
                /^\d{2}-?\d{8}-?\d$/.test(v.replace(/-/g, '').length === 11 ? v : '')
                    ? null
                    : 'CUIT inválido (11 dígitos)',
            razon_social: (v) => (v.trim().length >= 2 ? null : 'Mínimo 2 caracteres'),
            punto_de_venta: (v) => (v >= 1 && v <= 99999 ? null : 'Punto de venta entre 1 y 99999'),
        },
    });

    const handleSubmit = form.onSubmit(async (values) => {
        setError('');
        setLoading(true);
        try {
            await updateConfiguracionFiscal({
                cuit_emisor: values.cuit_emisor.replace(/-/g, ''),
                razon_social: values.razon_social,
                condicion_fiscal: values.condicion_fiscal,
                punto_de_venta: values.punto_de_venta,
                modo: 'homologacion',
            });
            setSaved(true);
            // Auto-advance after a brief moment
            setTimeout(onSaved, 800);
        } catch (e: unknown) {
            const msg = e instanceof Error ? e.message : 'Error al guardar. Intentá nuevamente.';
            setError(msg);
        } finally {
            setLoading(false);
        }
    });

    if (saved) {
        return (
            <Alert color="teal" variant="light" icon={<CheckCircle size={16} />}>
                Configuración fiscal guardada correctamente.
            </Alert>
        );
    }

    return (
        <form onSubmit={handleSubmit}>
            <Stack gap="md">
                {error && (
                    <Alert icon={<AlertCircle size={16} />} color="red" variant="light">
                        {error}
                    </Alert>
                )}

                <TextInput
                    label="CUIT"
                    placeholder="20-12345678-9"
                    required
                    {...form.getInputProps('cuit_emisor')}
                />
                <TextInput
                    label="Razón social"
                    placeholder="Mi Kiosco SRL"
                    required
                    {...form.getInputProps('razon_social')}
                />
                <Select
                    label="Condición fiscal"
                    data={[
                        { value: 'Monotributo', label: 'Monotributo' },
                        { value: 'Responsable Inscripto', label: 'Responsable Inscripto' },
                        { value: 'Exento', label: 'Exento' },
                    ]}
                    {...form.getInputProps('condicion_fiscal')}
                />
                <NumberInput
                    label="Punto de venta"
                    min={1}
                    max={99999}
                    required
                    {...form.getInputProps('punto_de_venta')}
                />

                <Group justify="space-between" mt="sm">
                    <Button variant="subtle" color="gray" onClick={onSkip}>
                        Saltear por ahora
                    </Button>
                    <Button type="submit" loading={loading}>
                        Guardar configuración
                    </Button>
                </Group>
            </Stack>
        </form>
    );
}

// ── Main page ───────────────────────────────────────────────────────────────

export function OnboardingPage() {
    const navigate = useNavigate();
    const [active, setActive] = useState(0);

    const isLast = active === STEPS.length - 1;
    const step = STEPS[active];

    const handleNext = () => {
        if (isLast) {
            navigate('/', { replace: true });
        } else {
            setActive((s) => s + 1);
        }
    };

    const renderStepContent = () => {
        if (step.kind === 'fiscal') {
            return (
                <Stack gap="md">
                    <Group gap="sm" align="flex-start">
                        <ThemeIcon size="lg" radius="xl" color="blue" variant="light">
                            {step.icon}
                        </ThemeIcon>
                        <div>
                            <Title order={4}>{step.title}</Title>
                            <Text size="sm" c="dimmed" mt={4}>{step.description}</Text>
                        </div>
                    </Group>
                    <FiscalConfigForm
                        onSaved={handleNext}
                        onSkip={handleNext}
                    />
                </Stack>
            );
        }

        if (step.kind === 'catalog') {
            return (
                <Stack gap="md">
                    <Group gap="sm" align="flex-start">
                        <ThemeIcon size="lg" radius="xl" color="teal" variant="light">
                            <Package size={20} />
                        </ThemeIcon>
                        <div>
                            <Title order={4}>{step.title}</Title>
                            <Text size="sm" c="dimmed" mt={4}>{step.description}</Text>
                        </div>
                    </Group>

                    <Alert color="teal" variant="light" icon={<CheckCircle size={16} />}>
                        Tu negocio esta listo. Cargamos categorias y productos de ejemplo para que arranques.
                    </Alert>

                    <List
                        spacing="xs"
                        size="sm"
                        icon={
                            <ThemeIcon size={16} radius="xl" color="teal" variant="light">
                                <CheckCircle size={10} />
                            </ThemeIcon>
                        }
                    >
                        {step.bullets.map((b) => (
                            <List.Item key={b}>{b}</List.Item>
                        ))}
                    </List>
                    <Group gap="xs">
                        <Badge color="teal" variant="light" leftSection={<Upload size={10} />}>
                            Importacion CSV disponible
                        </Badge>
                    </Group>
                </Stack>
            );
        }

        // InfoStep (welcome, users)
        return (
            <Stack gap="md">
                <Group gap="sm" align="flex-start">
                    <ThemeIcon size="lg" radius="xl" color="blue" variant="light">
                        {step.icon}
                    </ThemeIcon>
                    <div>
                        <Title order={4}>{step.title}</Title>
                        <Text size="sm" c="dimmed" mt={4}>{step.description}</Text>
                    </div>
                </Group>
                {'bullets' in step && (
                    <List
                        spacing="xs"
                        size="sm"
                        icon={
                            <ThemeIcon size={16} radius="xl" color="teal" variant="light">
                                <CheckCircle size={10} />
                            </ThemeIcon>
                        }
                    >
                        {step.bullets.map((b: string) => (
                            <List.Item key={b}>{b}</List.Item>
                        ))}
                    </List>
                )}
            </Stack>
        );
    };

    return (
        <Center style={{ minHeight: '100vh', background: 'var(--mantine-color-body)', padding: '2rem 1rem' }}>
            <Box w={560}>
                <Stack gap="xs" mb="xl" align="center">
                    <Title order={1} c="blue.4" fw={800} style={{ letterSpacing: '-1px' }}>
                        BlendPOS
                    </Title>
                    <Text c="dimmed" size="sm">Configuración inicial</Text>
                </Stack>

                <Paper p="xl" radius="md" withBorder>
                    <Stepper active={active} mb="xl" size="sm">
                        {STEPS.map((s) => (
                            <Stepper.Step key={s.label} label={s.label} icon={s.icon} />
                        ))}
                    </Stepper>

                    {renderStepContent()}

                    {/* Navigation buttons — hidden for the fiscal step (it has its own buttons) */}
                    {step.kind !== 'fiscal' && (
                        <Group justify="space-between" mt="xl">
                            <Button
                                variant="subtle"
                                color="gray"
                                disabled={active === 0}
                                onClick={() => setActive((s) => s - 1)}
                            >
                                Anterior
                            </Button>
                            <Button onClick={handleNext} color={isLast ? 'teal' : 'blue'}>
                                {isLast ? 'Empezar a vender!' : 'Siguiente'}
                            </Button>
                        </Group>
                    )}
                </Paper>

                <Text size="xs" c="dimmed" ta="center" mt="md">
                    Podés saltear la configuración y volver a esta guía desde el panel de administración.
                </Text>
            </Box>
        </Center>
    );
}
