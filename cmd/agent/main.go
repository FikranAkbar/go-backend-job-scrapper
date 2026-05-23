// Command agent is the entrypoint for the autonomous job-finding agent.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FikranAkbar/go-backend-job-scrapper/internal/ai"
	"github.com/FikranAkbar/go-backend-job-scrapper/internal/config"
	"github.com/FikranAkbar/go-backend-job-scrapper/internal/filter"
	"github.com/FikranAkbar/go-backend-job-scrapper/internal/notifier"
	"github.com/FikranAkbar/go-backend-job-scrapper/internal/reporter"
	"github.com/FikranAkbar/go-backend-job-scrapper/internal/scraper"
	"github.com/FikranAkbar/go-backend-job-scrapper/internal/store"
)

const digestInterval = 7 * 24 * time.Hour

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	if err := run(cfg); err != nil {
		slog.Error("agent exited with error", "err", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	db, err := store.New(ctx, cfg.DBUrl)
	if err != nil {
		return err
	}
	defer db.Close()

	scrapers := []scraper.Scraper{
		// Global remote boards
		scraper.NewRemotive(),
		scraper.NewWeWorkRemotely(),
		scraper.NewRemoteOK(),
		scraper.NewHimalayas(),
		// Social / regional job boards
		scraper.NewLinkedIn(),
		scraper.NewGlints(),
		scraper.NewJobStreet(),
	}
	scorer := ai.NewScorer(cfg.GeminiAPIKey)

	// Initialize reporters based on config
	var reporters []reporter.Reporter
	switch cfg.ReportFormat {
	case "telegram":
		if cfg.TelegramBotToken != "" && cfg.TelegramChatID != "" {
			tg := notifier.NewTelegram(cfg.TelegramBotToken, cfg.TelegramChatID)
			reporters = append(reporters, tg)
			slog.Info("reporter: telegram enabled")
		} else {
			slog.Warn("telegram config missing, skipping telegram reporter")
		}
	case "html":
		reporters = append(reporters, reporter.NewHTMLReporter(cfg.ReportOutputDir))
		slog.Info("reporter: html enabled", "output_dir", cfg.ReportOutputDir)
	case "csv":
		reporters = append(reporters, reporter.NewCSVReporter(cfg.ReportOutputDir))
		slog.Info("reporter: csv enabled", "output_dir", cfg.ReportOutputDir)
	case "all":
		reporters = append(reporters, reporter.NewHTMLReporter(cfg.ReportOutputDir))
		reporters = append(reporters, reporter.NewCSVReporter(cfg.ReportOutputDir))
		if cfg.TelegramBotToken != "" && cfg.TelegramChatID != "" {
			tg := notifier.NewTelegram(cfg.TelegramBotToken, cfg.TelegramChatID)
			reporters = append(reporters, tg)
		}
		slog.Info("reporter: all formats enabled", "output_dir", cfg.ReportOutputDir)
	default:
		return fmt.Errorf("invalid REPORT_FORMAT: %q (must be telegram, html, csv, or all)", cfg.ReportFormat)
	}

	if len(reporters) == 0 {
		return fmt.Errorf("no reporters configured")
	}

	interval := time.Duration(cfg.ScanIntervalHours) * time.Hour

	// Initialize lastDigest so the first weekly digest fires after one full week.
	lastDigest := time.Now()

	for {
		if err := runOnce(ctx, scrapers, db, scorer, reporters); err != nil {
			slog.Error("scan failed", "err", err)
		}

		// Send weekly digest when 7 days have elapsed since the last one.
		if time.Since(lastDigest) >= digestInterval {
			if err := sendWeeklyDigest(db, reporters); err != nil {
				slog.Error("weekly digest failed", "err", err)
			} else {
				lastDigest = time.Now()
			}
		}

		slog.Info("next scan", "in", interval)

		select {
		case <-ctx.Done():
			slog.Info("shutting down gracefully")
			return nil
		case <-time.After(interval):
		}
	}
}

// sendWeeklyDigest fetches the top 10 jobs from the past week and sends a digest to all reporters.
func sendWeeklyDigest(db store.Store, reporters []reporter.Reporter) error {
	since := time.Now().Add(-digestInterval)
	jobs, err := db.TopJobs(since, 10)
	if err != nil {
		return fmt.Errorf("weekly digest: fetch top jobs: %w", err)
	}
	if len(jobs) == 0 {
		slog.Info("weekly digest: no scored jobs to report")
		return nil
	}

	for _, r := range reporters {
		if err := r.GenerateDigest(jobs); err != nil {
			slog.Error("weekly digest: reporter failed", "err", err)
			// Continue with other reporters even if one fails
		}
	}

	slog.Info("weekly digest sent", "count", len(jobs))
	return nil
}

func runOnce(
	ctx context.Context,
	scrapers []scraper.Scraper,
	db store.Store,
	scorer *ai.Scorer,
	reporters []reporter.Reporter,
) error {
	// 1. Scrape
	all := scraper.FetchAll(scrapers)
	slog.Info("scraped jobs", "total", len(all))

	// 2. Keyword filter
	filtered := filter.Apply(all)
	slog.Info("after keyword filter", "count", len(filtered))

	// 3. Dedup — only keep jobs not yet in DB
	fresh, err := db.FilterSeen(filtered)
	if err != nil {
		return err
	}
	slog.Info("new jobs after dedup", "count", len(fresh))

	if len(fresh) == 0 {
		return nil
	}

	// 4. AI scoring
	scored := scorer.ScoreAll(fresh)

	// 5. Persist all scored jobs
	if err := db.Save(scored); err != nil {
		return err
	}

	// 6. Count high-scoring jobs
	highScoreCount := 0
	for _, j := range scored {
		if j.AIScore >= 6 {
			highScoreCount++
		}
	}

	// 7. Generate reports
	stats := reporter.ReportStats{
		TotalScraped:   len(all),
		TotalFiltered:  len(filtered),
		TotalNew:       len(fresh),
		TotalHighScore: highScoreCount,
	}

	for _, r := range reporters {
		if err := r.Generate(scored, stats); err != nil {
			slog.Error("reporter failed", "err", err)
			// Continue with other reporters even if one fails
		}
	}

	slog.Info("scan complete", "new_jobs", len(fresh), "scored", len(scored), "high_score", highScoreCount)
	return nil
}
