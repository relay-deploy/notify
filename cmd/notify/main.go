package main

import (
	"context"
	"os"

	"github.com/relay-deploy/chassis/go/hosting"
	"github.com/relay-deploy/chassis/go/logging"
	chassispostgres "github.com/relay-deploy/chassis/go/postgres"
	notifypostgres "github.com/relay-deploy/notify/internal/adapter/postgres"
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

	return hosting.Run(logger, hosting.ManagementServer(loaded.managementPort, health))
}
