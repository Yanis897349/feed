package main

import (
	"strings"
	"testing"
	"time"
)

func TestArticleFromRSSItem(t *testing.T) {
	got, ok := articleFromRSSItem(rssItem{
		Title:   "  Market    update ",
		Link:    " https://example.test/story ",
		PubDate: "Tue, 21 Jul 2026 09:30:00 +0000",
	})
	if !ok || got.Title != "Market update" || got.Link != "https://example.test/story" {
		t.Fatalf("articleFromRSSItem() = %#v, %t", got, ok)
	}
}

func TestSanitizeTextRemovesTerminalControls(t *testing.T) {
	if got, want := sanitizeText("  Market\x1b[31m   update\n"), "Market [31m update"; got != want {
		t.Fatalf("sanitizeText() = %q, want %q", got, want)
	}
}

func TestSanitizeTextRemovesUnicodeFormatting(t *testing.T) {
	if got, want := sanitizeText("Market\u202ereversed\u202c update"), "Market reversed update"; got != want {
		t.Fatalf("sanitizeText() = %q, want %q", got, want)
	}
}

func TestParsePublishedDate(t *testing.T) {
	for _, value := range []string{
		"Tue, 21 Jul 2026 09:30:00 +0000",
		"Tue, 21 Jul 2026 09:30:00 GMT",
		"21 Jul 26 09:30 GMT",
		"Mon, 2 Jan 2006 15:04:05 +0000",
		"Mon, 2 Jan 2006 15:04:05 GMT",
		"2 Jan 06 15:04 GMT",
		"2 Jan 2006 15:04:05 GMT",
		"Mon, 2 Jan 2006 15:04 GMT",
		"2 Jan 2006 15:04 GMT",
		"Mon, 2 Jan 06 15:04:05 Z",
		"2026-07-21T09:30:00Z",
	} {
		if _, err := parsePublishedDate(value); err != nil {
			t.Fatalf("parsePublishedDate(%q) error = %v", value, err)
		}
	}
	if _, err := parsePublishedDate("July-ish"); err == nil {
		t.Fatal("parsePublishedDate() accepted an unsupported date")
	}
	if _, err := parsePublishedDate("Mon, 2 Jan 2006 15:04:05 XYZ"); err == nil {
		t.Fatal("parsePublishedDate() accepted an unknown named timezone")
	}
}

func TestParsePublishedDateUsesNamedZoneOffset(t *testing.T) {
	got, err := parsePublishedDate("Tue, 21 Jul 2026 09:30:00 EST")
	if err != nil {
		t.Fatalf("parsePublishedDate() error = %v", err)
	}
	want := time.Date(2026, 7, 21, 14, 30, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("parsePublishedDate() = %v, want %v", got, want)
	}
}

func TestValidArticleLink(t *testing.T) {
	for _, value := range []string{"javascript:alert(1)", "https://example.test/a b", "https://example.test/\u202Efile"} {
		if validArticleLink(value) {
			t.Fatalf("validArticleLink(%q) = true", value)
		}
	}
	if !validArticleLink("https://example.test/story") {
		t.Fatal("validArticleLink() rejected an HTTPS URL")
	}
}

func TestParsePublishedDateRejectsInvalidNumericZone(t *testing.T) {
	for _, value := range []string{"+0A00", "+000", strings.Repeat("0", 5)} {
		if _, err := parsePublishedDate("Tue, 21 Jul 2026 09:30:00 " + value); err == nil {
			t.Fatalf("parsePublishedDate() accepted zone %q", value)
		}
	}
}
