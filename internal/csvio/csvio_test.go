package csvio

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/kamilch1k/driftwatch/internal/timeseries"
)

func TestReadPoints(t *testing.T) {
	input := strings.NewReader(`timestamp,series,value
2026-06-19T09:00:00Z,api.latency.p95,100.5
2026-06-19T09:01:00Z,api.latency.p95,101.25
`)

	points, err := ReadPoints(input)
	if err != nil {
		t.Fatalf("ReadPoints returned error: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("expected two points, got %d", len(points))
	}
	if points[0].Series != "api.latency.p95" || points[0].Value != 100.5 {
		t.Fatalf("unexpected first point: %#v", points[0])
	}
}

func TestReadPointsRejectsMissingHeaders(t *testing.T) {
	_, err := ReadPoints(strings.NewReader("timestamp,value\n2026-06-19T09:00:00Z,100\n"))
	if err == nil {
		t.Fatal("expected missing header error")
	}
}

func TestWriteDetections(t *testing.T) {
	var buffer bytes.Buffer
	err := WriteDetections(&buffer, []timeseries.Detection{
		{
			Timestamp:   testTime(t, "2026-06-19T09:00:00Z"),
			Series:      "api.latency.p95",
			Value:       190,
			Signal:      timeseries.SignalAnomaly,
			Median:      100,
			MAD:         1,
			RobustZ:     60.7,
			EWMA:        120,
			DriftScore:  12,
			Explanation: "robust z-score exceeded threshold",
		},
	})
	if err != nil {
		t.Fatalf("WriteDetections returned error: %v", err)
	}
	output := buffer.String()
	if !strings.Contains(output, "api.latency.p95") || !strings.Contains(output, "anomaly") {
		t.Fatalf("unexpected csv output: %s", output)
	}
}

func testTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse helper time: %v", err)
	}
	return parsed
}
