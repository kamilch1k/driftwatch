package csvio

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/kamilch1k/driftwatch/internal/timeseries"
)

var requiredHeaders = []string{"timestamp", "series", "value"}

func ReadPoints(reader io.Reader) ([]timeseries.Point, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true

	headers, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("read csv header: %w", err)
	}
	indexes, err := headerIndexes(headers)
	if err != nil {
		return nil, err
	}

	var points []timeseries.Point
	line := 1
	for {
		line++
		record, err := csvReader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read csv line %d: %w", line, err)
		}
		point, err := parsePoint(record, indexes)
		if err != nil {
			return nil, fmt.Errorf("parse csv line %d: %w", line, err)
		}
		points = append(points, point)
	}
	return points, nil
}

func WriteDetections(writer io.Writer, detections []timeseries.Detection) error {
	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write([]string{
		"timestamp",
		"series",
		"value",
		"signal",
		"median",
		"mad",
		"robust_z",
		"ewma",
		"drift_score",
		"explanation",
	}); err != nil {
		return err
	}
	for _, detection := range detections {
		if err := csvWriter.Write([]string{
			detection.Timestamp.Format(time.RFC3339),
			detection.Series,
			formatFloat(detection.Value),
			string(detection.Signal),
			formatFloat(detection.Median),
			formatFloat(detection.MAD),
			formatFloat(detection.RobustZ),
			formatFloat(detection.EWMA),
			formatFloat(detection.DriftScore),
			detection.Explanation,
		}); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
}

func headerIndexes(headers []string) (map[string]int, error) {
	indexes := map[string]int{}
	for index, header := range headers {
		indexes[normalizeHeader(header)] = index
	}
	for _, required := range requiredHeaders {
		if _, ok := indexes[required]; !ok {
			return nil, fmt.Errorf("missing required csv header %q", required)
		}
	}
	return indexes, nil
}

func parsePoint(record []string, indexes map[string]int) (timeseries.Point, error) {
	get := func(name string) (string, error) {
		index := indexes[name]
		if index >= len(record) {
			return "", fmt.Errorf("missing %q field", name)
		}
		return strings.TrimSpace(record[index]), nil
	}

	timestampText, err := get("timestamp")
	if err != nil {
		return timeseries.Point{}, err
	}
	timestamp, err := time.Parse(time.RFC3339Nano, timestampText)
	if err != nil {
		return timeseries.Point{}, fmt.Errorf("invalid timestamp %q: %w", timestampText, err)
	}

	valueText, err := get("value")
	if err != nil {
		return timeseries.Point{}, err
	}
	value, err := strconv.ParseFloat(valueText, 64)
	if err != nil {
		return timeseries.Point{}, fmt.Errorf("invalid value %q: %w", valueText, err)
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return timeseries.Point{}, fmt.Errorf("value must be finite, got %q", valueText)
	}

	series, err := get("series")
	if err != nil {
		return timeseries.Point{}, err
	}

	return timeseries.Point{
		Timestamp: timestamp,
		Series:    series,
		Value:     value,
	}, nil
}

func normalizeHeader(header string) string {
	return strings.ToLower(strings.TrimSpace(header))
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
