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
		scraper.NewRemotive(),
		scraper.NewWeWorkRemotely(),
		scraper.NewRemoteOK(),
		scraper.NewHimalayas(),
	}
	scorer := ai.NewScorer(cfg.GeminiAPIKey)
	tg := notifier.NewTelegram(cfg.TelegramBotToken, cfg.TelegramChatID)
	interval := time.Duration(cfg.ScanIntervalHours) * time.Hour

	// Initialize lastDigest so the first weekly digest fires after one full week.
	lastDigest := time.Now()

	for {
		if err := runOnce(ctx, scrapers, db, scorer, tg); err != nil {
			slog.Error("scan failed", "err", err)
		}

		// Send weekly digest when 7 days have elapsed since the last one.
		if time.Since(lastDigest) >= digestInterval {
			if err := sendWeeklyDigest(db, tg); err != nil {
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

// sendWeeklyDigest fetches the top 10 jobs from the past week and sends a digest to Telegram.
func sendWeeklyDigest(db store.Store, tg *notifier.TelegramNotifier) error {
	since := time.Now().Add(-digestInterval)
	jobs, err := db.TopJobs(since, 10)
	if err != nil {
		return fmt.Errorf("weekly digest: fetch top jobs: %w", err)
	}
	if len(jobs) == 0 {
		slog.Info("weekly digest: no scored jobs to report")
		return nil
	}
	if err := tg.SendDigest(jobs); err != nil {
		return fmt.Errorf("weekly digest: %w", err)
	}
	slog.Info("weekly digest sent", "count", len(jobs))
	return nil
}

func runOnce(
	ctx context.Context,
	scrapers []scraper.Scraper,
	db store.Store,
	scorer *ai.Scorer,
	tg *notifier.TelegramNotifier,
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

	// 6. Notify on score >= 6
	var toNotify []store.Job
	for _, j := range scored {
		if j.AIScore >= 6 {
			toNotify = append(toNotify, j)
		}
	}

	if len(toNotify) == 0 {
		slog.Info("scan complete", "new_jobs", len(fresh), "scored", len(scored), "notified", 0)
		return nil
	}

	if err := tg.Send(toNotify); err != nil {
		return err
	}

	notifiedIDs := make([]int, 0, len(toNotify))
	for _, j := range toNotify {
		notifiedIDs = append(notifiedIDs, j.ID)
	}
	if err := db.MarkNotified(notifiedIDs); err != nil {
		return err
	}

	slog.Info("scan complete", "new_jobs", len(fresh), "scored", len(scored), "notified", len(toNotify))
	return nil
}
