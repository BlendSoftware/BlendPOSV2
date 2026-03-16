# Propuesta: BlendPOS SaaS Multi-Tenant

## Resumen ejecutivo

BlendPOS evoluciona de sistema POS single-tenant a plataforma SaaS multi-tenant por suscripción, posicionada como el "Anti-Alegra" para comercios de alta rotación (kioscos, despensas, almacenes). La propuesta capitaliza tres fallas sistémicas de la competencia: lentitud en flujos de venta intensivos, dependencia de la nube para operación básica, y restricciones de acceso a datos históricos por tier de pago.

La ventaja competitiva de BlendPOS es estructural: offline-first real con 48h de autonomía garantizada, flujo de cobro de 2 interacciones (escaneo → cobro), y analytics histórico completo desde el plan más bajo.

---

## Problema

### Fallas de la competencia (Alegra POS y similares)

| Eje | Alegra POS | Impacto en el negocio |
|-----|-----------|----------------------|
| **Velocidad** | 3–8 seg por transacción en hora pico | Filas, abandono, pérdida de ventas |
| **Offline** | Degradado funcional sin conexión | Caja paralizada ante cortes de Fibertel/Movistar |
| **Datos** | Reportes históricos bloqueados por plan | El dueño no puede ver sus propias ventas de hace 6 meses |
| **Multi-dispositivo** | Sincronización lenta entre cajas | Inventario inconsistente entre terminales |

### Contexto del mercado argentino

- Cortes de internet frecuentes (promedio 4.2h/semana en AMBA según ENACOM 2024)
- Comercios de alta rotación: 80–400 transacciones/día en kioscos urbanos
- Sensibilidad precio: el dueño del kiosco no paga más por ver sus propios datos
- AFIP/ARCA: obligación de facturación electrónica para monotributistas desde cat. D

---

## Solución

### BlendPOS SaaS: tres pilares no negociables

**1. Velocidad sub-segundo**
- Registro de venta en IndexedDB local: `< 100ms` (p99)
- Sincronización en background: no bloquea el flujo de cobro
- UI de cobro: máximo 2 interacciones — escaneo de código de barras + confirmación de pago
- Precarga del catálogo completo en memoria al abrir la caja

**2. Offline-first inquebrantable**
- Autonomía mínima garantizada: **48 horas sin internet**
- Catálogo completo en IndexedDB (sin límite de productos en local)
- Facturación AFIP encolada: se emite el CAE cuando se restaura la conexión
- Comprobantes en PDF generados localmente si el backend no responde
- Sin degradación funcional: el cajero no nota si hay o no internet

**3. Analytics transparente sin paywalls**
- Todos los reportes disponibles desde el plan más bajo
- Datos históricos sin límite de fecha
- Exportación a CSV/PDF sin costo adicional
- Dashboard en tiempo real de ventas, stock y caja

---

## Non-goals

- **No** se propone rediseño de stack (se mantiene Go/React/Python 100%)
- **No** incluye marketplace o integraciones con e-commerce en esta fase
- **No** cubre gestión de RRHH o nómina
- **No** incluye módulo contable (no es ERP)
- **No** propone migración a microservicios independientes — el monolito Go es intencional y suficiente para la escala objetivo (< 10.000 tenants en 18 meses)
- **No** incluye soporte para múltiples puntos de venta en distintas provincias (IIBB multi-jurisdiccional) en esta fase

---

## Módulos del producto

### Módulo 1: Facturación Ultra-Rápida

**Descripción:** Terminal de cobro optimizada para velocidad máxima. Un cajero experimentado debe poder procesar 600 transacciones en una jornada de 8 horas (1.25 trans/min promedio, con picos de 3/min).

**Historias de usuario:**

- Como **cajero de kiosco**, quiero escanear un código de barras y confirmar el cobro en 2 toques, para no hacer esperar a la fila en hora pico.
- Como **cajero**, quiero que el sistema acepte el pago aunque no haya internet, para no paralizar la caja durante cortes de luz o señal.
- Como **cajero**, quiero ver el precio actualizado instantáneamente al escanear, para evitar cobrar precios desactualizados.
- Como **supervisor**, quiero anular una venta en menos de 30 segundos, para corregir errores sin demorar la atención.

**Criterios de aceptación:**

