# Driftwatch

Driftwatch is a Go CLI and HTTP service for finding anomalies and sustained drift in backend telemetry streams.

It is intentionally small enough to understand, but not a tutorial clone: it has a tested detection core, CSV ingestion, a JSON API, Docker support, CI, and sample telemetry that proves the system works without needing a real production service.

## Why this project exists

Backend teams still need engineers who can reason about production behavior, not only build CRUD endpoints. Driftwatch demonstrates:

- Go service design with standard-library HTTP
- rolling per-series state for streaming data
- robust statistics with median/MAD anomaly scoring
- Page-Hinkley-style sustained drift detection
- CI-friendly CLI behavior and machine-readable reports
- Dockerized deployment without external services

## Quick start

```powershell
go test ./...
go run ./cmd/driftwatch -input samples/latency.csv
```

Write a report:

```powershell
go run ./cmd/driftwatch -input samples/latency.csv -out reports/latency-report.json
```

Run the API:

```powershell
go run ./cmd/api -addr :8080
```

Post the sample request:

```powershell
Invoke-RestMethod -Method Post -Uri http://localhost:8080/detect -ContentType application/json -InFile samples/payload.json
```

## CSV input format

```csv
timestamp,series,value
2026-06-19T09:00:00Z,api.latency.p95,100
2026-06-19T09:01:00Z,api.latency.p95,101
```

Timestamps must be RFC3339 or RFC3339Nano. Values must be finite floats.

## API

`GET /health`

Returns:

```json
{ "status": "ok" }
```

`POST /detect`

Request:

```json
{
  "config": {
    "windowSize": 8,
    "minSamples": 4,
    "madThreshold": 6,
    "ewmaAlpha": 0.25,
    "pageHinkleyDelta": 0.1,
    "driftThreshold": 25
  },
  "points": [
    { "timestamp": "2026-06-19T09:00:00Z", "series": "api.latency.p95", "value": 100 }
  ]
}
```

The endpoint returns HTTP `200` when no signals are found and HTTP `422` when anomalies or drift are detected.

## Repository structure

```text
cmd/api              HTTP server entrypoint
cmd/driftwatch       CLI entrypoint
internal/csvio       CSV ingestion and report export
internal/detector    detection engine and tests
internal/httpapi     API handler and tests
internal/timeseries  shared telemetry models
samples              synthetic datasets and API payloads
docs                 algorithm and testing notes
```

## Algorithm

See [docs/algorithm.md](docs/algorithm.md) for the math and reasoning behind the rolling robust z-score, EWMA baseline, and Page-Hinkley-style drift score.

## Security

The repository contains no credentials and the sample data is synthetic. `.env` files are ignored by default, and CI uses only local tests/builds.
