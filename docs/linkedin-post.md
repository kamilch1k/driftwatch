# LinkedIn Post Draft

I built Driftwatch, a Go CLI and HTTP service for detecting anomalies and sustained drift in backend telemetry streams.

The project uses rolling median/MAD scoring for spike detection, EWMA baselines for short-term movement, and a Page-Hinkley-style score for sustained level shifts. It includes a tested detector core, CSV ingestion, a JSON API, sample telemetry, Docker support, and CI.

What I wanted to show with it:

- backend service design in Go
- production-oriented thinking around latency/error metrics
- explainable statistical detection instead of a black-box demo
- a clean CLI/API surface that can be tested locally

Repo: https://github.com/kamilch1k/driftwatch
