package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTUIOptionsFromEnv(t *testing.T) {
	t.Setenv("ACCESSIBLE", "1")
	t.Setenv("NO_COLOR", "")

	options := tuiOptionsFromEnv()
	if !options.Accessible || options.Styled || options.FeedURL != defaultFeedURL {
		t.Fatalf("tuiOptionsFromEnv() = %#v, want accessible unstyled defaults", options)
	}
}

func TestRunTUIAcceptsInjectedInput(t *testing.T) {
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `<rss><channel></channel></rss>`)
	}))
	defer server.Close()

	var output strings.Builder
	err := runTUI(
		context.Background(),
		strings.NewReader("1\n"),
		&output,
		server.Client(),
		func() time.Time { return now },
		tuiOptions{Accessible: true, FeedURL: server.URL},
	)
	if err != nil {
		t.Fatalf("runTUI() error = %v", err)
	}
	if !strings.Contains(output.String(), "No articles found") {
		t.Fatalf("runTUI() output = %q", output.String())
	}
}

func TestChooseDateRangeAccessibleFlow(t *testing.T) {
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	var output strings.Builder
	selected, err := chooseDateRange(
		context.Background(),
		strings.NewReader("1\n"),
		&output,
		func() time.Time { return now },
		true,
	)
	if err != nil {
		t.Fatalf("chooseDateRange() error = %v", err)
	}
	if !selected.Start.Equal(now.Add(-30*time.Minute)) || !selected.End.Equal(now) {
		t.Fatalf("chooseDateRange() = %#v, want the last 30 minutes", selected)
	}
	if !strings.Contains(output.String(), "Last 30 minutes") {
		t.Fatalf("chooseDateRange() output = %q, want accessible options", output.String())
	}
}

func TestChooseDateRangeAccessibleCustomDefaults(t *testing.T) {
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	selected, err := chooseDateRange(
		context.Background(),
		strings.NewReader("7\n\n\n"),
		io.Discard,
		func() time.Time { return now },
		true,
	)
	if err != nil {
		t.Fatalf("chooseDateRange() error = %v", err)
	}
	if !selected.Start.Equal(now.Add(-time.Hour)) || !selected.End.Equal(now) {
		t.Fatalf("chooseDateRange() = %#v, want the default custom range", selected)
	}
}

func TestChooseDateRangeHonorsCanceledContext(t *testing.T) {
	for _, accessible := range []bool{false, true} {
		t.Run(fmt.Sprintf("accessible=%t", accessible), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, err := chooseDateRange(
				ctx,
				strings.NewReader(""),
				io.Discard,
				time.Now,
				accessible,
			)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("chooseDateRange() error = %v, want context.Canceled", err)
			}
		})
	}
}

func TestAccessiblePromptCancelsWhileReading(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reader := &blockingReader{started: make(chan struct{}), release: make(chan struct{})}
	done := make(chan error, 1)
	go func() {
		_, err := chooseDateRange(ctx, reader, io.Discard, time.Now, true)
		done <- err
	}()

	<-reader.started
	cancel()
	err := <-done
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("chooseDateRange() error = %v, want context.Canceled", err)
	}
}
