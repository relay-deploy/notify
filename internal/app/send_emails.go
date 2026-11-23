package app

import (
	"context"
	"fmt"
	"time"

	"github.com/relay-deploy/chassis/go/fault"
	"github.com/relay-deploy/chassis/go/logging"
	"github.com/relay-deploy/notify/internal/domain"
)

const (
	claimLease        = 2 * time.Minute
	claimLimit        = 20
	firstRetryDelay   = 30 * time.Second
	longestRetryDelay = 15 * time.Minute
	giveUpAfter       = 24 * time.Hour
)

var senderLogger = logging.New("notify.sender")

type PendingEmail struct {
	ID        string
	Email     domain.Email
	Rendered  *Rendered
	Attempts  int
	CreatedAt time.Time
}

type EmailStore interface {
	ClaimDue(ctx context.Context, now time.Time, lease time.Duration, limit int) ([]PendingEmail, error)
	SaveRendered(ctx context.Context, id string, r Rendered) error
	MarkSent(ctx context.Context, id, providerID string, at time.Time) error
	RetryAt(ctx context.Context, id string, at time.Time, reason string) error
	Abandon(ctx context.Context, id, reason string) error
}

type Directory interface {
	Names(ctx context.Context, appID, projectID string) (Names, error)
}

type Mailer interface {
	Send(ctx context.Context, idempotencyKey string, r Rendered) (string, error)
}

type SendEmails struct {
	Store             EmailStore
	Directory         Directory
	Composer          Composer
	Mailer            Mailer
	NameFallbackAfter time.Duration
}

func (s SendEmails) SendDue(ctx context.Context, now time.Time) error {
	for {
		batch, err := s.Store.ClaimDue(ctx, now, claimLease, claimLimit)
		if err != nil {
			return err
		}
		for _, pending := range batch {
			if err := s.deliver(ctx, pending, now); err != nil {
				return err
			}
		}
		if len(batch) < claimLimit {
			return nil
		}
	}
}

func (s SendEmails) deliver(ctx context.Context, pending PendingEmail, now time.Time) error {
	rendered, ready, err := s.render(ctx, pending, now)
	if err != nil || !ready {
		return err
	}
	providerID, err := s.Mailer.Send(ctx, pending.ID, rendered)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	age := now.Sub(pending.CreatedAt)
	switch {
	case err == nil:
		return s.Store.MarkSent(ctx, pending.ID, providerID, now)
	case fault.IsUnavailable(err) && age < giveUpAfter:
		retryAt := now.Add(retryDelay(pending.Attempts))
		senderLogger.Warn(fmt.Sprintf("Sending email %s failed (%s), retrying at %s", pending.ID, err, retryAt.UTC().Format(time.RFC3339)))
		return s.Store.RetryAt(ctx, pending.ID, retryAt, err.Error())
	case fault.IsUnavailable(err):
		return s.abandon(ctx, pending, fmt.Sprintf("still unsent %.0f h after it was queued: %s", giveUpAfter.Hours(), err))
	default:
		return s.abandon(ctx, pending, err.Error())
	}
}

func (s SendEmails) render(ctx context.Context, pending PendingEmail, now time.Time) (Rendered, bool, error) {
	if pending.Rendered != nil {
		return *pending.Rendered, true, nil
	}
	names, ready, err := s.names(ctx, pending, now)
	if err != nil || !ready {
		return Rendered{}, false, err
	}
	rendered, err := s.Composer.Compose(pending.Email, names)
	if err != nil {
		return Rendered{}, false, s.abandon(ctx, pending, err.Error())
	}
	if err := s.Store.SaveRendered(ctx, pending.ID, rendered); err != nil {
		return Rendered{}, false, err
	}
	return rendered, true, nil
}

func (s SendEmails) names(ctx context.Context, pending PendingEmail, now time.Time) (Names, bool, error) {
	email := pending.Email
	names, err := s.Directory.Names(ctx, email.AppID, email.ProjectID)
	switch {
	case ctx.Err() != nil:
		return Names{}, false, ctx.Err()
	case err == nil:
		return names, true, nil
	}
	fallbackAt := pending.CreatedAt.Add(s.NameFallbackAfter)
	if fault.IsUnavailable(err) && now.Before(fallbackAt) {
		retryAt := earliest(now.Add(retryDelay(pending.Attempts)), fallbackAt)
		senderLogger.Warn(fmt.Sprintf("Names for email %s are unavailable (%s), retrying at %s", pending.ID, err, retryAt.UTC().Format(time.RFC3339)))
		return Names{}, false, s.Store.RetryAt(ctx, pending.ID, retryAt, err.Error())
	}
	s.reportShortIDs(pending, err)
	return Names{App: domain.ShortID(email.AppID), Project: domain.ShortID(email.ProjectID)}, true, nil
}

func (s SendEmails) reportShortIDs(pending PendingEmail, cause error) {
	message := fmt.Sprintf("Rendering email %s with short ids (%s)", pending.ID, cause)
	if fault.IsUnavailable(cause) {
		senderLogger.Warn(message)
		return
	}
	senderLogger.Info(message)
}

func (s SendEmails) abandon(ctx context.Context, pending PendingEmail, reason string) error {
	senderLogger.Error(fmt.Sprintf("Abandoned email %s (%s) for app %s: %s", pending.ID, pending.Email.Kind, pending.Email.AppID, reason))
	return s.Store.Abandon(ctx, pending.ID, reason)
}

func retryDelay(attempts int) time.Duration {
	delay := firstRetryDelay
	for range max(attempts-1, 0) {
		if delay >= longestRetryDelay {
			break
		}
		delay *= 2
	}
	return min(delay, longestRetryDelay)
}

func earliest(first, second time.Time) time.Time {
	if second.Before(first) {
		return second
	}
	return first
}
