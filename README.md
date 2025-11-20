# OmniAPI Platform

OmniAPI is a dual-stack platform built around a Go backend for real-time IoT/industrial data ingestion and a React + Vite frontend that lets operators configure tenants, sites, connectors, and external services. The repository is intentionally split into two top-level folders so each stack can evolve independently while sharing a single Git history.

## Repository layout

| Path        | Description                                                                                                         | Key tech                                    |
| ----------- | ------------------------------------------------------------------------------------------------------------------- | ------------------------------------------- |
| `backend/`  | Go services, connectors, WebSocket hub, MongoDB persistence, Prometheus metrics, extensive docs                     | Go 1.24+, MongoDB 7, Gorilla WS, Prometheus |
| `frontend/` | Admin dashboard that consumes the OmniAPI REST/WebSocket surface; includes auth, setup wizard, and management views | React 18, TypeScript, Vite, Axios           |

## Backend (Go)

### Highlights

- Multi-tenant architecture with connector framework (MQTT feed, REST climate, dummy adapters, etc.).
- Dual-queue design: a requester queue for upstream polling plus a status-pusher queue for health heartbeats.
- JSON Schema validation, WebSocket broadcasting (`/ws` + `/ws/test`), Prometheus metrics, and MongoDB repositories.
- Rich documentation inside `backend/README.md` and `internal/websocket/PROTOCOL.md`.

### Prerequisites

- Go **1.24+**
- MongoDB **7.0+** (local or remote)
- Optional: Docker / docker-compose if you prefer containerized services.

### Environment variables

```bash
cd backend
cp .env.example .env
# edit the values below with your secrets
# - MONGODB_URI / MONGODB_DATABASE / MONGODB_TIMEOUT
# - JWT_SECRET (used for API + WebSocket auth)
# - Connector/API secrets such as MQTT_FEED_SECRET, WEATHER_API_KEY, etc.
```

### Run locally

```bash
cd backend
go mod tidy
go run ./cmd/api
```

By default the API listens on `http://localhost:3000`. Health/info endpoints and the WebSocket hub are documented in `backend/README.md`.

### Tests & tooling

```bash
cd backend
go test ./...
```

Add `-run` or `-count=1` flags as needed when iterating on specific packages.

## Frontend (React + Vite)

### What the frontend does

The dashboard lives under `frontend/` and is composed of:

- **Setup Wizard (`/setup`)** – Onboards the first admin user by calling the backend bootstrap endpoints.
- **Auth flow (`/login`)** – Uses `AuthContext`, persists JWTs in `localStorage`, and guards routes via `ProtectedRoute`.
- **Dashboard shell (`/dashboard`)** – Provides the main layout, quick stats, and shortcuts for daily operations.
- **Tenants module** – CRUD UI for salmon companies with search, modal-based forms, and status badges (`pages/Tenants.tsx`).
- **Sites module** – Management UI for farming sites, including location metadata and tenant linkage.
- **External services & connectors** – Screens to register ScaleAQ/Innovex-like services, run connection tests, and monitor statuses.
- **Services view** – Overview of registered adapters plus placeholders for future connectors/settings pages.

All API calls go through `src/services/api.ts`, which injects the JWT header and automatically redirects to `/login` on 401 responses.

### Prerequisites

- Node.js **18+** (or the current LTS that Vite supports)
- npm / pnpm / yarn (examples below assume npm)

### Environment variables

Create a `.env` file in `frontend/` with at least:

```dotenv
VITE_API_URL=http://localhost:3000
```

Point `VITE_API_URL` to wherever the Go API is running (adjust when deploying).

### Run locally

```bash
cd frontend
npm install
npm run dev -- --host
```

Vite serves the SPA on `http://localhost:5173` (or the next available port); use `npm run build` + `npm run preview` to test production bundles.

## Developing both stacks together

1. Start MongoDB (and any upstream services you depend on).
2. In one terminal: `cd backend && go run ./cmd/api`.
3. In another terminal: `cd frontend && npm run dev -- --host`.
4. Log in via `http://localhost:5173/login` (or run the setup wizard) and begin managing tenants, sites, services, or triggering connector flows. The frontend proxies all API calls to the backend URL defined in `VITE_API_URL`.

## Additional documentation

- `backend/README.md` – Deep dive into architecture, connector lifecycle, Prometheus queries, troubleshooting, etc.
- `backend/internal/websocket/PROTOCOL.md` – Full WebSocket contract (DATA vs STATUS events, throttling, keep-latest policy, and sample clients).
- `docs/CONNECTORS_MANUAL.md` – Manual de uso del dashboard de Conectores y discovery ScaleAQ.
- Component-level docs live alongside the code (`internal/**`, `frontend/src/pages/**`).

With this top-level README you can quickly understand how the repository is organized, what each stack is responsible for, and how to run or extend the platform end-to-end.
