import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import {
    Center, Paper, Title, Text, TextInput, PasswordInput,
    Button, Stack, Alert, Box, Anchor,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { AlertCircle } from 'lucide-react';
import { registerTenant } from '../services/api/tenant';
import { tokenStore } from '../store/tokenStore';
import { useAuthStore } from '../store/useAuthStore';

export function RegisterPage() {
    const navigate = useNavigate();
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

    const form = useForm({
        initialValues: {
            nombre_negocio: '',
            slug: '',
            nombre: '',
            username: '',
            email: '',
            password: '',
            confirm_password: '',
        },
        validate: {
            nombre_negocio: (v) => (v.trim().length >= 2 ? null : 'Mínimo 2 caracteres'),
            slug: (v) => (/^[a-z0-9]{2,63}$/.test(v) ? null : 'Solo letras minúsculas y números, 2-63 caracteres'),
            nombre: (v) => (v.trim().length >= 2 ? null : 'Mínimo 2 caracteres'),
            username: (v) => (v.trim().length >= 3 ? null : 'Mínimo 3 caracteres'),
            password: (v) => (v.length >= 8 ? null : 'Mínimo 8 caracteres'),
            confirm_password: (v, values) => (v === values.password ? null : 'Las contraseñas no coinciden'),
            email: (v) => (!v || /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v) ? null : 'Email inválido'),
        },
    });

    const handleSubmit = form.onSubmit(async (values) => {
        setError('');
        setLoading(true);
        try {
            const resp = await registerTenant({
                nombre_negocio: values.nombre_negocio,
                slug: values.slug.toLowerCase(),
                nombre: values.nombre,
                username: values.username,
                password: values.password,
                email: values.email || undefined,
            });
            // Log in immediately — tokens issued by registration
            tokenStore.setTokens(resp.access_token, resp.refresh_token);
            // Sync auth store state via initAuth (reads token from memory)
            await useAuthStore.getState().refresh();
            navigate('/onboarding', { replace: true });
        } catch (e: unknown) {
            const msg = e instanceof Error ? e.message : 'Error al registrar. Intente nuevamente.';
            setError(msg);
        } finally {
            setLoading(false);
        }
    });

    return (
        <Center style={{ minHeight: '100vh', background: 'var(--mantine-color-body)', padding: '2rem 0' }}>
            <Box w={420}>
                <Stack gap="xs" mb="xl" align="center">
                    <Title order={1} c="blue.4" fw={800} style={{ letterSpacing: '-1px' }}>
                        BlendPOS
                    </Title>
                    <Text c="dimmed" size="sm">Creá tu cuenta — es gratis para empezar</Text>
                </Stack>

                <Paper p="xl" radius="md" withBorder>
                    <Title order={3} mb="lg">Registrar negocio</Title>

                    {error && (
                        <Alert icon={<AlertCircle size={16} />} color="red" mb="md" variant="light">
                            {error}
                        </Alert>
                    )}

                    <form onSubmit={handleSubmit}>
                        <Stack gap="md">
                            <TextInput
                                label="Nombre del negocio"
                                placeholder="Mi Kiosco"
                                required
                                {...form.getInputProps('nombre_negocio')}
                                data-autofocus
                            />
                            <TextInput
                                label="Identificador único (slug)"
                                placeholder="mikiosco"
                                description="Solo letras minúsculas y números, sin espacios"
                                required
                                {...form.getInputProps('slug')}
                                onChange={(e) => form.setFieldValue('slug', e.currentTarget.value.toLowerCase().replace(/[^a-z0-9]/g, ''))}
                            />
                            <TextInput
                                label="Tu nombre"
                                placeholder="Juan Pérez"
                                required
                                {...form.getInputProps('nombre')}
                            />
                            <TextInput
                                label="Usuario administrador"
                                placeholder="admin"
                                required
                                {...form.getInputProps('username')}
                            />
                            <TextInput
                                label="Email (opcional)"
                                placeholder="juan@mikiosco.com"
                                type="email"
                                {...form.getInputProps('email')}
                            />
                            <PasswordInput
                                label="Contraseña"
                                placeholder="Mínimo 8 caracteres"
                                required
                                {...form.getInputProps('password')}
                            />
                            <PasswordInput
                                label="Confirmar contraseña"
                                placeholder="Repetir contraseña"
                                required
                                {...form.getInputProps('confirm_password')}
                            />
                            <Button type="submit" fullWidth loading={loading} mt="sm">
                                Crear cuenta
                            </Button>
                        </Stack>
                    </form>

                    <Text size="sm" c="dimmed" ta="center" mt="md">
                        ¿Ya tenés cuenta?{' '}
                        <Anchor component={Link} to="/login" size="sm">
                            Iniciá sesión
                        </Anchor>
                    </Text>
                </Paper>
            </Box>
        </Center>
    );
}
