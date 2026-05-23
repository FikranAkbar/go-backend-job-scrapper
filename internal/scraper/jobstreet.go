// Package scraper provides a JobStreet (SEEK-powered) job scraper.
// JobStreet Indonesia uses the SEEK Chalice Search v4 API for listings.
package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/FikranAkbar/go-backend-job-scrapper/internal/store"
)

const jobstreetSearchURL = "https://id.jobstreet.com/api/chalice-search/v4/search"

// JobStreetScraper fetches jobs from JobStreet Indonesia (SEEK-powered).
type JobStreetScraper struct {
	client   *http.Client
	keywords []string
}

// NewJobStreet creates a JobStreetScraper searching for Go/backend roles.
func NewJobStreet() *JobStreetScraper {
	return &JobStreetScraper{
		client:   &http.Client{Timeout: 20 * time.Second},
		keywords: []string{"golang backend", "go backend", "backend engineer"},
	}
}

// Name returns the scraper identifier.
func (j *JobStreetScraper) Name() string { return "jobstreet" }

// Fetch retrieves job listings from the JobStreet SEEK search API.
func (j *JobStreetScraper) Fetch() ([]store.Job, error) {
	seen := make(map[string]bool)
	var jobs []store.Job

	for _, kw := range j.keywords {
		results, err := j.fetchKeyword(kw)
		if err != nil {
			continue
		}
		for _, job := range results {
			if !seen[job.SourceID] {
				seen[job.SourceID] = true
				jobs = append(jobs, job)
			}
		}
	}

	if len(jobs) == 0 {
		return nil, fmt.Errorf("scraper jobstreet: no jobs retrieved")
	}
	return jobs, nil
}

func (j *JobStreetScraper) fetchKeyword(keyword string) ([]store.Job, error) {
	params := url.Values{}
	params.Set("siteKey", "ID-Main")
	params.Set("sourcesystem", "houston")
	params.Set("keywords", keyword)
	params.Set("page", "1")
	params.Set("pageSize", "30")
	params.Set("locale", "en-ID")

	req, err := http.NewRequest(http.MethodGet, jobstreetSearchURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("scraper jobstreet: build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; job-agent/1.0)")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://id.jobstreet.com/")

	resp, err := j.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("scraper jobstreet: fetch %q: %w", keyword, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scraper jobstreet: unexpected status %d for keyword %q", resp.StatusCode, keyword)
	}

	// SEEK Chalice Search v4 response structure
	var result struct {
		Data struct {
			Jobs []struct {
				ID         string `json:"id"`
				Title      string `json:"title"`
				Advertiser struct {
					Description string `json:"description"`
				} `json:"advertiser"`
				Location  string   `json:"location"`
				Suburb    string   `json:"suburb"`
				Teaser    string   `json:"teaser"`
				WorkType  string   `json:"workType"`
				Tags      []struct {
					Label string `json:"label"`
				} `json:"tags"`
				BulletPoints []string `json:"bulletPoints"`
				// Some API versions use these fields
				SolMetadata struct {
					JobID string `json:"jobId"`
				} `json:"solMetadata"`
			} `json:"jobs"`
			TotalCount int `json:"totalCount"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("scraper jobstreet: decode response: %w", err)
	}

	jobs := make([]store.Job, 0, len(result.Data.Jobs))
	for _, item := range result.Data.Jobs {
		if item.Title == "" || item.ID == "" {
			continue
		}

		jobURL := "https://id.jobstreet.com/en/job/" + item.ID

		// Build location: prefer suburb if available
		location := item.Location
		if item.Suburb != "" && !strings.Contains(location, item.Suburb) {
			location = item.Suburb + ", " + location
		}

		// Collect tags from tag labels
		tags := make([]string, 0, len(item.Tags))
		for _, t := range item.Tags {
			if t.Label != "" {
				tags = append(tags, t.Label)
			}
		}

		// Build description from teaser + bullet points
		desc := item.Teaser
		if len(item.BulletPoints) > 0 {
			desc += "\n" + strings.Join(item.BulletPoints, "\n")
		}

		sid := sourceIDFromURL("jobstreet", jobURL)
		jobs = append(jobs, store.Job{
			SourceID:    sid,
			Source:      "jobstreet",
			Title:       item.Title,
			Company:     item.Advertiser.Description,
			Location:    location,
			URL:         jobURL,
			Tags:        tags,
			Description: strings.TrimSpace(desc),
		})
	}
	return jobs, nil
}

