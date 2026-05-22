// Package reporter provides report generation in multiple formats (HTML, CSV, Telegram).
package reporter

import (
	"github.com/FikranAkbar/go-backend-job-scrapper/internal/store"
)

// Reporter is the interface for all report generators.
type Reporter interface {
	// Generate creates a report from the given jobs and stats.
	Generate(jobs []store.Job, stats ReportStats) error
	// GenerateDigest creates a weekly digest report.
	GenerateDigest(jobs []store.Job) error
}

// ReportStats holds summary statistics for a scan.
type ReportStats struct {
	TotalScraped   int
	TotalFiltered  int
	TotalNew       int
	TotalHighScore int
}

