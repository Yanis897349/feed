package main

import (
	"strings"
	"testing"
	"time"
)

func TestParseLocalDateTime(t *testing.T) {
	location := time.FixedZone("test", 2*60*60)
	got, err := parseLocalDateTime("2026-07-21 09:30", location)
	if err != nil {
		t.Fatalf("parseLocalDateTime() error = %v", err)
	}
	if got.Hour() != 9 || got.Location() != location {
		t.Fatalf("parseLocalDateTime() = %v, want 09:30 in supplied location", got)
	}
}

func TestParseLocalDateTimeRejectsDSTTransitionTimes(t *testing.T) {
	location, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Fatalf("time.LoadLocation() error = %v", err)
	}
	for name, value := range map[string]string{
		"nonexistent": "2026-03-29 02:30",
		"ambiguous":   "2026-10-25 02:30",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseLocalDateTime(value, location); err == nil || !strings.Contains(err.Error(), "daylight-saving") {
				t.Fatalf("parseLocalDateTime(%q) error = %v, want a daylight-saving error", value, err)
			}
		})
	}
}

func TestParseCustomRangeRequiresIncreasingRange(t *testing.T) {
	location := time.UTC
	if _, err := parseCustomRange("2026-07-21 10:00", "2026-07-21 09:00", location); err == nil {
		t.Fatal("parseCustomRange() accepted a decreasing range")
	}
}

func TestPresetRange(t *testing.T) {
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)

	today, err := presetRange("today", now)
	if err != nil {
		t.Fatalf("presetRange(today) error = %v", err)
	}
	if got, want := today.Start.Hour(), 0; got != want {
		t.Fatalf("presetRange(today) start hour = %d, want %d", got, want)
	}

	recent, err := presetRange("6h", now)
	if err != nil {
		t.Fatalf("presetRange(6h) error = %v", err)
	}
	if got, want := recent.Start, now.Add(-6*time.Hour); !got.Equal(want) {
		t.Fatalf("presetRange(6h) start = %v, want %v", got, want)
	}

	if _, err := presetRange("later", now); err == nil {
		t.Fatal("presetRange() accepted an invalid preset")
	}
}

func TestPresetRangeAtUsesCurrentClockValue(t *testing.T) {
	launchTime := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	selectionTime := launchTime.Add(45 * time.Minute)

	selected, err := presetRangeAt("30m", func() time.Time { return selectionTime })
	if err != nil {
		t.Fatalf("presetRangeAt() error = %v", err)
	}
	if !selected.End.Equal(selectionTime) || !selected.Start.Equal(selectionTime.Add(-30*time.Minute)) {
		t.Fatalf("presetRangeAt() = %#v, want a range relative to %v", selected, selectionTime)
	}
}
