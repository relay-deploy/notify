package main

import (
	"context"
	"os"

	chassisevents "github.com/relay-deploy/chassis/go/events"
	"github.com/relay-deploy/chassis/go/hosting"
	"github.com/relay-deploy/chassis/go/logging"
	chassispostgres "github.com/relay-deploy/chassis/go/postgres"
	notifypostgres "github.com/relay-deploy/notify/internal/adapter/postgres"
	"github.com/relay-deploy/notify/internal/app"
	"github.com/relay-deploy/notify/internal/transport/events"
)

const schema = "notify"

func main() {
	os.Exit(run())
}

func run() int {
	logger := logging.New("notify")
	loaded, err := loadSettings()
	if err != nil {
		logger.Error("Invalid configuration", "error", err)
		return 1
	}

	ctx := context.Background()
	pool, err := chassispostgres.Open(ctx, loaded.connectionString, schema)
	if err != nil {
		logger.Error("Cannot open the database", "error", err)
		return 1
	}
	defer pool.Close()

	if err := chassispostgres.Migrate(ctx, pool, schema, notifypostgres.Migrations); err != nil {
		logger.Error("Cannot migrate the database", "error", err)
		return 1
	}

	health := hosting.NewHealth()
	health.AddReadiness("postgres", chassispostgres.Readiness(pool))

	if loaded.resendAPIKey == "" {
		logger.Warn("No Resend API key, emails are logged instead of sent")
	}

	dispatcher := events.Dispatcher(app.WorkloadEvents{NewLoopAfter: loaded.newLoopAfter}, app.ReleaseEvents{})
	components := []hosting.Component{hosting.ManagementServer(loaded.managementPort, health)}
	for _, source := range loaded.sources {
		components = append(components, chassisevents.Subscription(pool, schema, source, dispatcher, chassisevents.DefaultOptions()))
	}

	pass := app.AlertPass{
		Transactor:       notifypostgres.NewTransactor(pool),
		Quiet:            loaded.recoveryQuietTime,
		ReminderInterval: loaded.reminderInterval,
		Expiry:           loaded.alertExpiry,
	}
	components = append(components, alertPassComponent(pass, loaded.alertPassInterval))

	return hosting.Run(logger, components...)
}
