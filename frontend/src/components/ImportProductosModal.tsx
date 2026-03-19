// ─────────────────────────────────────────────────────────────────────────────
// ImportProductosModal — CSV bulk import wizard for products.
// Steps: Upload CSV → Preview & validate → Confirm import
// ─────────────────────────────────────────────────────────────────────────────

import { useState, useCallback } from 'react';
import {
    Modal, Stepper, Stack, Text, Button, Group, FileInput,
    Table, Badge, Alert, Anchor, Progress, ScrollArea,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { Upload, FileSpreadsheet, CheckCircle, AlertCircle, Download } from 'lucide-react';
import { crearProductoBulk, type CrearProductoRequest, type BulkImportResponse } from '../services/api/products';

// ── Types ────────────────────────────────────────────────────────────────────

interface ParsedRow {
    rowNumber: number;
    nombre: string;
    codigo_barras: string;
    precio_venta: number;
    categoria: string;
    stock_actual: number;
    errors: string[];
}

interface ImportProductosModalProps {
    opened: boolean;
    onClose: () => void;
    onImported: () => void;
}

// ── CSV Parsing ──────────────────────────────────────────────────────────────

const EXPECTED_HEADERS = ['nombre', 'codigo_barras', 'precio_venta', 'categoria', 'stock_actual'];

const SAMPLE_CSV = `nombre,codigo_barras,precio_venta,categoria,stock_actual
Coca Cola 500ml,7790895000591,850.00,Bebidas,24
Pan Lactal Bimbo,7790040913004,1200.50,Almacen,10
Galletitas Oreo,7622210713285,950.00,Golosinas,15`;

function parseCsv(text: string): { rows: ParsedRow[]; headerError: string | null } {
    const lines = text.split(/\r?\n/).filter((l) => l.trim().length > 0);

    if (lines.length < 2) {
        return { rows: [], headerError: 'El archivo debe tener al menos un encabezado y una fila de datos.' };
    }

    // Normalize headers
    const rawHeaders = lines[0].split(',').map((h) => h.trim().toLowerCase().replace(/\s+/g, '_'));

    // Check required columns exist
    const missing = EXPECTED_HEADERS.filter((h) => !rawHeaders.includes(h));
    if (missing.length > 0) {
        return {
            rows: [],
            headerError: `Columnas faltantes: ${missing.join(', ')}. Columnas esperadas: ${EXPECTED_HEADERS.join(', ')}`,
        };
    }

    const headerIndex = Object.fromEntries(rawHeaders.map((h, i) => [h, i]));

    const rows: ParsedRow[] = [];
    for (let i = 1; i < lines.length; i++) {
        const cols = lines[i].split(',').map((c) => c.trim());
        const errors: string[] = [];

        const nombre = cols[headerIndex['nombre']] ?? '';
        const codigo_barras = cols[headerIndex['codigo_barras']] ?? '';
        const precio_venta_raw = cols[headerIndex['precio_venta']] ?? '';
        const categoria = cols[headerIndex['categoria']] ?? '';
        const stock_actual_raw = cols[headerIndex['stock_actual']] ?? '';

        if (!nombre) errors.push('nombre es requerido');
        if (!codigo_barras) errors.push('codigo_barras es requerido');

        const precio_venta = parseFloat(precio_venta_raw);
        if (isNaN(precio_venta) || precio_venta <= 0) {
            errors.push('precio_venta debe ser un numero mayor a 0');
        }

        const stock_actual = parseInt(stock_actual_raw, 10);
        if (isNaN(stock_actual) || stock_actual < 0) {
            errors.push('stock_actual debe ser un entero >= 0');
        }

        if (!categoria) errors.push('categoria es requerida');

        rows.push({
            rowNumber: i + 1,
            nombre,
            codigo_barras,
            precio_venta: isNaN(precio_venta) ? 0 : precio_venta,
            categoria,
            stock_actual: isNaN(stock_actual) ? 0 : stock_actual,
            errors,
        });
    }

    return { rows, headerError: null };
}

// ── Component ────────────────────────────────────────────────────────────────

export function ImportProductosModal({ opened, onClose, onImported }: ImportProductosModalProps) {
    const [active, setActive] = useState(0);
    const [file, setFile] = useState<File | null>(null);
    const [parsedRows, setParsedRows] = useState<ParsedRow[]>([]);
    const [parseError, setParseError] = useState<string | null>(null);
    const [importing, setImporting] = useState(false);
    const [importProgress, setImportProgress] = useState(0);
    const [importResult, setImportResult] = useState<BulkImportResponse | null>(null);

    const validRows = parsedRows.filter((r) => r.errors.length === 0);
    const invalidRows = parsedRows.filter((r) => r.errors.length > 0);

    const reset = useCallback(() => {
        setActive(0);
        setFile(null);
        setParsedRows([]);
        setParseError(null);
        setImporting(false);
        setImportProgress(0);
        setImportResult(null);
    }, []);

    const handleClose = () => {
        reset();
        onClose();
    };

    const handleFileChange = (f: File | null) => {
        setFile(f);
        setParsedRows([]);
        setParseError(null);
    };

    const handleParse = async () => {
        if (!file) return;

        const text = await file.text();
        const { rows, headerError } = parseCsv(text);

        if (headerError) {
            setParseError(headerError);
            return;
        }

        setParsedRows(rows);
        setParseError(null);
        setActive(1);
    };

    const handleImport = async () => {
        if (validRows.length === 0) return;

        setImporting(true);
        setImportProgress(0);

        const products: CrearProductoRequest[] = validRows.map((r) => ({
            codigo_barras: r.codigo_barras,
            nombre: r.nombre,
            precio_costo: 0,
            precio_venta: r.precio_venta,
            categoria: r.categoria,
            stock_actual: r.stock_actual,
            stock_minimo: 0,
        }));

        try {
            const result = await crearProductoBulk(products);
            setImportResult(result);
            setImportProgress(100);
            setActive(2);

            const created = result.results.filter((r) => r.success).length;
            const failed = result.results.filter((r) => !r.success).length;

            if (failed === 0) {
                notifications.show({
                    title: 'Importacion completada',
                    message: `${created} producto${created !== 1 ? 's' : ''} creado${created !== 1 ? 's' : ''} correctamente.`,
                    color: 'teal',
                    icon: <CheckCircle size={14} />,
                });
            } else {
                notifications.show({
                    title: 'Importacion parcial',
                    message: `${created} creados, ${failed} fallaron. Revisa los detalles.`,
                    color: 'orange',
                    icon: <AlertCircle size={14} />,
                });
            }

            onImported();
        } catch (e: unknown) {
            const msg = e instanceof Error ? e.message : 'Error durante la importacion';
            notifications.show({ title: 'Error', message: msg, color: 'red' });
        } finally {
            setImporting(false);
        }
    };

    const handleDownloadTemplate = () => {
        const blob = new Blob([SAMPLE_CSV], { type: 'text/csv;charset=utf-8;' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'plantilla_productos.csv';
        a.click();
        URL.revokeObjectURL(url);
    };

    return (
        <Modal
            opened={opened}
            onClose={handleClose}
            title={<Text fw={700}>Importar productos desde CSV</Text>}
            size="lg"
            centered
        >
            <Stack gap="md">
                <Stepper active={active} size="sm">
                    <Stepper.Step label="Subir CSV" icon={<Upload size={16} />} />
                    <Stepper.Step label="Revisar" icon={<FileSpreadsheet size={16} />} />
                    <Stepper.Step label="Resultado" icon={<CheckCircle size={16} />} />
                </Stepper>

                {/* ── Step 0: Upload ──────────────────────────────────────── */}
                {active === 0 && (
                    <Stack gap="md">
                        <Text size="sm" c="dimmed">
                            Subi un archivo CSV con las columnas: <strong>nombre</strong>, <strong>codigo_barras</strong>,{' '}
                            <strong>precio_venta</strong>, <strong>categoria</strong>, <strong>stock_actual</strong>.
                        </Text>

                        <Anchor component="button" size="sm" onClick={handleDownloadTemplate}>
                            <Group gap={4}>
                                <Download size={14} />
                                Descargar plantilla CSV
                            </Group>
                        </Anchor>

                        <FileInput
                            label="Archivo CSV"
                            placeholder="Selecciona un archivo .csv"
                            accept=".csv,text/csv"
                            value={file}
                            onChange={handleFileChange}
                            leftSection={<Upload size={14} />}
                        />

                        {parseError && (
                            <Alert icon={<AlertCircle size={16} />} color="red" variant="light">
                                {parseError}
                            </Alert>
                        )}

                        <Group justify="flex-end">
                            <Button variant="subtle" onClick={handleClose}>Cancelar</Button>
                            <Button onClick={handleParse} disabled={!file}>
                                Analizar archivo
                            </Button>
                        </Group>
                    </Stack>
                )}

                {/* ── Step 1: Preview ─────────────────────────────────────── */}
                {active === 1 && (
                    <Stack gap="md">
                        <Group gap="md">
                            <Badge color="teal" variant="light" size="lg">
                                {validRows.length} valido{validRows.length !== 1 ? 's' : ''}
                            </Badge>
                            {invalidRows.length > 0 && (
                                <Badge color="red" variant="light" size="lg">
                                    {invalidRows.length} con errores
                                </Badge>
                            )}
                        </Group>

                        <ScrollArea h={300}>
                            <Table striped highlightOnHover verticalSpacing="xs" withTableBorder>
                                <Table.Thead>
                                    <Table.Tr>
                                        <Table.Th style={{ width: 50 }}>Fila</Table.Th>
                                        <Table.Th>Nombre</Table.Th>
                                        <Table.Th>Codigo</Table.Th>
                                        <Table.Th>Precio</Table.Th>
                                        <Table.Th>Categoria</Table.Th>
                                        <Table.Th>Stock</Table.Th>
                                        <Table.Th>Estado</Table.Th>
                                    </Table.Tr>
                                </Table.Thead>
                                <Table.Tbody>
                                    {parsedRows.slice(0, 50).map((row) => (
                                        <Table.Tr
                                            key={row.rowNumber}
                                            style={row.errors.length > 0 ? { background: 'var(--mantine-color-red-light)' } : undefined}
                                        >
                                            <Table.Td>{row.rowNumber}</Table.Td>
                                            <Table.Td>{row.nombre || <Text c="red" size="xs">vacio</Text>}</Table.Td>
                                            <Table.Td><Text size="xs" ff="monospace">{row.codigo_barras}</Text></Table.Td>
                                            <Table.Td>${row.precio_venta.toFixed(2)}</Table.Td>
                                            <Table.Td>{row.categoria}</Table.Td>
                                            <Table.Td>{row.stock_actual}</Table.Td>
                                            <Table.Td>
                                                {row.errors.length === 0 ? (
                                                    <Badge color="teal" size="xs" variant="light">OK</Badge>
                                                ) : (
                                                    <Badge color="red" size="xs" variant="light">
                                                        {row.errors.join('; ')}
                                                    </Badge>
                                                )}
                                            </Table.Td>
                                        </Table.Tr>
                                    ))}
                                </Table.Tbody>
                            </Table>
                        </ScrollArea>

                        {parsedRows.length > 50 && (
                            <Text size="xs" c="dimmed" ta="center">
                                Mostrando primeras 50 filas de {parsedRows.length} total.
                            </Text>
                        )}

                        {invalidRows.length > 0 && (
                            <Alert icon={<AlertCircle size={16} />} color="orange" variant="light">
                                {invalidRows.length} fila{invalidRows.length !== 1 ? 's' : ''} con errores seran ignorada{invalidRows.length !== 1 ? 's' : ''}.
                                Solo se importaran las {validRows.length} fila{validRows.length !== 1 ? 's' : ''} valida{validRows.length !== 1 ? 's' : ''}.
                            </Alert>
                        )}

                        <Group justify="space-between">
                            <Button variant="subtle" onClick={() => setActive(0)}>Volver</Button>
                            <Button
                                onClick={handleImport}
                                disabled={validRows.length === 0}
                                loading={importing}
                            >
                                Importar {validRows.length} producto{validRows.length !== 1 ? 's' : ''}
                            </Button>
                        </Group>

                        {importing && <Progress value={importProgress} animated />}
                    </Stack>
                )}

                {/* ── Step 2: Result ──────────────────────────────────────── */}
                {active === 2 && importResult && (
                    <Stack gap="md">
                        <Alert
                            icon={<CheckCircle size={16} />}
                            color="teal"
                            variant="light"
                        >
                            Importacion finalizada: {importResult.results.filter((r) => r.success).length} creados,{' '}
                            {importResult.results.filter((r) => !r.success).length} fallaron.
                        </Alert>

                        {importResult.results.some((r) => !r.success) && (
                            <ScrollArea h={200}>
                                <Table striped verticalSpacing="xs" withTableBorder>
                                    <Table.Thead>
                                        <Table.Tr>
                                            <Table.Th>Indice</Table.Th>
                                            <Table.Th>Error</Table.Th>
                                        </Table.Tr>
                                    </Table.Thead>
                                    <Table.Tbody>
                                        {importResult.results
                                            .map((r, i) => ({ ...r, index: i }))
                                            .filter((r) => !r.success)
                                            .map((r) => (
                                                <Table.Tr key={r.index}>
                                                    <Table.Td>{r.index + 1}</Table.Td>
                                                    <Table.Td><Text size="xs" c="red">{r.error}</Text></Table.Td>
                                                </Table.Tr>
                                            ))}
                                    </Table.Tbody>
                                </Table>
                            </ScrollArea>
                        )}

                        <Group justify="flex-end">
                            <Button onClick={handleClose}>Cerrar</Button>
                        </Group>
                    </Stack>
                )}
            </Stack>
        </Modal>
    );
}
