package app

import (
	"context"
	"time"

	"github.com/relay-deploy/chassis/go/logging"
	"github.com/relay-deploy/notify/internal/domain"
)

var alertPassLogger = logging.New("notify.alert_pass")

type Transactor interface {
	InUnit(ctx context.Context, work func(Unit) error) error
}

type AlertPass struct {
	Transactor       Transactor
	Quiet            time.Duration
	ReminderInterval time.Duration
	Expiry           time.Duration
}

func (p AlertPass) Run(ctx context.Context, now time.Time) error {
	return p.Transactor.InUnit(ctx, func(unit Unit) error {
		alerts, err := unit.Alerts().All(ctx)
		if err != nil {
			return err
		}
		for _, alert := range alerts {
			if err := p.settle(ctx, unit, alert, now); err != nil {
				return err
			}
		}
		return nil
	})
}

func (p AlertPass) settle(ctx context.Context, unit Unit, alert *domain.CrashAlert, now time.Time) error {
	switch {
	case alert.RecoveryDue(now, p.Quiet):
		if err := unit.Emails().Enqueue(ctx, domain.RecoveredEmail(alert)); err != nil {
			return err
		}
		return unit.Alerts().Close(ctx, alert.AppID)
	case alert.Expired(now, p.Expiry):
		alertPassLogger.Info("Closed stale alert for app " + alert.AppID)
		return unit.Alerts().Close(ctx, alert.AppID)
	case alert.ReminderDue(now, p.ReminderInterval):
		if err := unit.Emails().Enqueue(ctx, domain.StillCrashingEmail(alert, now)); err != nil {
			return err
		}
		alert.Remind(now)
		return unit.Alerts().Save(ctx, alert)
	}
	return nil
}
