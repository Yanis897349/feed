package main

import (
	"fmt"
	"io"
	"time"

	"charm.land/lipgloss/v2"
)

var (
	accentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
	boldStyle   = lipgloss.NewStyle().Bold(true)
	mutedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#737373"))
	countStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#2E9B65")).Bold(true)
)

func renderPlainArticleResults(output io.Writer, articles []article, windowLabel string) error {
	if len(articles) == 0 {
		if _, err := fmt.Fprintf(output, "No articles found in the last %s.\n", windowLabel); err != nil {
			return fmt.Errorf("write results: %w", err)
		}
		return nil
	}

	for _, article := range articles {
		if _, err := fmt.Fprintf(
			output,
			"Title:     %s\nPublished: %s\nLink:      %s\n\n",
			article.Title,
			article.Published.UTC().Format(time.RFC3339),
			article.Link,
		); err != nil {
			return fmt.Errorf("write results: %w", err)
		}
	}
	return nil
}

func renderArticleResults(output io.Writer, selectedRange dateRange, articles []article, styled bool) error {
	period := fmt.Sprintf(
		"%s to %s",
		formatRangeEndpoint(selectedRange.Start),
		formatRangeEndpoint(selectedRange.End),
	)
	count := fmt.Sprintf("%d article%s", len(articles), pluralSuffix(len(articles)))

	heading := "INVESTINGLIVE NEWS"
	label := selectedRange.Label
	if styled {
		heading = accentStyle.Render(heading)
		label = boldStyle.Render(label)
		period = mutedStyle.Render(period)
		count = countStyle.Render(count)
	}

	if _, err := fmt.Fprintf(output, "\n%s\n%s\n%s  |  %s\n\n", heading, label, period, count); err != nil {
		return fmt.Errorf("write results: %w", err)
	}

	if len(articles) == 0 {
		if _, err := fmt.Fprintln(output, "No articles found in this range. Try a wider date range."); err != nil {
			return fmt.Errorf("write results: %w", err)
		}
		return nil
	}

	for index, article := range articles {
		title := article.Title
		metadata := fmt.Sprintf(
			"%s  %s",
			article.Published.In(selectedRange.Start.Location()).Format("Jan 02, 15:04 MST"),
			article.Link,
		)
		marker := fmt.Sprintf("%2d", index+1)
		if styled {
			marker = accentStyle.Render(marker)
			title = boldStyle.Render(title)
			metadata = mutedStyle.Render(metadata)
		}
		if _, err := fmt.Fprintf(output, "%s  %s\n    %s\n", marker, title, metadata); err != nil {
			return fmt.Errorf("write results: %w", err)
		}
		if index < len(articles)-1 {
			if _, err := fmt.Fprintln(output); err != nil {
				return fmt.Errorf("write results: %w", err)
			}
		}
	}
	if _, err := fmt.Fprintln(output); err != nil {
		return fmt.Errorf("write results: %w", err)
	}
	return nil
}

func formatRangeEndpoint(value time.Time) string {
	zone, _ := value.Zone()
	return fmt.Sprintf("%s (%s)", value.Format(dateTimeLayout), zone)
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
