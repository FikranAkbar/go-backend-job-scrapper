// Package notifier delivers job notifications via the Telegram Bot API.
package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/FikranAkbar/go-backend-job-scrapper/internal/reporter"
	"github.com/FikranAkbar/go-backend-job-scrapper/internal/store"
)

const telegramAPIBase = "https://api.telegram.org/bot"

// Notifier is the contract for sending job notifications.
type Notifier interface {
	// Send delivers the given jobs to the notification channel.
	Send(jobs []store.Job) error
	// SendDigest delivers a weekly summary of top-scored jobs.
	SendDigest(jobs []store.Job) error
	// Generate creates a notification report (implements reporter.Reporter interface).
	Generate(jobs []store.Job, stats reporter.ReportStats) error
	// GenerateDigest creates a weekly digest report (implements reporter.Reporter interface).
	GenerateDigest(jobs []store.Job) error
}

// TelegramNotifier sends job notifications via the Telegram Bot API.
type TelegramNotifier struct {
	token  string
	chatID string
	client *http.Client
}

// NewTelegram creates a new TelegramNotifier.
func NewTelegram(token, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		token:  token,
		chatID: chatID,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Send delivers each job as a Telegram message.
func (t *TelegramNotifier) Send(jobs []store.Job) error {
	for _, j := range jobs {
		msg := formatMessage(j)
		if err := t.sendMessage(msg); err != nil {
			return fmt.Errorf("notifier telegram: send job %q: %w", j.SourceID, err)
		}
	}
	return nil
}

// SendDigest delivers a weekly summary of the top-scored jobs as a single Telegram message.
func (t *TelegramNotifier) SendDigest(jobs []store.Job) error {
	if len(jobs) == 0 {
		return nil
	}
	if err := t.sendMessage(formatDigest(jobs)); err != nil {
		return fmt.Errorf("notifier telegram: send digest: %w", err)
	}
	return nil
}

// Generate implements reporter.Reporter interface - sends jobs as Telegram messages.
func (t *TelegramNotifier) Generate(jobs []store.Job, stats reporter.ReportStats) error {
	// Filter only high-scoring jobs (>= 6)
	var toSend []store.Job
	for _, j := range jobs {
		if j.AIScore >= 6 {
			toSend = append(toSend, j)
		}
	}
	return t.Send(toSend)
}

// GenerateDigest implements reporter.Reporter interface - sends weekly digest via Telegram.
func (t *TelegramNotifier) GenerateDigest(jobs []store.Job) error {
	return t.SendDigest(jobs)
}

func (t *TelegramNotifier) sendMessage(text string) error {
	payload, err := json.Marshal(map[string]any{
		"chat_id":    t.chatID,
		"text":       text,
		"parse_mode": "HTML",
	})
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	url := telegramAPIBase + t.token + "/sendMessage"
	resp, err := t.client.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram returned status %d", resp.StatusCode)
	}
	return nil
}

// formatMessage builds the Telegram notification text for a single job.
func formatMessage(j store.Job) string {
	label := scoreLabel(j.AIScore)
	tags := strings.Join(j.Tags, " · ")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s [%d/10] %s — %s\n", label, j.AIScore, j.Title, j.Company))
	sb.WriteString(fmt.Sprintf("📍 %s\n", j.Location))
	if tags != "" {
		sb.WriteString(fmt.Sprintf("🛠  %s\n", tags))
	}
	sb.WriteString(fmt.Sprintf("🔗 %s\n", j.URL))
	if j.AIReason != "" {
		sb.WriteString(fmt.Sprintf("\n💬 %s", j.AIReason))
	}
	return sb.String()
}

// formatDigest builds a weekly summary message from the top-scored jobs.
func formatDigest(jobs []store.Job) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 <b>Weekly Job Digest</b> — Top %d matches this week\n\n", len(jobs)))
	for i, j := range jobs {
		label := scoreLabel(j.AIScore)
		sb.WriteString(fmt.Sprintf(
			"%d. %s [%d/10] <a href=\"%s\">%s</a> — %s\n   📍 %s\n",
			i+1, label, j.AIScore, j.URL, j.Title, j.Company, j.Location,
		))
	}
	return sb.String()
}

func scoreLabel(score int) string {
	switch {
	case score >= 8:
		return "🟢"
	case score >= 6:
		return "🟡"
	default:
		return "⚪"
	}
}

