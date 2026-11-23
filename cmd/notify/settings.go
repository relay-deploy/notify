package main

import (
	"errors"
	"net/mail"
	"time"

	"github.com/relay-deploy/chassis/go/config"
	"github.com/relay-deploy/chassis/go/grpcclient"
	"github.com/relay-deploy/chassis/go/postgres"
	"github.com/relay-deploy/notify/internal/adapter/resend"
)

const (
	platformAddressKey = "Relay__Platform__Address"
	sourcesKey         = "Relay__Notify__Sources"
	emailToKey         = "Relay__Notify__Email__To"
	emailFromKey       = "Relay__Notify__Email__From"
	resendBaseURLKey   = "Relay__Notify__Resend__BaseUrl"
)

var errNoSources = errors.New("lists no sources")

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
		managementPort:    reader.Port("Relay__Endpoints__ManagementPort", 8080),
		platformAddress:   reader.Required(platformAddressKey),
		sources:           reader.List(sourcesKey, []string{"runtime_docker", "platform"}),
		emailTo:           reader.Required(emailToKey),
		emailFrom:         reader.String(emailFromKey, "relay <onboarding@resend.dev>"),
		resendAPIKey:      reader.String("Relay__Notify__Resend__ApiKey", ""),
		resendBaseURL:     reader.String(resendBaseURLKey, "https://api.resend.com"),
		location:          reader.Location("Relay__Notify__TimeZone", "UTC"),
		reminderInterval:  reader.PositiveDuration("Relay__Notify__ReminderInterval", 30*time.Minute),
		recoveryQuietTime: reader.PositiveDuration("Relay__Notify__RecoveryQuietPeriod", 5*time.Minute),
		newLoopAfter:      reader.PositiveDuration("Relay__Notify__NewLoopAfter", time.Hour),
		nameFallbackAfter: reader.PositiveDuration("Relay__Notify__NameFallbackAfter", 2*time.Minute),
		alertExpiry:       reader.PositiveDuration("Relay__Notify__AlertExpiry", 24*time.Hour),
		alertPassInterval: reader.PositiveDuration("Relay__Notify__AlertPassInterval", time.Minute),
	}
	check(reader, loaded)
	return loaded, reader.Err()
}

func check(reader *config.Reader, loaded settings) {
	if loaded.platformAddress != "" {
		reader.Check(platformAddressKey, grpcclient.ValidAddress(loaded.platformAddress))
	}
	if len(loaded.sources) == 0 {
		reader.Check(sourcesKey, errNoSources)
	}
	for _, source := range loaded.sources {
		reader.Check(sourcesKey, postgres.ValidSchema(source))
	}
	if loaded.emailTo != "" {
		reader.Check(emailToKey, validEmailAddress(loaded.emailTo))
	}
	reader.Check(emailFromKey, validEmailAddress(loaded.emailFrom))
	if loaded.resendAPIKey != "" {
		reader.Check(resendBaseURLKey, resend.ValidBaseURL(loaded.resendBaseURL))
	}
}

func validEmailAddress(address string) error {
	_, err := mail.ParseAddress(address)
	return err
}
