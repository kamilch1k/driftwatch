package detector

import (
	"testing"
	"time"

	"github.com/kamilch1k/driftwatch/internal/timeseries"
)

var testBaseTime = time.Date(2026, 6, 19, 9, 0, 0, 0, time.UTC)

func testPoint(offset int, series string, value float64) timeseries.Point {
	return timeseries.Point{
		Timestamp: testBaseTime.Add(time.Duration(offset) * time.Minute),
		Series:    series,
		Value:     value,
	}
}

func TestAnalyzeDetectsSpikeAnomaly(t *testing.T) {
	points := []timeseries.Point{
		testPoint(0, "api.latency.p95", 100),
		testPoint(1, "api.latency.p95", 101),
		testPoint(2, "api.latency.p95", 99),
		testPoint(3, "api.latency.p95", 100),
		testPoint(4, "api.latency.p95", 102),
		testPoint(5, "api.latency.p95", 101),
		testPoint(6, "api.latency.p95", 99),
		testPoint(7, "api.latency.p95", 100),
		testPoint(8, "api.latency.p95", 190),
	}

	report := Analyze(points, Config{
		WindowSize:     8,
		MinSamples:     8,
		MADThreshold:   6,
		DriftThreshold: 1_000,
	})

	if report.Anomalies != 1 {
		t.Fatalf("expected one anomaly, got %d: %#v", report.Anomalies, report.Detections)
	}
	last := report.Detections[len(report.Detections)-1]
	if last.Signal != timeseries.SignalAnomaly {
		t.Fatalf("expected last point to be an anomaly, got %s", last.Signal)
	}
	if last.RobustZ <= 6 {
		t.Fatalf("expected robust z-score above threshold, got %.2f", last.RobustZ)
	}
}

func TestAnalyzeDetectsSustainedDrift(t *testing.T) {
	points := make([]timeseries.Point, 0, 32)
	for i := range 16 {
		points = append(points, testPoint(i, "checkout.errors", 10+float64(i%2)))
	}
	for i := range 16 {
		points = append(points, testPoint(i+16, "checkout.errors", 28+float64(i%3)))
	}

	report := Analyze(points, Config{
		WindowSize:       16,
		MinSamples:       8,
		MADThreshold:     1_000,
		PageHinkleyDelta: 0.1,
		DriftThreshold:   35,
	})

	if report.Drifts == 0 {
		t.Fatalf("expected sustained level shift to trigger drift: %#v", report.Detections)
	}
	if report.Detections[len(report.Detections)-1].Signal == timeseries.SignalAnomaly {
		t.Fatalf("drift test should not rely on spike anomalies")
	}
}

func TestAnalyzeDoesNotMutateInputOrder(t *testing.T) {
	points := []timeseries.Point{
		testPoint(3, "b", 2),
		testPoint(1, "a", 1),
		testPoint(2, "a", 2),
	}
	originalFirst := points[0]

	_ = Analyze(points, Config{MinSamples: 2})

	if points[0] != originalFirst {
		t.Fatalf("Analyze should not reorder the caller's slice")
	}
}

func TestDetectorKeepsSeriesBaselinesSeparate(t *testing.T) {
	points := []timeseries.Point{
		testPoint(0, "api-a", 100),
		testPoint(1, "api-a", 101),
		testPoint(2, "api-a", 99),
		testPoint(3, "api-a", 100),
		testPoint(4, "api-a", 190),
		testPoint(0, "api-b", 300),
		testPoint(1, "api-b", 301),
		testPoint(2, "api-b", 299),
		testPoint(3, "api-b", 300),
		testPoint(4, "api-b", 301),
	}

	report := Analyze(points, Config{
		WindowSize:     4,
		MinSamples:     4,
		MADThreshold:   6,
		DriftThreshold: 1_000,
	})

	if report.SeriesScanned != 2 {
		t.Fatalf("expected two series, got %d", report.SeriesScanned)
	}
	if report.Anomalies != 1 {
		t.Fatalf("expected one anomaly isolated to api-a, got %d", report.Anomalies)
	}
	for _, detection := range report.Detections {
		if detection.Series == "api-b" && detection.Signal != timeseries.SignalNormal {
			t.Fatalf("api-b should stay normal, got %#v", detection)
		}
	}
}
