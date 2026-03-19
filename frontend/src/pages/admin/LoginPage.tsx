import { useState, useEffect } from 'react';
import { useNavigate, useLocation, Link } from 'react-router-dom';
import {
    Paper, Title, Text, TextInput, PasswordInput,
    Button, Stack, Alert, Box, Anchor,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { AlertCircle, ShieldAlert, WifiOff, Building2, FileCheck } from 'lucide-react';
import { useAuthStore } from '../../store/useAuthStore';
import { changePasswordApi } from '../../services/api/auth';
import classes from './LoginPage.module.css';

const FEATURES = [
    {
        icon: <WifiOff size={20} />,
        label: 'Funciona offline',
        description: '48hs de autonomía sin internet',
    },
    {
        icon: <Building2 size={20} />,
        label: 'Multi-sucursal',
        description: 'Gestioná todos tus locales desde un lugar',
    },
    {
        icon: <FileCheck size={20} />,
        label: 'AFIP integrado',
        description: 'Facturación electrónica automática',
    },
];

export function LoginPage() {
    const navigate = useNavigate();
    const location = useLocation();
    const { login, isAuthenticated, user, mustChangePassword, clearMustChangePassword } = useAuthStore();
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

    // Route guard: redirect if already authenticated AND password change is not required
    useEffect(() => {
        if (isAuthenticated && !mustChangePassword) {
            const isAdminRole = user?.rol === 'admin' || user?.rol === 'supervisor';
            navigate(isAdminRole ? '/admin/dashboard' : '/', { replace: true });
        }
    }, [isAuthenticated, user, mustChangePassword, navigate]);

    const from = (location.state as { from?: { pathname: string } })?.from?.pathname ?? '/';

    const form = useForm<{ email: string; password: string }>({
        initialValues: { email: '', password: '' },
        validate: {
            email: (v) => (v.trim().length >= 2 ? null : 'Ingrese usuario o email'),
            password: (v) => (v.length >= 4 ? null : 'Mínimo 4 caracteres'),
        },
    });

    const pwForm = useForm<{ newPassword: string; confirmPassword: string }>({
        initialValues: { newPassword: '', confirmPassword: '' },
        validate: {
            newPassword: (v) => (v.length >= 8 ? null : 'Mínimo 8 caracteres'),
            confirmPassword: (v, values) =>
                v === values.newPassword ? null : 'Las contraseñas no coinciden',
        },
    });

    const handleSubmit = form.onSubmit(async ({ email, password }) => {
        setError('');
        setLoading(true);
        const ok = await login(email, password);
        setLoading(false);
        if (ok) {
            // If must_change_password is set, the useEffect won't redirect —
            // we stay on this page and show the password change form.
            const needsChange = useAuthStore.getState().mustChangePassword;
            if (!needsChange) {
                const updatedUser = useAuthStore.getState().user;
                const isAdminRole = updatedUser?.rol === 'admin' || updatedUser?.rol === 'supervisor';
                const isAdminRoute = from.startsWith('/admin');
                if (isAdminRole && !isAdminRoute) {
                    navigate('/admin/dashboard', { replace: true });
                } else {
                    navigate(from === '/login' ? '/' : from, { replace: true });
                }
            }
        } else {
            setError('Credenciales inválidas o usuario inactivo.');
        }
    });

    const handleChangePassword = pwForm.onSubmit(async ({ newPassword }) => {
        setError('');
        setLoading(true);
        try {
            await changePasswordApi(newPassword);
            clearMustChangePassword();
            // Now redirect normally
            const updatedUser = useAuthStore.getState().user;
            const isAdminRole = updatedUser?.rol === 'admin' || updatedUser?.rol === 'supervisor';
            navigate(isAdminRole ? '/admin/dashboard' : '/', { replace: true });
        } catch {
            setError('Error al cambiar la contraseña. Intente nuevamente.');
        } finally {
            setLoading(false);
        }
    });

    // ── SEC-03: Forced password change form ──────────────────────────────────
    if (isAuthenticated && mustChangePassword) {
        return (
            <div className={classes.wrapper}>
                <div className={classes.hero}>
                    <div className={classes.heroContent}>
                        <div className={classes.logo}>BlendPOS</div>
                        <p className={classes.tagline}>
                            Tu POS inteligente para el comercio argentino
                        </p>
                    </div>
                </div>

                <div className={classes.formSide}>
                    <div className={`${classes.formContainer} ${classes.fadeIn}`}>
                        {/* Mobile branding */}
                        <div className={classes.mobileBranding}>
                            <div className={classes.mobileLogo}>BlendPOS</div>
                            <span className={classes.mobileTagline}>Cambio de contraseña obligatorio</span>
                        </div>

                        <Paper p="xl" radius="md" withBorder className={classes.formCard}>
                            <Title order={3} mb="lg">Cambiar contraseña</Title>

                            <Alert icon={<ShieldAlert size={16} />} color="orange" mb="md" variant="light">
                                Por seguridad, debe cambiar su contraseña antes de continuar.
                            </Alert>

                            {error && (
                                <Alert icon={<AlertCircle size={16} />} color="red" mb="md" variant="light">
                                    {error}
                                </Alert>
                            )}

                            <form onSubmit={handleChangePassword}>
                                <Stack gap="md">
                                    <PasswordInput
                                        label="Nueva contraseña"
                                        placeholder="Mínimo 8 caracteres"
                                        {...pwForm.getInputProps('newPassword')}
                                        data-autofocus
                                    />
                                    <PasswordInput
                                        label="Confirmar contraseña"
                                        placeholder="Repetir contraseña"
                                        {...pwForm.getInputProps('confirmPassword')}
                                    />
                                    <Button type="submit" fullWidth loading={loading} mt="sm" color="orange">
                                        Cambiar contraseña
                                    </Button>
                                </Stack>
                            </form>
                        </Paper>
                    </div>
                </div>
            </div>
        );
    }

    // ── Normal login form ────────────────────────────────────────────────────
    return (
        <div className={classes.wrapper}>
            {/* Left column: Hero / Branding */}
            <div className={classes.hero}>
                <div className={`${classes.heroContent} ${classes.fadeIn}`}>
                    <div className={classes.logo}>BlendPOS</div>
                    <p className={classes.tagline}>
                        Tu POS inteligente para el comercio argentino
                    </p>

                    <div className={classes.featureList}>
                        {FEATURES.map((f) => (
                            <div key={f.label} className={classes.featureItem}>
                                <div className={classes.featureIcon}>
                                    {f.icon}
                                </div>
                                <div>
                                    <div className={classes.featureLabel}>{f.label}</div>
                                    <div className={classes.featureText}>{f.description}</div>
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
            </div>

            {/* Right column: Form */}
            <div className={classes.formSide}>
                <div className={`${classes.formContainer} ${classes.fadeInDelay}`}>
                    {/* Mobile branding */}
                    <div className={classes.mobileBranding}>
                        <div className={classes.mobileLogo}>BlendPOS</div>
                        <span className={classes.mobileTagline}>Panel de Administración</span>
                    </div>

                    <Paper p="xl" radius="md" withBorder className={classes.formCard}>
                        <Title order={3} mb="xs">Iniciar sesión</Title>
                        <Text size="sm" c="dimmed" mb="lg">
                            Ingresá tus credenciales para acceder
                        </Text>

                        {error && (
                            <Alert icon={<AlertCircle size={16} />} color="red" mb="md" variant="light">
                                {error}
                            </Alert>
                        )}

                        <form onSubmit={handleSubmit}>
                            <Stack gap="md">
                                <TextInput
                                    label="Usuario o Email"
                                    placeholder="admin"
                                    {...form.getInputProps('email')}
                                    data-autofocus
                                />
                                <PasswordInput
                                    label="Contraseña"
                                    placeholder="Tu contraseña"
                                    {...form.getInputProps('password')}
                                />
                                <Button type="submit" fullWidth loading={loading} mt="sm" size="md">
                                    Ingresar
                                </Button>
                            </Stack>
                        </form>

                        <Text size="sm" c="dimmed" ta="center" mt="lg">
                            ¿No tenés cuenta?{' '}
                            <Anchor component={Link} to="/register" size="sm" fw={600}>
                                Creá tu negocio gratis
                            </Anchor>
                        </Text>
                    </Paper>
                </div>
            </div>
        </div>
    );
}
