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
	alerts alertStore
	emails emailQueue
}

func UnitOf(tx pgx.Tx) app.Unit {
	return unit{alerts: alertStore{db: tx}, emails: emailQueue{db: tx}}
}

func (u unit) Alerts() app.AlertStore {
	return u.alerts
}

func (u unit) Emails() app.EmailQueue {
	return u.emails
}
