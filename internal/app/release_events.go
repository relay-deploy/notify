package app

import (
	"context"
	"time"

	"github.com/relay-deploy/notify/internal/domain"
)

type ReleaseEvents struct{}

func (ReleaseEvents) Succeeded(ctx context.Context, unit Unit, appID, projectID string, facts domain.Facts, at time.Time) error {
	if err := unit.LockApp(ctx, appID); err != nil {
		return err
	}
	if err := unit.Emails().Enqueue(ctx, domain.DeployedEmail(appID, projectID, facts, at)); err != nil {
		return err
	}
	if err := unit.Releases().MarkLive(ctx, appID, facts.ReleaseID); err != nil {
		return err
	}
	alert, err := unit.Alerts().Find(ctx, appID)
	if err != nil || alert == nil {
		return err
	}
	old, err := unit.Releases().Superseded(ctx, appID, alert.ReleaseID)
	if err != nil || !old {
		return err
	}
	return unit.Alerts().Close(ctx, appID)
}

func (ReleaseEvents) Failed(ctx context.Context, unit Unit, appID, projectID string, facts domain.Facts, at time.Time) error {
	return unit.Emails().Enqueue(ctx, domain.DeployFailedEmail(appID, projectID, facts, at))
}
