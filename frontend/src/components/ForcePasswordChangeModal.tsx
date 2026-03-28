// ─────────────────────────────────────────────────────────────────────────────
// ForcePasswordChangeModal — SEC-03
// Non-dismissible modal that forces password change on first login.
// Renders when mustChangePassword is true in the auth store.
// ─────────────────────────────────────────────────────────────────────────────

import { useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { Modal, Stack, Title, Text, PasswordInput, Button, Alert } from '@mantine/core';
import { useForm } from '@mantine/form';
import { AlertCircle, Lock } from 'lucide-react';
import { useAuthStore } from '../store/useAuthStore';
import { changePasswordApi } from '../services/api/auth';

interface FormValues {
    newPassword: string;
    confirmPassword: string;
}

export function ForcePasswordChangeModal() {
    const mustChangePassword = useAuthStore((s) => s.mustChangePassword);
    const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
    const clearMustChangePassword = useAuthStore((s) => s.clearMustChangePassword);
    const navigate = useNavigate();
    const location = useLocation();

    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

    const form = useForm<FormValues>({
        initialValues: {
            newPassword: '',
            confirmPassword: '',
        },
        validate: {
            newPassword: (v) => (v.length >= 8 ? null : 'Mínimo 8 caracteres'),
            confirmPassword: (v, values) =>
                v === values.newPassword ? null : 'Las contraseñas no coinciden',
        },
    });

    const handleSubmit = form.onSubmit(async (values) => {
        setError('');
        setLoading(true);
        try {
            await changePasswordApi(values.newPassword);
            clearMustChangePassword();
            navigate('/onboarding', { replace: true });
        } catch (e: unknown) {
            const msg = e instanceof Error ? e.message : 'Error al cambiar la contraseña';
            setError(msg);
        } finally {
            setLoading(false);
        }
    });

    // Don't render when on login page — LoginPage handles its own password change UI
    if (!isAuthenticated || !mustChangePassword || location.pathname === '/login') return null;

    return (
        <Modal
            opened
            onClose={() => {/* Non-dismissible */}}
            withCloseButton={false}
            closeOnClickOutside={false}
            closeOnEscape={false}
            centered
            size="sm"
            overlayProps={{ backgroundOpacity: 0.65, blur: 4 }}
        >
            <form onSubmit={handleSubmit}>
                <Stack gap="md">
                    <Stack gap={4} align="center">
                        <Lock size={32} />
                        <Title order={3} ta="center">Cambia tu contraseña</Title>
                        <Text size="sm" c="dimmed" ta="center">
                            Por seguridad, tenés que cambiar tu contraseña antes de continuar.
                        </Text>
                    </Stack>

                    {error && (
                        <Alert icon={<AlertCircle size={16} />} color="red" variant="light">
                            {error}
                        </Alert>
                    )}

                    <PasswordInput
                        label="Nueva contraseña"
                        placeholder="Mínimo 8 caracteres"
                        required
                        {...form.getInputProps('newPassword')}
                    />
                    <PasswordInput
                        label="Confirmar nueva contraseña"
                        placeholder="Repetí la nueva contraseña"
                        required
                        {...form.getInputProps('confirmPassword')}
                    />

                    <Button type="submit" fullWidth loading={loading} mt="sm">
                        Cambiar contraseña
                    </Button>
                </Stack>
            </form>
        </Modal>
    );
}
