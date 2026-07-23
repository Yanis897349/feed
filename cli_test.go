package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := map[string]time.Duration{
		"30m":      30 * time.Minute,
		"90min":    90 * time.Minute,
		"2h":       2 * time.Hour,
		"1.5hours": 90 * time.Minute,
		"1,5 hr":   90 * time.Minute,
	}

	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			got, err := parseDuration(input)
			if err != nil {
				t.Fatalf("parseDuration() error = %v", err)
			}
			if got != want {
				t.Fatalf("parseDuration() = %v, want %v", got, want)
			}
		})
	}
}

func TestParseDurationRejectsInvalidValues(t *testing.T) {
	for _, input := range []string{
		"",
		"0m",
		"-1h",
		"tomorrow",
		"0.00000000001m",
		"2562047.7880152157h",
		"999999999999999999999999h",
	} {
		t.Run(input, func(t *testing.T) {
			if _, err := parseDuration(input); err == nil {
				t.Fatalf("parseDuration(%q) succeeded, want an error", input)
			}
		})
	}
}

func TestRunVersionDoesNotFetch(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("version flag unexpectedly made an HTTP request")
		return nil, nil
	})}

	var output strings.Builder
	if err := run(context.Background(), []string{"--version"}, &output, io.Discard, client, time.Now); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if got, want := output.String(), appName+" "+version+"\n"; got != want {
		t.Fatalf("run() output = %q, want %q", got, want)
	}
}

func TestRunHelpDoesNotFetchOrReturnAnError(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("help flag unexpectedly made an HTTP request")
		return nil, nil
	})}

	var stderr strings.Builder
	if err := run(context.Background(), []string{"-help"}, io.Discard, &stderr, client, time.Now); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("run() help output = %q, want usage text", stderr.String())
	}
}

func TestRunRejectsConflictingWindows(t *testing.T) {
	err := run(
		context.Background(),
		[]string{"-last", "1h", "30m"},
		io.Discard,
		io.Discard,
		http.DefaultClient,
		time.Now,
	)
	if err == nil || !strings.Contains(err.Error(), "either positionally") {
		t.Fatalf("run() error = %v, want a conflicting window error", err)
	}
}

func TestRunReportsNoMatches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `<rss><channel></channel></rss>`)
	}))
	defer server.Close()

	var output strings.Builder
	err := run(context.Background(), []string{"-feed", server.URL}, &output, io.Discard, server.Client(), time.Now)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !strings.Contains(output.String(), "No articles found") {
		t.Fatalf("run() output = %q, want an empty result message", output.String())
	}
}

func TestRunFiltersAndSortsArticles(t *testing.T) {
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	feed := `<?xml version="1.0"?>
<rss><channel>
  <item><title>Older match</title><link>https://example.test/older</link><pubDate>Tue, 21 Jul 2026 11:15:00 +0000</pubDate></item>
  <item><title>Too old</title><link>https://example.test/old</link><pubDate>Tue, 21 Jul 2026 09:00:00 +0000</pubDate></item>
  <item><title>Newest &amp; decoded</title><link>https://example.test/new</link><pubDate>Tue, 21 Jul 2026 11:50:00 +0000</pubDate></item>
</channel></rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = io.WriteString(w, feed)
	}))
	defer server.Close()

	var output strings.Builder
	err := run(
		context.Background(),
		[]string{"-feed", server.URL, "1h"},
		&output,
		io.Discard,
		server.Client(),
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := output.String()
	if strings.Contains(got, "Too old") {
		t.Fatalf("output contains an article outside the window: %s", got)
	}
	if strings.Index(got, "Newest & decoded") > strings.Index(got, "Older match") {
		t.Fatalf("articles are not newest first: %s", got)
	}
	if !strings.Contains(got, "Published: 2026-07-21T11:50:00Z") {
		t.Fatalf("output does not contain an RFC3339 UTC publication time: %s", got)
	}
}
