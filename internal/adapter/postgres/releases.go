package postgres

import "context"

type releaseStore struct {
	db queryer
}

func (s releaseStore) MarkLive(ctx context.Context, appID, releaseID string) error {
	_, err := s.db.Exec(ctx, `INSERT INTO live_releases AS live (app_id, release_id) VALUES ($1::uuid, $2::uuid)
ON CONFLICT (app_id) DO UPDATE SET release_id = excluded.release_id
WHERE live.release_id < excluded.release_id`, appID, releaseID)
	return err
}

func (s releaseStore) Superseded(ctx context.Context, appID, releaseID string) (bool, error) {
	var superseded bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM live_releases WHERE app_id = $1::uuid AND release_id > $2::uuid)`,
		appID, releaseID).Scan(&superseded)
	return superseded, err
}
