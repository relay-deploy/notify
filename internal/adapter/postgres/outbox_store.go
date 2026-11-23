package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/relay-deploy/notify/internal/app"
	"github.com/relay-deploy/notify/internal/domain"
)

const claimDueSQL = `WITH claimed AS (
    UPDATE emails SET next_attempt_at = $2, attempts = attempts + 1
    WHERE id IN (
        SELECT id FROM emails
        WHERE status = 'pending' AND next_attempt_at <= $1
        ORDER BY created_at
        LIMIT $3
        FOR UPDATE SKIP LOCKED)
    RETURNING id, kind, app_id, project_id, facts, occurred_at, subject, html, text_body, attempts, created_at)
SELECT id::text, kind, app_id::text, project_id::text, facts, occurred_at, subject, html, text_body, attempts, created_at
FROM claimed
ORDER BY created_at`

type emailStore struct {
	db queryer
}

func NewEmailStore(pool *pgxpool.Pool) app.EmailStore {
	return emailStore{db: pool}
}

func (s emailStore) ClaimDue(ctx context.Context, now time.Time, lease time.Duration, limit int) ([]app.PendingEmail, error) {
	rows, err := s.db.Query(ctx, claimDueSQL, now, now.Add(lease), limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanPendingEmail)
}

func (s emailStore) SaveRendered(ctx context.Context, id string, r app.Rendered) error {
	_, err := s.db.Exec(ctx, `UPDATE emails SET subject = $2, html = $3, text_body = $4
WHERE id = $1::uuid AND status = 'pending'`, id, r.Subject, r.HTML, r.Text)
	return err
}

func (s emailStore) MarkSent(ctx context.Context, id, providerID string, at time.Time) error {
	_, err := s.db.Exec(ctx, `UPDATE emails SET status = 'sent', sent_at = $3, provider_id = $2
WHERE id = $1::uuid AND status = 'pending'`, id, providerID, at)
	return err
}

func (s emailStore) RetryAt(ctx context.Context, id string, at time.Time, reason string) error {
	_, err := s.db.Exec(ctx, `UPDATE emails SET next_attempt_at = $2, last_error = $3
WHERE id = $1::uuid AND status = 'pending'`, id, at, reason)
	return err
}

func (s emailStore) Abandon(ctx context.Context, id, reason string) error {
	_, err := s.db.Exec(ctx, `UPDATE emails SET status = 'abandoned', last_error = $2
WHERE id = $1::uuid AND status = 'pending'`, id, reason)
	return err
}

func scanPendingEmail(row pgx.CollectableRow) (app.PendingEmail, error) {
	var pending app.PendingEmail
	var kind string
	var facts []byte
	var subject, html, text *string
	err := row.Scan(&pending.ID, &kind, &pending.Email.AppID, &pending.Email.ProjectID, &facts, &pending.Email.At,
		&subject, &html, &text, &pending.Attempts, &pending.CreatedAt)
	if err != nil {
		return pending, err
	}
	pending.Email.Kind = domain.EmailKind(kind)
	if err := json.Unmarshal(facts, &pending.Email.Facts); err != nil {
		return pending, err
	}
	if subject != nil && html != nil && text != nil {
		pending.Rendered = &app.Rendered{Subject: *subject, HTML: *html, Text: *text}
	}
	return pending, nil
}
