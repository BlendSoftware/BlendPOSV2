import { useEffect, useState } from 'react';
import { Button, Group, ScrollArea } from '@mantine/core';
import { searchCatalogProducts } from '../../offline/catalog';
import type { LocalProduct } from '../../offline/db';
import { formatCurrency } from '../../utils/format';

interface QuickProductsProps {
    onSelect: (product: LocalProduct) => void;
}

export function QuickProducts({ onSelect }: QuickProductsProps) {
    const [products, setProducts] = useState<LocalProduct[]>([]);

    useEffect(() => {
        searchCatalogProducts('', 10).then(setProducts).catch(() => {});
    }, []);

    if (products.length === 0) return null;

    return (
        <ScrollArea scrollbarSize={4} type="auto" offsetScrollbars>
            <Group gap={6} wrap="nowrap" py={4}>
                {products.map((p) => (
                    <Button
                        key={p.id}
                        variant="light"
                        size="compact-sm"
                        style={{ flexShrink: 0 }}
                        onClick={() => onSelect(p)}
                    >
                        {p.nombre} · {formatCurrency(p.precio)}
                    </Button>
                ))}
            </Group>
        </ScrollArea>
    );
}
