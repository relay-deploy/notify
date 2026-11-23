package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/relay-deploy/chassis/go/hosting"
	"github.com/relay-deploy/chassis/go/logging"
	chassispostgres "github.com/relay-deploy/chassis/go/postgres"
	"github.com/relay-deploy/notify/internal/app"
)

const (
	emailChannel     = "notify_emails"
	senderPoll       = 15 * time.Second
	listenRetryDelay = 5 * time.Second
)

func senderComponent(pool *pgxpool.Pool, sender app.SendEmails) hosting.Component {
	logger := logging.New("notify.sender")
	return hosting.Component{
		Name: "sender",
		Run: func(ctx context.Context) error {
			wake := make(chan struct{}, 1)
			var listener sync.WaitGroup
			listener.Go(func() { listenForEmails(ctx, pool, wake, logger) })
			defer listener.Wait()
			poll := time.NewTicker(senderPoll)
			defer poll.Stop()
			for {
				if err := sender.SendDue(ctx, time.Now().UTC()); err != nil && ctx.Err() == nil {
					logger.Warn(fmt.Sprintf("Sending due emails failed (%s), retrying on the next wake-up", err))
				}
				select {
				case <-ctx.Done():
					return nil
				case <-wake:
				case <-poll.C:
				}
			}
		},
	}
}

func listenForEmails(ctx context.Context, pool *pgxpool.Pool, wake chan<- struct{}, logger *slog.Logger) {
	for {
		err := chassispostgres.Listen(ctx, pool, emailChannel, wake)
		if ctx.Err() != nil {
			return
		}
		message := fmt.Sprintf("Listening for queued emails failed (%s), retrying in %s", err, listenRetryDelay)
		if chassispostgres.IsUnavailable(err) {
			logger.Warn(message)
		} else {
			logger.Error(message)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(listenRetryDelay):
		}
	}
}
