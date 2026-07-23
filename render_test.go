package main

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRenderArticleResultsPlainText(t *testing.T) {
	start := time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC)
	selectedRange := dateRange{Start: start, End: start.Add(time.Hour), Label: "Morning news"}
	articles := []article{{
		Title:     "Markets open higher",
		Link:      "https://example.test/markets",
		Published: start.Add(30 * time.Minute),
	}}

	var output strings.Builder
	if err := renderArticleResults(&output, selectedRange, articles, false); err != nil {
		t.Fatalf("renderArticleResults() error = %v", err)
	}
	got := output.String()
	for _, want := range []string{"INVESTINGLIVE NEWS", "Morning news", "1 article", "Markets open higher", "https://example.test/markets"} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderArticleResults() output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("plain output contains ANSI escapes: %q", got)
	}
}

func TestRenderArticleResultsLabelsBothDSTZones(t *testing.T) {
	location, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Fatalf("time.LoadLocation() error = %v", err)
	}
	selectedRange := dateRange{
		Start: time.Date(2026, 3, 29, 1, 30, 0, 0, location),
		End:   time.Date(2026, 3, 29, 3, 30, 0, 0, location),
		Label: "DST transition",
	}

	var output strings.Builder
	if err := renderArticleResults(&output, selectedRange, nil, false); err != nil {
		t.Fatalf("renderArticleResults() error = %v", err)
	}
	want := "2026-03-29 01:30 (CET) to 2026-03-29 03:30 (CEST)"
	if !strings.Contains(output.String(), want) {
		t.Fatalf("renderArticleResults() output = %q, want %q", output.String(), want)
	}
}

func TestRenderArticleResultsDisambiguatesRepeatedDSTHour(t *testing.T) {
	location, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Fatalf("time.LoadLocation() error = %v", err)
	}
	selectedRange := dateRange{
		Start: time.Date(2026, 10, 25, 1, 30, 0, 0, location),
		End:   time.Date(2026, 10, 25, 3, 30, 0, 0, location),
		Label: "DST transition",
	}
	articles := []article{
		{Title: "First 02:30", Link: "https://example.test/first", Published: time.Date(2026, 10, 25, 0, 30, 0, 0, time.UTC)},
		{Title: "Second 02:30", Link: "https://example.test/second", Published: time.Date(2026, 10, 25, 1, 30, 0, 0, time.UTC)},
	}

	var output strings.Builder
	if err := renderArticleResults(&output, selectedRange, articles, false); err != nil {
		t.Fatalf("renderArticleResults() error = %v", err)
	}
	for _, want := range []string{"Oct 25, 02:30 CEST", "Oct 25, 02:30 CET"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("renderArticleResults() output = %q, want %q", output.String(), want)
		}
	}
}

func TestRenderersReportWriteFailures(t *testing.T) {
	writeErr := errors.New("write failed")
	writer := failingWriter{err: writeErr}
	start := time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC)

	if err := renderPlainArticleResults(writer, nil, "1h"); !errors.Is(err, writeErr) {
		t.Fatalf("renderPlainArticleResults() error = %v, want %v", err, writeErr)
	}
	if err := renderArticleResults(writer, dateRange{Start: start, End: start.Add(time.Hour)}, nil, false); !errors.Is(err, writeErr) {
		t.Fatalf("renderArticleResults() error = %v, want %v", err, writeErr)
	}
}
