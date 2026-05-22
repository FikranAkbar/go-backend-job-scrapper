// Package store provides the domain model and PostgreSQL persistence layer.
package store

import "time"

// Job represents a single job listing fetched from any source.
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

