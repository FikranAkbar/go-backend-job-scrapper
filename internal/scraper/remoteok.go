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

// remoteokURLs lists tag-filtered RemoteOK API endpoints.
// RemoteOK supports ?tags= filtering so we fetch Go and backend jobs directly
// rather than downloading the entire board and hoping for matches.
// Multiple URLs are tried in sequence; a single success is enough.
var remoteokURLs = []string{
	"https://remoteok.com/api?tags=golang",
	"https://remoteok.com/api?tags=backend",
}

// RemoteOKScraper fetches jobs from the RemoteOK public JSON API.
type RemoteOKScraper struct {
	client *http.Client
}

// NewRemoteOK creates a new RemoteOKScraper with a 30-second timeout.
// Timeout is generous because RemoteOK can be slow to respond from container IPs.
func NewRemoteOK() *RemoteOKScraper {
	return &RemoteOKScraper{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the scraper identifier.
func (r *RemoteOKScraper) Name() string { return "remoteok" }

// Fetch retrieves job listings from the RemoteOK API.
// It tries multiple tag-filtered endpoints and merges results, deduplicating by slug.
func (r *RemoteOKScraper) Fetch() ([]store.Job, error) {
	seen := make(map[string]bool)
	var jobs []store.Job
	var lastErr error

	for _, url := range remoteokURLs {
		results, err := r.fetchURL(url)
		if err != nil {
			lastErr = err
			continue
		}
		for _, j := range results {
			if !seen[j.SourceID] {
				seen[j.SourceID] = true
				jobs = append(jobs, j)
			}
		}
	}

	if len(jobs) == 0 && lastErr != nil {
		return nil, fmt.Errorf("scraper remoteok: all endpoints failed, last error: %w", lastErr)
	}
	return jobs, nil
}

func (r *RemoteOKScraper) fetchURL(apiURL string) ([]store.Job, error) {
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("scraper remoteok: create request: %w", err)
	}
	// RemoteOK tarpits plain/bot User-Agents — send a realistic browser UA.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://remoteok.com/")

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

