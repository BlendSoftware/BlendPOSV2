# BlendPOS V2 — Prompt de QA End-to-End

Sos un QA Engineer senior. Tu trabajo es testear la aplicación BlendPOS V2 de punta a punta, desde el registro hasta la venta. La app corre en `http://localhost:5173`. El backend corre en `http://localhost:8080`.

**Reglas generales:**
- Tomá screenshot después de CADA paso importante
- Si algo falla, documentá: qué hiciste, qué esperabas, qué pasó, y screenshot del error
- Probá tanto el happy path como los edge cases y validaciones
- Probá en dark mode Y light mode (hay un toggle de tema en cada página)
- Si encontrás un bug, seguí testeando — no pares

---

## FASE 1: REGISTRO DE CUENTA

### 1.1 — Navegación inicial
1. Abrí `http://localhost:5173/login`
2. Verificá que carga la página de login correctamente
3. Verificá que se ve el branding "BlendPOS" con el logo
4. Verificá las 3 feature cards del hero (offline 48h, multi-sucursal, facturación AFIP)
5. Buscá el link "Creá tu negocio gratis" y hacé click

### 1.2 — Página de registro: UI
1. Verificá que se ve el indicador de pasos (Paso 1 de 3)
2. Verificá el botón "Volver al login" arriba a la izquierda
3. Verificá que se muestran las 4 tarjetas de tipo de negocio:
   - **Kiosco** (icono tienda, azul) — "Golosinas, bebidas, cigarrillos y más"
   - **Carnicería** (icono carne, rojo) — "Cortes, embutidos y fiambres"
   - **Minimarket** (icono carrito, teal) — "Almacén con variedad de productos"
   - **Verdulería** (icono manzana, verde) — "Frutas, verduras y productos frescos"
4. Verificá que cada tarjeta muestra cantidad de categorías y productos
5. Verificá que "Kiosco" viene preseleccionado (borde azul, check icon)
6. Clickeá en cada tipo de negocio y verificá que cambia la selección visual (borde coloreado, check)
7. Si hay categorías cargadas, verificá que se muestran como badges debajo de las tarjetas

### 1.3 — Registro: Validaciones (edge cases)
1. Sin llenar nada, intentá enviar el formulario → deben aparecer errores de validación
2. Probá estos campos con datos inválidos:
   - **Nombre del negocio**: 1 solo caracter → debe mostrar "Mínimo 2 caracteres"
   - **Slug**: con espacios o mayúsculas (ej: "Mi Kiosco") → debe rechazar, solo acepta `[a-z0-9]`, 2-63 chars
   - **Slug**: un solo caracter → debe rechazar
   - **Tu nombre**: 1 caracter → "Mínimo 2 caracteres"
   - **Usuario**: 2 caracteres → "Mínimo 3 caracteres"
   - **Email**: texto sin formato de email (ej: "hola") → "Email inválido"
   - **Email**: dejalo vacío → debe pasar (es opcional)
   - **Contraseña**: menos de 8 caracteres → "Mínimo 8 caracteres"
   - **Confirmar contraseña**: distinta a la contraseña → "Las contraseñas no coinciden"
3. Verificá que el campo slug auto-formatea: si escribís "Mi Kiosco 123", debe quedar "mikiosco123"

### 1.4 — Registro: Happy path
1. Seleccioná tipo de negocio: **Kiosco**
2. Completá los datos:
   - Nombre del negocio: `QA Test Kiosco`
   - Slug: `qatestkiosco` (o uno que no exista)
   - Tu nombre: `Tester QA`
   - Usuario: `qaadmin`
   - Email: (dejalo vacío o poné `qa@test.com`)
   - Contraseña: `Test1234!`
   - Confirmar contraseña: `Test1234!`
3. Click en "Crear cuenta gratis"
4. Verificá que muestra loading en el botón
5. Verificá que redirige a `/onboarding` al completar
6. Si da error (ej: slug duplicado), documentá el error y probá con otro slug

