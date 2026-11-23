package app

import (
	"context"
	"time"

	"github.com/relay-deploy/notify/internal/domain"
)

type ReleaseEvents struct{}

func (ReleaseEvents) Succeeded(ctx context.Context, unit Unit, appID, projectID string, facts domain.Facts, at time.Time) error {
	if err := unit.Emails().Enqueue(ctx, domain.DeployedEmail(appID, projectID, facts, at)); err != nil {
		return err
	}
	alert, err := unit.Alerts().Find(ctx, appID)
	if err != nil || alert == nil || alert.ReleaseID == facts.ReleaseID {
		return err
	}
	return unit.Alerts().Close(ctx, appID)
}

func (ReleaseEvents) Failed(ctx context.Context, unit Unit, appID, projectID string, facts domain.Facts, at time.Time) error {
	return unit.Emails().Enqueue(ctx, domain.DeployFailedEmail(appID, projectID, facts, at))
}
