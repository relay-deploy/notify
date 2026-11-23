package app

import (
	"context"

	"github.com/relay-deploy/notify/internal/domain"
)

type AlertStore interface {
	Find(ctx context.Context, appID string) (*domain.CrashAlert, error)
	Save(ctx context.Context, alert *domain.CrashAlert) error
	Close(ctx context.Context, appID string) error
	All(ctx context.Context) ([]*domain.CrashAlert, error)
}

type EmailQueue interface {
	Enqueue(ctx context.Context, email domain.Email) error
}

type ReleaseStore interface {
	MarkLive(ctx context.Context, appID, releaseID string) error
	Superseded(ctx context.Context, appID, releaseID string) (bool, error)
}

type Unit interface {
	LockApp(ctx context.Context, appID string) error
	Alerts() AlertStore
	Emails() EmailQueue
	Releases() ReleaseStore
}
