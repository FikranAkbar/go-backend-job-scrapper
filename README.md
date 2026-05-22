# Job Agent

An autonomous job-finding agent written in Go. Scrapes job listings on a schedule,
deduplicates via PostgreSQL, pre-filters by keyword, scores with Gemini Flash AI,
and delivers results to Telegram.

## Quick Start

```bash
cp .env.example .env
# Fill in DB_URL, TELEGRAM_BOT_TOKEN, TELEGRAM_CHAT_ID (and optionally GEMINI_API_KEY)
docker compose up --build
```

## Project Structure

```
cmd/agent/          — entrypoint & scheduler loop
internal/config/    — env var loading & validation
internal/store/     — PostgreSQL models & queries
internal/scraper/   — job source scrapers
internal/filter/    — keyword pre-filter
internal/ai/        — Gemini Flash scoring
internal/notifier/  — Telegram notifications
migrations/         — SQL schema (auto-applied by Docker)
```

## Pipeline

`Scrape → Keyword Filter → Dedup → AI Score → Notify`

## Tech Stack

- Go 1.23 · Docker · PostgreSQL 16 · pgx/v5 · Gemini Flash API · Telegram Bot API

# go-backend-job-scrapper
