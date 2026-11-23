package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/relay-deploy/notify/internal/domain"
)

const alertColumns = `app_id::text, project_id::text, release_id::text, opened_at, last_crash_at, crash_count,
    last_exit_code, ran_out_of_memory, last_notified_at, last_notified_crash_count, recovered_at`

type alertStore struct {
	db queryer
}

func (s alertStore) Find(ctx context.Context, appID string) (*domain.CrashAlert, error) {
	rows, err := s.db.Query(ctx, `SELECT `+alertColumns+` FROM crash_alerts WHERE app_id = $1::uuid FOR UPDATE`, appID)
	if err != nil {
		return nil, err
	}
	alert, err := pgx.CollectOneRow(rows, scanAlert)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return alert, err
}

func (s alertStore) Save(ctx context.Context, alert *domain.CrashAlert) error {
	_, err := s.db.Exec(ctx, `INSERT INTO crash_alerts (app_id, project_id, release_id, opened_at, last_crash_at,
    crash_count, last_exit_code, ran_out_of_memory, last_notified_at, last_notified_crash_count, recovered_at)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (app_id) DO UPDATE SET
    project_id = excluded.project_id,
    release_id = excluded.release_id,
    opened_at = excluded.opened_at,
    last_crash_at = excluded.last_crash_at,
    crash_count = excluded.crash_count,
    last_exit_code = excluded.last_exit_code,
    ran_out_of_memory = excluded.ran_out_of_memory,
    last_notified_at = excluded.last_notified_at,
    last_notified_crash_count = excluded.last_notified_crash_count,
    recovered_at = excluded.recovered_at`,
		alert.AppID, alert.ProjectID, alert.ReleaseID, alert.OpenedAt, alert.LastCrashAt, alert.CrashCount,
		alert.LastExitCode, alert.RanOutOfMemory, alert.LastNotifiedAt, alert.LastNotifiedCrashCount, alert.RecoveredAt)
	return err
}

func (s alertStore) Close(ctx context.Context, appID string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM crash_alerts WHERE app_id = $1::uuid`, appID)
	return err
}

func (s alertStore) All(ctx context.Context) ([]*domain.CrashAlert, error) {
	rows, err := s.db.Query(ctx, `SELECT `+alertColumns+` FROM crash_alerts ORDER BY opened_at FOR UPDATE`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanAlert)
}

func scanAlert(row pgx.CollectableRow) (*domain.CrashAlert, error) {
	var alert domain.CrashAlert
	err := row.Scan(&alert.AppID, &alert.ProjectID, &alert.ReleaseID, &alert.OpenedAt, &alert.LastCrashAt,
		&alert.CrashCount, &alert.LastExitCode, &alert.RanOutOfMemory, &alert.LastNotifiedAt,
		&alert.LastNotifiedCrashCount, &alert.RecoveredAt)
	return &alert, err
}
