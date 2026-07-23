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

func TestFetchArticlesBetweenUsesBothBounds(t *testing.T) {
	feed := `<rss><channel>
  <item><title>Before</title><link>https://example.test/before</link><pubDate>Tue, 21 Jul 2026 08:59:00 +0000</pubDate></item>
  <item><title>Inside</title><link>https://example.test/inside</link><pubDate>Tue, 21 Jul 2026 09:30:00 +0000</pubDate></item>
  <item><title>After</title><link>https://example.test/after</link><pubDate>Tue, 21 Jul 2026 10:01:00 +0000</pubDate></item>
</channel></rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, feed)
	}))
	defer server.Close()

	start := time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC)
	articles, err := fetchArticlesBetween(context.Background(), server.Client(), server.URL, start, end)
	if err != nil {
		t.Fatalf("fetchArticlesBetween() error = %v", err)
	}
	if len(articles) != 1 || articles[0].Title != "Inside" {
		t.Fatalf("fetchArticlesBetween() = %#v, want only Inside", articles)
	}
}

func TestFetchArticlesBetweenRequiresIncreasingRange(t *testing.T) {
	now := time.Now()
	for name, end := range map[string]time.Time{
		"same time": now,
		"before":    now.Add(-time.Minute),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := fetchArticlesBetween(context.Background(), http.DefaultClient, "https://example.test/feed", now, end)
			if err == nil || !strings.Contains(err.Error(), "after") {
				t.Fatalf("fetchArticlesBetween() error = %v, want an invalid range error", err)
			}
		})
	}
}

func TestFetchArticlesBetweenRejectsUnsafeLinksAndCleansTitles(t *testing.T) {
	feed := `<rss><channel>
  <item><title>  Valid    headline  </title><link>https://example.test/story</link><pubDate>Tue, 21 Jul 2026 09:30:00 +0000</pubDate></item>
  <item><title>Unsafe link</title><link>javascript:alert(1)</link><pubDate>Tue, 21 Jul 2026 09:45:00 +0000</pubDate></item>
  <item><title>Terminal control</title><link>https://example.test/&#x9B;31mred</link><pubDate>Tue, 21 Jul 2026 09:46:00 +0000</pubDate></item>
  <item><title>Bidirectional formatting</title><link>https://example.test/&#x202E;gnp.exe</link><pubDate>Tue, 21 Jul 2026 09:47:00 +0000</pubDate></item>
</channel></rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, feed)
	}))
	defer server.Close()

	start := time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC)
	articles, err := fetchArticlesBetween(context.Background(), server.Client(), server.URL, start, end)
	if err != nil {
		t.Fatalf("fetchArticlesBetween() error = %v", err)
	}
	if len(articles) != 1 || articles[0].Title != "Valid headline" {
		t.Fatalf("fetchArticlesBetween() = %#v, want one cleaned, safe article", articles)
	}
}

func TestFetchArticlesBetweenReportsHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	now := time.Now()
	_, err := fetchArticlesBetween(context.Background(), server.Client(), server.URL, now.Add(-time.Hour), now)
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("fetchArticlesBetween() error = %v, want an HTTP 503 error", err)
	}
}

func TestFetchArticlesBetweenLimitsResponseSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, strings.Repeat("x", maxFeedSize+1))
	}))
	defer server.Close()

	now := time.Now()
	_, err := fetchArticlesBetween(context.Background(), server.Client(), server.URL, now.Add(-time.Hour), now)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("fetchArticlesBetween() error = %v, want a response size error", err)
	}
}

func TestDecodeFeed(t *testing.T) {
	items, err := decodeFeed(strings.NewReader(`<rss><channel>
  <item><title>Headline</title><link>https://example.test/story</link><pubDate>Tue, 21 Jul 2026 09:30:00 +0000</pubDate></item>
</channel></rss>`))
	if err != nil {
		t.Fatalf("decodeFeed() error = %v", err)
	}
	if len(items) != 1 || items[0].Title != "Headline" {
		t.Fatalf("decodeFeed() = %#v, want one decoded item", items)
	}
}

func TestDecodeFeedSupportsDeclaredLegacyCharset(t *testing.T) {
	feed := "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?><rss><channel>" +
		"<item><title>Caf\xe9 markets</title><link>https://example.test/story</link>" +
		"<pubDate>Tue, 21 Jul 2026 09:30:00 +0000</pubDate></item>" +
		"</channel></rss>"
	items, err := decodeFeed(strings.NewReader(feed))
	if err != nil {
		t.Fatalf("decodeFeed() error = %v", err)
	}
	if len(items) != 1 || items[0].Title != "Café markets" {
		t.Fatalf("decodeFeed() = %#v, want a decoded ISO-8859-1 title", items)
	}
}

func TestDecodeFeedRejectsMalformedXML(t *testing.T) {
	if _, err := decodeFeed(strings.NewReader(`<rss><channel>`)); err == nil {
		t.Fatal("decodeFeed() accepted malformed XML")
	}
}

func TestDecodeFeedRejectsNonRSSDocuments(t *testing.T) {
	for name, document := range map[string]string{
		"HTML response":     `<html><body>upstream error</body></html>`,
		"missing channel":   `<rss version="2.0"></rss>`,
		"multiple elements": `<rss><channel/></rss><html/>`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeFeed(strings.NewReader(document)); err == nil {
				t.Fatalf("decodeFeed() accepted %s", document)
			}
		})
	}
}
