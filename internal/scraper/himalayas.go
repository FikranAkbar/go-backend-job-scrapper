// Package scraper provides the Himalayas scraper (public JSON API).
package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/FikranAkbar/go-backend-job-scrapper/internal/store"
)

// himalayasURLs are keyword-filtered endpoints so we only receive backend/Go roles.
// Using ?q= means Himalayas does the heavy lifting server-side, reducing noise.
var himalayasURLs = []string{
	"https://himalayas.app/jobs/api?q=golang&limit=50",
	"https://himalayas.app/jobs/api?q=backend+engineer&limit=50",
}

// HimalayasScraper fetches jobs from the Himalayas public JSON API.
type HimalayasScraper struct {
	client *http.Client
}

// NewHimalayas creates a new HimalayasScraper with a 15-second timeout.
func NewHimalayas() *HimalayasScraper {
	return &HimalayasScraper{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Name returns the scraper identifier.
func (h *HimalayasScraper) Name() string { return "himalayas" }

// Fetch retrieves job listings from the Himalayas API using keyword-filtered queries.
func (h *HimalayasScraper) Fetch() ([]store.Job, error) {
	seen := make(map[string]bool)
	var jobs []store.Job
	var lastErr error

	for _, apiURL := range himalayasURLs {
		results, err := h.fetchURL(apiURL)
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
		return nil, lastErr
	}
	return jobs, nil
}

func (h *HimalayasScraper) fetchURL(apiURL string) ([]store.Job, error) {
	resp, err := h.client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("scraper himalayas: fetch jobs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scraper himalayas: unexpected status %d", resp.StatusCode)
	}

	var payload struct {
		Jobs []struct {
			Slug    string `json:"slug"`
			Title   string `json:"title"`
			Company struct {
				Name string `json:"name"`
			} `json:"company"`
			LocationRestrictions []string `json:"locationRestrictions"`
			URL                  string   `json:"applicationLink"`
			Tags                 []string `json:"tags"`
			Description          string   `json:"description"`
		} `json:"jobs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("scraper himalayas: decode response: %w", err)
	}

	jobs := make([]store.Job, 0, len(payload.Jobs))
	for _, j := range payload.Jobs {
		jobURL := j.URL
		if jobURL == "" {
			jobURL = "https://himalayas.app/jobs/" + j.Slug
		}
		location := strings.Join(j.LocationRestrictions, ", ")
		sid := sourceIDFromURL("himalayas", jobURL)
		jobs = append(jobs, store.Job{
			SourceID:    sid,
			Source:      "himalayas",
			Title:       j.Title,
			Company:     j.Company.Name,
			Location:    location,
			URL:         jobURL,
			Tags:        j.Tags,
			Description: stripHTMLTags(j.Description),
		})
	}
	return jobs, nil
}
