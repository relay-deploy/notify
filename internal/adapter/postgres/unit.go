package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/relay-deploy/notify/internal/app"
)

type queryer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row
}

type unit struct {
	db       queryer
	alerts   alertStore
	emails   emailQueue
	releases releaseStore
}

func UnitOf(tx pgx.Tx) app.Unit {
	return unit{db: tx, alerts: alertStore{db: tx}, emails: emailQueue{db: tx}, releases: releaseStore{db: tx}}
}

func (u unit) LockApp(ctx context.Context, appID string) error {
	_, err := u.db.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('notify.app.' || $1::uuid::text, 0))`, appID)
	return err
}

func (u unit) Alerts() app.AlertStore {
	return u.alerts
}

func (u unit) Emails() app.EmailQueue {
	return u.emails
}

func (u unit) Releases() app.ReleaseStore {
	return u.releases
}
