import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { OnboardingPage } from './OnboardingPage';

// ── Mocks ────────────────────────────────────────────────────────────────────

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
    const actual = await vi.importActual('react-router-dom');
    return { ...actual, useNavigate: () => mockNavigate };
});

vi.mock('../services/api/configuracion_fiscal', () => ({
    updateConfiguracionFiscal: vi.fn(),
    getConfiguracionFiscal: vi.fn(),
}));

function renderPage() {
    return render(
        <MantineProvider>
            <MemoryRouter initialEntries={['/onboarding']}>
                <OnboardingPage />
            </MemoryRouter>
        </MantineProvider>,
    );
}

describe('OnboardingPage', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('renders stepper with all steps', () => {
        renderPage();
        expect(screen.getByText('Bienvenido')).toBeInTheDocument();
        expect(screen.getByText('Facturación')).toBeInTheDocument();
        expect(screen.getByText('Productos')).toBeInTheDocument();
        expect(screen.getByText('Usuarios')).toBeInTheDocument();
    });

    it('shows welcome step content initially', () => {
        renderPage();
        expect(screen.getByText(/bienvenido a blendpos/i)).toBeInTheDocument();
        expect(screen.getByText(/plan starter gratuito/i)).toBeInTheDocument();
    });

    it('navigates to fiscal config step on Siguiente', async () => {
        renderPage();
        const user = userEvent.setup();

        await user.click(screen.getByText('Siguiente'));

        await waitFor(() => {
            expect(screen.getByText(/configura la facturacion afip/i)).toBeInTheDocument();
        });
    });

    it('shows fiscal config form with CUIT and punto de venta fields', async () => {
        renderPage();
        const user = userEvent.setup();

        // Go to fiscal step
        await user.click(screen.getByText('Siguiente'));

        await waitFor(() => {
            expect(screen.getByLabelText(/cuit/i)).toBeInTheDocument();
            expect(screen.getByLabelText(/razon social/i)).toBeInTheDocument();
            expect(screen.getByLabelText(/punto de venta/i)).toBeInTheDocument();
        });
    });

    it('allows skipping fiscal config step', async () => {
        renderPage();
        const user = userEvent.setup();

        // Go to fiscal step
        await user.click(screen.getByText('Siguiente'));

        await waitFor(() => {
            expect(screen.getByText(/saltear por ahora/i)).toBeInTheDocument();
        });

        await user.click(screen.getByText(/saltear por ahora/i));

        // Should advance to Productos step
        await waitFor(() => {
            expect(screen.getByText(/carga tus productos/i)).toBeInTheDocument();
        });
    });

    it('shows catalog step with CSV coming soon indicator', async () => {
        renderPage();
        const user = userEvent.setup();

        // Go to fiscal step, skip it, land on catalog
        await user.click(screen.getByText('Siguiente'));
        await waitFor(() => {
            expect(screen.getByText(/saltear por ahora/i)).toBeInTheDocument();
        });
        await user.click(screen.getByText(/saltear por ahora/i));

        await waitFor(() => {
            // Text appears in both a list item and a badge
            const matches = screen.getAllByText(/proximamente/i);
            expect(matches.length).toBeGreaterThanOrEqual(1);
        });
    });

    it('navigates to POS on last step completion', async () => {
        renderPage();
        const user = userEvent.setup();

        // Step 0 -> 1 (welcome -> fiscal)
        await user.click(screen.getByText('Siguiente'));
        // Step 1 -> 2 (fiscal -> catalog, via skip)
        await waitFor(() => {
            expect(screen.getByText(/saltear por ahora/i)).toBeInTheDocument();
        });
        await user.click(screen.getByText(/saltear por ahora/i));
        // Step 2 -> 3 (catalog -> users)
        await waitFor(() => {
            expect(screen.getByText('Siguiente')).toBeInTheDocument();
        });
        await user.click(screen.getByText('Siguiente'));
        // Step 3 -> POS (users -> done)
        await waitFor(() => {
            expect(screen.getByText(/empezar a vender/i)).toBeInTheDocument();
        });
        await user.click(screen.getByText(/empezar a vender/i));

        expect(mockNavigate).toHaveBeenCalledWith('/', { replace: true });
    });

    it('submits fiscal config and shows success', async () => {
        const { updateConfiguracionFiscal } = await import('../services/api/configuracion_fiscal');
        vi.mocked(updateConfiguracionFiscal).mockResolvedValueOnce({
            message: 'Configuracion guardada',
        });

        renderPage();
        const user = userEvent.setup();

        // Go to fiscal step
        await user.click(screen.getByText('Siguiente'));

        await waitFor(() => {
            expect(screen.getByLabelText(/cuit/i)).toBeInTheDocument();
        });

        await user.type(screen.getByLabelText(/cuit/i), '20-12345678-9');
        await user.type(screen.getByLabelText(/razon social/i), 'Mi Kiosco SRL');
        // punto_de_venta defaults to 1, leave it

        await user.click(screen.getByText(/guardar configuracion/i));

        await waitFor(() => {
            expect(vi.mocked(updateConfiguracionFiscal)).toHaveBeenCalledWith(
                expect.objectContaining({
                    cuit_emisor: '20123456789',
                    razon_social: 'Mi Kiosco SRL',
                    punto_de_venta: 1,
                }),
            );
        });

        // Should show success message
        await waitFor(() => {
            expect(screen.getByText(/guardada correctamente/i)).toBeInTheDocument();
        });
    });
});
