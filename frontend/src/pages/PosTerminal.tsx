import { useRef, useCallback, useState, useEffect } from 'react';
import { TextInput, Loader, ActionIcon, Tooltip } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { ScanLine, AlertCircle, Search, Percent, CreditCard, Clock } from 'lucide-react';
import { useIsTouchDevice } from '../hooks/useIsTouchDevice';

import { PosHeader } from '../components/pos/PosHeader';
import { SalesTable } from '../components/pos/SalesTable';
import { TotalPanel } from '../components/pos/TotalPanel';
import { HotkeysFooter } from '../components/pos/HotkeysFooter';
import { PaymentModal } from '../components/pos/PaymentModal';
// ComprobanteModal ya no es necesario - se selecciona el tipo dentro de PaymentModal
// import { ComprobanteModal } from '../components/pos/ComprobanteModal';
import { ProductSearch } from '../components/pos/ProductSearch';
import { PriceCheckModal } from '../components/pos/PriceCheckModal';
import { DiscountModal } from '../components/pos/DiscountModal';
import { SaleHistoryModal } from '../components/pos/SaleHistoryModal';
import { PostSaleModal } from '../components/pos/PostSaleModal';
import { AbrirCajaModal } from '../components/pos/AbrirCajaModal';
import { WeightInputModal } from '../components/pos/WeightInputModal';
import { QuickProducts } from '../components/pos/QuickProducts';

import { useCartStore } from '../store/useCartStore';
import type { UnidadMedida } from '../store/useCartStore';
import { usePOSUIStore } from '../store/usePOSUIStore';
import { useSaleStore } from '../store/useSaleStore';
import { useAuthStore } from '../store/useAuthStore';
import { useCajaStore } from '../store/useCajaStore';
import { usePromocionesStore } from '../store/usePromocionesStore';
import { usePosThemeStore } from '../store/usePosThemeStore';
import { usePosFocus } from '../hooks/usePosFocus';
import { findCatalogProductByBarcode, searchCatalogProducts, seedCatalogFromMocksIfEmpty, forceRefreshCatalog } from '../offline/catalog';
import { getPrecioPorBarcode } from '../services/api/products';

import styles from './PosTerminal.module.css';

function formatCurrency(value: number): string {
    return new Intl.NumberFormat('es-AR', {
        style: 'currency',
        currency: 'ARS',
        minimumFractionDigits: 2,
    }).format(value);
}

