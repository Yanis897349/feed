package main

import (
	"fmt"
	"net/mail"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"
)

type article struct {
	Title     string
	Link      string
	Published time.Time
}

var publishedZoneOffsets = map[string]int{
	"UT":  0,
	"UTC": 0,
	"GMT": 0,
	"Z":   0,
	"EST": -5 * 60 * 60,
	"EDT": -4 * 60 * 60,
	"CST": -6 * 60 * 60,
	"CDT": -5 * 60 * 60,
	"MST": -7 * 60 * 60,
	"MDT": -6 * 60 * 60,
	"PST": -8 * 60 * 60,
	"PDT": -7 * 60 * 60,
}

func filterArticlesBetween(items []rssItem, start, end time.Time) []article {
	articles := make([]article, 0, len(items))
	for _, item := range items {
		article, ok := articleFromRSSItem(item)
		if !ok || article.Published.Before(start) || article.Published.After(end) {
			continue
		}
		articles = append(articles, article)
	}

	sort.Slice(articles, func(i, j int) bool {
		return articles[i].Published.After(articles[j].Published)
	})
	return articles
}

func articleFromRSSItem(item rssItem) (article, bool) {
	published, err := parsePublishedDate(item.PubDate)
	if err != nil {
		return article{}, false
	}

	title := sanitizeText(item.Title)
	link := strings.TrimSpace(item.Link)
	if title == "" || !validArticleLink(link) {
		return article{}, false
	}

	return article{Title: title, Link: link, Published: published}, true
}

func sanitizeText(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return ' '
		}
		return r
	}, value)
	return strings.Join(strings.Fields(value), " ")
}

func validArticleLink(value string) bool {
	if strings.IndexFunc(value, func(r rune) bool {
		return unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || unicode.IsSpace(r)
	}) >= 0 {
		return false
	}

	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func parsePublishedDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if published, err := time.Parse(time.RFC3339, value); err == nil {
		return published, nil
	}

	fields := strings.Fields(value)
	if len(fields) == 0 {
		return time.Time{}, fmt.Errorf("unsupported publication date %q", value)
	}

	zone := fields[len(fields)-1]
	if offset, ok := publishedZoneOffsets[zone]; ok {
		zoneStart := strings.LastIndex(value, zone)
		value = value[:zoneStart] + numericZoneOffset(offset)
	} else if !validNumericZone(zone) {
		return time.Time{}, fmt.Errorf("unsupported publication date %q", value)
	}

	if published, err := mail.ParseDate(value); err == nil {
		return published, nil
	}
	return time.Time{}, fmt.Errorf("unsupported publication date %q", value)
}

func numericZoneOffset(offset int) string {
	sign := '+'
	if offset < 0 {
		sign = '-'
		offset = -offset
	}
	return fmt.Sprintf("%c%02d%02d", sign, offset/(60*60), offset/60%60)
}

func validNumericZone(value string) bool {
	if len(value) != 5 || (value[0] != '+' && value[0] != '-') {
		return false
	}
	for index := 1; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}
