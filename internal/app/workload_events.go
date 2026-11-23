package app

import (
	"context"
	"time"

	"github.com/relay-deploy/notify/internal/domain"
)

type WorkloadEvents struct {
	NewLoopAfter time.Duration
	Quiet        time.Duration
}

func (w WorkloadEvents) Crashed(ctx context.Context, unit Unit, crash domain.Crash) error {
	alert, err := unit.Alerts().Find(ctx, crash.AppID)
	if err != nil {
		return err
	}
	switch {
	case alert == nil:
		return w.open(ctx, unit, crash)
	case alert.RecoveryDue(crash.At, w.Quiet):
		if err := unit.Emails().Enqueue(ctx, domain.RecoveredEmail(alert)); err != nil {
			return err
		}
		return w.reopen(ctx, unit, crash)
	case alert.IsStale(crash, w.NewLoopAfter):
		return w.reopen(ctx, unit, crash)
	}
	alert.Add(crash)
	return unit.Alerts().Save(ctx, alert)
}

func (w WorkloadEvents) reopen(ctx context.Context, unit Unit, crash domain.Crash) error {
	if err := unit.Alerts().Close(ctx, crash.AppID); err != nil {
		return err
	}
	return w.open(ctx, unit, crash)
}

func (w WorkloadEvents) open(ctx context.Context, unit Unit, crash domain.Crash) error {
	alert := domain.OpenAlert(crash)
	if err := unit.Alerts().Save(ctx, alert); err != nil {
		return err
	}
	return unit.Emails().Enqueue(ctx, domain.CrashedEmail(alert))
}

func (w WorkloadEvents) Recovered(ctx context.Context, unit Unit, appID string, at time.Time) error {
	alert, err := unit.Alerts().Find(ctx, appID)
	if err != nil || alert == nil {
		return err
	}
	alert.Recover(at)
	return unit.Alerts().Save(ctx, alert)
}

func (w WorkloadEvents) Stopped(ctx context.Context, unit Unit, appID string) error {
	return unit.Alerts().Close(ctx, appID)
}