---

## FASE 2: ONBOARDING

### 2.1 — Paso 1: Bienvenida
1. Verificá que estás en la pantalla de bienvenida con 4 features:
   - 2 terminales incluidas
   - Facturación AFIP habilitada
   - Catálogo ilimitado
   - Modo offline automático
2. Avanzá al siguiente paso

### 2.2 — Paso 2: Facturación AFIP
1. Verificá los campos:
   - CUIT (11 dígitos, acepta guiones)
   - Razón social (mín 2 chars)
   - Condición fiscal (dropdown: Monotributo, Responsable Inscripto, Exento)
   - Punto de venta (1-99999)
2. Probá validaciones:
   - CUIT con menos de 11 dígitos → error
   - CUIT con letras → error
   - Punto de venta 0 o 100000 → error
3. Probá "Saltear por ahora" → debe avanzar sin guardar
4. O completá datos ficticios y guardá:
   - CUIT: `20-12345678-9`
   - Razón social: `QA Test SRL`
   - Condición: `Monotributo`
   - Punto de venta: `1`

### 2.3 — Paso 3: Productos
1. Verificá que muestra instrucciones sobre el catálogo precargado
2. Verificá mención de importación CSV
3. Avanzá

### 2.4 — Paso 4: Usuarios
1. Verificá info sobre roles (Cajero, Supervisor, Admin)
2. Completá el onboarding
3. Verificá que redirige correctamente (probablemente a `/admin/dashboard` o `/`)

---

## FASE 3: PANEL DE ADMINISTRACIÓN

### 3.1 — Dashboard
1. Navegá a `/admin/dashboard`
2. Verificá que carga sin errores
3. Verificá que se muestran KPIs (aunque estén en 0 porque es cuenta nueva)
4. Verificá el sidebar con todas las secciones de navegación

### 3.2 — Sidebar de navegación
Verificá que existen y son clickeables TODOS estos items:
- [ ] Dashboard
- [ ] Productos
- [ ] Categorías
- [ ] Inventario
- [ ] Transferencias
- [ ] Stock Sucursal
- [ ] Vencimientos
- [ ] Proveedores
- [ ] Compras
- [ ] Facturación
- [ ] Reportes
- [ ] Clientes/Fiado
- [ ] Cierre de Caja
- [ ] Config. Fiscal
- [ ] Usuarios
- [ ] Sucursales
- [ ] Asistente IA

Hacé click en CADA uno y verificá que:
- La página carga sin errores
- Se ve contenido (aunque sea un empty state)
- La URL cambia correctamente
- El item queda marcado como activo en el sidebar

### 3.3 — Gestión de Productos (CRUD completo)
1. Ir a Productos
2. Verificá el listado (puede tener productos del preset o estar vacío)
3. **Crear producto:**
   - Click en botón de crear/agregar producto
   - Completá: nombre, código de barras, precio, categoría, stock
   - Guardá y verificá que aparece en la lista
4. **Editar producto:**
   - Seleccioná un producto existente
   - Modificá el precio
   - Guardá y verificá que el cambio persiste
5. **Eliminar producto:**
   - Eliminá un producto
   - Verificá que desaparece de la lista
6. Creá al menos **5 productos** con datos variados para poder testear el POS después:
   - Producto 1: `Coca Cola 500ml` — código `7790895000591` — precio `$1500`
   - Producto 2: `Alfajor Havanna` — código `7790310800102` — precio `$2800`
   - Producto 3: `Agua Mineral 1.5L` — código `7798159540012` — precio `$900`
   - Producto 4: `Galletitas Oreo` — código `7622210713001` — precio `$1200`
   - Producto 5: `Yerba Mate 1kg` — código `7790150000723` — precio `$4500`

