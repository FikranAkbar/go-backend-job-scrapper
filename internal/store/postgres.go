// Package store provides the domain model and PostgreSQL persistence layer.
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store defines the contract for all database operations.
type Store interface {
	// FilterSeen returns only jobs whose source_id is not yet in the DB.
	FilterSeen(jobs []Job) ([]Job, error)
	// Save upserts jobs by source_id.
	Save(jobs []Job) error
	// MarkNotified sets notified=true for the given job IDs.
	MarkNotified(ids []int) error
	// TopJobs returns the top-scored jobs created since the given time, ordered by ai_score DESC.
	TopJobs(since time.Time, limit int) ([]Job, error)
	// Close releases the connection pool.
	Close()
}

// PostgresStore implements Store using pgx/v5 connection pool.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// New creates a new PostgresStore, connecting to the given DB URL.
// Returns an error if the connection cannot be established.
func New(ctx context.Context, dbURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("store: connect to postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("store: ping postgres: %w", err)
	}
	return &PostgresStore{pool: pool}, nil
}

// Close releases the underlying connection pool.
func (s *PostgresStore) Close() {
	s.pool.Close()
}

// FilterSeen returns only jobs whose source_id does not exist in the DB.
func (s *PostgresStore) FilterSeen(jobs []Job) ([]Job, error) {
	if len(jobs) == 0 {
		return []Job{}, nil
	}

	ids := make([]string, len(jobs))
	for i, j := range jobs {
		ids[i] = j.SourceID
	}

	rows, err := s.pool.Query(context.Background(),
		`SELECT source_id FROM jobs WHERE source_id = ANY($1)`, ids)
	if err != nil {
		return nil, fmt.Errorf("store: filter seen: query: %w", err)
	}
	defer rows.Close()

	seen := make(map[string]bool)
	for rows.Next() {
		var sid string
		if err := rows.Scan(&sid); err != nil {
			return nil, fmt.Errorf("store: filter seen: scan: %w", err)
		}
		seen[sid] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: filter seen: rows: %w", err)
	}

	fresh := make([]Job, 0, len(jobs))
	for _, j := range jobs {
		if !seen[j.SourceID] {
			fresh = append(fresh, j)
		}
	}
	return fresh, nil
}

// Save upserts jobs into the database by source_id.
func (s *PostgresStore) Save(jobs []Job) error {
	if len(jobs) == 0 {
		return nil
	}

	ctx := context.Background()
	for _, j := range jobs {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO jobs (source_id, source, title, company, location, url, description, tags, ai_score, ai_reason, notified)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
			ON CONFLICT (source_id) DO UPDATE
			  SET ai_score   = EXCLUDED.ai_score,
			      ai_reason  = EXCLUDED.ai_reason,
			      notified   = EXCLUDED.notified,
			      description = EXCLUDED.description`,
			j.SourceID, j.Source, j.Title, j.Company, j.Location, j.URL,
			j.Description, j.Tags, j.AIScore, j.AIReason, j.Notified,
		)
		if err != nil {
			return fmt.Errorf("store: save job %q: %w", j.SourceID, err)
		}
	}
	return nil
}

// MarkNotified sets notified=true for the given job IDs.
func (s *PostgresStore) MarkNotified(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := s.pool.Exec(context.Background(),
		`UPDATE jobs SET notified=true WHERE id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("store: mark notified: %w", err)
	}
	return nil
}

// TopJobs returns up to limit top-scored (ai_score > 0) jobs created since the given time,
// ordered by ai_score descending.
func (s *PostgresStore) TopJobs(since time.Time, limit int) ([]Job, error) {
	rows, err := s.pool.Query(context.Background(), `
		SELECT id, source_id, source, title, company, location, url, description, tags, ai_score, ai_reason, notified, created_at
		FROM jobs
		WHERE created_at >= $1 AND ai_score > 0
		ORDER BY ai_score DESC
		LIMIT $2`, since, limit)
	if err != nil {
		return nil, fmt.Errorf("store: top jobs: query: %w", err)
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(
			&j.ID, &j.SourceID, &j.Source, &j.Title, &j.Company,
			&j.Location, &j.URL, &j.Description, &j.Tags, &j.AIScore, &j.AIReason,
			&j.Notified, &j.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("store: top jobs: scan: %w", err)
		}
		jobs = append(jobs, j)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: top jobs: rows: %w", err)
	}
	return jobs, nil
}

