package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const dateTimeLayout = "2006-01-02 15:04"

type dateRange struct {
	Start time.Time
	End   time.Time
	Label string
}

type rangeKind uint8

const (
	rangeDuration rangeKind = iota
	rangeToday
	rangeCustom
)

type rangePreset struct {
	MenuLabel string
	Value     string
	Kind      rangeKind
	Duration  time.Duration
}

var rangePresets = [...]rangePreset{
	{MenuLabel: "Last 30 minutes", Value: "30m", Kind: rangeDuration, Duration: 30 * time.Minute},
	{MenuLabel: "Last hour", Value: "1h", Kind: rangeDuration, Duration: time.Hour},
	{MenuLabel: "Last 6 hours", Value: "6h", Kind: rangeDuration, Duration: 6 * time.Hour},
	{MenuLabel: "Last 12 hours", Value: "12h", Kind: rangeDuration, Duration: 12 * time.Hour},
	{MenuLabel: "Last 24 hours", Value: "24h", Kind: rangeDuration, Duration: 24 * time.Hour},
	{MenuLabel: "Today", Value: "today", Kind: rangeToday},
	{MenuLabel: "Custom date and time range", Value: "custom", Kind: rangeCustom},
}

func findRangePreset(value string) (rangePreset, bool) {
	for _, preset := range rangePresets {
		if preset.Value == value {
			return preset, true
		}
	}
	return rangePreset{}, false
}

func invalidRangePresetError(value string) error {
	return fmt.Errorf("invalid range preset %q", value)
}

func presetRangeAt(value string, clock func() time.Time) (dateRange, error) {
	return presetRange(value, clock())
}

func presetRange(value string, now time.Time) (dateRange, error) {
	preset, ok := findRangePreset(value)
	if !ok || preset.Kind == rangeCustom {
		return dateRange{}, invalidRangePresetError(value)
	}
	if preset.Kind == rangeToday {
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return dateRange{Start: start, End: now, Label: "Today's articles"}, nil
	}

	return dateRange{
		Start: now.Add(-preset.Duration),
		End:   now,
		Label: "Articles from the last " + friendlyRangeLabel(preset.Duration),
	}, nil
}

func customRangeDefaults(now time.Time) (string, string) {
	return now.Add(-time.Hour).Format(dateTimeLayout), now.Format(dateTimeLayout)
}

func parseCustomRange(startValue, endValue string, location *time.Location) (dateRange, error) {
	start, err := parseLocalDateTime(startValue, location)
	if err != nil {
		return dateRange{}, err
	}
	end, err := parseLocalDateTime(endValue, location)
	if err != nil {
		return dateRange{}, err
	}
	if !end.After(start) {
		return dateRange{}, errors.New("end must be after start")
	}
	return dateRange{Start: start, End: end, Label: "Articles in custom range"}, nil
}

func validateDateTime(location *time.Location) func(string) error {
	return func(value string) error {
		_, err := parseLocalDateTime(value, location)
		return err
	}
}

func validateRangeEnd(startValue *string, location *time.Location) func(string) error {
	return func(endValue string) error {
		_, err := parseCustomRange(*startValue, endValue, location)
		return err
	}
}

func parseLocalDateTime(value string, location *time.Location) (time.Time, error) {
	value = strings.TrimSpace(value)
	parsed, err := time.ParseInLocation(dateTimeLayout, value, location)
	if err != nil {
		return time.Time{}, errors.New("use YYYY-MM-DD HH:MM, for example 2026-07-21 09:30")
	}
	if parsed.Format(dateTimeLayout) != value {
		return time.Time{}, errors.New("that local time does not exist because of a daylight-saving transition")
	}
	if ambiguousLocalDateTime(parsed, value, location) {
		return time.Time{}, errors.New("that local time occurs twice because of a daylight-saving transition; choose an unambiguous time")
	}
	return parsed, nil
}

func ambiguousLocalDateTime(parsed time.Time, value string, location *time.Location) bool {
	offsets := make(map[int]struct{}, 3)
	for _, candidate := range []time.Time{parsed.Add(-24 * time.Hour), parsed, parsed.Add(24 * time.Hour)} {
		_, offset := candidate.Zone()
		offsets[offset] = struct{}{}
	}

	wallTime := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), parsed.Hour(), parsed.Minute(), 0, 0, time.UTC)
	matches := make(map[int64]struct{}, len(offsets))
	for offset := range offsets {
		candidate := wallTime.Add(-time.Duration(offset) * time.Second).In(location)
		if candidate.Format(dateTimeLayout) == value {
			matches[candidate.Unix()] = struct{}{}
		}
	}
	return len(matches) > 1
}

func friendlyRangeLabel(duration time.Duration) string {
	if duration < time.Hour {
		return fmt.Sprintf("%.0f minutes", duration.Minutes())
	}
	if duration == time.Hour {
		return "hour"
	}
	return fmt.Sprintf("%.0f hours", duration.Hours())
}
