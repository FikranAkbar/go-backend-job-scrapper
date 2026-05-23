// Package scraper provides a generic RSS scraper for LinkedIn, JobStreet, Glints, etc.
package scraper

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"github.com/FikranAkbar/go-backend-job-scrapper/internal/store"
)

// RSSFeed represents a parsed RSS/Atom feed.
type rssFeed struct {
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	GUID    string `xml:"guid"`
	Desc    string `xml:"description"`
}

// RSSScraper is a generic RSS scraper that can target any job RSS feed URL.
type RSSScraper struct {
	client *http.Client
	name   string
	feedURL string
}

// NewRSS creates a new RSSScraper for the given source name and feed URL.
func NewRSS(name, feedURL string) *RSSScraper {
	return &RSSScraper{
		client:  &http.Client{Timeout: 15 * time.Second},
		name:    name,
		feedURL: feedURL,
	}
}

// Name returns the scraper identifier.
func (r *RSSScraper) Name() string { return r.name }

// Fetch retrieves job listings from the RSS feed URL.
func (r *RSSScraper) Fetch() ([]store.Job, error) {
	resp, err := r.client.Get(r.feedURL)
	if err != nil {
		return nil, fmt.Errorf("scraper rss %s: fetch jobs: %w", r.name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scraper rss %s: unexpected status %d", r.name, resp.StatusCode)
	}

	var feed rssFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("scraper rss %s: decode feed: %w", r.name, err)
	}

	jobs := make([]store.Job, 0, len(feed.Channel.Items))
	for _, item := range feed.Channel.Items {
		link := item.Link
		if link == "" {
			link = item.GUID
		}
		if link == "" {
			continue
		}
		sid := sourceIDFromURL(r.name, link)
		jobs = append(jobs, store.Job{
			SourceID:    sid,
			Source:      r.name,
			Title:       item.Title,
			URL:         link,
			Description: stripHTMLTags(item.Desc),
		})
	}
	return jobs, nil
}

