package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/relay-deploy/notify/internal/app"
)

type transactor struct {
	pool *pgxpool.Pool
}

func NewTransactor(pool *pgxpool.Pool) app.Transactor {
	return transactor{pool: pool}
}

func (t transactor) InUnit(ctx context.Context, work func(app.Unit) error) error {
	return pgx.BeginFunc(ctx, t.pool, func(tx pgx.Tx) error {
		return work(UnitOf(tx))
	})
}
