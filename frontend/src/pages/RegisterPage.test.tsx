import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { RegisterPage } from './RegisterPage';

// ── Mocks ────────────────────────────────────────────────────────────────────

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
    const actual = await vi.importActual('react-router-dom');
    return { ...actual, useNavigate: () => mockNavigate };
});

vi.mock('../services/api/tenant', () => ({
    registerTenant: vi.fn(),
}));

vi.mock('../store/tokenStore', () => ({
    tokenStore: {
        setTokens: vi.fn(),
        getAccessToken: vi.fn(() => null),
        getRefreshToken: vi.fn(() => null),
        clearTokens: vi.fn(),
    },
}));

vi.mock('../store/useAuthStore', () => ({
    useAuthStore: Object.assign(
        vi.fn((selector: (s: Record<string, unknown>) => unknown) =>
            selector({ user: null, isAuthenticated: false }),
        ),
        {
            getState: vi.fn(() => ({
                refresh: vi.fn(async () => true),
                user: null,
                isAuthenticated: false,
            })),
        },
    ),
}));

vi.mock('@mantine/notifications', () => ({
    notifications: { show: vi.fn() },
}));

function renderPage() {
    return render(
        <MantineProvider>
            <MemoryRouter initialEntries={['/register']}>
                <RegisterPage />
            </MemoryRouter>
        </MantineProvider>,
    );
}

describe('RegisterPage', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('renders the registration form with required fields', () => {
        renderPage();

        expect(screen.getByLabelText(/nombre del negocio/i)).toBeInTheDocument();
        expect(screen.getByLabelText(/identificador único/i)).toBeInTheDocument();
        expect(screen.getByLabelText(/tu nombre/i)).toBeInTheDocument();
        expect(screen.getByLabelText(/usuario administrador/i)).toBeInTheDocument();
        expect(screen.getByText(/crear cuenta/i)).toBeInTheDocument();
    });

    it('validates that password must be at least 8 characters', async () => {
        renderPage();
        const user = userEvent.setup();

        // Fill all required fields but with a short password
        await user.type(screen.getByLabelText(/nombre del negocio/i), 'Mi Kiosco');
        await user.type(screen.getByLabelText(/identificador único/i), 'mikiosco');
        await user.type(screen.getByLabelText(/tu nombre/i), 'Juan Perez');
        await user.type(screen.getByLabelText(/usuario administrador/i), 'admin');

        // Password fields - get them by label
        const passwordInputs = screen.getAllByLabelText(/contraseña/i);
        const passwordField = passwordInputs[0];
        const confirmField = passwordInputs[1];

        await user.type(passwordField, '1234');
        await user.type(confirmField, '1234');

        await user.click(screen.getByText(/crear cuenta/i));

        await waitFor(() => {
            expect(screen.getByText(/mínimo 8 caracteres/i)).toBeInTheDocument();
        });
    });

    it('requires password confirmation to match', async () => {
        renderPage();
        const user = userEvent.setup();

        await user.type(screen.getByLabelText(/nombre del negocio/i), 'Mi Kiosco');
        await user.type(screen.getByLabelText(/identificador único/i), 'mikiosco');
        await user.type(screen.getByLabelText(/tu nombre/i), 'Juan Perez');
        await user.type(screen.getByLabelText(/usuario administrador/i), 'admin');

        const passwordInputs = screen.getAllByLabelText(/contraseña/i);
        await user.type(passwordInputs[0], 'securepass123');
        await user.type(passwordInputs[1], 'differentpass');

        await user.click(screen.getByText(/crear cuenta/i));

        await waitFor(() => {
            expect(screen.getByText(/las contraseñas no coinciden/i)).toBeInTheDocument();
        });
    });

    it('calls registerTenant on valid submission', async () => {
        const { registerTenant } = await import('../services/api/tenant');
        const mockRegister = vi.mocked(registerTenant);
        mockRegister.mockResolvedValueOnce({
            tenant: {
                id: 'tenant-1',
                slug: 'mikiosco',
                nombre: 'Mi Kiosco',
                activo: true,
                created_at: '2026-01-01T00:00:00Z',
            },
            access_token: 'eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJ0aWQiOiJ0ZW5hbnQtMSJ9.fake',
            refresh_token: 'refresh-fake',
            token_type: 'Bearer',
            expires_in: 3600,
        });

        renderPage();
        const user = userEvent.setup();

        await user.type(screen.getByLabelText(/nombre del negocio/i), 'Mi Kiosco');
        await user.type(screen.getByLabelText(/identificador único/i), 'mikiosco');
        await user.type(screen.getByLabelText(/tu nombre/i), 'Juan Perez');
        await user.type(screen.getByLabelText(/usuario administrador/i), 'admin');

        const passwordInputs = screen.getAllByLabelText(/contraseña/i);
        await user.type(passwordInputs[0], 'securepass123');
        await user.type(passwordInputs[1], 'securepass123');

        await user.click(screen.getByText(/crear cuenta/i));

        await waitFor(() => {
            expect(mockRegister).toHaveBeenCalledWith(
                expect.objectContaining({
                    nombre_negocio: 'Mi Kiosco',
                    slug: 'mikiosco',
                    nombre: 'Juan Perez',
                    username: 'admin',
                    password: 'securepass123',
                }),
            );
        });

        await waitFor(() => {
            expect(mockNavigate).toHaveBeenCalledWith('/onboarding', { replace: true });
        });
    });

    it('shows error message when registration fails', async () => {
        const { registerTenant } = await import('../services/api/tenant');
        vi.mocked(registerTenant).mockRejectedValueOnce(new Error('El slug ya existe'));

        renderPage();
        const user = userEvent.setup();

        await user.type(screen.getByLabelText(/nombre del negocio/i), 'Mi Kiosco');
        await user.type(screen.getByLabelText(/identificador único/i), 'mikiosco');
        await user.type(screen.getByLabelText(/tu nombre/i), 'Juan');
        await user.type(screen.getByLabelText(/usuario administrador/i), 'admin');

        const passwordInputs = screen.getAllByLabelText(/contraseña/i);
        await user.type(passwordInputs[0], 'securepass123');
        await user.type(passwordInputs[1], 'securepass123');

        await user.click(screen.getByText(/crear cuenta/i));

        await waitFor(() => {
            expect(screen.getByText(/el slug ya existe/i)).toBeInTheDocument();
        });
    });
});
