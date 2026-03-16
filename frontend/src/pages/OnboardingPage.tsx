import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    Center, Paper, Title, Text, Button, Stack, Group,
    Stepper, ThemeIcon, Box, List,
} from '@mantine/core';
import { CheckCircle, ShoppingCart, Users, Receipt, Zap } from 'lucide-react';

const STEPS = [
    {
        label: 'Bienvenido',
        icon: <Zap size={20} />,
        title: '¡Bienvenido a BlendPOS!',
        description: 'Tu cuenta fue creada con el plan Starter gratuito. En pocos pasos vas a estar listo para vender.',
        bullets: [
            '2 terminales de caja incluidas',
            'Facturación AFIP habilitada',
            'Catálogo ilimitado de productos',
            'Modo offline automático',
        ],
    },
    {
        label: 'Productos',
        icon: <ShoppingCart size={20} />,
        title: 'Cargá tus productos',
        description: 'Podés agregar productos desde el Panel Admin. Cada producto puede tener precio, stock, código de barras y categoría.',
        bullets: [
            'Ir a Admin → Productos → Nuevo producto',
            'O importar masivamente desde CSV via Proveedores',
            'Las categorías ayudan a organizar el catálogo en el POS',
        ],
    },
    {
        label: 'Usuarios',
        icon: <Users size={20} />,
        title: 'Invitá a tu equipo',
        description: 'Podés crear cajeros y supervisores para que usen el POS sin acceder al panel de administración.',
        bullets: [
            'Ir a Admin → Usuarios → Nuevo usuario',
            'Roles: Cajero (solo POS), Supervisor (POS + reportes), Administrador (todo)',
            'Cada cajero puede tener su propio punto de venta',
        ],
    },
    {
        label: 'Facturación',
        icon: <Receipt size={20} />,
        title: 'Configurá la facturación AFIP',
        description: 'Para emitir comprobantes electrónicos, necesitás cargar los datos fiscales de tu negocio.',
        bullets: [
            'Ir a Admin → Configuración Fiscal',
            'Cargar CUIT, punto de venta y certificado AFIP',
            'Si no facturás aún, podés saltear este paso',
        ],
    },
];

export function OnboardingPage() {
    const navigate = useNavigate();
    const [active, setActive] = useState(0);

    const isLast = active === STEPS.length - 1;
    const step = STEPS[active];

    const handleNext = () => {
        if (isLast) {
            navigate('/admin/dashboard', { replace: true });
        } else {
            setActive((s) => s + 1);
        }
    };

    return (
        <Center style={{ minHeight: '100vh', background: 'var(--mantine-color-body)', padding: '2rem 1rem' }}>
            <Box w={520}>
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
                    </Stack>

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
                            {isLast ? '¡Empezar a vender!' : 'Siguiente'}
                        </Button>
                    </Group>
                </Paper>

                <Text size="xs" c="dimmed" ta="center" mt="md">
                    Podés saltear la configuración y volver a esta guía desde el panel de administración.
                </Text>
            </Box>
        </Center>
    );
}
