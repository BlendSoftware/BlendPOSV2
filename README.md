# BlendPOS

Sistema de punto de venta (POS) completo con soporte offline-first, facturación AFIP, y gestión de inventario.

## Requisitos

| Software | Versión | Notas |
|----------|---------|-------|
| Docker Desktop | >= 4.x | **Unico requisito obligatorio** |
| Git | >= 2.x | Para clonar el repo |
| Node.js | >= 20 | Solo si queres correr el frontend fuera de Docker |
| Go | >= 1.24 | Solo si queres correr el backend fuera de Docker |

## Quick Start (3 pasos)

```bash
# 1. Clonar el repositorio
git clone <repository-url> BlendPOSV2
cd BlendPOSV2

# 2. Levantar todos los servicios
docker compose up -d
# Esperar ~30-60 segundos a que el backend compile (Go)

# 3. Crear usuario admin de prueba
docker compose exec backend go run cmd/seeduser/main.go
```

**Listo. Abrir http://localhost:5173 y loguearse con:**

| Campo | Valor |
|-------|-------|
| Usuario | `admin@blendpos.com` |
| Contraseña | `1234` |

### Que pasa cuando haces `docker compose up`?

1. **PostgreSQL** arranca y queda healthy en `:5432`
2. **Redis** arranca y queda healthy en `:6379`
3. **Backend (Go)** compila automaticamente, corre las migraciones y escucha en `:8000`
4. **Frontend (Vite)** instala dependencias (`npm ci`) y sirve en `:5173`
5. **AFIP Sidecar** no arranca (necesita certificados AFIP, ver seccion opcional abajo)

> **NO necesitas correr migraciones manualmente.** El backend las ejecuta automaticamente al arrancar.
>
> **NO necesitas crear archivos `.env`.** Todas las variables de desarrollo estan en `docker-compose.yml`.

## Verificar que funciona

```bash
# Ver estado de todos los containers
docker compose ps

# Deberias ver algo asi:
# blendposv2-backend-1    Up    0.0.0.0:8000->8000/tcp
# blendposv2-frontend-1   Up    0.0.0.0:5173->5173/tcp
# blendposv2-postgres-1   Up    0.0.0.0:5432->5432/tcp
# blendposv2-redis-1      Up    0.0.0.0:6379->6379/tcp

# Si el backend no arranco, ver logs:
docker compose logs backend

# Probar que la API responde:
curl -s http://localhost:8000/v1/auth/login -X POST \
  -H "Content-Type: application/json" \
  -d '{"username":"admin@blendpos.com","password":"1234"}'
# Debe devolver un JSON con access_token
```

## Arquitectura

```
┌────────────┐     ┌──────────┐     ┌──────────────┐
│  Frontend  │────>│ Backend  │────>│  PostgreSQL   │
│ React+Vite │     │  Go/Gin  │     │    :5432      │
│   :5173    │     │  :8000   │──┐  └──────────────┘
└────────────┘     └──────────┘  │  ┌──────────────┐
                        │        └─>│    Redis      │
                        v           │    :6379      │
                   ┌──────────┐     └──────────────┘
                   │  AFIP    │
                   │ Sidecar  │  (opcional en dev)
                   │  :8001   │
                   └──────────┘
```

| Servicio | Puerto | Tecnologia |
|----------|--------|------------|
| Frontend | 5173 | React 19, Vite, Mantine UI, Zustand, Dexie.js (IndexedDB) |
| Backend | 8000 | Go 1.24, Gin, GORM, shopspring/decimal |
| PostgreSQL | 5432 | PostgreSQL 15 Alpine |
| Redis | 6379 | Redis 7 Alpine |
| AFIP Sidecar | 8001 | Python/FastAPI + pyafipws (opcional, necesita certificados) |

## Estructura del Proyecto