export function PosTerminal() {
    const scannerRef = useRef<HTMLInputElement>(null);
    const searchRef = useRef<HTMLInputElement>(null);
    const [scannerValue, setScannerValue] = useState('');
    const [scannerFeedback, setScannerFeedback] = useState<'idle' | 'success' | 'error'>('idle');
    const [searchVisible, setSearchVisible] = useState(false);
    const [searchInitialQuery, setSearchInitialQuery] = useState('');
    const [historyOpen, setHistoryOpen] = useState(false);

    // ── Weight input modal state ────────────────────────────────────────
    const [weightModalOpen, setWeightModalOpen] = useState(false);
    const [weightProduct, setWeightProduct] = useState<{
        id: string;
        nombre: string;
        precio: number;
        codigoBarras: string;
        unidadMedida: 'kg' | 'gramo';
    } | null>(null);

    const {
        cart,
        total,
        totalConDescuento,
        addItem,
        addWeightItem,
        clearCart,
        moveSelectionUp,
        moveSelectionDown,
        removeSelectedItem,
        selectedRowIndex,
        updateQuantity,
        setPromoDiscounts,
    } = useCartStore();

    const totalUnits = cart.reduce((sum, item) => sum + item.cantidad, 0);
    const ahorroActual = Math.max(0, total - totalConDescuento);
    const selectedItem = selectedRowIndex >= 0 && selectedRowIndex < cart.length ? cart[selectedRowIndex] : null;

    const {
        isPaymentModalOpen,
        isPriceCheckModalOpen,
        isDiscountModalOpen,
        // isComprobanteModalOpen, // Ya no se usa
        openPaymentModal,
        closePaymentModal,
        // openComprobanteModal, // Ya no se usa
        // closeComprobanteModal, // Ya no se usa
        openPriceCheckModal,
        closePriceCheckModal,
        openDiscountModal,
        closeDiscountModal,
        openItemDiscountModal,
    } = usePOSUIStore();

    const setCajero = useSaleStore((s) => s.setCajero);
    const syncTicketCounter = useSaleStore((s) => s.syncTicketCounter);
    const { user } = useAuthStore();
    const { sesionId, restaurar } = useCajaStore();
    const { fetchActivePromociones, promociones, computePromoDescuentos } = usePromocionesStore();
    const activeTheme = usePosThemeStore((s) => s.activeTheme);
    const [isInitializing, setIsInitializing] = useState(true);

    // Al montar, sincronizar sesión de caja con el backend (limpia localStorage obsoleto)
    useEffect(() => {
        restaurar()
            .catch((err) => {
                console.warn('Error al restaurar sesión de caja:', err);
            })
            .finally(() => setIsInitializing(false));
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    // Sincronizar el contador de tickets con el backend al iniciar
    useEffect(() => {
        syncTicketCounter().catch(console.warn);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    // Mostrar modal de apertura de caja si no hay sesión activa.
    // SOLO se evalúa cuando isInitializing es false (después de consultar el backend).
    const [cajaModalOpen, setCajaModalOpen] = useState(false);
    useEffect(() => {
        if (isInitializing) return;
        // Pequeño delay para evitar condición de carrera
        const timer = setTimeout(() => {
            const shouldOpen = !sesionId;
            setCajaModalOpen(shouldOpen);
        }, 100);
        return () => clearTimeout(timer);
    }, [sesionId, isInitializing]);

    // Sincronizar catálogo desde el backend en cada apertura del POS.
    // Usa forceRefreshCatalog para garantizar que nuevos productos del admin
    // sean siempre visibles sin importar el estado del IndexedDB local.
    // También precargamos las promociones activas para aplicarlas al agregar productos.
    useEffect(() => {
        forceRefreshCatalog().catch(() => seedCatalogFromMocksIfEmpty().catch(console.warn));
        fetchActivePromociones();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    // Sync the cashier name from the auth store into the sale store
    useEffect(() => {
        if (user?.nombre) setCajero(user.nombre);
    }, [user?.nombre, setCajero]);

    // ── Combo / quantity promotion detection ────────────────────────────
    // Re-runs when the cart's product composition or quantities change, or when
    // the promotions list is refreshed (e.g. on POS mount).
    const cartKey = cart.map((c) => `${c.id}:${c.cantidad}`).sort().join(',');
    useEffect(() => {
        const cartProductIds = cart.map((c) => c.id);
        const priceMap: Record<string, number> = {};
        const quantityMap: Record<string, number> = {};
        cart.forEach((c) => { priceMap[c.id] = c.precio; quantityMap[c.id] = c.cantidad; });
        const { descuentos, promoNombres } = computePromoDescuentos(cartProductIds, priceMap, quantityMap);
        setPromoDiscounts(descuentos, promoNombres);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [cartKey, promociones]);

    const { isTouch } = useIsTouchDevice();

    const anyModalOpen = isPaymentModalOpen || isPriceCheckModalOpen || isDiscountModalOpen || historyOpen || weightModalOpen || searchVisible;

    // Sticky focus: auto-return to scanner after 2s inactivity
    usePosFocus(scannerRef, anyModalOpen);

    // ── Helper: add product or open weight modal ─────────────────────
    const addOrPromptWeight = useCallback(
        (product: { id: string; nombre: string; precio: number; codigoBarras: string; unidadMedida?: string }) => {
            const um = (product.unidadMedida ?? 'unidad') as UnidadMedida;
            if (um === 'kg' || um === 'gramo') {
                setWeightProduct({
                    id: product.id,
                    nombre: product.nombre,
                    precio: product.precio,
                    codigoBarras: product.codigoBarras,
                    unidadMedida: um,
                });
                setWeightModalOpen(true);
            } else {
                addItem({ id: product.id, nombre: product.nombre, precio: product.precio, codigoBarras: product.codigoBarras, unidadMedida: um });
            }
        },
        [addItem],
    );

    // ── Añadir producto por código de barras o nombre ─────────────────
    const handleAddProduct = useCallback(
        async (value: string): Promise<boolean> => {
            const trimmed = value.trim();
            if (!trimmed) return false;

            // 1. Precio en tiempo real desde el backend (con datos frescos)
            if (import.meta.env.VITE_API_BASE && navigator.onLine) {
                try {
                    const apiProduct = await getPrecioPorBarcode(trimmed);
                    // Check stock before adding
                    if (apiProduct.stock_disponible <= 0) {
                        notifications.show({
                            title: 'Sin stock',
                            message: `"${apiProduct.nombre}" no tiene stock disponible`,
                            color: 'orange',
                            icon: <AlertCircle size={16} />,
                            autoClose: 3000,
                        });
                        return false;
                    }
                    const local = await findCatalogProductByBarcode(trimmed);
                    if (local) {
                        addOrPromptWeight({ id: local.id, nombre: apiProduct.nombre, precio: apiProduct.precio_venta, codigoBarras: trimmed, unidadMedida: local.unidadMedida });
                        return true;
                    }
                } catch {
                    // No encontrado por barcode exacto - continuar
                }
            }

            // 2. Catalogo local (IndexedDB sincronizado desde backend) - por barcode
            // findCatalogProductByBarcode ya filtra stock > 0
            const product = await findCatalogProductByBarcode(trimmed);
            if (product) {
                addOrPromptWeight({ id: product.id, nombre: product.nombre, precio: product.precio, codigoBarras: product.codigoBarras, unidadMedida: product.unidadMedida });
                return true;
            }

            // 3. Catalogo local - busqueda por nombre parcial
            // searchCatalogProducts ya filtra stock > 0
            const results = await searchCatalogProducts(trimmed, 1);
            const match = results[0];
            if (match) {
                addOrPromptWeight({ id: match.id, nombre: match.nombre, precio: match.precio, codigoBarras: match.codigoBarras, unidadMedida: match.unidadMedida });
                return true;
            }

            // Producto no encontrado
            notifications.show({
                title: 'Producto no encontrado',
                message: `No se encontró ningún producto con stock para: "${trimmed}"`,
                color: 'red',
                icon: <AlertCircle size={16} />,
                autoClose: 3000,
            });
            return false;
        },
        [addOrPromptWeight]
    );

    // ── Scanner input handler ─────────────────────────────────────────
    const handleScannerKeyDown = useCallback(
        (e: React.KeyboardEvent<HTMLInputElement>) => {
            // Dejar que el listener global en window maneje los hotkeys F, Escape y +/-
            if (e.key.startsWith('F') || e.key === 'Escape') {
                e.preventDefault();
                return;
            }

            // Flechas: navegar tabla sin mover el cursor del input
            if (e.key === 'ArrowUp' || e.key === 'ArrowDown') {
                e.preventDefault();
                return; // el window listener lo maneja
            }

            // +/- quantity hotkeys: only if nothing typed yet in the scanner field
            if ((e.key === '+' || e.key === '-' || e.key === 'Add' || e.key === 'Subtract') &&
                (scannerValue === '' || e.key === 'Add' || e.key === 'Subtract')) {
                return; // el window listener lo maneja
            }

            if (e.key !== 'Enter') return;
            handleAddProduct(scannerValue).then((added) => {
                if (added) {
                    setScannerValue('');
                    setScannerFeedback('success');
                } else {
                    setScannerFeedback('error');
                }
                window.setTimeout(() => setScannerFeedback('idle'), 180);
            });
        },
        [handleAddProduct, scannerValue]
    );

    const openSearch = useCallback((initialQuery = '') => {
        setSearchInitialQuery(initialQuery);
        setSearchVisible(true);
    }, []);

    const closeSearch = useCallback(() => {
        setSearchVisible(false);
        setSearchInitialQuery('');
        // Clear the scanner input via state (controlled component)
        setScannerValue('');
        setTimeout(() => scannerRef.current?.focus(), 50);
    }, []);

    // ── Global hotkeys (window-level — funciona aunque el foco esté en el input) ──
    useEffect(() => {
        const handleKey = (e: KeyboardEvent) => {
            switch (e.key) {
                case 'F2':
                    e.preventDefault();
                    if (!anyModalOpen) openSearch();
                    break;

                case 'F3':
                    e.preventDefault();
                    if (!anyModalOpen && cart.length > 0) {
                        const safeIndex = selectedRowIndex >= 0 && selectedRowIndex < cart.length ? selectedRowIndex : 0;
                        const item = cart[safeIndex];
                        if (item) openItemDiscountModal(item.id);
                    }
                    break;

                case 'F5':
                    e.preventDefault();
                    if (!anyModalOpen) openPriceCheckModal();
                    break;

                case 'F7':
                    e.preventDefault();
                    if (!anyModalOpen) setHistoryOpen(true);
                    break;

                case 'F8':
                    e.preventDefault();
                    if (!anyModalOpen && cart.length > 0) openDiscountModal();
                    break;

                case 'F10':
                    e.preventDefault();
                    if (!anyModalOpen && cart.length > 0) openPaymentModal();
                    break;

                case 'Escape':
                    e.preventDefault();
                    if (isPaymentModalOpen) closePaymentModal();
                    else if (isPriceCheckModalOpen) closePriceCheckModal();
                    else if (isDiscountModalOpen) closeDiscountModal();
                    else if (historyOpen) setHistoryOpen(false);
                    else if (searchVisible) closeSearch();
                    // Clear scanner input via state to prevent "null" text
                    setScannerValue('');
                    // NOTE: No clearCart() on bare Escape to prevent accidental cart deletion
                    break;

                case 'ArrowUp':
                    if (anyModalOpen) break;
                    e.preventDefault();
                    moveSelectionUp();
                    break;

                case 'ArrowDown':
                    if (anyModalOpen) break;
                    e.preventDefault();
                    moveSelectionDown();
                    break;

                case 'Delete':
                    if (anyModalOpen) break;
                    e.preventDefault();
                    if (cart.length > 0) removeSelectedItem();
                    break;

                case '+':
                case 'Add':
                    if (anyModalOpen) break;
                    if (selectedRowIndex >= 0 && selectedRowIndex < cart.length) {
                        e.preventDefault();
                        const itemP = cart[selectedRowIndex];
                        updateQuantity(itemP.id, itemP.cantidad + 1);
                    }
                    break;

                case '-':
                case 'Subtract':
                    if (anyModalOpen) break;
                    if (selectedRowIndex >= 0 && selectedRowIndex < cart.length) {
                        e.preventDefault();
                        const itemM = cart[selectedRowIndex];
                        if (itemM.cantidad > 1) updateQuantity(itemM.id, itemM.cantidad - 1);
                    }
                    break;
            }
        };

        window.addEventListener('keydown', handleKey);
        return () => window.removeEventListener('keydown', handleKey);
    }, [
        anyModalOpen,
        cart,
        isPaymentModalOpen, isPriceCheckModalOpen, isDiscountModalOpen, historyOpen,
        openSearch, closeSearch,
        openPaymentModal, closePaymentModal,
        openPriceCheckModal, closePriceCheckModal,
        openDiscountModal, closeDiscountModal,
        openItemDiscountModal,
        clearCart, removeSelectedItem,
        moveSelectionUp, moveSelectionDown,
        selectedRowIndex, updateQuantity,
    ]);

    // ── EARLY RETURN: Mientras se verifica la sesión de caja, NO renderizar el POS.
    // Esto evita el deadlock donde el modal de apertura se renderiza antes
    // de que restaurar() termine de consultar el backend.
    if (isInitializing) {
        return (
            <div style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                height: '100vh',
                width: '100vw',
                background: 'var(--mantine-color-body)',
            }}>
                <Loader size="xl" color="blue" type="dots" />
            </div>
        );
    }

    return (
        <div className={styles.posLayout} data-pos-theme={activeTheme.id}>
            <div className={styles.workspaceFrame}>
                <PosHeader />

                <main className={styles.mainContent}>
                    <section className={styles.salesSection}>
                        <div className={styles.scannerDock}>
                            <div className={styles.sectionLabel}>Ingreso de Productos</div>
                            <TextInput
                                ref={scannerRef}
                                value={scannerValue}
                                placeholder="Escanee código o escriba producto para búsqueda inmediata"
                                leftSection={<ScanLine size={18} />}
                                size="md"
                                className={`${styles.scannerInput} ${scannerFeedback === 'success' ? styles.scannerInputSuccess : ''} ${scannerFeedback === 'error' ? styles.scannerInputError : ''}`}
                                classNames={{ input: `${styles.scannerInputField} ${scannerFeedback === 'success' ? styles.scannerInputFieldSuccess : ''} ${scannerFeedback === 'error' ? styles.scannerInputFieldError : ''}` }}
                                onKeyDown={handleScannerKeyDown}
                                onChange={(e) => {
                                    const val = e.currentTarget.value ?? '';
                                    setScannerValue(val);
                                    // Si el valor contiene letras, abrir búsqueda automáticamente
                                    if (val && /[a-zA-ZáéíóúüñÁÉÍÓÚÜÑ]/.test(val) && !searchVisible && !anyModalOpen) {
                                        openSearch(val);
                                    }
                                }}
                                autoFocus
                            />

                            <div className={styles.quickStatsRow}>
                                <div className={styles.quickPill}>
                                    <span className={styles.quickLabel}>Items</span>
                                    <strong className={styles.quickValue}>{totalUnits}</strong>
                                </div>
                                <div className={styles.quickPill}>
                                    <span className={styles.quickLabel}>Productos</span>
                                    <strong className={styles.quickValue}>{cart.length}</strong>
                                </div>
                                <div className={styles.quickPill}>
                                    <span className={styles.quickLabel}>Ahorro</span>
                                    <strong className={styles.quickValue}>{formatCurrency(ahorroActual)}</strong>
                                </div>
                            </div>

                            <div className={styles.contextHint}>
                                {selectedItem
                                    ? `Seleccionado: ${selectedItem.nombre} • ${selectedItem.cantidad} ud. • +/- para ajustar`
                                    : 'Tip: Enter agrega al carrito, F2 abre búsqueda y F10 cobra rápido'}
                            </div>
                        </div>

                        <div className={styles.tableShell}>
                            <div className={styles.sectionLabel}>Detalle de Venta</div>
                            <div className={styles.tableArea}>
                                <SalesTable />
                            </div>
                        </div>
                    </section>

                    <aside className={styles.totalSectionWrap}>
                        <div className={styles.sectionLabel}>Resumen y Cobro</div>
                        <TotalPanel />
                    </aside>
                </main>

                <HotkeysFooter />
            </div>

            {/* ── Modales ────────────────────────────────────────── */}
            {/* ComprobanteModal removido - se selecciona tipo dentro de PaymentModal */}
            <PaymentModal />
            <PriceCheckModal />
            <DiscountModal />
            <SaleHistoryModal opened={historyOpen} onClose={() => setHistoryOpen(false)} />
            <PostSaleModal />

            {/* ── Modal apertura de caja ─────────────────────────── */}
            <AbrirCajaModal
                opened={cajaModalOpen}
                onSuccess={() => setCajaModalOpen(false)}
            />

            {/* ── Modal peso (báscula) ─────────────────────────────── */}
            <WeightInputModal
                opened={weightModalOpen}
                onClose={() => { setWeightModalOpen(false); setWeightProduct(null); }}
                onConfirm={(peso) => {
                    if (weightProduct) {
                        addWeightItem({
                            id: weightProduct.id,
                            nombre: weightProduct.nombre,
                            precio: weightProduct.precio,
                            codigoBarras: weightProduct.codigoBarras,
                            unidadMedida: weightProduct.unidadMedida,
                            peso,
                        });
                    }
                    setWeightModalOpen(false);
                    setWeightProduct(null);
                }}
                productName={weightProduct?.nombre ?? ''}
                precioUnitario={weightProduct?.precio ?? 0}
                unidadMedida={weightProduct?.unidadMedida ?? 'kg'}
            />

            {/* ── Búsqueda flotante (F2) ──────────────────────────── */}
            {searchVisible && (
                <ProductSearch onClose={closeSearch} inputRef={searchRef} initialQuery={searchInitialQuery} onAddProduct={addOrPromptWeight} />
            )}

            {/* ── Touch action bar (replaces F-key shortcuts on touch devices) ── */}
            {isTouch && (
                <div className={styles.touchActionBar}>
                    <Tooltip label="Buscar producto">
                        <ActionIcon
                            size="xl"
                            variant="light"
                            color="blue"
                            className={styles.touchActionBtn}
                            onClick={() => openSearch()}
                            aria-label="Buscar"
                        >
                            <Search size={22} />
                        </ActionIcon>
                    </Tooltip>
                    <Tooltip label="Descuento">
                        <ActionIcon
                            size="xl"
                            variant="light"
                            color="orange"
                            className={styles.touchActionBtn}
                            onClick={openDiscountModal}
                            disabled={cart.length === 0}
                            aria-label="Descuento"
                        >
                            <Percent size={22} />
                        </ActionIcon>
                    </Tooltip>
                    <Tooltip label="Cobrar">
                        <ActionIcon
                            size="xl"
                            variant="filled"
                            color="green"
                            className={styles.touchActionBtn}
                            onClick={openPaymentModal}
                            disabled={cart.length === 0}
                            aria-label="Cobrar"
                        >
                            <CreditCard size={22} />
                        </ActionIcon>
                    </Tooltip>
                    <Tooltip label="Historial">
                        <ActionIcon
                            size="xl"
                            variant="light"
                            color="gray"
                            className={styles.touchActionBtn}
                            onClick={() => setHistoryOpen(true)}
                            aria-label="Historial"
                        >
                            <Clock size={22} />
                        </ActionIcon>
                    </Tooltip>
                </div>
            )}
        </div>
    );
}
