// Package ai provides AI-based job scoring using the Google Gemini Flash REST API.
package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/FikranAkbar/go-backend-job-scrapper/internal/store"
)

const geminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent"

// Scorer scores jobs using the Gemini Flash API.
type Scorer struct {
	apiKey   string
	endpoint string
	client   *http.Client
}

// NewScorer creates a new Scorer. If apiKey is empty, Score() is a no-op.
func NewScorer(apiKey string) *Scorer {
	return &Scorer{
		apiKey:   apiKey,
		endpoint: geminiEndpoint,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// NewScorerWithEndpoint creates a Scorer with a custom endpoint URL.
// Intended for use in tests (e.g. with net/http/httptest).
func NewScorerWithEndpoint(endpoint, apiKey string) *Scorer {
	return &Scorer{
		apiKey:   apiKey,
		endpoint: endpoint,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// scoreResponse is the JSON structure returned by Gemini.
type scoreResponse struct {
	Score  int    `json:"score"`
	Reason string `json:"reason"`
}

// Score sends the job to Gemini and returns its AI score (1–10) and reasoning.
// Returns (0, "", nil) if no API key is configured.
func (s *Scorer) Score(j store.Job) (int, string, error) {
	if s.apiKey == "" {
		return 0, "", nil
	}

	prompt := buildPrompt(j)

	requestBody, err := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]any{
					{"text": prompt},
				},
			},
		},
	})
	if err != nil {
		return 0, "", fmt.Errorf("ai scorer: marshal request: %w", err)
	}

	url := s.endpoint + "?key=" + s.apiKey
	resp, err := s.client.Post(url, "application/json", bytes.NewReader(requestBody))
	if err != nil {
		return 0, "", fmt.Errorf("ai scorer: post to gemini: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("ai scorer: gemini returned status %d", resp.StatusCode)
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return 0, "", fmt.Errorf("ai scorer: decode gemini response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return 0, "", fmt.Errorf("ai scorer: empty candidates in gemini response")
	}

	rawText := geminiResp.Candidates[0].Content.Parts[0].Text
	rawText = strings.TrimSpace(rawText)

	var scored scoreResponse
	if err := json.Unmarshal([]byte(rawText), &scored); err != nil {
		return 0, "", fmt.Errorf("ai scorer: parse score JSON %q: %w", rawText, err)
	}
	if scored.Score < 1 || scored.Score > 10 {
		return 0, "", fmt.Errorf("ai scorer: score %d out of range 1–10", scored.Score)
	}

	return scored.Score, scored.Reason, nil
}

// ScoreAll scores a slice of jobs and returns them with AIScore and AIReason set.
// Individual errors are logged and do not abort the run — the original job (score=0) is returned on error.
func (s *Scorer) ScoreAll(jobs []store.Job) []store.Job {
	out := make([]store.Job, len(jobs))
	for i, j := range jobs {
		score, reason, err := s.Score(j)
		if err != nil {
			slog.Error("ai scorer: score failed", "title", j.Title, "source_id", j.SourceID, "err", err)
			j.AIScore = 0
			j.AIReason = ""
		} else {
			j.AIScore = score
			j.AIReason = reason
		}
		out[i] = j
	}
	return out
}

func buildPrompt(j store.Job) string {
	tags := strings.Join(j.Tags, ", ")
	desc := j.Description
	if len([]rune(desc)) > 500 {
		desc = string([]rune(desc)[:500]) + "..."
	}
	return fmt.Sprintf(`You are a job fit scorer. Rate this job 1–10 for a candidate with:
- 2–3 years Go (Golang) backend experience
- Strong: microservices, Saga pattern, SQL optimization, gRPC, REST API design
- Open to other backend languages (Java, Rust, Python) if solving similar problems
- Targeting remote or international roles (US or Japan preferred)
- Also open to Indonesia-based hybrid/onsite roles in Jakarta

Job Title: %s
Company: %s
Location: %s
Tags: %s
Description (first 500 chars): %s

Respond ONLY in valid JSON with no markdown or preamble:
{"score": 8, "reason": "Strong Golang microservices role, fully remote, aligns well."}`,
		j.Title, j.Company, j.Location, tags, desc)
}
