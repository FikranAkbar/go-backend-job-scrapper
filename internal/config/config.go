// Package config loads and validates application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all runtime configuration for the job agent.
type Config struct {
	DBUrl             string
	TelegramBotToken  string
	TelegramChatID    string
	GeminiAPIKey      string
	ScanIntervalHours int    // default: 6
	ReportFormat      string // "telegram" | "html" | "csv" | "all" (default: "html")
	ReportOutputDir   string // default: "./reports"
}

// Load reads environment variables, validates required fields, and returns a Config.
// Returns an error if any required variable is missing.
func Load() (*Config, error) {
	cfg := &Config{
		DBUrl:            os.Getenv("DB_URL"),
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatID:   os.Getenv("TELEGRAM_CHAT_ID"),
		GeminiAPIKey:     os.Getenv("GEMINI_API_KEY"),
		ReportFormat:     os.Getenv("REPORT_FORMAT"),
		ReportOutputDir:  os.Getenv("REPORT_OUTPUT_DIR"),
	}

	if cfg.DBUrl == "" {
		return nil, fmt.Errorf("config: DB_URL is required")
	}

	// Telegram is now optional if using file-based reports
	if cfg.ReportFormat == "telegram" {
		if cfg.TelegramBotToken == "" {
			return nil, fmt.Errorf("config: TELEGRAM_BOT_TOKEN is required when REPORT_FORMAT=telegram")
		}
		if cfg.TelegramChatID == "" {
			return nil, fmt.Errorf("config: TELEGRAM_CHAT_ID is required when REPORT_FORMAT=telegram")
		}
	}

	// Default report format
	if cfg.ReportFormat == "" {
		cfg.ReportFormat = "html"
	}

	// Default output directory
	if cfg.ReportOutputDir == "" {
		cfg.ReportOutputDir = "./reports"
	}

	interval := os.Getenv("SCAN_INTERVAL_HOURS")
	if interval == "" {
		cfg.ScanIntervalHours = 6
	} else {
		n, err := strconv.Atoi(interval)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("config: SCAN_INTERVAL_HOURS must be a positive integer, got %q", interval)
		}
		cfg.ScanIntervalHours = n
	}

	return cfg, nil
}

