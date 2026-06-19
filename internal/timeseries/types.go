package timeseries

import "time"

type Point struct {
	Timestamp time.Time `json:"timestamp"`
	Series    string    `json:"series"`
	Value     float64   `json:"value"`
}

type Signal string

const (
	SignalNormal  Signal = "normal"
	SignalAnomaly Signal = "anomaly"
	SignalDrift   Signal = "drift"
)

type Detection struct {
	Timestamp   time.Time `json:"timestamp"`
	Series      string    `json:"series"`
	Value       float64   `json:"value"`
	Signal      Signal    `json:"signal"`
	Median      float64   `json:"median"`
	MAD         float64   `json:"mad"`
	RobustZ     float64   `json:"robustZ"`
	EWMA        float64   `json:"ewma"`
	DriftScore  float64   `json:"driftScore"`
	WindowCount int       `json:"windowCount"`
	Explanation string    `json:"explanation"`
	Warmup      bool      `json:"warmup"`
}

type Report struct {
	PointsScanned int         `json:"pointsScanned"`
	SeriesScanned int         `json:"seriesScanned"`
	Anomalies     int         `json:"anomalies"`
	Drifts        int         `json:"drifts"`
	Detections    []Detection `json:"detections"`
}

func (r Report) HasSignals() bool {
	return r.Anomalies > 0 || r.Drifts > 0
}
