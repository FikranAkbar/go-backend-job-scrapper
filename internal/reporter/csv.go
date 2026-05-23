// Package reporter provides CSV report generation for job results.
package reporter

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/FikranAkbar/go-backend-job-scrapper/internal/store"
)

// CSVReporter generates CSV reports for jobs.
type CSVReporter struct {
	outputDir string
}

// NewCSVReporter creates a new CSVReporter that saves files to the given directory.
func NewCSVReporter(outputDir string) *CSVReporter {
	return &CSVReporter{outputDir: outputDir}
}

// Generate creates a CSV file with the given jobs.
func (c *CSVReporter) Generate(jobs []store.Job, stats ReportStats) error {
	if err := os.MkdirAll(c.outputDir, 0755); err != nil {
		return fmt.Errorf("csv reporter: create output dir: %w", err)
	}

	filename := fmt.Sprintf("job_report_%s.csv", time.Now().Format("2006-01-02_15-04-05"))
	filepath := filepath.Join(c.outputDir, filename)

	f, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("csv reporter: create file: %w", err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// Write header
	header := []string{
		"ID",
		"Source",
		"Title",
		"Company",
		"Location",
		"AI Score",
		"AI Reason",
		"Tags",
		"Description",
		"URL",
		"Created At",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("csv reporter: write header: %w", err)
	}

	// Write jobs
	for _, j := range jobs {
		record := []string{
			strconv.Itoa(j.ID),
			j.Source,
			j.Title,
			j.Company,
			j.Location,
			strconv.Itoa(j.AIScore),
			j.AIReason,
			strings.Join(j.Tags, "; "),
			j.Description,
			j.URL,
			j.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("csv reporter: write record: %w", err)
		}
	}

	fmt.Printf("📊 CSV report saved to: %s\n", filepath)
	return nil
}

// GenerateDigest creates a CSV report for the weekly digest.
func (c *CSVReporter) GenerateDigest(jobs []store.Job) error {
	if len(jobs) == 0 {
		return nil
	}

	if err := os.MkdirAll(c.outputDir, 0755); err != nil {
		return fmt.Errorf("csv reporter: create output dir: %w", err)
	}

	filename := fmt.Sprintf("weekly_digest_%s.csv", time.Now().Format("2006-01-02"))
	filepath := filepath.Join(c.outputDir, filename)

	f, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("csv reporter: create digest file: %w", err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// Write header
	header := []string{
		"Rank",
		"AI Score",
		"Title",
		"Company",
		"Location",
		"Source",
		"Tags",
		"AI Reason",
		"Description",
		"URL",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("csv reporter: write digest header: %w", err)
	}

	// Write jobs
	for i, j := range jobs {
		record := []string{
			strconv.Itoa(i + 1),
			strconv.Itoa(j.AIScore),
			j.Title,
			j.Company,
			j.Location,
			j.Source,
			strings.Join(j.Tags, "; "),
			j.AIReason,
			j.Description,
			j.URL,
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("csv reporter: write digest record: %w", err)
		}
	}

	fmt.Printf("📊 Weekly digest CSV saved to: %s\n", filepath)
	return nil
}

