package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kamilch1k/driftwatch/internal/detector"
	"github.com/kamilch1k/driftwatch/internal/timeseries"
)

func TestHealth(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	NewHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestDetectReturnsUnprocessableWhenSignalsExist(t *testing.T) {
	payload := DetectRequest{
		Config: detector.Config{
			WindowSize:     4,
			MinSamples:     4,
			MADThreshold:   6,
			DriftThreshold: 1_000,
		},
		Points: []timeseries.Point{
			apiPoint(0, 100),
			apiPoint(1, 101),
			apiPoint(2, 99),
			apiPoint(3, 100),
			apiPoint(4, 190),
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/detect", bytes.NewReader(body))
	recorder := httptest.NewRecorder()

	NewHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	var response DetectResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Signals || response.Report.Anomalies != 1 {
		t.Fatalf("expected one anomaly response, got %#v", response)
	}
}

func TestDetectRejectsEmptyPayload(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/detect", bytes.NewReader([]byte(`{"points":[]}`)))
	recorder := httptest.NewRecorder()

	NewHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func apiPoint(offset int, value float64) timeseries.Point {
	return timeseries.Point{
		Timestamp: time.Date(2026, 6, 19, 9, offset, 0, 0, time.UTC),
		Series:    "api.latency.p95",
		Value:     value,
	}
}