| Métrica | Valor requerido |
|---------|----------------|
| Registro en IndexedDB local | `< 100ms` (p99) |
| Render del precio al escanear | `< 50ms` desde keypress |
| Confirmación visual de venta OK | `< 200ms` end-to-end (local) |
| Flujo de cobro | máximo 2 interacciones |
| Tiempo de anulación (supervisor) | `< 30 segundos` |
| Precarga catálogo al abrir caja | `< 3 segundos` (hasta 50.000 productos) |

**Reglas de negocio:**

- RN-F1: Una venta registrada localmente es **inmutable** hasta ser sincronizada. No se puede editar en IndexedDB, solo anular.
- RN-F2: El `offline_id` se genera en el frontend como `{tenant_id}:{device_id}:{timestamp_ms}:{random_4}` — garantiza unicidad global sin coordinación con servidor.
- RN-F3: Si el producto no está en catálogo local, se permite ingresar precio manual con flag `precio_manual: true`.
- RN-F4: La anulación de una venta ya sincronizada requiere rol `supervisor` o superior y genera un movimiento de caja negativo.

---

### Módulo 2: Catálogo Resiliente

**Descripción:** El catálogo de productos debe estar disponible offline en su totalidad. Las actualizaciones de precios y stock se sincronizan en background sin interrumpir la operación.

**Historias de usuario:**

- Como **dueño**, quiero actualizar el precio de un producto desde el panel web y que la caja lo refleje en menos de 60 segundos, para no cobrar precios incorrectos.
- Como **cajero**, quiero que la caja funcione con el catálogo del día anterior si no hay internet, para no perder ventas por una falla de conectividad.
- Como **dueño**, quiero recibir una alerta cuando el stock de un producto cae por debajo del mínimo definido, para gestionar el reabastecimiento a tiempo.

**Criterios de aceptación:**

| Métrica | Valor requerido |
|---------|----------------|
| Propagación de precio actualizado (online) | `< 60 segundos` a todas las cajas activas |
| Tamaño máximo de catálogo en IndexedDB | Sin límite técnico (Dexie soporta GBs) |
| Staleness máximo del catálogo en offline | 48 horas (última sincronización exitosa) |
| Alerta de stock mínimo | Notificación en dashboard `< 5 minutos` |

**Reglas de negocio:**

- RN-C1: El catálogo local tiene siempre un `catalog_version` (timestamp de última sync). Si supera 48h, el cajero ve un banner de advertencia pero **puede seguir vendiendo**.
- RN-C2: Cambios de precio en el servidor aplican solo a ventas nuevas. Las ventas en cola offline conservan el precio al momento del registro.
- RN-C3: El stock en el servidor es la fuente de verdad. El stock local es optimista y se reconcilia al sincronizar.

---

### Módulo 3: Sincronización Offline

**Descripción:** El motor de sincronización es la pieza más crítica del sistema. Debe garantizar que ninguna venta se pierda, ninguna se duplique, y el orden causal se preserve.

**Historias de usuario:**

- Como **dueño**, quiero que todas las ventas registradas sin internet se suban automáticamente al restaurarse la conexión, para no perder ningún registro de mi caja.
- Como **sistema**, quiero detectar y resolver colisiones entre ventas offline de múltiples terminales, para mantener el inventario consistente.
- Como **cajero**, quiero ver un indicador visual del estado de sincronización, para saber si mis ventas están respaldadas en la nube.

**Criterios de aceptación:**

| Métrica | Valor requerido |
|---------|----------------|
| Pérdida de ventas en sync | 0 (cero absoluto) |
| Duplicados tras reconexión | 0 (garantizado por `offline_id`) |
| Ventana de retry en background | cada 30 segundos hasta éxito |
| Batch máximo por sync | 500 ventas por request |
| Indicador visual de sync en UI | verde/amarillo/rojo visible en PosHeader |

**Reglas de negocio:**

- RN-S1: Una venta con `offline_id` ya existente en el servidor retorna `200 OK` con los datos existentes (idempotente). El cliente no genera error.
- RN-S2: Colisión de stock: si dos terminales venden el último item simultáneamente en offline, ambas ventas se registran. El stock puede quedar negativo temporalmente. Se genera una alerta de `stock_negativo` para revisión del dueño.
- RN-S3: El orden de sincronización es por `created_at` ascendente dentro de cada `device_id`. No se garantiza orden global entre terminales distintas.
- RN-S4: Las ventas con `estado: pendiente_afip` se sincronizan primero; la facturación AFIP se encola después (worker asíncrono).
- RN-S5: Si el sync falla por error de servidor (5xx), se usa exponential backoff: 30s → 60s → 120s → 300s. Si supera 1h sin éxito, se notifica al dueño por email.

