package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kamilch1k/driftwatch/internal/detector"
	"github.com/kamilch1k/driftwatch/internal/timeseries"
)

type DetectRequest struct {
	Config detector.Config    `json:"config,omitempty"`
	Points []timeseries.Point `json:"points"`
}

type DetectResponse struct {
	Signals bool              `json:"signals"`
	Report  timeseries.Report `json:"report"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("POST /detect", handleDetect)
	return mux
}

func handleHealth(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func handleDetect(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	request.Body = http.MaxBytesReader(writer, request.Body, 2<<20)

	var payload DetectRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		status := http.StatusBadRequest
		if errors.As(err, new(*http.MaxBytesError)) {
			status = http.StatusRequestEntityTooLarge
		}
		writeJSON(writer, status, errorResponse{Error: err.Error()})
		return
	}
	if len(payload.Points) == 0 {
		writeJSON(writer, http.StatusBadRequest, errorResponse{Error: "points must not be empty"})
		return
	}

	report := detector.Analyze(payload.Points, payload.Config)
	status := http.StatusOK
	if report.HasSignals() {
		status = http.StatusUnprocessableEntity
	}
	writeJSON(writer, status, DetectResponse{
		Signals: report.HasSignals(),
		Report:  report,
	})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
