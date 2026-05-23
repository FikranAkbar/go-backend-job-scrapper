// Package scraper provides the RemoteOK scraper (public JSON API).
package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/FikranAkbar/go-backend-job-scrapper/internal/store"
)

const remoteokURL = "https://remoteok.com/api"

// RemoteOKScraper fetches jobs from the RemoteOK public JSON API.
type RemoteOKScraper struct {
	client *http.Client
}

// NewRemoteOK creates a new RemoteOKScraper with a 15-second timeout.
func NewRemoteOK() *RemoteOKScraper {
	return &RemoteOKScraper{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Name returns the scraper identifier.
func (r *RemoteOKScraper) Name() string { return "remoteok" }

// Fetch retrieves job listings from the RemoteOK API.
func (r *RemoteOKScraper) Fetch() ([]store.Job, error) {
	req, err := http.NewRequest(http.MethodGet, remoteokURL, nil)
	if err != nil {
		return nil, fmt.Errorf("scraper remoteok: create request: %w", err)
	}
	// RemoteOK requires a non-empty User-Agent
	req.Header.Set("User-Agent", "job-agent/1.0")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("scraper remoteok: fetch jobs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scraper remoteok: unexpected status %d", resp.StatusCode)
	}

	// The API returns a JSON array; the first element is a legal notice object, skip it.
	var raw []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("scraper remoteok: decode response: %w", err)
	}

	type remoteOKJob struct {
		Slug        string   `json:"slug"`
		URL         string   `json:"url"`
		Position    string   `json:"position"`
		Company     string   `json:"company"`
		Location    string   `json:"location"`
		Tags        []string `json:"tags"`
		Description string   `json:"description"`
	}

	jobs := make([]store.Job, 0, len(raw))
	for _, msg := range raw {
		var j remoteOKJob
		if err := json.Unmarshal(msg, &j); err != nil || j.Slug == "" {
			continue
		}
		jobURL := j.URL
		if jobURL == "" {
			jobURL = "https://remoteok.com/remote-jobs/" + j.Slug
		}
		sid := sourceIDFromURL("remoteok", jobURL)
		jobs = append(jobs, store.Job{
			SourceID:    sid,
			Source:      "remoteok",
			Title:       j.Position,
			Company:     j.Company,
			Location:    strings.TrimSpace(j.Location),
			URL:         jobURL,
			Tags:        j.Tags,
			Description: stripHTMLTags(j.Description),
		})
	}
	return jobs, nil
}

