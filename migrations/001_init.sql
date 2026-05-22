-- migrations/001_init.sql
-- Initial schema for the job agent.

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
CREATE INDEX IF NOT EXISTS idx_jobs_notified  ON jobs (notified);
CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs (created_at);

