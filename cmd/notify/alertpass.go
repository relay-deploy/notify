package main

import (
	"context"
	"time"

	"github.com/relay-deploy/chassis/go/hosting"
	"github.com/relay-deploy/chassis/go/logging"
	"github.com/relay-deploy/notify/internal/app"
)

func alertPassComponent(pass app.AlertPass, interval time.Duration) hosting.Component {
	logger := logging.New("notify.alert_pass")
	return hosting.Component{
		Name: "alert-pass",
		Run: func(ctx context.Context) error {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				if err := pass.Run(ctx, time.Now().UTC()); err != nil && ctx.Err() == nil {
					logger.Warn("Alert pass failed, retrying on the next tick", "error", err)
				}
				select {
				case <-ctx.Done():
					return nil
				case <-ticker.C:
				}
			}
		},
	}
}
