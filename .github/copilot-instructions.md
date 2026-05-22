# Copilot Instructions — Job Agent

> Place this file at `.github/copilot-instructions.md` in your project root.
> Copilot reads this automatically on every prompt in Agent Mode.

---

## Project Summary

An autonomous job-finding agent written in Go. It runs on a schedule inside
Docker, scrapes job listings from multiple sources, deduplicates them via
PostgreSQL, pre-filters by keyword, scores relevant jobs using the Gemini Flash
API, and delivers results to a personal Telegram Bot — without any manual
intervention.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.26 |
| Container | Docker + Docker Compose |
| Database | PostgreSQL 16 (Alpine) |
| DB Driver | `github.com/jackc/pgx/v5` |
| AI Scoring | Google Gemini Flash REST API (free tier) |
| Notification | Telegram Bot API (direct HTTP, no third-party SDK) |
| Scheduling | In-process loop using `time.Sleep` |
| HTTP Client | Go standard library `net/http` only |
| Config | Environment variables via `os.Getenv` |
| Testing | Go standard `testing` package, table-driven tests |

---

## Project Layout

Follow the standard Go project layout strictly.

```
job-agent/
├── .github/
│   └── copilot-instructions.md   ← this file
├── cmd/
│   └── agent/
│       └── main.go               # entrypoint, wires dependencies, runs scheduler loop
├── internal/
│   ├── config/
│   │   └── config.go             # loads and validates all env vars into a Config struct
│   ├── scraper/
│   │   ├── scraper.go            # Scraper interface + FetchAll() aggregator
│   │   ├── remotive.go           # Remotive public JSON API
│   │   ├── weworkremotely.go     # We Work Remotely RSS feed
│   │   ├── remoteok.go           # RemoteOK public JSON API
│   │   ├── himalayas.go          # Himalayas public JSON API
│   │   └── rss.go                # Generic RSS parser (LinkedIn, JobStreet, Glints)
│   ├── filter/
│   │   └── filter.go             # Keyword-based pre-filter, runs before any AI call
│   ├── ai/
│   │   └── scorer.go             # Gemini Flash API call, JSON response parsing, score 1–10
│   ├── store/
│   │   ├── postgres.go           # DB connection, queries, deduplication logic
│   │   └── models.go             # Job struct (shared domain model)
│   └── notifier/
│       └── telegram.go           # Telegram Bot API sender via net/http
├── migrations/
│   └── 001_init.sql              # jobs table schema, run automatically by Docker
├── docker-compose.yml
├── Dockerfile
├── .env.example
├── go.mod
├── go.sum
└── README.md
```

---

## Domain Model

All packages share this single Job struct. Do not duplicate it.

```go
// internal/store/models.go
package store

import "time"

type Job struct {
    ID        int
    SourceID  string    // unique external ID or URL hash — used for dedup
    Source    string    // "remotive" | "weworkremotely" | "remoteok" | "himalayas" | "rss"
    Title     string
    Company   string
    Location  string
    URL       string
    Tags      []string  // tech stack tags from the source
    AIScore   int       // 1–10, 0 means not yet scored
    AIReason  string    // short reasoning from Gemini
    Notified  bool
    CreatedAt time.Time
}
```

---

## Architecture Rules

These rules define how components interact. Always follow them.

1. **Scraper → Filter → Dedup → AI → Notify** — this is the only allowed flow order.
2. **Filter runs before AI** — never call the Gemini API on unfiltered jobs.
3. **Dedup runs before AI** — never score jobs already in the DB.
4. **AI scores only the shortlist** — keyword filter should reduce to ~10–20 jobs max per run.
5. **Notifier receives only scored jobs** — pass jobs with `AIScore >= 7` to Telegram.
6. **Scraper interface is the contract** — all scrapers implement `Scraper`, aggregated by `FetchAll()`.
7. **Store is the only package that touches PostgreSQL** — no raw SQL outside `internal/store`.
8. **Config is loaded once at startup** — pass `config.Config` struct into constructors, never call `os.Getenv` outside `internal/config`.

---

## Interfaces

### Scraper

```go
// internal/scraper/scraper.go
package scraper

import "github.com/fikran/job-agent/internal/store"

type Scraper interface {
    Name() string
    Fetch() ([]store.Job, error)
}

// FetchAll runs all scrapers concurrently and merges results
func FetchAll(scrapers []Scraper) []store.Job
```

### Store

```go
// internal/store/postgres.go
package store

type Store interface {
    FilterSeen(jobs []Job) ([]Job, error)  // returns only jobs not yet in DB
    Save(jobs []Job) error                  // upsert by source_id
    MarkNotified(ids []int) error
}
```

### Notifier

