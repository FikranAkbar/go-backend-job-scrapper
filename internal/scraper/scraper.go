// Package scraper defines the Scraper interface and the FetchAll aggregator.
package scraper

import (
	"log/slog"
	"sync"

	"github.com/FikranAkbar/go-backend-job-scrapper/internal/store"
)

// Scraper is the contract every job-source scraper must implement.
type Scraper interface {
	// Name returns a short identifier for this scraper (e.g. "remotive").
	Name() string
	// Fetch retrieves job listings from the source.
	Fetch() ([]store.Job, error)
}

// FetchAll runs all scrapers concurrently and merges their results.
// Errors from individual scrapers are logged but do not abort the run.
func FetchAll(scrapers []Scraper) []store.Job {
	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		allJobs []store.Job
	)

	for _, s := range scrapers {
		wg.Add(1)
		go func(s Scraper) {
			defer wg.Done()
			jobs, err := s.Fetch()
			if err != nil {
				slog.Error("scraper failed", "source", s.Name(), "err", err)
				return
			}
			mu.Lock()
			allJobs = append(allJobs, jobs...)
			mu.Unlock()
		}(s)
	}

	wg.Wait()
	return allJobs
}