```
BlendPOSV2/
├── backend/
│   ├── cmd/
│   │   ├── server/          # Punto de entrada principal
│   │   ├── seeduser/        # Utilidad: crear usuario admin de prueba
│   │   └── genhash/         # Utilidad: generar bcrypt hashes
│   ├── internal/
│   │   ├── handler/         # Controladores HTTP (Gin)
│   │   ├── service/         # Logica de negocio
│   │   ├── repository/      # Acceso a datos (GORM)
│   │   ├── model/           # Modelos de BD
│   │   ├── dto/             # Data Transfer Objects
│   │   ├── config/          # Carga de env vars (Viper)
│   │   ├── worker/          # Workers async (facturacion, emails)
│   │   └── infra/           # Infra (DB, Redis, PDF)
│   ├── migrations/          # Migraciones SQL (auto-aplicadas al arrancar)
│   └── Dockerfile.dev       # Imagen dev con Air hot-reload
├── frontend/
│   ├── src/
│   │   ├── pages/           # Paginas (POS, Dashboard, Admin)
│   │   ├── components/      # Componentes reutilizables
│   │   ├── services/api/    # Clientes API tipados
│   │   ├── store/           # Zustand stores
│   │   ├── offline/         # Offline-first (Dexie, sync queue)
│   │   └── api/             # API client centralizado (axios)
│   └── .env.example
├── afip-sidecar/            # Microservicio facturacion AFIP (opcional)
├── docker-compose.yml       # Dev environment
├── docker-compose.override.yml  # Override: excluye afip-sidecar en dev
├── docker-compose.prod.yml  # Produccion (con Traefik)
└── README.md
```

## Variables de Entorno

### Para desarrollo: NO necesitas archivos .env

Todo esta configurado en `docker-compose.yml`. Solo necesitas `.env` si queres habilitar funcionalidades opcionales.

### .env opcionales

Si queres habilitar features opcionales, crea un archivo `.env` en la raiz:

```bash
cp .env.example .env
# Editar con tus valores
```

| Variable | Requerida? | Default en docker-compose | Descripcion |
|----------|-----------|---------------------------|-------------|
| **SMTP_HOST** | Opcional | (vacio) | Host SMTP para envio de emails |
| **SMTP_PORT** | Opcional | `587` | Puerto SMTP |
| **SMTP_USER** | Opcional | (vacio) | Usuario SMTP |
| **SMTP_PASSWORD** | Opcional | (vacio) | Password SMTP |
| **MISTRAL_API_KEY** | Opcional | (vacio) | API key de Mistral AI para el Asistente IA |

### Variables ya configuradas en docker-compose.yml (NO tocar en dev)

Estas variables ya tienen valores correctos para desarrollo. No necesitas crearlas:

| Variable | Valor en dev | Descripcion |
|----------|-------------|-------------|
| `DATABASE_URL` | `postgres://blendpos:blendpos@postgres:5432/blendpos?sslmode=disable` | Conexion PostgreSQL |
| `REDIS_URL` | `redis://redis:6379/0` | Conexion Redis |
| `JWT_SECRET` | `dev_secret_change_in_production!_32chars` | Secreto JWT (cambiar en prod!) |
| `PORT` | `8000` | Puerto del backend |
| `ALLOWED_ORIGINS` | `http://localhost:5173` | CORS origins permitidos |
| `AFIP_SIDECAR_URL` | `http://afip-sidecar:8001` | URL del sidecar AFIP |
| `AFIP_CUIT_EMISOR` | `20471955575` | CUIT emisor para homologacion |
| `INTERNAL_API_TOKEN` | `dev_internal_token_change_in_production` | Token interno backend-sidecar |
| `VITE_API_BASE` | `http://localhost:8000` | URL del backend para el frontend |

### Frontend .env (solo si corres fuera de Docker)

Si corres el frontend con `npm run dev` (sin Docker), crea `frontend/.env`:

```env
VITE_API_BASE=http://localhost:8000
```

### AFIP Sidecar (opcional, solo si necesitas facturacion)

El sidecar AFIP **no es necesario para desarrollo general**. Solo se usa para facturacion electronica.

