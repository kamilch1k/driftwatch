package detector

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/kamilch1k/driftwatch/internal/timeseries"
)

type Config struct {
	WindowSize       int     `json:"windowSize"`
	MinSamples       int     `json:"minSamples"`
	MADThreshold     float64 `json:"madThreshold"`
	EWMAAlpha        float64 `json:"ewmaAlpha"`
	PageHinkleyDelta float64 `json:"pageHinkleyDelta"`
	DriftThreshold   float64 `json:"driftThreshold"`
}

func DefaultConfig() Config {
	return Config{
		WindowSize:       24,
		MinSamples:       8,
		MADThreshold:     6,
		EWMAAlpha:        0.25,
		PageHinkleyDelta: 0.05,
		DriftThreshold:   35,
	}
}

func (c Config) WithDefaults() Config {
	defaults := DefaultConfig()
	if c.WindowSize <= 0 {
		c.WindowSize = defaults.WindowSize
	}
	if c.MinSamples <= 0 {
		c.MinSamples = defaults.MinSamples
	}
	if c.MADThreshold <= 0 {
		c.MADThreshold = defaults.MADThreshold
	}
	if c.EWMAAlpha <= 0 || c.EWMAAlpha > 1 {
		c.EWMAAlpha = defaults.EWMAAlpha
	}
	if c.PageHinkleyDelta < 0 {
		c.PageHinkleyDelta = defaults.PageHinkleyDelta
	}
	if c.DriftThreshold <= 0 {
		c.DriftThreshold = defaults.DriftThreshold
	}
	return c
}

type Detector struct {
	config Config
	series map[string]*seriesState
}

func New(config Config) *Detector {
	return &Detector{
		config: config.WithDefaults(),
		series: map[string]*seriesState{},
	}
}

func Analyze(points []timeseries.Point, config Config) timeseries.Report {
	ordered := slices.Clone(points)
	slices.SortFunc(ordered, func(a, b timeseries.Point) int {
		if c := cmp.Compare(a.Series, b.Series); c != 0 {
			return c
		}
		return a.Timestamp.Compare(b.Timestamp)
	})

	detector := New(config)
	report := timeseries.Report{PointsScanned: len(points)}
	seenSeries := map[string]struct{}{}
	for _, point := range ordered {
		seenSeries[normalizeSeries(point.Series)] = struct{}{}
		detection := detector.Add(point)
		report.Detections = append(report.Detections, detection)
		switch detection.Signal {
		case timeseries.SignalAnomaly:
			report.Anomalies++
		case timeseries.SignalDrift:
			report.Drifts++
		}
	}
	report.SeriesScanned = len(seenSeries)
	return report
}

func (d *Detector) Add(point timeseries.Point) timeseries.Detection {
	series := normalizeSeries(point.Series)
	state := d.series[series]
	if state == nil {
		state = &seriesState{}
		d.series[series] = state
	}

	detection := state.detect(point, series, d.config)
	state.observe(point.Value, d.config)
	return detection
}

type seriesState struct {
	window  []float64
	ewma    float64
	hasEWMA bool

	phCount      int
	phMean       float64
	phCumulative float64
	phMin        float64
}

func (s *seriesState) detect(point timeseries.Point, series string, config Config) timeseries.Detection {
	median, mad := medianMAD(s.window)
	robustZ := 0.0
	warmup := len(s.window) < config.MinSamples
	if !warmup {
		scale := math.Max(mad*1.4826, 1e-9)
		robustZ = math.Abs(point.Value-median) / scale
	}

	driftScore := s.nextDriftScore(point.Value, config)
	ewma := point.Value
	if s.hasEWMA {
		ewma = config.EWMAAlpha*point.Value + (1-config.EWMAAlpha)*s.ewma
	}

	detection := timeseries.Detection{
		Timestamp:   point.Timestamp,
		Series:      series,
		Value:       point.Value,
		Signal:      timeseries.SignalNormal,
		Median:      median,
		MAD:         mad,
		RobustZ:     robustZ,
		EWMA:        ewma,
		DriftScore:  driftScore,
		WindowCount: len(s.window),
		Warmup:      warmup,
		Explanation: "within rolling baseline",
	}

	if !warmup && robustZ >= config.MADThreshold {
		detection.Signal = timeseries.SignalAnomaly
		detection.Explanation = fmt.Sprintf("robust z-score %.2f exceeded threshold %.2f", robustZ, config.MADThreshold)
	}
	if !warmup && driftScore >= config.DriftThreshold {
		detection.Signal = timeseries.SignalDrift
		detection.Explanation = fmt.Sprintf("Page-Hinkley drift score %.2f exceeded threshold %.2f", driftScore, config.DriftThreshold)
	}
	if warmup {
		detection.Explanation = fmt.Sprintf("warming up: %d/%d baseline samples", len(s.window), config.MinSamples)
	}

	return detection
}

func (s *seriesState) observe(value float64, config Config) {
	if s.hasEWMA {
		s.ewma = config.EWMAAlpha*value + (1-config.EWMAAlpha)*s.ewma
	} else {
		s.ewma = value
		s.hasEWMA = true
	}

	s.window = append(s.window, value)
	if len(s.window) > config.WindowSize {
		s.window = s.window[len(s.window)-config.WindowSize:]
	}

	s.updateDrift(value, config)
}

func (s *seriesState) nextDriftScore(value float64, config Config) float64 {
	count := s.phCount + 1
	mean := s.phMean
	if count == 1 {
		mean = value
	} else {
		mean += (value - mean) / float64(count)
	}
	cumulative := s.phCumulative + value - mean - config.PageHinkleyDelta
	minimum := math.Min(s.phMin, cumulative)
	return cumulative - minimum
}

func (s *seriesState) updateDrift(value float64, config Config) {
	s.phCount++
	if s.phCount == 1 {
		s.phMean = value
		s.phCumulative = 0
		s.phMin = 0
		return
	}

	s.phMean += (value - s.phMean) / float64(s.phCount)
	s.phCumulative += value - s.phMean - config.PageHinkleyDelta
	s.phMin = math.Min(s.phMin, s.phCumulative)
	if s.phCumulative-s.phMin >= config.DriftThreshold {
		s.phCount = 1
		s.phMean = value
		s.phCumulative = 0
		s.phMin = 0
	}
}

func medianMAD(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}

	sorted := slices.Clone(values)
	slices.Sort(sorted)
	median := percentile(sorted, 0.5)

	deviations := make([]float64, len(sorted))
	for i, value := range sorted {
		deviations[i] = math.Abs(value - median)
	}
	slices.Sort(deviations)
	return median, percentile(deviations, 0.5)
}

func percentile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}

	position := q * float64(len(sorted)-1)
	lower := int(math.Floor(position))
	upper := int(math.Ceil(position))
	if lower == upper {
		return sorted[lower]
	}
	weight := position - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

func normalizeSeries(series string) string {
	series = strings.TrimSpace(series)
	if series == "" {
		return "default"
	}
	return series
}
