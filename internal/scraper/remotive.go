// Package scraper provides the Remotive scraper (public JSON API).
package scraper

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/FikranAkbar/go-backend-job-scrapper/internal/store"
)

const remotiveURL = "https://remotive.com/api/remote-jobs?limit=100"

// RemotiveScraper fetches jobs from the Remotive public JSON API.
type RemotiveScraper struct {
	client *http.Client
}

// NewRemotive creates a new RemotiveScraper with a 15-second timeout.
func NewRemotive() *RemotiveScraper {
	return &RemotiveScraper{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Name returns the scraper identifier.
func (r *RemotiveScraper) Name() string { return "remotive" }

// Fetch retrieves job listings from the Remotive API.
func (r *RemotiveScraper) Fetch() ([]store.Job, error) {
	resp, err := r.client.Get(remotiveURL)
	if err != nil {
		return nil, fmt.Errorf("scraper remotive: fetch jobs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scraper remotive: unexpected status %d", resp.StatusCode)
	}

	var payload struct {
		Jobs []struct {
			ID          int      `json:"id"`
			URL         string   `json:"url"`
			Title       string   `json:"job_title"`
			CompanyName string   `json:"company_name"`
			Location    string   `json:"candidate_required_location"`
			Tags        []string `json:"tags"`
		} `json:"jobs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("scraper remotive: decode response: %w", err)
	}

	jobs := make([]store.Job, 0, len(payload.Jobs))
	for _, j := range payload.Jobs {
		sourceID := sourceIDFromURL("remotive", j.URL)
		jobs = append(jobs, store.Job{
			SourceID: sourceID,
			Source:   "remotive",
			Title:    j.Title,
			Company:  j.CompanyName,
			Location: j.Location,
			URL:      j.URL,
			Tags:     j.Tags,
		})
	}
	return jobs, nil
}

// sourceIDFromURL creates a stable source ID by hashing the URL with a prefix.
func sourceIDFromURL(prefix, rawURL string) string {
	h := sha256.Sum256([]byte(rawURL))
	return fmt.Sprintf("%s-%x", prefix, h[:8])
}

// tagsFromCSV splits a comma-separated tag string into a slice.
func tagsFromCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

