package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"charm.land/huh/v2"
	"charm.land/huh/v2/spinner"
)

type tuiOptions struct {
	Accessible bool
	Styled     bool
	FeedURL    string
}

func tuiOptionsFromEnv() tuiOptions {
	accessible := os.Getenv("ACCESSIBLE") != ""
	return tuiOptions{
		Accessible: accessible,
		Styled:     !accessible && os.Getenv("NO_COLOR") == "",
		FeedURL:    defaultFeedURL,
	}
}

func runTUI(
	ctx context.Context,
	input io.Reader,
	output io.Writer,
	client *http.Client,
	clock func() time.Time,
	options tuiOptions,
) error {
	selectedRange, err := chooseDateRange(ctx, input, output, clock, options.Accessible)
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil
		}
		return err
	}

	var articles []article
	fetch := spinner.New().
		Title(" Fetching the latest articles...").
		Type(spinner.MiniDot).
		Context(ctx).
		WithOutput(output).
		WithAccessible(options.Accessible).
		ActionWithErr(func(actionCtx context.Context) error {
			var fetchErr error
			articles, fetchErr = fetchArticlesBetween(actionCtx, client, options.FeedURL, selectedRange.Start, selectedRange.End)
			return fetchErr
		})
	if err := fetch.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}

	return renderArticleResults(output, selectedRange, articles, options.Styled)
}

func chooseDateRange(
	ctx context.Context,
	input io.Reader,
	output io.Writer,
	clock func() time.Time,
	accessible bool,
) (dateRange, error) {
	if accessible {
		return chooseAccessibleDateRange(ctx, input, output, clock)
	}

	presetValue := "1h"
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("InvestingLive article range").
				Description("Choose how far back to look in the live news feed.").
				Options(presetOptions()...).
				Value(&presetValue),
		),
	).WithInput(input).WithOutput(output)

	if err := runInteractiveForm(ctx, form); err != nil {
		return dateRange{}, err
	}

	preset, ok := findRangePreset(presetValue)
	if !ok {
		return dateRange{}, invalidRangePresetError(presetValue)
	}
	if preset.Kind != rangeCustom {
		return presetRangeAt(presetValue, clock)
	}

	return chooseCustomRange(ctx, input, output, clock())
}

func runInteractiveForm(ctx context.Context, form *huh.Form) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := form.RunWithContext(ctx)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

func presetOptions() []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(rangePresets))
	for _, preset := range rangePresets {
		options = append(options, huh.NewOption(preset.MenuLabel, preset.Value))
	}
	return options
}

func chooseCustomRange(ctx context.Context, input io.Reader, output io.Writer, now time.Time) (dateRange, error) {
	startValue, endValue := customRangeDefaults(now)
	location := now.Location()

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Start (local time)").
				Description("Use YYYY-MM-DD HH:MM in your local timezone.").
				Value(&startValue).
				Validate(validateDateTime(location)),
			huh.NewInput().
				Title("End (local time)").
				Description("Use YYYY-MM-DD HH:MM in your local timezone.").
				Value(&endValue).
				Validate(validateRangeEnd(&startValue, location)),
		),
	).WithInput(input).WithOutput(output)

	if err := runInteractiveForm(ctx, form); err != nil {
		return dateRange{}, err
	}
	return parseCustomRange(startValue, endValue, location)
}