### 3.4 — Categorías
1. Verificá que hay categorías precargadas del preset
2. Intentá crear una categoría nueva
3. Intentá editar una existente
4. Intentá eliminar una (verificá qué pasa si tiene productos asociados)

### 3.5 — Usuarios
1. Ir a Usuarios
2. Crear un usuario cajero:
   - Nombre: `Cajero Test`
   - Username: `cajero1`
   - Password: `Cajero123!`
   - Rol: Cajero
3. Crear un usuario supervisor:
   - Nombre: `Super Test`
   - Username: `super1`
   - Password: `Super123!`
   - Rol: Supervisor
4. Verificá que ambos aparecen en la lista

### 3.6 — Sucursales
1. Ir a Sucursales
2. Verificá que existe al menos la sucursal principal
3. Intentá crear una sucursal nueva si la UI lo permite

### 3.7 — Clientes/Fiado
1. Ir a Clientes
2. Crear un cliente:
   - Nombre: `Cliente Fiado Test`
   - Límite de crédito: `$50000`
3. Verificá que aparece en la lista

### 3.8 — Configuración Fiscal
1. Ir a Config. Fiscal
2. Verificá que se muestran los datos cargados en onboarding (o vacío si salteaste)
3. Si están vacíos, completá:
   - CUIT: `20-12345678-9`
   - Razón social: `QA Test SRL`
   - Condición fiscal: Monotributo
   - Punto de venta: `1`

### 3.9 — Reportes
1. Ir a Reportes
2. Verificá que la página carga
3. Si hay filtros de fecha, probá distintos rangos
4. Verificá que no hay errores de JS en consola

### 3.10 — Otras páginas admin
Visitá cada una de estas y verificá que cargan sin errores:
- Inventario
- Transferencias
- Stock Sucursal
- Vencimientos
- Proveedores
- Compras
- Facturación

---

## FASE 4: TERMINAL POS

### 4.1 — Ingreso al POS
1. Desde el admin, buscá el link para ir al POS (o navegá a `http://localhost:5173/`)
2. Verificá que carga el terminal POS
3. Si aparece el modal "Abrir Caja", completá el proceso para abrir la sesión de caja
4. Verificá el header:
   - Branding "BlendPOS"
   - Info del operador (nombre + rol en badge)
   - Badge de conectividad (verde = "Conectado")
   - Reloj en tiempo real
   - Botones: impresora, cierre caja, panel admin, tema, cerrar sesión

### 4.2 — Búsqueda y carga de productos
1. Verificá que el campo de scanner tiene auto-focus
2. **Por código de barras:**
   - Escribí `7790895000591` (Coca Cola) y presioná Enter
   - Verificá que se agrega a la tabla de venta
   - Verificá que se muestra el último producto escaneado
3. **Por búsqueda (F2):**
   - Presioná F2
   - Verificá que se abre el modal de búsqueda
   - Escribí "Alfajor"
   - Verificá que aparecen resultados
   - Seleccioná con flechas ↑↓ y Enter
   - Verificá que se agrega al carrito
4. **Por nombre en scanner:**
   - En el campo de scanner, escribí letras (ej: "gall")
   - Verificá que auto-abre la búsqueda (F2)
5. Agregá los 5 productos que creaste