```go
// internal/notifier/telegram.go
package notifier

import "github.com/fikran/job-agent/internal/store"

type Notifier interface {
    Send(jobs []store.Job) error
}
```

---

## Config

All configuration comes from environment variables. No hardcoded values.

```go
// internal/config/config.go
package config

type Config struct {
    DBUrl               string
    TelegramBotToken    string
    TelegramChatID      string
    GeminiAPIKey        string
    ScanIntervalHours   int    // default: 6
}

func Load() (*Config, error)  // returns error if required fields are missing
```

Required env vars: `DB_URL`, `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`
Optional env vars: `GEMINI_API_KEY` (if empty, skip AI scoring), `SCAN_INTERVAL_HOURS`

---

## Code Style

### General

- Use Go 1.26 standard library where possible — prefer stdlib over third-party packages.
- Only allowed external dependencies: `pgx/v5` for PostgreSQL, `encoding/xml` for RSS (stdlib).
- No global variables or `init()` functions. Pass all dependencies via struct constructors.
- All exported types and functions must have a doc comment.
- Group imports: stdlib first, then external, separated by a blank line.

### Error Handling

Always wrap errors with context. Never swallow errors silently.

```go
// correct
if err != nil {
    return fmt.Errorf("scraper remotive: fetch jobs: %w", err)
}

// wrong — no context
if err != nil {
    return err
}

// wrong — swallowed
if err != nil {
    log.Println(err)
}
```

### Logging

Use `log/slog` (stdlib, available since Go 1.21). Structured logging only.

```go
slog.Info("scan complete", "new_jobs", len(fresh), "scored", len(scored), "notified", notified)
slog.Error("scraper failed", "source", s.Name(), "err", err)
```

### HTTP Calls

Use the standard `net/http` package with explicit timeouts. Never use `http.DefaultClient`.

```go
client := &http.Client{Timeout: 15 * time.Second}
```

### Concurrency

Use `sync.WaitGroup` + `errgroup` for concurrent scraper runs. Protect shared state with mutexes.

---

## Scheduler Loop (main.go pattern)

```go
func run(cfg *config.Config) error {
    // wire up dependencies
    store := store.New(cfg.DBUrl)
    scrapers := []scraper.Scraper{
        scraper.NewRemotive(),
        scraper.NewWeWorkRemotely(),
        scraper.NewRemoteOK(),
    }
    scorer := ai.NewScorer(cfg.GeminiAPIKey)
    notifier := notifier.NewTelegram(cfg.TelegramBotToken, cfg.TelegramChatID)
    interval := time.Duration(cfg.ScanIntervalHours) * time.Hour

    for {
        if err := runOnce(scrapers, store, scorer, notifier); err != nil {
            slog.Error("scan failed", "err", err)
        }
        slog.Info("next scan", "in", interval)
        time.Sleep(interval)
    }
}
```

---

## AI Scoring

### Gemini Flash REST endpoint

```
POST https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key={GEMINI_API_KEY}
```

### Prompt template

```
You are a job fit scorer. Rate this job 1–10 for a candidate with:
- 2–3 years Go (Golang) backend experience
- Strong: microservices, Saga pattern, SQL optimization, gRPC, REST API design
- Open to other backend languages (Java, Rust, Python) if solving similar problems
- Targeting remote or international roles (US or Japan preferred)
- Also open to Indonesia-based hybrid/onsite roles in Jakarta

Job Title: {title}
Company: {company}
Location: {location}
Tags: {tags}
Description (first 300 chars): {description}

Respond ONLY in valid JSON with no markdown or preamble:
{"score": 8, "reason": "Strong Golang microservices role, fully remote, aligns well."}
```

### Score routing

| Score | Action |
|---|---|
| 8–10 | Send to Telegram — label: 🟢 Strong Match |
| 6–7 | Send to Telegram — label: 🟡 Maybe |
| 1–5 | Save to DB only, no notification |
| Error | Log and skip, do not block the run |

---

## Keyword Filter Rules

Run this before any AI call. Case-insensitive matching against title + tags + description.

### Include (any match passes)

```
golang, " go ", go backend, backend engineer, backend developer,
platform engineer, microservices, api engineer, distributed systems,
grpc, rest api, software engineer backend
```

### Exclude (any match is rejected)

```
frontend, react, vue, angular, mobile, ios, android,
flutter, data scientist, machine learning, devops only,
qa engineer, manual tester, ux designer, ui designer
```

### Location rules

Accept: `remote`, `worldwide`, `asia`, `indonesia`, `japan`, `united states`, `singapore`
Reject: any role explicitly marked `onsite` outside Jakarta

---

## Database Schema

