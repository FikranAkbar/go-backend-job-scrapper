package main

import (
	"fmt"
	"log"
	"time"

	"github.com/FikranAkbar/go-backend-job-scrapper/internal/reporter"
	"github.com/FikranAkbar/go-backend-job-scrapper/internal/store"
)

func main() {
	// Sample jobs to demonstrate HTML report
	jobs := []store.Job{
		{
			ID:       1,
			SourceID: "demo-1",
			Source:   "remotive",
			Title:    "Senior Backend Engineer (Go)",
			Company:  "Acme Corp",
			Location: "Remote (Worldwide)",
			URL:      "https://example.com/job/1",
			Tags:     []string{"golang", "microservices", "kubernetes", "grpc"},
			AIScore:  9,
			AIReason: "Excellent match: Strong Golang focus with microservices architecture. Fully remote and international company.",
		},
		{
			ID:       2,
			SourceID: "demo-2",
			Source:   "weworkremotely",
			Title:    "Backend Developer - Platform Team",
			Company:  "Tech Startup Inc",
			Location: "Remote (US/Europe)",
			URL:      "https://example.com/job/2",
			Tags:     []string{"golang", "api", "postgresql", "docker"},
			AIScore:  8,
			AIReason: "Very good fit: Backend role with Go, platform focus aligns well with experience.",
		},
		{
			ID:       3,
			SourceID: "demo-3",
			Source:   "remoteok",
			Title:    "Software Engineer - Backend (Jakarta)",
			Company:  "Indonesian Tech Company",
			Location: "Jakarta, Indonesia",
			URL:      "https://example.com/job/3",
			Tags:     []string{"golang", "rest api", "redis", "sql"},
			AIScore:  7,
			AIReason: "Good match: Local Jakarta role with Golang. Salary may be lower but good for Indonesia market.",
		},
		{
			ID:       4,
			SourceID: "demo-4",
			Source:   "himalayas",
			Title:    "Platform Engineer",
			Company:  "Cloud Systems Ltd",
			Location: "Remote (Asia)",
			URL:      "https://example.com/job/4",
			Tags:     []string{"golang", "aws", "terraform", "cicd"},
			AIScore:  6,
			AIReason: "Potential fit: Platform role but more DevOps-focused. Asia timezone friendly.",
		},
	}

	stats := reporter.ReportStats{
		TotalScraped:   120,
		TotalFiltered:  25,
		TotalNew:       4,
		TotalHighScore: 4,
	}

	// Generate HTML report
	htmlReporter := reporter.NewHTMLReporter("./reports")
	if err := htmlReporter.Generate(jobs, stats); err != nil {
		log.Fatalf("Failed to generate HTML report: %v", err)
	}

	// Generate CSV report
	csvReporter := reporter.NewCSVReporter("./reports")
	if err := csvReporter.Generate(jobs, stats); err != nil {
		log.Fatalf("Failed to generate CSV report: %v", err)
	}

	fmt.Println("\n✅ Demo reports generated successfully!")
	fmt.Println("📊 Check ./reports/ directory for:")
	fmt.Printf("   - job_report_%s.html\n", time.Now().Format("2006-01-02"))
	fmt.Printf("   - job_report_%s.csv\n", time.Now().Format("2006-01-02"))
	fmt.Println("\nOpen the HTML file in your browser to see the visual report!")
}

