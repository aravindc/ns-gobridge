# ns-gobridge

A lightweight Go bridge that polls the [Dexcom Share](https://www.dexcom.com/) API for continuous glucose monitor (CGM) readings and stores them in a Postgres database, using a schema compatible with [Nightscout](https://github.com/nightscout/cgm-remote-monitor). It also exposes a REST API for the stored readings, a set of derived glucose insights (quartiles, variability, patterns, trend, data quality), and logging carbs/insulin treatments alongside them.

## How it works

1. On startup, the app loads configuration from an `.env` file.
2. It authenticates against the Dexcom Share service (`share2.dexcom.com` for the US region, `shareous1.dexcom.com` for outside the US) to obtain a session ID.
3. Every minute, it polls the Dexcom Share API for the latest glucose readings.
4. Each new reading (deduplicated by timestamp) is written to the `nightscoutdb` table in Postgres.
5. A REST API (served concurrently) exposes the stored readings, computed insights, and carbs/insulin treatments (the latter logged directly via the API, not polled from Dexcom).

## Project layout

- [main.go](main.go) — entry point; loads config, runs the polling loop, and starts the REST API.
- [bridge/](bridge/) — Dexcom Share API client (authentication, session handling, fetching readings).
- [db/](db/) — Postgres storage layer (built on [uptrace/bun](https://github.com/uptrace/bun)): connecting, checking for existing entries, inserting/selecting readings (including latest-entry and time-range queries) and carbs/insulin treatments.
- [model/](model/) — data models for Dexcom readings, the Nightscout-style DB record, a MySugr export format, carbs/insulin treatments, and computed glucose insights (stats, quartiles, hour-of-day and day-of-week patterns, variability, rate-of-change, rolling trend, data quality).
- [common/](common/) — shared helpers (trend/date parsing utilities, trend-to-arrow display).
- [web/](web/) — REST API ([gin](https://github.com/gin-gonic/gin)) exposing stored readings, derived insights, and treatment logging.

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

[docker-compose.yaml](docker-compose.yaml) runs the app alongside its own Postgres container and a [Caddy](https://caddyserver.com/) reverse proxy, with the `nightscoutdb` and `treatments` tables created automatically from [db/init/](db/init/) on first start. These init scripts only run against a fresh Postgres data volume — an already-initialized volume needs the new table(s) created manually (or the volume recreated) after pulling schema changes.

The `postgres` and `ns-gobridge` services load their environment via `env_file` rather than inline `environment:` blocks, so before first use, create these two untracked files (matching the treatment of [.env.development](.env.development) — never commit real values):

```bash
# .env.postgres — consumed by the official postgres image
cat > .env.postgres <<'EOF'
POSTGRES_USER=postgres
POSTGRES_PASSWORD=secret
POSTGRES_DB=health
EOF

# .env.ns-gobridge — compose-specific overrides for ns-gobridge, on top of .env.development
cat > .env.ns-gobridge <<'EOF'
NS_ENV=development
PG_HOST=postgres
PG_PORT=5432
PG_SSLMODE=disable
EOF
```

```bash
docker compose up --build
```

This uses [.env.development](.env.development) for Dexcom/app configuration (mounted read-only into the container) and connects to the bundled Postgres over plain TCP (`PG_SSLMODE=disable`, since it's a local container, not a TLS-terminated managed database). Update `.env.development` with real Dexcom credentials before starting.

Neither `ns-gobridge` (port 8080) nor `postgres` (port 5432) publish their ports to the host directly — both are reachable only from other containers on the internal compose network. The [Caddyfile](Caddyfile)-configured `proxy` service is the single public entrypoint, listening on `80`/`443` and forwarding `/api/*` to `ns-gobridge:8080`. All other paths return a 404 from the proxy. (For local debugging access to Postgres directly, e.g. via `psql` or a GUI client, temporarily add a `ports: ["5432:5432"]` mapping back to the `postgres` service.)

Caddy is configured for the domain `api.health.pers.dev` and requests/renews a real Let's Encrypt certificate for it automatically (Caddy's "automatic HTTPS"). For this to work in a real deployment:

- DNS for `api.health.pers.dev` must point at the host running this compose stack.
- Ports 80 and 443 must be reachable from the internet (port 80 is used for the ACME HTTP-01 challenge as well as HTTP→HTTPS redirects).
- The `caddy_data`/`caddy_config` named volumes persist the issued certificate across restarts — don't remove them casually, or Caddy will re-request a cert from Let's Encrypt (which is rate-limited).

For local development without a real domain/public DNS, edit [Caddyfile](Caddyfile) to use `localhost` (or `:80`) with `tls internal` instead, which makes Caddy issue a self-signed certificate from its own internal CA rather than requesting one from Let's Encrypt.

The REST API is then available at `https://api.health.pers.dev/api/...` (or `http://localhost/api/...` if using the local-dev Caddyfile variant above).

## REST API

The app serves a REST API on `PORT` (default `8080`) alongside the Dexcom polling loop. If `API_KEY` is set, all routes except `/api/health` require it via an `X-API-Key` header.

### Readings

| Endpoint | Description |
|---|---|
| `GET /api/health` | Liveness check. Never requires auth. |
| `GET /api/current` | Latest glucose reading, trend, and direction arrow. |
| `GET /api/device/current` | Minimal flat JSON for constrained IoT clients (e.g. M5Stack): `{"sgv":120,"dir":"→","mins_ago":3}`. |
| `GET /api/entries?from=&to=` | Readings between two RFC3339 timestamps (defaults to the last 24 hours). |

`/api/current` and `/api/device/current` are served from a 10-second in-process cache, so many clients polling concurrently (e.g. a fleet of IoT devices) only translate to one Postgres query per cache window, not one per request.

### Insights

All insight endpoints accept `?period=24h|1wk|1mth|3mths` (a lookback window ending now) instead of explicit `from=`/`to=` timestamps, except `/api/stats`, which keeps the original `from=`/`to=` range params. Each endpoint's default period is chosen for what's statistically meaningful for that computation (e.g. hour-of-day patterns default to a month, since a single day gives at most one or two samples per hour bucket).

| Endpoint | Description |
|---|---|
| `GET /api/stats?from=&to=` | Computed insights over a range (defaults to the last 24 hours): average glucose, min/max, estimated HbA1c, GMI, time-in-range/below/above percentages (70–180 mg/dL), and low/high episode counts. For the standard 90-day GMI reporting window, pass `from=` set to 90 days ago. Default range is served from the same 10-second cache as `/api/current`. |
| `GET /api/quartiles?period=` | Glucose quartiles (Q1/median/Q3) plus min/max over a period (default `24h`). |
| `GET /api/variability?period=` | Glycemic variability over a period (default `24h`): standard deviation and coefficient of variation (CV%). Per the ADA/ATTD consensus, CV% ≤ 36% indicates stable control. |
| `GET /api/rate-of-change?period=` | Dexcom trend-code distribution and computed rate-of-change statistics (mg/dL per minute, from consecutive readings) over a period (default `24h`), including rapid rise/fall episode counts (≥ 2 mg/dL/min). |
| `GET /api/patterns/hourly?period=` | Glucose statistics bucketed by hour-of-day (0–23) over a period (default `1mth`) — surfaces recurring patterns like the dawn phenomenon or post-meal spikes that a single whole-period stat would smooth over. |
| `GET /api/patterns/day-of-week?period=` | Glucose statistics bucketed by day of week (Sunday–Saturday) over a period (default `1mth`) — surfaces weekday-vs-weekend differences in control. |
| `GET /api/trend/rolling?period=` | Slices a period (default `3mths`) into successive 7-day buckets, reporting average glucose/time-in-range/CV per bucket, so whether control is improving or worsening over time can be read off directly. |
| `GET /api/data-quality?period=` | Sensor coverage and gap detection over a period (default `1wk`): gaps between consecutive readings (more than 10 minutes apart), a coverage percentage, and the largest gap. Useful context for whether other endpoints' figures are based on representative data or skewed by a dropout (e.g. an overnight sensor loss). |

### Treatments (carbs & insulin)

| Endpoint | Description |
|---|---|
| `GET /api/treatments?from=&to=` | Logged carbs/insulin treatments between two RFC3339 timestamps (defaults to the last 24 hours). |
| `POST /api/treatments` | Logs a carbs/insulin treatment. Body: `carbs` (int, optional), `insulin` (number, optional), `mealType` (one of `breakfast`/`lunch`/`dinner`/`snack`, optional), `foodDescription` (string, optional), `datetime` (RFC3339, optional — defaults to now). `carbs` and `insulin` are independently optional, so a correction bolus with no food, or carbs logged without a dose, can both be recorded. Returns `201 Created` with the saved record. |

All glucose values (`sgv`, `averageSgv`, `minSgv`, `maxSgv`, quartiles, hourly/day-of-week stats, rolling-trend averages) are returned in `mg/dl` by default. Add `?units=mmol` to any glucose-returning endpoint to get values in mmol/L instead (rounded to 1–2 decimal places), or set `UNITS=mmol` to change the server-wide default. The response always includes a `units` field so clients can tell which one they got. Treatment endpoints and rate-of-change trend counts/percentages are unit-independent.

Examples:

```bash
curl -H "X-API-Key: $API_KEY" "http://localhost:8080/api/stats?from=2026-07-17T00:00:00Z&to=2026-07-18T00:00:00Z"
curl -H "X-API-Key: $API_KEY" "http://localhost:8080/api/device/current?units=mmol"
curl -H "X-API-Key: $API_KEY" "http://localhost:8080/api/patterns/hourly?period=1mth"
curl -H "X-API-Key: $API_KEY" "http://localhost:8080/api/variability?period=1wk"

curl -X POST -H "X-API-Key: $API_KEY" -H "Content-Type: application/json" \
  -d '{"carbs": 45, "insulin": 4.5, "mealType": "lunch", "foodDescription": "pasta with garlic bread"}' \
  "http://localhost:8080/api/treatments"
```

## Testing

```bash
go test -v ./...
```

## Releases

Tagged pushes are built and published via [GoReleaser](.goreleaser.yaml), with release artifacts signed using [cosign](https://github.com/sigstore/cosign) (see [.github/workflows/](.github/workflows/)).