Para habilitarlo necesitas:
1. Certificado AFIP (`.crt`) y clave privada (`.key`)
2. Ponerlos en `afip-sidecar/certs/afip.crt` y `afip-sidecar/certs/afip.key`
3. Levantar con: `docker compose --profile afip up -d`

## Base de Datos

Las migraciones se ejecutan **automaticamente** cuando el backend arranca. No necesitas correr nada manual.

Los archivos SQL estan en `backend/migrations/` (numerados). El backend los aplica en orden al iniciar.

### Verificar tablas

```bash
docker compose exec postgres psql -U blendpos -d blendpos -c "\dt"
# Deberia mostrar ~32 tablas
```

### Reset completo (borra todo y arranca de cero)

```bash
docker compose down -v          # Borra containers + volumenes (datos)
docker compose up -d            # Recrea todo desde cero
# Esperar ~30s y volver a crear el admin:
docker compose exec backend go run cmd/seeduser/main.go
```

## Credenciales de prueba

Despues de correr el seeder (`go run cmd/seeduser/main.go`):

| Rol | Usuario | Contraseña |
|-----|---------|------------|
| Administrador | `admin@blendpos.com` | `1234` |

## Desarrollo Local (sin Docker)

```bash
# Terminal 1: PostgreSQL + Redis (con Docker)
docker compose up -d postgres redis

# Terminal 2: Backend
cd backend
# Crear backend/.env con:
#   DATABASE_URL=postgres://blendpos:blendpos@localhost:5432/blendpos?sslmode=disable
#   REDIS_URL=redis://localhost:6379/0
#   JWT_SECRET=dev_secret_change_in_production!_32chars
go run cmd/server/main.go

# Terminal 3: Frontend
cd frontend
# Crear frontend/.env con:
#   VITE_API_BASE=http://localhost:8000
npm install
npm run dev
```

## Comandos Utiles

```bash
# Levantar todo
docker compose up -d

# Ver logs
docker compose logs -f                    # todos
docker compose logs -f backend            # solo backend

# Reiniciar un servicio
docker compose restart backend

# Rebuild (despues de cambios en Dockerfile)
docker compose up -d --build backend

# Acceder a psql
docker compose exec postgres psql -U blendpos -d blendpos

# Crear/resetear usuario admin
docker compose exec backend go run cmd/seeduser/main.go
```

## Troubleshooting

### "El backend no arranca"
```bash
docker compose logs backend
# Si dice "building..." y nada mas, esperar ~60 segundos (compila Go)
# Si dice error de conexion a postgres, verificar que postgres este healthy:
docker compose ps
```

### "AFIP sidecar falla"
Es normal. El sidecar necesita certificados AFIP reales. El resto del sistema funciona sin el.

### "SMTP warnings al hacer docker compose up"
```
The "SMTP_HOST" variable is not set. Defaulting to a blank string.
```
Son inofensivos. Solo aparecen porque no configuraste SMTP (emails), que es opcional.

### "El frontend carga pero el login falla"
1. Verificar que el backend este corriendo: `docker compose ps`
2. Verificar que el admin existe: `docker compose exec backend go run cmd/seeduser/main.go`
3. Credenciales: usuario `admin@blendpos.com`, password `1234`

### Reset total
```bash
docker compose down -v && docker compose up -d
# Esperar ~30s, luego:
docker compose exec backend go run cmd/seeduser/main.go
```

## Notas Importantes

1. **Montos** — El backend serializa montos como strings JSON (`"650.00"` no `650.00`) usando `shopspring/decimal`.

2. **Offline-first** — Las ventas se persisten localmente en IndexedDB y se sincronizan via `/v1/ventas/sync-batch`. El POS funciona sin conexion.

3. **Cambios de esquema** — Crear nuevas migraciones en `backend/migrations/`. No modificar tablas directamente.

4. **Produccion** — Cambiar `JWT_SECRET`, `POSTGRES_PASSWORD`, y setear `AFIP_HOMOLOGACION=false`.
