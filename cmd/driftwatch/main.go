package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kamilch1k/driftwatch/internal/csvio"
	"github.com/kamilch1k/driftwatch/internal/detector"
	"github.com/kamilch1k/driftwatch/internal/timeseries"
)

const (
	exitOK       = 0
	exitUsage    = 64
	exitData     = 65
	exitSignaled = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("driftwatch", flag.ContinueOnError)
	flags.SetOutput(stderr)

	var config detector.Config
	input := flags.String("input", "", "CSV file with timestamp,series,value columns")
	output := flags.String("out", "", "optional path for a full JSON report")
	format := flags.String("format", "text", "output format: text or json")
	failOnSignal := flags.Bool("fail-on-signal", false, "exit with code 2 when anomalies or drift are detected")
	flags.IntVar(&config.WindowSize, "window", 24, "rolling baseline window size")
	flags.IntVar(&config.MinSamples, "min-samples", 8, "minimum samples before detection starts")
	flags.Float64Var(&config.MADThreshold, "mad-threshold", 6, "robust z-score threshold for anomalies")
	flags.Float64Var(&config.EWMAAlpha, "ewma-alpha", 0.25, "EWMA smoothing factor")
	flags.Float64Var(&config.PageHinkleyDelta, "page-hinkley-delta", 0.05, "Page-Hinkley tolerance")
	flags.Float64Var(&config.DriftThreshold, "drift-threshold", 35, "Page-Hinkley threshold for sustained drift")

	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	if strings.TrimSpace(*input) == "" {
		_, _ = fmt.Fprintln(stderr, "missing required -input path")
		return exitUsage
	}

	file, err := os.Open(*input)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "open input: %v\n", err)
		return exitData
	}
	defer file.Close()

	points, err := csvio.ReadPoints(file)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "read input: %v\n", err)
		return exitData
	}
	report := detector.Analyze(points, config)

	if strings.TrimSpace(*output) != "" {
		if err := writeReport(*output, report); err != nil {
			_, _ = fmt.Fprintf(stderr, "write report: %v\n", err)
			return exitData
		}
	}

	switch strings.ToLower(strings.TrimSpace(*format)) {
	case "text":
		printText(stdout, report)
	case "json":
		if err := json.NewEncoder(stdout).Encode(report); err != nil {
			_, _ = fmt.Fprintf(stderr, "write json: %v\n", err)
			return exitData
		}
	default:
		_, _ = fmt.Fprintf(stderr, "unknown -format %q\n", *format)
		return exitUsage
	}

	if *failOnSignal && report.HasSignals() {
		return exitSignaled
	}
	return exitOK
}

func writeReport(path string, report timeseries.Report) error {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func printText(writer io.Writer, report timeseries.Report) {
	_, _ = fmt.Fprintf(
		writer,
		"scanned=%d series=%d anomalies=%d drifts=%d\n",
		report.PointsScanned,
		report.SeriesScanned,
		report.Anomalies,
		report.Drifts,
	)
	for _, detection := range report.Detections {
		if detection.Signal == timeseries.SignalNormal {
			continue
		}
		_, _ = fmt.Fprintf(
			writer,
			"%s %-7s %-24s value=%8.2f median=%8.2f robust_z=%7.2f drift=%7.2f %s\n",
			detection.Timestamp.Format("2006-01-02T15:04:05Z"),
			detection.Signal,
			detection.Series,
			detection.Value,
			detection.Median,
			detection.RobustZ,
			detection.DriftScore,
			detection.Explanation,
		)
	}
}
