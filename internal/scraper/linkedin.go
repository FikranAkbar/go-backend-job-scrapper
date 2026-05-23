// Package scraper provides a LinkedIn job scraper using the public guest jobs API.
//
// Note: LinkedIn may rate-limit or block automated requests. This scraper uses
// the unauthenticated /jobs-guest API that powers LinkedIn's public job search.
// Scraping LinkedIn is subject to their User Agreement — use for personal/research
// purposes only.
package scraper

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/FikranAkbar/go-backend-job-scrapper/internal/store"
)

const linkedinGuestURL = "https://www.linkedin.com/jobs-guest/jobs/api/seeMoreJobPostings/search"

var (
	reLinkedInID      = regexp.MustCompile(`data-entity-urn="urn:li:jobPosting:(\d+)"`)
	reLinkedInURL     = regexp.MustCompile(`href="(https://www\.linkedin\.com/jobs/view/[^"?]+)"`)
	reLinkedInTitle   = regexp.MustCompile(`class="base-search-card__title"[^>]*>\s*([^<]+?)\s*<`)
	reLinkedInCompany = regexp.MustCompile(`class="hidden-nested-link"[^>]*>\s*([^<]+?)\s*<`)
	reLinkedInLoc     = regexp.MustCompile(`class="job-search-card__location"[^>]*>\s*([^<]+?)\s*<`)
	reLinkedInDesc    = regexp.MustCompile(`class="base-search-card__metadata"[^>]*>([\s\S]+?)</div>`)
)

// LinkedInScraper fetches jobs from LinkedIn's public unauthenticated guest API.
type LinkedInScraper struct {
	client   *http.Client
	keywords []string
}

// NewLinkedIn creates a LinkedInScraper that searches for Go/backend roles.
func NewLinkedIn() *LinkedInScraper {
	return &LinkedInScraper{
		client: &http.Client{Timeout: 20 * time.Second},
		keywords: []string{
			"golang backend",
			"go backend engineer",
		},
	}
}

// Name returns the scraper identifier.
func (l *LinkedInScraper) Name() string { return "linkedin" }

// Fetch retrieves job listings from LinkedIn's public guest jobs API.
func (l *LinkedInScraper) Fetch() ([]store.Job, error) {
	seen := make(map[string]bool)
	var jobs []store.Job

	for _, kw := range l.keywords {
		results, err := l.fetchKeyword(kw)
		if err != nil {
			// One keyword failing should not abort the whole scraper
			continue
		}
		for _, j := range results {
			if !seen[j.SourceID] {
				seen[j.SourceID] = true
				jobs = append(jobs, j)
			}
		}
	}

	if len(jobs) == 0 {
		return nil, fmt.Errorf("scraper linkedin: no jobs retrieved (may be rate-limited or blocked by LinkedIn)")
	}
	return jobs, nil
}

func (l *LinkedInScraper) fetchKeyword(keyword string) ([]store.Job, error) {
	params := url.Values{}
	params.Set("keywords", keyword)
	params.Set("location", "remote")
	params.Set("start", "0")
	params.Set("count", "25")

	req, err := http.NewRequest(http.MethodGet, linkedinGuestURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("scraper linkedin: build request: %w", err)
	}
	// Mimic a real browser to avoid immediate 429s
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("scraper linkedin: fetch %q: %w", keyword, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("scraper linkedin: rate limited (429) for keyword %q", keyword)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scraper linkedin: unexpected status %d for keyword %q", resp.StatusCode, keyword)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("scraper linkedin: read body: %w", err)
	}

	return parseLinkedInHTML(string(body)), nil
}

// parseLinkedInHTML extracts job listings from LinkedIn's HTML-fragment response.
func parseLinkedInHTML(body string) []store.Job {
	// Each job card starts with data-entity-urn
	cards := strings.Split(body, "data-entity-urn=")
	if len(cards) <= 1 {
		return nil
	}

	var jobs []store.Job
	for _, card := range cards[1:] {
		// Reconstruct the attribute so our compiled regexps can match it
		card = "data-entity-urn=" + card

		m := reLinkedInID.FindStringSubmatch(card)
		if len(m) < 2 {
			continue
		}
		jobID := m[1]
		jobURL := "https://www.linkedin.com/jobs/view/" + jobID

		// Prefer full URL with slug when available
		if u := reExtract(reLinkedInURL, card); u != "" {
			jobURL = u
		}

		title := reExtract(reLinkedInTitle, card)
		if title == "" {
			continue
		}
		company := reExtract(reLinkedInCompany, card)
		location := reExtract(reLinkedInLoc, card)

		// Best-effort description from the card metadata
		desc := stripHTMLTags(reExtract(reLinkedInDesc, card))

		sid := sourceIDFromURL("linkedin", jobURL)
		jobs = append(jobs, store.Job{
			SourceID:    sid,
			Source:      "linkedin",
			Title:       title,
			Company:     company,
			Location:    location,
			URL:         jobURL,
			Description: desc,
		})
	}
	return jobs
}