---

### Módulo 4: Analytics Transparente

**Descripción:** Reportería completa, sin paywalls, desde el día 1 de suscripción. El dueño del comercio tiene derecho irrestricto a sus propios datos.

**Historias de usuario:**

- Como **dueño**, quiero ver el resumen de ventas del día, semana y mes en un dashboard, para tomar decisiones de compra y precio.
- Como **dueño**, quiero exportar mis ventas históricas a CSV en cualquier rango de fechas, para presentarlas a mi contador o al banco.
- Como **dueño**, quiero ver qué productos vendí más en los últimos 30 días, para optimizar mi stock.
- Como **dueño**, quiero comparar el rendimiento de mis distintas sucursales (si aplica), para identificar cuál necesita atención.

**Criterios de aceptación:**

| Métrica | Valor requerido |
|---------|----------------|
| Acceso a datos históricos | Sin límite de fecha en todos los planes |
| Tiempo de carga del dashboard | `< 2 segundos` (hasta 1 año de datos) |
| Exportación CSV | Disponible en todos los planes, sin costo extra |
| Retención de datos | Mínimo 5 años (cumplimiento AFIP) |

**Reglas de negocio:**

- RN-A1: Los datos analíticos son propiedad del tenant. BlendPOS no los comercializa, agrega ni comparte con terceros.
- RN-A2: La exportación de datos debe estar disponible incluso si la suscripción vence, durante un período de gracia de 30 días.
- RN-A3: Queries de analytics se ejecutan sobre réplica de lectura (o índices específicos) para no impactar el rendimiento de la caja.

---

## Modelo de suscripción (planes)

| Plan | Precio/mes | Terminales | Usuarios | AFIP | Soporte |
|------|-----------|-----------|---------|------|---------|
| **Starter** | $0 (30 días) | 1 | 2 | Homologación | Email |
| **Kiosco** | $X | 1 | 3 | Producción | Email + Chat |
| **Negocio** | $2X | 3 | 10 | Producción | Prioritario |
| **Pro** | $4X | Ilimitadas | Ilimitados | Producción | Dedicado |

> Los reportes históricos y la exportación CSV están disponibles en **todos los planes sin excepción**.

---

## Matriz de trazabilidad: requerimientos → componentes

| ID Req | Descripción | Componente Backend | Componente Frontend | DB |
|--------|------------|-------------------|--------------------|----|
| RN-F1 | Inmutabilidad venta local | `venta_service.go` | `offline/sync.ts` | IndexedDB + `ventas` |
| RN-F2 | `offline_id` único global | — | `offline/sync.ts` (generación) | `ventas.offline_id` UNIQUE |
| RN-F3 | Precio manual | `handler/ventas.go` | `PosTerminal.tsx` | `venta_items.precio_manual` |
| RN-F4 | Anulación con rol | `middleware/auth.go` + `venta_service.go` | `store/useCartStore.ts` | `ventas.estado` |
| RN-C1 | `catalog_version` | `handler/productos.go` | `offline/catalog.ts` | `productos.updated_at` |
| RN-C2 | Precio al momento de registro | `venta_service.go` | `offline/sync.ts` | `venta_items.precio_unitario` snapshot |
| RN-C3 | Stock optimista | `inventario_service.go` | `offline/catalog.ts` | `productos.stock` |
| RN-S1 | Idempotencia offline_id | `venta_service.go` (upsert) | `offline/sync.ts` | UNIQUE constraint |
| RN-S2 | Stock negativo permitido | `inventario_service.go` | `PosHeader.tsx` (alerta) | `movimientos_stock` |
| RN-S4 | Facturación asíncrona | `worker/facturacion_worker.go` | `PosHeader.tsx` (estado) | `comprobantes.estado` |
| RN-S5 | Backoff en sync | — | `offline/sync.ts` | IndexedDB `sync_queue.retry_at` |
| RN-A1 | Propiedad de datos | `middleware/tenant.go` (RLS) | — | Row-Level Security |
| RN-A3 | Réplica de lectura | `infra/database.go` | `services/api/reportes.ts` | PostgreSQL read replica |
