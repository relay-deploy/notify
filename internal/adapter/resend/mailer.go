package resend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/relay-deploy/chassis/go/fault"
	"github.com/relay-deploy/notify/internal/app"
)

const (
	requestTimeout       = 10 * time.Second
	answerLimit          = 64 << 10
	concurrentRequests   = "concurrent_idempotent_requests"
	unreachableErrorCode = "resend.unreachable"
)

type Mailer struct {
	client   *http.Client
	endpoint string
	apiKey   string
	from     string
	to       string
}

func NewMailer(baseURL, apiKey, from, to string) *Mailer {
	return &Mailer{
		client:   &http.Client{Timeout: requestTimeout},
		endpoint: strings.TrimRight(baseURL, "/") + "/emails",
		apiKey:   apiKey,
		from:     from,
		to:       to,
	}
}

type message struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
	Text    string   `json:"text"`
}

type answer struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

func (m *Mailer) Send(ctx context.Context, idempotencyKey string, r app.Rendered) (string, error) {
	body, err := json.Marshal(message{From: m.from, To: []string{m.to}, Subject: r.Subject, HTML: r.HTML, Text: r.Text})
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, m.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+m.apiKey)
	request.Header.Set("Idempotency-Key", idempotencyKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := m.client.Do(request)
	if err != nil {
		return "", fault.New(fault.Unavailable, unreachableErrorCode, "Cannot reach Resend: "+err.Error())
	}
	defer response.Body.Close()
	reply := readAnswer(response.Body)
	if response.StatusCode == http.StatusOK {
		return reply.ID, nil
	}
	return "", refusal(response.StatusCode, reply)
}

func readAnswer(body io.Reader) answer {
	var reply answer
	raw, err := io.ReadAll(io.LimitReader(body, answerLimit))
	if err != nil || json.Unmarshal(raw, &reply) != nil {
		return answer{}
	}
	return reply
}

func refusal(statusCode int, reply answer) error {
	reason := reply.Name
	if reason == "" {
		reason = strconv.Itoa(statusCode)
	}
	text := fmt.Sprintf("Resend answered %d %s", statusCode, reason)
	if reply.Message != "" {
		text += ": " + reply.Message
	}
	if retryable(statusCode, reply.Name) {
		return fault.New(fault.Unavailable, "resend."+reason, text)
	}
	return fault.New(fault.Validation, "resend."+reason, text)
}

func retryable(statusCode int, name string) bool {
	return statusCode == http.StatusTooManyRequests ||
		statusCode >= http.StatusInternalServerError ||
		(statusCode == http.StatusConflict && name == concurrentRequests)
}
