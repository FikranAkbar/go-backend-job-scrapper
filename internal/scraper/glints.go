// Package scraper provides a Glints job scraper using the public GraphQL API.
package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/FikranAkbar/go-backend-job-scrapper/internal/store"
)

const glintsGraphQLURL = "https://glints.com/api/graphql"

// glintsQuery is the GraphQL query used to search for jobs on Glints.
const glintsQuery = `
query SearchOpportunities($keyword: String, $countryCode: String, $pageSize: Int, $pageNumber: Int) {
  opportunities(params: {
    query: $keyword
    countryCode: $countryCode
    pageSize: $pageSize
    pageNumber: $pageNumber
  }) {
    data {
      id
      title
      company { name }
      citySubDivision { name }
      country { name }
      workArrangementOptions
      skills { skill { name } }
      shortDescription
      links { url }
    }
  }
}`

// GlintsScraper fetches jobs from the Glints public GraphQL API.
type GlintsScraper struct {
	client      *http.Client
	countryCode string
	keywords    []string
}

// NewGlints creates a GlintsScraper targeting Indonesia (ID) by default.
func NewGlints() *GlintsScraper {
	return &GlintsScraper{
		client:      &http.Client{Timeout: 20 * time.Second},
		countryCode: "ID",
		keywords:    []string{"golang", "go backend", "backend engineer"},
	}
}

// Name returns the scraper identifier.
func (g *GlintsScraper) Name() string { return "glints" }

// Fetch retrieves job listings from the Glints GraphQL API.
func (g *GlintsScraper) Fetch() ([]store.Job, error) {
	seen := make(map[string]bool)
	var jobs []store.Job

	for _, kw := range g.keywords {
		results, err := g.fetchKeyword(kw)
		if err != nil {
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
		return nil, fmt.Errorf("scraper glints: no jobs retrieved")
	}
	return jobs, nil
}

func (g *GlintsScraper) fetchKeyword(keyword string) ([]store.Job, error) {
	payload := map[string]any{
		"operationName": "SearchOpportunities",
		"variables": map[string]any{
			"keyword":     keyword,
			"countryCode": g.countryCode,
			"pageSize":    30,
			"pageNumber":  0,
		},
		"query": glintsQuery,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("scraper glints: marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, glintsGraphQLURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("scraper glints: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; job-agent/1.0)")
	req.Header.Set("Accept", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("scraper glints: post graphql: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scraper glints: unexpected status %d", resp.StatusCode)
	}

	var gqlResp struct {
		Data struct {
			Opportunities struct {
				Data []struct {
					ID      string `json:"id"`
					Title   string `json:"title"`
					Company struct {
						Name string `json:"name"`
					} `json:"company"`
					CitySubDivision struct {
						Name string `json:"name"`
					} `json:"citySubDivision"`
					Country struct {
						Name string `json:"name"`
					} `json:"country"`
					WorkArrangementOptions []string `json:"workArrangementOptions"`
					Skills                 []struct {
						Skill struct {
							Name string `json:"name"`
						} `json:"skill"`
					} `json:"skills"`
					ShortDescription string `json:"shortDescription"`
					Links            []struct {
						URL string `json:"url"`
					} `json:"links"`
				} `json:"data"`
			} `json:"opportunities"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, fmt.Errorf("scraper glints: decode response: %w", err)
	}
	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("scraper glints: graphql error: %s", gqlResp.Errors[0].Message)
	}

	items := gqlResp.Data.Opportunities.Data
	jobs := make([]store.Job, 0, len(items))
	for _, item := range items {
		if item.Title == "" {
			continue
		}

		// Build URL — prefer the first link, else construct from ID
		jobURL := "https://glints.com/id/opportunities/jobs/" + item.ID
		if len(item.Links) > 0 && item.Links[0].URL != "" {
			jobURL = item.Links[0].URL
		}

		// Build location string
		location := item.CitySubDivision.Name
		if item.Country.Name != "" {
			if location != "" {
				location += ", " + item.Country.Name
			} else {
				location = item.Country.Name
			}
		}
		// Prepend work arrangement if remote
		for _, wa := range item.WorkArrangementOptions {
			if strings.EqualFold(wa, "REMOTE") || strings.EqualFold(wa, "HYBRID") {
				location = wa + " — " + location
				break
			}
		}

		// Collect skill tags
		tags := make([]string, 0, len(item.Skills))
		for _, s := range item.Skills {
			if s.Skill.Name != "" {
				tags = append(tags, strings.ToLower(s.Skill.Name))
			}
		}

		sid := sourceIDFromURL("glints", jobURL)
		jobs = append(jobs, store.Job{
			SourceID:    sid,
			Source:      "glints",
			Title:       item.Title,
			Company:     item.Company.Name,
			Location:    location,
			URL:         jobURL,
			Tags:        tags,
			Description: stripHTMLTags(item.ShortDescription),
		})
	}
	return jobs, nil
}

