// Package scraper provides the WeWorkRemotely scraper (RSS feed).
package scraper

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"github.com/FikranAkbar/go-backend-job-scrapper/internal/store"
)

const wwrURL = "https://weworkremotely.com/remote-jobs.rss"

// WWRScraper fetches jobs from the We Work Remotely RSS feed.
type WWRScraper struct {
	client *http.Client
}

// NewWeWorkRemotely creates a new WWRScraper with a 15-second timeout.
func NewWeWorkRemotely() *WWRScraper {
	return &WWRScraper{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Name returns the scraper identifier.
func (w *WWRScraper) Name() string { return "weworkremotely" }

// Fetch retrieves job listings from the We Work Remotely RSS feed.
func (w *WWRScraper) Fetch() ([]store.Job, error) {
	resp, err := w.client.Get(wwrURL)
	if err != nil {
		return nil, fmt.Errorf("scraper weworkremotely: fetch jobs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scraper weworkremotely: unexpected status %d", resp.StatusCode)
	}

	var feed struct {
		Channel struct {
			Items []struct {
				Title       string `xml:"title"`
				Link        string `xml:"link"`
				GUID        string `xml:"guid"`
				Region      string `xml:"region"`
				Type        string `xml:"type"`
				Description string `xml:"description"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("scraper weworkremotely: decode rss: %w", err)
	}

	jobs := make([]store.Job, 0, len(feed.Channel.Items))
	for _, item := range feed.Channel.Items {
		if item.Link == "" {
			continue
		}
		sid := sourceIDFromURL("weworkremotely", item.Link)
		jobs = append(jobs, store.Job{
			SourceID:    sid,
			Source:      "weworkremotely",
			Title:       item.Title,
			Location:    item.Region,
			URL:         item.Link,
			Description: stripHTMLTags(item.Description),
		})
	}
	return jobs, nil
}

