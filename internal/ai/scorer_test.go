package ai_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FikranAkbar/go-backend-job-scrapper/internal/ai"
	"github.com/FikranAkbar/go-backend-job-scrapper/internal/store"
)

// geminiResponse builds a fake Gemini API response wrapping the given text.
func geminiResponse(text string) map[string]any {
	return map[string]any{
		"candidates": []map[string]any{
			{
				"content": map[string]any{
					"parts": []map[string]any{
						{"text": text},
					},
				},
			},
		},
	}
}

func TestScore(t *testing.T) {
	tests := []struct {
		name        string
		handler     http.HandlerFunc
		wantScore   int
		wantReason  string
		wantErrFrag string // non-empty means we expect an error containing this substring
	}{
		{
			name: "valid score 8",
			handler: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(geminiResponse(`{"score": 8, "reason": "Great match"}`))
			},
			wantScore:  8,
			wantReason: "Great match",
		},
		{
			name: "valid score boundary 1",
			handler: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(geminiResponse(`{"score": 1, "reason": "Poor match"}`))
			},
			wantScore:  1,
			wantReason: "Poor match",
		},
		{
			name: "valid score boundary 10",
			handler: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(geminiResponse(`{"score": 10, "reason": "Perfect match"}`))
			},
			wantScore:  10,
			wantReason: "Perfect match",
		},
		{
			name: "non-200 status returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
			},
			wantErrFrag: "gemini returned status 429",
		},
		{
			name: "empty candidates returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(map[string]any{"candidates": []any{}})
			},
			wantErrFrag: "empty candidates",
		},
		{
			name: "invalid json text returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(geminiResponse(`not json at all`))
			},
			wantErrFrag: "parse score JSON",
		},
		{
			name: "score too high returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(geminiResponse(`{"score": 15, "reason": "invalid"}`))
			},
			wantErrFrag: "score 15 out of range",
		},
		{
			name: "score too low returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(geminiResponse(`{"score": 0, "reason": "invalid"}`))
			},
			wantErrFrag: "score 0 out of range",
		},
		{
			name: "malformed outer json returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{invalid`))
			},
			wantErrFrag: "decode gemini response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			scorer := ai.NewScorerWithEndpoint(srv.URL, "test-key")
			score, reason, err := scorer.Score(store.Job{
				Title:    "Golang Backend Engineer",
				Company:  "Acme Corp",
				Location: "Remote",
				Tags:     []string{"golang", "grpc"},
			})

			if tt.wantErrFrag != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErrFrag)
				}
				if !strings.Contains(err.Error(), tt.wantErrFrag) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantErrFrag)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if score != tt.wantScore {
				t.Errorf("Score() score = %d, want %d", score, tt.wantScore)
			}
			if reason != tt.wantReason {
				t.Errorf("Score() reason = %q, want %q", reason, tt.wantReason)
			}
		})
	}
}

func TestScoreNoAPIKey(t *testing.T) {
	scorer := ai.NewScorer("")
	score, reason, err := scorer.Score(store.Job{Title: "Golang Engineer"})
	if err != nil {
		t.Fatalf("unexpected error when apiKey is empty: %v", err)
	}
	if score != 0 {
		t.Errorf("Score() = %d, want 0 when no API key", score)
	}
	if reason != "" {
		t.Errorf("Score() reason = %q, want empty when no API key", reason)
	}
}

func TestScoreAll(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(geminiResponse(`{"score": 7, "reason": "Good match"}`))
	}))
	defer srv.Close()

	scorer := ai.NewScorerWithEndpoint(srv.URL, "test-key")
	jobs := []store.Job{
		{Title: "Golang Engineer", SourceID: "src-1"},
		{Title: "Backend Developer", SourceID: "src-2"},
		{Title: "Platform Engineer", SourceID: "src-3"},
	}

	result := scorer.ScoreAll(jobs)

	if len(result) != 3 {
		t.Fatalf("ScoreAll returned %d jobs, want 3", len(result))
	}
	for i, j := range result {
		if j.AIScore != 7 {
			t.Errorf("job[%d].AIScore = %d, want 7", i, j.AIScore)
		}
		if j.AIReason != "Good match" {
			t.Errorf("job[%d].AIReason = %q, want %q", i, j.AIReason, "Good match")
		}
	}
	if callCount != 3 {
		t.Errorf("expected 3 API calls, got %d", callCount)
	}
}

func TestScoreAllErrorsDontAbort(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 2 {
			// Simulate API error on second job
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(geminiResponse(`{"score": 9, "reason": "Excellent"}`))
	}))
	defer srv.Close()

	scorer := ai.NewScorerWithEndpoint(srv.URL, "test-key")
	jobs := []store.Job{
		{Title: "Golang Engineer", SourceID: "src-1"},
		{Title: "Backend Developer", SourceID: "src-2"}, // will error
		{Title: "Platform Engineer", SourceID: "src-3"},
	}

	result := scorer.ScoreAll(jobs)

	if len(result) != 3 {
		t.Fatalf("ScoreAll returned %d jobs, want 3 (errors must not abort)", len(result))
	}
	// Job 0 and 2 should be scored; job 1 should have score 0
	if result[0].AIScore != 9 {
		t.Errorf("job[0].AIScore = %d, want 9", result[0].AIScore)
	}
	if result[1].AIScore != 0 {
		t.Errorf("job[1].AIScore = %d, want 0 (error case)", result[1].AIScore)
	}
	if result[2].AIScore != 9 {
		t.Errorf("job[2].AIScore = %d, want 9", result[2].AIScore)
	}
}

func TestScoreAllNoAPIKey(t *testing.T) {
	scorer := ai.NewScorer("")
	jobs := []store.Job{
		{Title: "Golang Engineer"},
		{Title: "Backend Developer"},
	}
	result := scorer.ScoreAll(jobs)
	if len(result) != 2 {
		t.Fatalf("ScoreAll returned %d jobs, want 2", len(result))
	}
	for i, j := range result {
		if j.AIScore != 0 {
			t.Errorf("job[%d].AIScore = %d, want 0 when no API key", i, j.AIScore)
		}
	}
}