```sql
-- migrations/001_init.sql
CREATE TABLE IF NOT EXISTS jobs (
    id          SERIAL PRIMARY KEY,
    source_id   TEXT NOT NULL UNIQUE,
    source      TEXT NOT NULL,
    title       TEXT NOT NULL,
    company     TEXT,
    location    TEXT,
    url         TEXT NOT NULL,
    tags        TEXT[],
    ai_score    INT,
    ai_reason   TEXT,
    notified    BOOLEAN DEFAULT FALSE,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_jobs_source_id ON jobs (source_id);
CREATE INDEX IF NOT EXISTS idx_jobs_notified ON jobs (notified);
CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs (created_at);
```

---

## Telegram Message Format

```
🟢 [9/10] Senior Backend Engineer — Acme Corp
📍 Remote (Worldwide)
🛠  golang · microservices · grpc · postgresql
🔗 https://remotive.com/job/12345

💬 Strong Golang microservices match. Remote-first, aligns with US target.
```

---

## Testing Guidelines

- Write table-driven tests for `filter` and `ai` (scorer) packages — these have the most logic.
- Mock external HTTP calls in tests using `net/http/httptest`.
- Do not write tests that hit real external APIs or a real database.
- Test file naming: `filter_test.go` lives next to `filter.go` in the same package.

```go
// example table-driven test structure
func TestFilter(t *testing.T) {
    tests := []struct {
        name  string
        job   store.Job
        want  bool
    }{
        {"golang backend passes", store.Job{Title: "Golang Backend Engineer"}, true},
        {"frontend rejected", store.Job{Title: "React Frontend Developer"}, false},
        {"empty title rejected", store.Job{Title: ""}, false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := filter.IsRelevant(tt.job)
            if got != tt.want {
                t.Errorf("IsRelevant() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

---

## Docker

### Dockerfile (multi-stage)

```dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o agent ./cmd/agent

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/agent .
CMD ["./agent"]
```

### docker-compose.yml

```yaml
services:
  agent:
    build: .
    restart: unless-stopped
    env_file: .env
    depends_on:
      db:
        condition: service_healthy

  db:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_DB: jobagent
      POSTGRES_USER: agent
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U agent -d jobagent"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  pgdata:
```

---

## Environment Variables

```env
# .env.example

# Required
DB_PASSWORD=changeme
DB_URL=postgres://agent:changeme@db:5432/jobagent?sslmode=disable
TELEGRAM_BOT_TOKEN=        # from @BotFather on Telegram
TELEGRAM_CHAT_ID=          # your personal chat ID

# Optional
GEMINI_API_KEY=            # from aistudio.google.com — leave empty to skip AI scoring
SCAN_INTERVAL_HOURS=6      # how often to run, default 6
```

---

## Implementation Phases

Build in this order. Do not skip phases.

### Phase 1 — Core (no AI yet)
- [ ] `internal/config` — load and validate env vars
- [ ] `internal/store` — PostgreSQL connection, models, dedup, save
- [ ] `internal/scraper/remotive.go` — first scraper (JSON API, easiest)
- [ ] `internal/filter/filter.go` — keyword filter
- [ ] `internal/notifier/telegram.go` — Telegram sender
- [ ] `cmd/agent/main.go` — wire everything, scheduler loop
- [ ] `Dockerfile` + `docker-compose.yml`
- [ ] Run end-to-end: scrape → filter → notify via Telegram

### Phase 2 — More Sources
- [ ] `internal/scraper/weworkremotely.go` — RSS
- [ ] `internal/scraper/remoteok.go` — JSON API
- [ ] `internal/scraper/himalayas.go` — JSON API
- [ ] `internal/scraper/rss.go` — generic RSS for LinkedIn, JobStreet, Glints

### Phase 3 — AI Scoring
- [ ] `internal/ai/scorer.go` — Gemini Flash API integration
- [ ] Wire scorer into main loop between filter and notifier
- [ ] Add score-based routing (🟢 / 🟡 labels in Telegram)

### Phase 4 — Polish
- [ ] Concurrent `FetchAll()` using goroutines + errgroup
- [ ] Weekly digest summary to Telegram
- [ ] Graceful shutdown on SIGTERM
- [ ] Unit tests for filter and scorer packages

---

## What Copilot Should NOT Do

- Do not use `gorm` or any ORM — write raw SQL with `pgx/v5`.
- Do not use `cobra` or `viper` — config from env vars only.
- Do not use `gin`, `fiber`, or any HTTP framework — this is not a web server.
- Do not use `cron` libraries — use `time.Sleep` loop in main.
- Do not introduce new dependencies without a clear reason.
- Do not use `log.Fatal` inside library packages — only in `main.go`.
- Do not return `nil, nil` from functions that should always return a value.