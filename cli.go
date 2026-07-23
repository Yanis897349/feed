package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var friendlyDurationPattern = regexp.MustCompile(`(?i)^\s*(\d+(?:[.,]\d+)?)\s*(m|min|mins|minute|minutes|h|hr|hrs|hour|hours)\s*$`)

func run(ctx context.Context, args []string, stdout, stderr io.Writer, client *http.Client, clock func() time.Time) error {
	flags := flag.NewFlagSet(appName, flag.ContinueOnError)
	flags.SetOutput(stderr)

	last := flags.String("last", "1h", "time window, such as 30m, 2h, or 1.5hours")
	feedURL := flags.String("feed", defaultFeedURL, "RSS feed URL")
	showVersion := flags.Bool("version", false, "print version and exit")
	flags.Usage = func() {
		fmt.Fprintf(stderr, "Usage: %s [options] [window]\n", appName)
		fmt.Fprintln(stderr, "\nExamples:")
		fmt.Fprintf(stderr, "  %s 30m\n", appName)
		fmt.Fprintf(stderr, "  %s -last 2h\n", appName)
		fmt.Fprintln(stderr, "\nOptions:")
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *showVersion {
		if _, err := fmt.Fprintf(stdout, "%s %s\n", appName, version); err != nil {
			return fmt.Errorf("write version: %w", err)
		}
		return nil
	}

	positionals := flags.Args()
	if len(positionals) > 1 {
		flags.Usage()
		return errors.New("provide only one time window")
	}
	if len(positionals) == 1 {
		lastWasSet := false
		flags.Visit(func(f *flag.Flag) {
			if f.Name == "last" {
				lastWasSet = true
			}
		})
		if lastWasSet {
			return errors.New("provide the window either positionally or with -last, not both")
		}
		*last = positionals[0]
	}

	window, err := parseDuration(*last)
	if err != nil {
		return err
	}

	articles, err := fetchArticles(ctx, client, *feedURL, clock(), window)
	if err != nil {
		return err
	}

	return renderPlainArticleResults(stdout, articles, *last)
}

func parseDuration(value string) (time.Duration, error) {
	match := friendlyDurationPattern.FindStringSubmatch(value)
	if match == nil {
		return 0, fmt.Errorf("invalid time window %q; try 30m, 90min, 1h, or 2.5hours", value)
	}

	amount, err := strconv.ParseFloat(strings.ReplaceAll(match[1], ",", "."), 64)
	if err != nil || amount <= 0 {
		return 0, errors.New("the time window must be greater than zero")
	}

	unit := time.Hour
	if strings.HasPrefix(strings.ToLower(match[2]), "m") {
		unit = time.Minute
	}
	nanoseconds := amount * float64(unit)
	if nanoseconds >= float64(math.MaxInt64) {
		return 0, errors.New("the time window is too large")
	}

	duration := time.Duration(nanoseconds)
	if duration <= 0 {
		return 0, errors.New("the time window must be greater than zero")
	}
	return duration, nil
}
