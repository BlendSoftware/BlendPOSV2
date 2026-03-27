// ─────────────────────────────────────────────────────────────────────────────
// ForcePasswordChangeModal — SEC-03: Mandatory password change on first login.
// Non-dismissible modal that blocks all interaction until password is changed.
// ─────────────────────────────────────────────────────────────────────────────

import { useState } from 'react';
import { Modal, Stack, PasswordInput, Button, Alert, Text, Title } from '@mantine/core';
import { useForm } from '@mantine/form';
import { AlertCircle, Lock } from 'lucide-react';
import { useAuthStore } from '../../store/useAuthStore';
import { changePasswordApi } from '../../services/api/auth';

interface FormValues {
    currentPassword: string;
    newPassword: string;
    confirmPassword: string;
}

export function ForcePasswordChangeModal() {
    const mustChangePassword = useAuthStore((s) => s.mustChangePassword);
    const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
    const clearMustChangePassword = useAuthStore((s) => s.clearMustChangePassword);

    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

    const form = useForm<FormValues>({
        initialValues: {
            currentPassword: '',
            newPassword: '',
            confirmPassword: '',
        },
        validate: {
            currentPassword: (v) =>
                v.trim().length > 0 ? null : 'Ingresá tu contraseña actual',
            newPassword: (v) =>
                v.length >= 8 ? null : 'Mínimo 8 caracteres',
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
        } catch (e: unknown) {
            const msg = e instanceof Error ? e.message : 'Error al cambiar la contraseña. Intentá nuevamente.';
            setError(msg);
        } finally {
            setLoading(false);
        }
    });

    // Only show when authenticated AND must change password
    if (!isAuthenticated || !mustChangePassword) return null;

    return (
        <Modal
            opened
            onClose={() => {/* non-dismissible */}}
            withCloseButton={false}
            closeOnClickOutside={false}
            closeOnEscape={false}
            centered
            size="sm"
            overlayProps={{ backgroundOpacity: 0.7, blur: 4 }}
        >
            <form onSubmit={handleSubmit}>
                <Stack gap="md">
                    <Stack gap="xs" align="center">
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
                        label="Contrasena actual"
                        placeholder="Ingresá tu contraseña actual"
                        required
                        {...form.getInputProps('currentPassword')}
                    />
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

                    <Button type="submit" loading={loading} fullWidth mt="sm">
                        Cambiar contraseña
                    </Button>
                </Stack>
            </form>
        </Modal>
    );
}
