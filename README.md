# Go Backend Job Scrapper

An autonomous job-finding agent written in Go. Scrapes job listings on a schedule,
deduplicates via PostgreSQL, pre-filters by keyword, scores with Gemini Flash AI,
and delivers results via **HTML**, **CSV**, or **Telegram**.

## Quick Start

```bash
cp .env.example .env
# Edit .env and configure:
# - DB_URL (required)
# - REPORT_FORMAT (html, csv, telegram, or all)
# - For telegram: TELEGRAM_BOT_TOKEN & TELEGRAM_CHAT_ID
# - For AI scoring: GEMINI_API_KEY (optional)

docker compose up --build
```

## Features

✅ **4 Job Sources** — Remotive, WeWorkRemotely, RemoteOK, Himalayas  
✅ **Smart Filtering** — Keyword-based include/exclude rules  
✅ **AI Scoring** — Gemini Flash rates jobs 1-10 for match quality  
✅ **Deduplication** — PostgreSQL prevents duplicate processing  
✅ **Multiple Report Formats:**
  - 📊 **HTML** — Beautiful web reports with stats and styling
  - 📄 **CSV** — Spreadsheet-ready exports
  - 💬 **Telegram** — Real-time notifications

✅ **Weekly Digest** — Top 10 jobs sent weekly  
✅ **Docker Ready** — Compose orchestration with health checks  
✅ **Fully Tested** — 45 unit tests covering filter and AI scorer

## Report Formats

### HTML Reports (Default)
Generate beautiful HTML reports in `./reports/` directory:

```env
REPORT_FORMAT=html
REPORT_OUTPUT_DIR=./reports
```

**Output:** `reports/job_report_2026-05-22_14-30-00.html`

- Visual score badges (🟢 excellent, 🟡 maybe)
- Summary statistics dashboard
- Clickable job links
- Tags and AI reasoning display

### CSV Reports
Export to CSV for spreadsheet analysis:

```env
REPORT_FORMAT=csv
REPORT_OUTPUT_DIR=./reports
```

**Output:** `reports/job_report_2026-05-22_14-30-00.csv`

### Telegram Notifications
Real-time push notifications:

```env
REPORT_FORMAT=telegram
TELEGRAM_BOT_TOKEN=your_bot_token
TELEGRAM_CHAT_ID=your_chat_id
```

### All Formats
Enable everything at once:

```env
REPORT_FORMAT=all
```

## Project Structure

```
cmd/agent/          — entrypoint & scheduler loop
internal/config/    — env var loading & validation
internal/store/     — PostgreSQL models & queries
internal/scraper/   — job source scrapers (4 sources)
internal/filter/    — keyword pre-filter
internal/ai/        — Gemini Flash scoring
internal/reporter/  — HTML & CSV report generators
internal/notifier/  — Telegram notifications
migrations/         — SQL schema (auto-applied by Docker)
```

## Pipeline

**Scrape → Filter → Dedup → AI Score → Report**

1. Fetch jobs from 4 sources concurrently
2. Apply keyword filters (include/exclude rules)
3. Check database for duplicates (by source_id)
4. Score new jobs with Gemini Flash API (1-10)
5. Generate reports in configured format(s)
6. Weekly digest of top 10 jobs

## Tech Stack

**Core:** Go 1.25 · Docker · Docker Compose  
**Database:** PostgreSQL 16 · pgx/v5  
**AI:** Google Gemini Flash REST API  
**Reporting:** HTML templates · CSV · Telegram Bot API  
**Testing:** Go standard testing · table-driven tests

## Configuration

All configuration via environment variables. See `.env.example` for all options.

### Required
- `DB_URL` — PostgreSQL connection string

### Optional
- `REPORT_FORMAT` — `html` (default), `csv`, `telegram`, or `all`
- `REPORT_OUTPUT_DIR` — Where to save HTML/CSV (default: `./reports`)
- `TELEGRAM_BOT_TOKEN` — Required if `REPORT_FORMAT=telegram`
- `TELEGRAM_CHAT_ID` — Required if `REPORT_FORMAT=telegram`
- `GEMINI_API_KEY` — AI scoring (leave empty to skip)
- `SCAN_INTERVAL_HOURS` — How often to run (default: 6)

## Development

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Build binary
go build -o agent ./cmd/agent

# Run locally (requires running PostgreSQL)
./agent
```

## License

MIT

