package postgres

import (
	"context"
	"encoding/json"

	"github.com/relay-deploy/notify/internal/domain"
)

type emailQueue struct {
	db queryer
}

func (q emailQueue) Enqueue(ctx context.Context, email domain.Email) error {
	facts, err := json.Marshal(email.Facts)
	if err != nil {
		return err
	}
	_, err = q.db.Exec(ctx, `INSERT INTO emails (kind, app_id, project_id, facts, occurred_at, created_at, next_attempt_at)
VALUES ($1, $2::uuid, $3::uuid, $4, $5, now(), now())`,
		string(email.Kind), email.AppID, email.ProjectID, facts, email.At)
	if err != nil {
		return err
	}
	_, err = q.db.Exec(ctx, `SELECT pg_notify('notify_emails', '')`)
	return err
}
