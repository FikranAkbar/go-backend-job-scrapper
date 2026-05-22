// Package reporter provides HTML and CSV report generation for job results.
package reporter

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/FikranAkbar/go-backend-job-scrapper/internal/store"
)

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Job Scraper Report - {{.Timestamp}}</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: #f5f7fa;
            padding: 20px;
            line-height: 1.6;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
        }
        .header h1 { font-size: 28px; margin-bottom: 10px; }
        .header .meta { opacity: 0.9; font-size: 14px; }
        .summary {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            padding: 30px;
            background: #f8f9fa;
            border-bottom: 1px solid #e9ecef;
        }
        .stat {
            text-align: center;
        }
        .stat-value {
            font-size: 32px;
            font-weight: bold;
            color: #667eea;
            display: block;
        }
        .stat-label {
            font-size: 14px;
            color: #6c757d;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .jobs {
            padding: 30px;
        }
        .job-card {
            border: 1px solid #e9ecef;
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 20px;
            transition: all 0.2s;
        }
        .job-card:hover {
            box-shadow: 0 4px 12px rgba(0,0,0,0.1);
            transform: translateY(-2px);
        }
        .job-header {
            display: flex;
            align-items: flex-start;
            gap: 15px;
            margin-bottom: 15px;
        }
        .score-badge {
            background: #28a745;
            color: white;
            padding: 8px 16px;
            border-radius: 20px;
            font-weight: bold;
            font-size: 18px;
            min-width: 60px;
            text-align: center;
        }
        .score-badge.excellent { background: #28a745; }
        .score-badge.good { background: #17a2b8; }
        .score-badge.maybe { background: #ffc107; color: #333; }
        .job-info {
            flex: 1;
        }
        .job-title {
            font-size: 20px;
            font-weight: 600;
            color: #212529;
            margin-bottom: 5px;
        }
        .job-company {
            font-size: 16px;
            color: #6c757d;
            margin-bottom: 10px;
        }
        .job-meta {
            display: flex;
            flex-wrap: wrap;
            gap: 15px;
            font-size: 14px;
            color: #6c757d;
            margin-bottom: 10px;
        }
        .job-meta span {
            display: flex;
            align-items: center;
            gap: 5px;
        }
        .tags {
            display: flex;
            flex-wrap: wrap;
            gap: 8px;
            margin-bottom: 10px;
        }
        .tag {
            background: #e7f3ff;
            color: #0066cc;
            padding: 4px 12px;
            border-radius: 4px;
            font-size: 13px;
        }
        .ai-reason {
            background: #f8f9fa;
            border-left: 3px solid #667eea;
            padding: 12px;
            margin-top: 10px;
            font-style: italic;
            color: #495057;
        }
        .job-link {
            display: inline-block;
            margin-top: 10px;
            padding: 8px 16px;
            background: #667eea;
            color: white;
            text-decoration: none;
            border-radius: 6px;
            font-size: 14px;
            transition: background 0.2s;
        }
        .job-link:hover {
            background: #5568d3;
        }
        .empty-state {
            text-align: center;
            padding: 60px 20px;
            color: #6c757d;
        }
        .empty-state svg {
            width: 80px;
            height: 80px;
            margin-bottom: 20px;
            opacity: 0.5;
        }
        .footer {
            text-align: center;
            padding: 20px;
            background: #f8f9fa;
            color: #6c757d;
            font-size: 14px;
            border-top: 1px solid #e9ecef;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎯 Job Scraper Report</h1>
            <div class="meta">Generated on {{.Timestamp}}</div>
        </div>
        
        <div class="summary">
            <div class="stat">
                <span class="stat-value">{{.TotalScraped}}</span>
                <span class="stat-label">Total Scraped</span>
            </div>
            <div class="stat">
                <span class="stat-value">{{.TotalFiltered}}</span>
                <span class="stat-label">After Filter</span>
            </div>
            <div class="stat">
                <span class="stat-value">{{.TotalNew}}</span>
                <span class="stat-label">New Jobs</span>
            </div>
            <div class="stat">
                <span class="stat-value">{{.TotalHighScore}}</span>
                <span class="stat-label">High Score (≥6)</span>
            </div>
        </div>

        <div class="jobs">
            {{if .Jobs}}
                {{range .Jobs}}
                <div class="job-card">
                    <div class="job-header">
                        <div class="score-badge {{.ScoreClass}}">{{.AIScore}}/10</div>
                        <div class="job-info">
                            <div class="job-title">{{.Title}}</div>
                            <div class="job-company">{{.Company}}</div>
                            <div class="job-meta">
                                <span>📍 {{.Location}}</span>
                                <span>🔖 {{.Source}}</span>
                            </div>
                            {{if .Tags}}
                            <div class="tags">
                                {{range .Tags}}
                                <span class="tag">{{.}}</span>
                                {{end}}
                            </div>
                            {{end}}
                            {{if .AIReason}}
                            <div class="ai-reason">💬 {{.AIReason}}</div>
                            {{end}}
                            <a href="{{.URL}}" target="_blank" class="job-link">View Job →</a>
                        </div>
                    </div>
                </div>
                {{end}}
            {{else}}
                <div class="empty-state">
                    <div>📭</div>
                    <h3>No jobs to report</h3>
                    <p>No new jobs found in this scan.</p>
                </div>
            {{end}}
        </div>

        <div class="footer">
            Generated by Go Backend Job Scrapper | <a href="https://github.com/FikranAkbar/go-backend-job-scrapper" target="_blank" style="color: #667eea;">GitHub</a>
        </div>
    </div>
</body>
</html>`

// HTMLReporter generates HTML reports for jobs.
type HTMLReporter struct {
	outputDir string
}

// NewHTMLReporter creates a new HTMLReporter that saves files to the given directory.
func NewHTMLReporter(outputDir string) *HTMLReporter {
	return &HTMLReporter{outputDir: outputDir}
}

// templateData holds the data for the HTML template.
type templateData struct {
	Timestamp      string
	TotalScraped   int
	TotalFiltered  int
	TotalNew       int
	TotalHighScore int
	Jobs           []templateJob
}

type templateJob struct {
	store.Job
	ScoreClass string
}

// Generate creates an HTML report file with the given jobs and stats.
func (h *HTMLReporter) Generate(jobs []store.Job, stats ReportStats) error {
	if err := os.MkdirAll(h.outputDir, 0755); err != nil {
		return fmt.Errorf("html reporter: create output dir: %w", err)
	}

	tmpl, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("html reporter: parse template: %w", err)
	}

	// Convert jobs to template format
	templateJobs := make([]templateJob, len(jobs))
	for i, j := range jobs {
		scoreClass := "maybe"
		if j.AIScore >= 8 {
			scoreClass = "excellent"
		} else if j.AIScore >= 6 {
			scoreClass = "good"
		}
		templateJobs[i] = templateJob{Job: j, ScoreClass: scoreClass}
	}

	data := templateData{
		Timestamp:      time.Now().Format("Monday, January 2, 2006 at 3:04 PM"),
		TotalScraped:   stats.TotalScraped,
		TotalFiltered:  stats.TotalFiltered,
		TotalNew:       stats.TotalNew,
		TotalHighScore: stats.TotalHighScore,
		Jobs:           templateJobs,
	}

	filename := fmt.Sprintf("job_report_%s.html", time.Now().Format("2006-01-02_15-04-05"))
	filepath := filepath.Join(h.outputDir, filename)

	f, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("html reporter: create file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("html reporter: execute template: %w", err)
	}

	fmt.Printf("📊 HTML report saved to: %s\n", filepath)
	return nil
}

// GenerateDigest creates an HTML report for the weekly digest.
func (h *HTMLReporter) GenerateDigest(jobs []store.Job) error {
	if len(jobs) == 0 {
		return nil
	}

	stats := ReportStats{
		TotalScraped:   len(jobs),
		TotalFiltered:  len(jobs),
		TotalNew:       len(jobs),
		TotalHighScore: countHighScore(jobs),
	}

	if err := os.MkdirAll(h.outputDir, 0755); err != nil {
		return fmt.Errorf("html reporter: create output dir: %w", err)
	}

	tmpl, err := template.New("digest").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("html reporter: parse template: %w", err)
	}

	templateJobs := make([]templateJob, len(jobs))
	for i, j := range jobs {
		scoreClass := "maybe"
		if j.AIScore >= 8 {
			scoreClass = "excellent"
		} else if j.AIScore >= 6 {
			scoreClass = "good"
		}
		templateJobs[i] = templateJob{Job: j, ScoreClass: scoreClass}
	}

	data := templateData{
		Timestamp:      fmt.Sprintf("Weekly Digest - %s", time.Now().Format("January 2, 2006")),
		TotalScraped:   stats.TotalScraped,
		TotalFiltered:  stats.TotalFiltered,
		TotalNew:       stats.TotalNew,
		TotalHighScore: stats.TotalHighScore,
		Jobs:           templateJobs,
	}

	filename := fmt.Sprintf("weekly_digest_%s.html", time.Now().Format("2006-01-02"))
	filepath := filepath.Join(h.outputDir, filename)

	f, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("html reporter: create digest file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("html reporter: execute digest template: %w", err)
	}

	fmt.Printf("📊 Weekly digest saved to: %s\n", filepath)
	return nil
}

func countHighScore(jobs []store.Job) int {
	count := 0
	for _, j := range jobs {
		if j.AIScore >= 6 {
			count++
		}
	}
	return count
}
