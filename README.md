# warden-gateway

The **data ingestion layer** of the Warden smart-home platform. It simulates sensors,
streams readings to browsers over WebSocket, publishes them to NATS for downstream
services (warden-engine), and persists them to TimescaleDB. It also exposes a REST API
for historical queries.

Architecture: Go-idiomatic Hexagonal (Ben Johnson pattern) — domain types and interfaces
live at the root package; adapters are named by technology (`postgres/`, `nats/`),
transports by protocol (`http/`). See `CLAUDE.md` for the full layout and rationale.

---

## REST API contract

This API is **both** user-facing (dashboard) **and** service-to-service (e.g. warden-brain
discovers rooms before querying readings). The contracts below are stable; downstream
services depend on them.

Base URL (local): `http://localhost:8080`

### Authentication

`/api/*` endpoints are protected by the JWT verifier middleware. Callers must send:

```
Authorization: Bearer <jwt>
```

The token is verified against the issuer's JWKS (`JWKS_URL`), and must carry the expected
`iss` (`ISSUER`) and `aud` (`AUDIENCE`, default `warden-gateway`). Every query is
**tenant-scoped**: the `tenant` claim in the token determines which data the caller can see.
A token without a `tenant` claim is rejected with `401 Unauthorized`.

Service-to-service callers (warden-brain) obtain a token scoped to this audience via
warden-auth's `POST /token/exchange` (`audience: warden-gateway`), which preserves the
original tenant.

---

### `GET /api/readings`

Returns historical sensor readings for one room and sensor type within a time window,
newest first. Tenant-scoped.

**Query parameters** (all required):

| Param  | Type                  | Description                                  |
|--------|-----------------------|----------------------------------------------|
| `room` | string                | Room name, e.g. `bedroom`                    |
| `type` | string                | Sensor type: `temperature` \| `humidity` \| `motion` \| `co2` |
| `from` | RFC3339 timestamp     | Window start (inclusive), e.g. `2026-01-01T00:00:00Z` |
| `to`   | RFC3339 timestamp     | Window end (inclusive)                       |

**Response** `200 OK` — JSON array of readings:

```json
[
  {
    "tenant_id": "tenant-a",
    "sensor_id": "s1",
    "room": "bedroom",
    "type": "temperature",
    "value": 22.5,
    "unit": "°C",
    "timestamp": "2026-06-13T16:54:27Z"
  }
]
```

**Errors:** `400` (missing/invalid `room`, `type`, `from`, or `to`),
`401` (missing/invalid token or no `tenant` claim), `500` (query failure).

---

### `GET /api/rooms`

Returns the distinct room names that have readings for the caller's tenant, sorted
alphabetically. Tenant-scoped. Used by warden-brain to build a dynamic enum of valid
rooms for its `get_readings` tool.

**Query parameters:** none.

**Response** `200 OK` — JSON array of strings:

```json
["bedroom", "kitchen"]
```

When the tenant has no readings yet, the response is an empty array `[]` (never `null`),
so consumers can always parse an array.

**Errors:** `401` (missing/invalid token or no `tenant` claim), `500` (query failure).

---

### `GET /ws`

WebSocket upgrade. Once connected, the client receives real-time sensor readings pushed
by the gateway as they are produced.

---

### Operational endpoints (unauthenticated)

| Endpoint        | Description                                                        |
|-----------------|-------------------------------------------------------------------|
| `GET /health/live`  | Liveness probe. Always `200 ok` while the process is up.      |
| `GET /health/ready` | Readiness probe. `200` only if PostgreSQL and NATS are reachable; otherwise `503`. |
| `GET /metrics`      | Prometheus metrics exposition.                                |

---

## Configuration

Loaded from environment variables (with defaults for local dev) — see
`internal/config/config.go`:

| Variable            | Default                                                      | Purpose                                  |
|---------------------|-------------------------------------------------------------|------------------------------------------|
| `DATABASE_URL`      | `postgres://postgres:postgres@localhost:5432/warden_gateway_db` | TimescaleDB connection string         |
| `NATS_URL`          | `nats://localhost:4222`                                      | NATS server                              |
| `HTTP_PORT`         | `:8080`                                                      | HTTP listen address                      |
| `JAEGER_ENDPOINT`   | `localhost:4318`                                             | OTLP trace exporter endpoint             |
| `JWKS_URL`          | `http://localhost:8082/.well-known/jwks.json`               | JWKS for JWT verification                |
| `ISSUER`            | `warden-auth`                                                | Expected token `iss`                     |
| `AUDIENCE`          | `warden-gateway`                                             | Expected token `aud`                     |
| `SIMULATOR_ENABLED` | `true`                                                       | Run the in-process sensor simulator      |
| `SENSOR_INTERVAL`   | `500ms`                                                      | Interval between simulated readings      |

---

## Development

```sh
go build ./...
go vet ./...
go test ./...        # postgres/ integration tests require Docker (testcontainers)
sqlc generate        # regenerate db/generated/ after editing db/queries/
```
