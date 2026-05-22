// Package scraper provides the Himalayas scraper (public JSON API).
package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/FikranAkbar/go-backend-job-scrapper/internal/store"
)

const himalayasURL = "https://himalayas.app/jobs/api?limit=100"

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

// Fetch retrieves job listings from the Himalayas API.
func (h *HimalayasScraper) Fetch() ([]store.Job, error) {
	resp, err := h.client.Get(himalayasURL)
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
			Location string   `json:"locationRestrictions"`
			URL      string   `json:"applicationLink"`
			Tags     []string `json:"tags"`
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
		sid := sourceIDFromURL("himalayas", jobURL)
		jobs = append(jobs, store.Job{
			SourceID: sid,
			Source:   "himalayas",
			Title:    j.Title,
			Company:  j.Company.Name,
			Location: j.Location,
			URL:      jobURL,
			Tags:     j.Tags,
		})
	}
	return jobs, nil
}
