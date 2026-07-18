# ns-gobridge

A lightweight Go bridge that polls the [Dexcom Share](https://www.dexcom.com/) API for continuous glucose monitor (CGM) readings and stores them in a Postgres database, using a schema compatible with [Nightscout](https://github.com/nightscout/cgm-remote-monitor).

## How it works

1. On startup, the app loads configuration from an `.env` file.
2. It authenticates against the Dexcom Share service (`share2.dexcom.com` for the US region, `shareous1.dexcom.com` for outside the US) to obtain a session ID.
3. Every minute, it polls the Dexcom Share API for the latest glucose readings.
4. Each new reading (deduplicated by timestamp) is written to the `nightscoutdb` table in Postgres.
5. A REST API (served concurrently) exposes the stored readings and computed insights.

## Project layout

- [main.go](main.go) — entry point; loads config, runs the polling loop, and starts the REST API.
- [bridge/](bridge/) — Dexcom Share API client (authentication, session handling, fetching readings).
- [db/](db/) — Postgres storage layer (built on [uptrace/bun](https://github.com/uptrace/bun)): connecting, checking for existing entries, inserting/selecting readings (including latest-entry and time-range queries).
- [model/](model/) — data models for Dexcom readings, the Nightscout-style DB record, a MySugr export format, and computed glucose statistics.
- [common/](common/) — shared helpers (trend/date parsing utilities, trend-to-arrow display).
- [web/](web/) — REST API ([gin](https://github.com/gin-gonic/gin)) exposing stored readings and derived insights.

## Configuration

Configuration is provided via environment variables, loaded from an env file selected by `NS_ENV`:

- `development` (default) → `.env.development`
- `production` → `.env`
- `test` → `.env.test`

Required environment variables:

| Variable | Description |
|---|---|
| `NS_ENV` | Environment name: `development`, `production`, or `test`. |
| `BRIDGE_SERVER` | Dexcom Share region: `US` or any other value for outside the US. |
| `BRIDGE_USER` / `BRIDGE_PASS` | Dexcom Share account credentials. |
| `APPLICATION_ID` | Dexcom Share application ID. |
| `RECORD_COUNT` | Max number of readings to fetch per poll (defaults to `3`). |
| `PG_HOST` / `PG_PORT` / `PG_USER` / `PG_PASS` / `PG_DB` | Postgres connection details for storing readings. |
| `PG_SSLMODE` | `disable` to connect to Postgres without TLS (e.g. a local/Docker Postgres), or anything else (e.g. `require`) to connect with TLS (e.g. managed providers like Neon). Defaults to `require`. |
| `PORT` | Port for the REST API (defaults to `8080`). |
| `API_KEY` | If set, requires this value in an `X-API-Key` header on all `/api/*` routes except `/api/health`. Leave unset to disable auth (e.g. local development only). |
| `UNITS` | Default glucose display unit for the REST API: `mg/dl` (default) or `mmol`. Overridable per-request with `?units=`. |

See [.env.development](.env.development) for an example (development) configuration.

> **Note:** Never commit real credentials to this file or the repository — use placeholder values in version control and inject real secrets via your deployment environment or a secrets manager.

## Building and running

```bash
go build -o ns-gobridge .
NS_ENV=development ./ns-gobridge
```

### Docker

```bash
docker build -t ns-gobridge .
```

### Docker Compose

[docker-compose.yaml](docker-compose.yaml) runs the app alongside its own Postgres container, with the `nightscoutdb` schema created automatically from [db/init/](db/init/) on first start.

```bash
docker compose up --build
```

This uses [.env.development](.env.development) for Dexcom/app configuration (mounted read-only into the container) and connects to the bundled Postgres over plain TCP (`PG_SSLMODE=disable`, since it's a local container, not a TLS-terminated managed database). Update `.env.development` with real Dexcom credentials before starting. The REST API is then available at `http://localhost:8080`.

## REST API

The app serves a REST API on `PORT` (default `8080`) alongside the Dexcom polling loop. If `API_KEY` is set, all routes except `/api/health` require it via an `X-API-Key` header.

| Endpoint | Description |
|---|---|
| `GET /api/health` | Liveness check. Never requires auth. |
| `GET /api/current` | Latest glucose reading, trend, and direction arrow. |
| `GET /api/device/current` | Minimal flat JSON for constrained IoT clients (e.g. M5Stack): `{"sgv":120,"dir":"→","mins_ago":3}`. |
| `GET /api/entries?from=&to=` | Readings between two RFC3339 timestamps (defaults to the last 24 hours). |
| `GET /api/stats?from=&to=` | Computed insights over a range (defaults to the last 24 hours): average glucose, min/max, estimated HbA1c, GMI, time-in-range/below/above percentages (70–180 mg/dL), and low/high episode counts. For the standard 90-day GMI reporting window, pass `from=` set to 90 days ago. |

`/api/current`, `/api/device/current`, and `/api/stats` (default range only) are served from a 10-second in-process cache, so many clients polling concurrently (e.g. a fleet of IoT devices) only translate to one Postgres query per cache window, not one per request.

All glucose values (`sgv`, `averageSgv`, `minSgv`, `maxSgv`) are returned in `mg/dl` by default. Add `?units=mmol` to any endpoint to get values in mmol/L instead (rounded to 1 decimal place), or set `UNITS=mmol` to change the server-wide default. The response always includes a `units` field so clients can tell which one they got.

Example:

```bash
curl -H "X-API-Key: $API_KEY" "http://localhost:8080/api/stats?from=2026-07-17T00:00:00Z&to=2026-07-18T00:00:00Z"
curl -H "X-API-Key: $API_KEY" "http://localhost:8080/api/device/current?units=mmol"
```

## Testing

```bash
go test -v ./...
```

## Releases

Tagged pushes are built and published via [GoReleaser](.goreleaser.yaml), with release artifacts signed using [cosign](https://github.com/sigstore/cosign) (see [.github/workflows/](.github/workflows/)).
