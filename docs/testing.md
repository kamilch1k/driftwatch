# Testing Guide

## CLI smoke test

```powershell
go run ./cmd/driftwatch -input samples/latency.csv
```

Expected result: a one-line summary plus anomaly or drift rows for the synthetic spike and sustained shift.

Write a JSON report:

```powershell
go run ./cmd/driftwatch -input samples/latency.csv -out reports/latency-report.json
```

Fail a pipeline when signals exist:

```powershell
go build -o bin/driftwatch.exe ./cmd/driftwatch
.\bin\driftwatch.exe -input samples/latency.csv -fail-on-signal
```

The last command should exit with code `2` because the sample data intentionally contains signals.

## HTTP smoke test

Start the API:

```powershell
go run ./cmd/api -addr :8080
```

Submit the sample payload from another terminal:

```powershell
Invoke-RestMethod -Method Post -Uri http://localhost:8080/detect -ContentType application/json -InFile samples/payload.json
```

Expected result: HTTP `422` with a JSON report because the payload contains a deliberate anomaly.

## Unit tests

```powershell
go test ./...
```

The tests cover CSV parsing, spike anomaly detection, sustained drift detection, per-series baselines, input order safety, and HTTP response behavior.
