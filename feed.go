package main

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html/charset"
)

type rssFeed struct {
	XMLName xml.Name    `xml:"rss"`
	Channel *rssChannel `xml:"channel"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	PubDate string `xml:"pubDate"`
}

func fetchArticles(ctx context.Context, client *http.Client, feedURL string, now time.Time, window time.Duration) ([]article, error) {
	return fetchArticlesBetween(ctx, client, feedURL, now.Add(-window), now)
}

func fetchArticlesBetween(ctx context.Context, client *http.Client, feedURL string, start, end time.Time) ([]article, error) {
	if !end.After(start) {
		return nil, errors.New("the end of the date range must be after its start")
	}

	items, err := fetchFeedItems(ctx, client, feedURL)
	if err != nil {
		return nil, err
	}
	return filterArticlesBetween(items, start, end), nil
}

func fetchFeedItems(ctx context.Context, client *http.Client, feedURL string) ([]rssItem, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create feed request: %w", err)
	}
	request.Header.Set("Accept", "application/rss+xml, application/xml;q=0.9")
	request.Header.Set("User-Agent", fmt.Sprintf("%s/%s", appName, version))

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch feed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("fetch feed: server returned %s", response.Status)
	}

	items, err := decodeFeed(response.Body)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func decodeFeed(reader io.Reader) ([]rssItem, error) {
	contents, err := io.ReadAll(io.LimitReader(reader, maxFeedSize+1))
	if err != nil {
		return nil, fmt.Errorf("read RSS feed: %w", err)
	}
	if len(contents) > maxFeedSize {
		return nil, fmt.Errorf("parse RSS feed: response exceeds %d MiB", maxFeedSize>>20)
	}

	var feed rssFeed
	decoder := xml.NewDecoder(bytes.NewReader(contents))
	decoder.CharsetReader = charset.NewReaderLabel
	if err := decoder.Decode(&feed); err != nil {
		return nil, fmt.Errorf("parse RSS feed: %w", err)
	}
	if err := requireXMLDocumentEnd(decoder); err != nil {
		return nil, err
	}
	if feed.Channel == nil {
		return nil, errors.New("parse RSS feed: missing channel")
	}
	return feed.Channel.Items, nil
}

func requireXMLDocumentEnd(decoder *xml.Decoder) error {
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("parse RSS feed: %w", err)
		}
		switch value := token.(type) {
		case xml.CharData:
			if strings.TrimSpace(string(value)) == "" {
				continue
			}
		case xml.Comment, xml.ProcInst:
			continue
		}
		return errors.New("parse RSS feed: unexpected content after rss element")
	}
}
