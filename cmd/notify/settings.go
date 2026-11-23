package main

import (
	"time"

	"github.com/relay-deploy/chassis/go/config"
)

type settings struct {
	connectionString  string
	managementPort    int
	platformAddress   string
	sources           []string
	emailTo           string
	emailFrom         string
	resendAPIKey      string
	resendBaseURL     string
	location          *time.Location
	reminderInterval  time.Duration
	recoveryQuietTime time.Duration
	newLoopAfter      time.Duration
	nameFallbackAfter time.Duration
	alertExpiry       time.Duration
	alertPassInterval time.Duration
}

func loadSettings() (settings, error) {
	reader := config.FromEnvironment()
	loaded := settings{
		connectionString:  reader.Required("Relay__Postgres__ConnectionString"),
		managementPort:    reader.Int("Relay__Endpoints__ManagementPort", 8080),
		platformAddress:   reader.Required("Relay__Platform__Address"),
		sources:           reader.List("Relay__Notify__Sources", []string{"runtime_docker", "platform"}),
		emailTo:           reader.Required("Relay__Notify__Email__To"),
		emailFrom:         reader.String("Relay__Notify__Email__From", "relay <onboarding@resend.dev>"),
		resendAPIKey:      reader.String("Relay__Notify__Resend__ApiKey", ""),
		resendBaseURL:     reader.String("Relay__Notify__Resend__BaseUrl", "https://api.resend.com"),
		location:          reader.Location("Relay__Notify__TimeZone", "UTC"),
		reminderInterval:  reader.Duration("Relay__Notify__ReminderInterval", 30*time.Minute),
		recoveryQuietTime: reader.Duration("Relay__Notify__RecoveryQuietPeriod", 5*time.Minute),
		newLoopAfter:      reader.Duration("Relay__Notify__NewLoopAfter", time.Hour),
		nameFallbackAfter: reader.Duration("Relay__Notify__NameFallbackAfter", 2*time.Minute),
		alertExpiry:       reader.Duration("Relay__Notify__AlertExpiry", 24*time.Hour),
		alertPassInterval: reader.Duration("Relay__Notify__AlertPassInterval", time.Minute),
	}
	return loaded, reader.Err()
}
