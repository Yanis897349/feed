package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/muesli/cancelreader"
)

func chooseAccessibleDateRange(
	ctx context.Context,
	input io.Reader,
	output io.Writer,
	clock func() time.Time,
) (dateRange, error) {
	reader, closeReader, err := cancellableLineReader(ctx, input)
	if err != nil {
		return dateRange{}, fmt.Errorf("prepare input: %w", err)
	}
	defer closeReader()

	if _, err := fmt.Fprintln(output, "InvestingLive article range"); err != nil {
		return dateRange{}, fmt.Errorf("write prompt: %w", err)
	}
	if _, err := fmt.Fprintln(output, "Choose how far back to look in the live news feed."); err != nil {
		return dateRange{}, fmt.Errorf("write prompt: %w", err)
	}
	for index, preset := range rangePresets {
		if _, err := fmt.Fprintf(output, "%d. %s\n", index+1, preset.MenuLabel); err != nil {
			return dateRange{}, fmt.Errorf("write prompt: %w", err)
		}
	}

	defaultChoice := defaultRangeChoice()
	for {
		if _, err := fmt.Fprintf(output, "Enter a number between 1 and %d [%d]: ", len(rangePresets), defaultChoice); err != nil {
			return dateRange{}, fmt.Errorf("write prompt: %w", err)
		}
		value, err := readLine(ctx, reader)
		if err != nil {
			return dateRange{}, err
		}
		value = strings.TrimSpace(value)
		choice := defaultChoice
		if value != "" {
			choice, err = strconv.Atoi(value)
		}
		if err != nil || choice < 1 || choice > len(rangePresets) {
			if _, writeErr := fmt.Fprintln(output, "Enter a valid option number."); writeErr != nil {
				return dateRange{}, fmt.Errorf("write prompt: %w", writeErr)
			}
			continue
		}

		preset := rangePresets[choice-1]
		if preset.Kind == rangeCustom {
			return chooseAccessibleCustomRange(ctx, reader, output, clock())
		}
		return presetRangeAt(preset.Value, clock)
	}
}

func defaultRangeChoice() int {
	for index, preset := range rangePresets {
		if preset.Value == "1h" {
			return index + 1
		}
	}
	return 1
}

func chooseAccessibleCustomRange(
	ctx context.Context,
	reader *bufio.Reader,
	output io.Writer,
	now time.Time,
) (dateRange, error) {
	defaultStart, defaultEnd := customRangeDefaults(now)
	location := now.Location()

	startValue, err := promptAccessibleValue(
		ctx,
		reader,
		output,
		"Start (local time)",
		defaultStart,
		validateDateTime(location),
	)
	if err != nil {
		return dateRange{}, err
	}
	endValue, err := promptAccessibleValue(
		ctx,
		reader,
		output,
		"End (local time)",
		defaultEnd,
		validateRangeEnd(&startValue, location),
	)
	if err != nil {
		return dateRange{}, err
	}

	return parseCustomRange(startValue, endValue, location)
}

func promptAccessibleValue(
	ctx context.Context,
	reader *bufio.Reader,
	output io.Writer,
	label string,
	defaultValue string,
	validate func(string) error,
) (string, error) {
	for {
		if _, err := fmt.Fprintf(output, "%s [%s]: ", label, defaultValue); err != nil {
			return "", fmt.Errorf("write prompt: %w", err)
		}
		value, err := readLine(ctx, reader)
		if err != nil {
			return "", err
		}
		value = strings.TrimSpace(value)
		if value == "" {
			value = defaultValue
		}
		if err := validate(value); err != nil {
			if _, writeErr := fmt.Fprintln(output, err); writeErr != nil {
				return "", fmt.Errorf("write prompt: %w", writeErr)
			}
			continue
		}
		return value, nil
	}
}

func cancellableLineReader(ctx context.Context, input io.Reader) (*bufio.Reader, func(), error) {
	reader, ok := input.(cancelreader.CancelReader)
	if !ok {
		var err error
		reader, err = cancelreader.NewReader(input)
		if err != nil {
			return nil, nil, err
		}
	}

	done := make(chan struct{})
	watcherDone := make(chan struct{})
	go func() {
		defer close(watcherDone)
		select {
		case <-ctx.Done():
			reader.Cancel()
		case <-done:
		}
	}()

	closeReader := func() {
		close(done)
		<-watcherDone
		_ = reader.Close()
	}
	return bufio.NewReader(reader), closeReader, nil
}

func readLine(ctx context.Context, reader *bufio.Reader) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	value, err := reader.ReadString('\n')
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if err != nil && !(errors.Is(err, io.EOF) && value != "") {
		return "", err
	}
	return strings.TrimRight(value, "\r\n"), nil
}