### 4.3 — Tabla de ventas (carrito)
1. Verificá que se muestran todos los productos agregados con:
   - Número de línea (#)
   - Código de barras
   - Nombre del producto
   - Precio unitario
   - Cantidad (con botones +/-)
   - Subtotal
   - Botón eliminar (tacho)
2. **Modificar cantidad:**
   - Usá los botones +/- para cambiar cantidad
   - Verificá que el subtotal se recalcula
   - Probá editar la cantidad inline (click en el número)
3. **Navegación por teclado:**
   - Usá ↑↓ para navegar entre filas
   - Verificá que la fila seleccionada se resalta
   - Presioná + y - para ajustar cantidad de la fila seleccionada
   - Presioná Supr (Delete) para eliminar la fila seleccionada
4. **Quick stats:**
   - Verificá que las píldoras superiores muestran: Items, Productos, Ahorro
   - Verificá que se actualizan al agregar/quitar productos

### 4.4 — Panel de totales
1. Verificá que se muestra:
   - Cantidad de artículos
   - Descuento (0% si no hay)
   - Total en verde (grande)
2. Verificá que los botones están habilitados:
   - COBRAR (F10) — verde
   - DESCUENTO (F8) — azul
   - CANCELAR — rojo outline
3. Con carrito vacío, verificá que los botones están deshabilitados

### 4.5 — Descuento global (F8)
1. Presioná F8 o click en "DESCUENTO"
2. Verificá que se abre el modal de descuento
3. Verificá los elementos:
   - Total original
   - Input de porcentaje
   - Slider (0-50%)
   - Badges rápidos: 5%, 10%, 15%, 20%, 25%, 30%
   - Total final calculado
4. **Probá:**
   - Click en badge 10% → verificá que el input y slider se actualizan
   - Mové el slider → verificá que el input se actualiza
   - Escribí 25 en el input → verificá cálculo correcto
   - Escribí 101 → debe rechazar (máx 100%)
   - Click "Aplicar"
   - Verificá que el total en el panel se muestra con descuento (tachado original + nuevo total)
   - Click "Quitar descuento" para resetear

### 4.6 — Descuento por ítem (F3)
1. Seleccioná un producto en la tabla (↑↓)
2. Presioná F3
3. Verificá que el modal muestra el nombre del producto específico
4. Aplicá un descuento del 15%
5. Verificá que solo ese producto tiene badge de descuento en la tabla
6. Verificá que el total se recalculó correctamente

### 4.7 — Consulta de precio (F5)
1. Presioná F5
2. Verificá que se abre el modal de consulta de precio
3. Buscá un producto y verificá que muestra el precio sin agregarlo al carrito

### 4.8 — COBRAR: Pago en efectivo
1. Asegurate de tener productos en el carrito (total conocido)
2. Presioná F10 o click en "COBRAR"
3. Verificá el modal de pago:
   - Resumen (artículos, subtotal, descuento, TOTAL)
   - Tipo de comprobante (dropdown: Auto, Ticket interno, Factura C, etc.)
   - Método de pago (dropdown)
4. Seleccioná método: **Efectivo**
5. Verificá que aparece el campo "Monto recibido" con botones rápidos ($1000, $2000, $5000, $10000, $20000, Exacto)
6. **Probá:**
   - Click "Exacto" → monto = total, vuelto = $0
   - Escribí un monto menor al total → debe mostrar error "Monto insuficiente"
   - Escribí un monto mayor → verificá que el vuelto se calcula bien (ej: total $10900, pagás $11000, vuelto $100)
   - Click en $5000 → verificá que se setea el monto
7. Con monto válido, click "Confirmar Pago"
8. Verificá que la venta se procesa, el carrito se vacía, y se muestra confirmación

### 4.9 — COBRAR: Pago con tarjeta
1. Cargá productos al carrito
2. F10 → método de pago: **Tarjeta de Débito**
3. Verificá que NO pide monto recibido (no hay vuelto con tarjeta)
4. Confirmá y verificá que la venta se procesa

### 4.10 — COBRAR: Pago QR / Transferencia
1. Cargá productos
2. F10 → método: **QR**
3. Confirmá y verificá
4. Repetí con **Transferencia**

### 4.11 — COBRAR: Pago Fiado (cuenta corriente)
1. Cargá productos
2. F10 → método: **Fiado (Cuenta Corriente)**
3. Verificá que aparece un buscador de clientes
4. Buscá y seleccioná "Cliente Fiado Test" (el que creaste antes)
5. Verificá que muestra: límite de crédito, saldo disponible
6. Si el total no supera el límite, confirmá el pago
7. **Edge case:** intentá una venta que supere el límite de crédito → debe dar error

### 4.12 — COBRAR: Pago Mixto
1. Cargá productos (que sumen un total alto, ej: $10000+)
2. F10 → método: **Mixto**
3. Verificá que aparecen 4 inputs: Débito, Crédito, QR, Transferencia
4. Poné $3000 en Débito y $2000 en QR
5. Verificá que calcula cuánto queda en efectivo
6. Completá el efectivo y verificá el vuelto
7. Confirmá la venta

### 4.13 — COBRAR: Comprobante fiscal
1. Cargá productos
2. F10
3. Cambiá tipo de comprobante a **Factura C**
4. Verificá que aparecen campos adicionales:
   - Tipo documento (DNI/CUIT)
   - Número de documento
   - Nombre/Razón social (mín 3 chars)
   - Domicilio (mín 5 chars)
5. **Validaciones:**
   - DNI: 7-8 dígitos
   - CUIT: 11 dígitos exactos
   - Nombre < 3 chars → error
   - Domicilio < 5 chars → error
6. Completá datos válidos y confirmá
7. Si sos Monotributo, verificá que solo podés elegir Ticket o Factura C (no A ni B)

### 4.14 — Email de recibo (opcional)
1. En el modal de pago, buscá el campo de email (opcional)
2. Poné un email inválido → debe mostrar error
3. Poné un email válido o dejalo vacío
4. Confirmá la venta

### 4.15 — Historial de ventas (F7)
1. Presioná F7
2. Verificá que se abre el modal de historial
3. Verificá stats: "Ventas hoy" (count) y "Total cobrado"
4. Verificá que las ventas que hiciste aparecen como tarjetas expandibles
5. Expandí una venta y verificá:
   - Número de ticket
   - Hora
   - Método de pago
   - Lista de productos con cantidades y precios
   - Vuelto si fue en efectivo

### 4.16 — Cancelar carrito
1. Cargá productos al carrito
2. Click en "CANCELAR" o presioná Esc
3. Si el carrito tiene items, verificá que pide confirmación (no borra directo)
4. Confirmá la cancelación
5. Verificá que el carrito queda vacío

### 4.17 — Atajos de teclado (todos)
Probá CADA atajo y verificá que funciona:
- [ ] **F2** → Abre búsqueda de productos
- [ ] **F3** → Descuento por ítem (con producto seleccionado)
- [ ] **F5** → Consulta de precio
- [ ] **F7** → Historial de ventas
- [ ] **F8** → Descuento global
- [ ] **F10** → Cobrar
- [ ] **↑↓** → Navegar tabla
- [ ] **+/-** → Ajustar cantidad
- [ ] **Supr** → Eliminar item seleccionado
- [ ] **Esc** → Cerrar modal / cancelar carrito
- [ ] **Enter** → Confirmar en modales

### 4.18 — Toggle dark/light mode
1. Encontrá el botón de tema en el header
2. Cambiá entre dark y light mode
3. Verificá que TODOS los elementos del POS se ven correctamente en ambos modos:
   - Header
   - Tabla de ventas
   - Panel de totales
   - Modales
   - Inputs y botones

---

## FASE 5: ROLES Y PERMISOS

### 5.1 — Test con usuario Cajero
1. Cerrá sesión (botón logout en header del POS)
2. Logueate con `cajero1` / `Cajero123!`
3. Verificá que redirige al POS (no al admin)
4. Verificá que en el header NO aparece el botón de "Panel admin"
5. Hacé una venta normalmente
6. Intentá aplicar descuento mayor a 30% → debe bloquearte (cajero limitado a 30%)
7. Intentá acceder a `/admin/dashboard` manualmente → debe redirigir o bloquear

### 5.2 — Test con usuario Supervisor
1. Cerrá sesión
2. Logueate con `super1` / `Super123!`
3. Verificá que redirige a `/admin/dashboard`
4. Verificá que puede acceder a reportes y POS
5. Verificá que NO puede acceder a Sucursales (solo admin)
6. Verificá que puede aplicar descuentos sin límite de 30%

### 5.3 — Forzar cambio de contraseña
1. Si algún usuario es nuevo y tiene flag de cambio obligatorio, verificá que aparece el modal ForcePasswordChangeModal
2. Verificá validaciones: mín 8 chars, contraseñas deben coincidir

---

## FASE 6: CONECTIVIDAD Y EDGE CASES

### 6.1 — Indicador de conectividad
1. Verificá el badge en el POS header:
   - **Verde** "Conectado" cuando hay conexión
   - Si podés simular offline (desconectar red): verificá **Rojo** "Sin conexión"

### 6.2 — Login sin conexión
1. Si podés simular offline, intentá loguearte → verificá mensaje de error descriptivo

### 6.3 — Errores del servidor
1. Si el backend está caído, verificá que los mensajes de error son informativos
2. Verificá que no hay pantallas blancas de crash

### 6.4 — Producto no encontrado
1. En el POS, escaneá un código de barras que no existe (ej: `0000000000000`)
2. Verificá que muestra un mensaje de error claro, no un crash

### 6.5 — Doble click en botones
1. En el modal de pago, intentá hacer doble click rápido en "Confirmar Pago"
2. Verificá que no se procesa la venta dos veces (el botón debe deshabilitarse mientras procesa)

---

## FASE 7: RESPONSIVE Y UX

### 7.1 — Scroll en todas las páginas
1. En CADA página que tenga mucho contenido, verificá que se puede scrollear correctamente
2. Especialmente: Register, Onboarding, modales con mucho contenido

### 7.2 — Modales
1. En CADA modal, verificá:
   - Se puede cerrar con Esc
   - Se puede cerrar con el botón X o "Cancelar"
   - No se puede interactuar con el fondo mientras está abierto
   - El contenido es scrolleable si no entra

### 7.3 — Consola del navegador
1. Abrí DevTools → Console
2. Navegá por TODA la app
3. Reportá cualquier error de JS, warning, o request fallido
4. Ignorá warnings de React dev mode que no sean de la app

---

## FORMATO DE REPORTE

Para cada fase, reportá:

```
### FASE X: [nombre]

**Estado:** ✅ PASS | ⚠️ PARCIAL | ❌ FAIL

#### Tests pasados:
- [lista de lo que funcionó correctamente]

#### Bugs encontrados:
1. **[BUG-001] Título descriptivo**
   - Pasos para reproducir: ...
   - Resultado esperado: ...
   - Resultado actual: ...
   - Severidad: Crítico / Alto / Medio / Bajo
   - Screenshot: [adjuntar]

#### Observaciones:
- [Cosas que no son bugs pero podrían mejorar]
```

---

## DATOS DE TEST

| Campo | Valor |
|-------|-------|
| URL | `http://localhost:5173` |
| Negocio | `QA Test Kiosco` |
| Slug | `qatestkiosco` |
| Admin user | `qaadmin` |
| Admin pass | `Test1234!` |
| Cajero user | `cajero1` |
| Cajero pass | `Cajero123!` |
| Supervisor user | `super1` |
| Supervisor pass | `Super123!` |
| Cliente fiado | `Cliente Fiado Test` / Límite: $50000 |
| CUIT test | `20-12345678-9` |

## PRODUCTOS DE TEST

| Producto | Código | Precio |
|----------|--------|--------|
| Coca Cola 500ml | 7790895000591 | $1500 |
| Alfajor Havanna | 7790310800102 | $2800 |
| Agua Mineral 1.5L | 7798159540012 | $900 |
| Galletitas Oreo | 7622210713001 | $1200 |
| Yerba Mate 1kg | 7790150000723 | $4500 |
