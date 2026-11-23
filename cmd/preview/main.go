package main

import (
	"flag"
	"os"
	"path/filepath"
	"time"

	"github.com/relay-deploy/chassis/go/logging"
	"github.com/relay-deploy/notify/internal/adapter/email"
	"github.com/relay-deploy/notify/internal/app"
	"github.com/relay-deploy/notify/internal/domain"
)

const (
	sampleAppID         = "019ab3c2-7d41-7e0a-9f3b-2c5d8e1a4b6f"
	sampleProjectID     = "019ab3c1-0b2e-7f4d-a6c8-1e3f5a7b9d02"
	sampleReleaseID     = "019ab3c4-1a2b-7c3d-8e4f-5a6b3e9d27c1"
	sampleNextReleaseID = "019ab3d9-5c6e-7a1b-9d2f-4e6a8c0b51f7"
	sampleImage         = "ghcr.io/crashlab/web:1.4.2"
	sampleNextImage     = "ghcr.io/crashlab/web:1.4.3"
	sampleDigest        = "sha256:9c1d4e7a2b3f6a8d0e5c7b9a1f3d5e7c9b2a4f6d8e0c2a4b6d8f0e2c4a6b8d0f"
	sampleFailureCode   = "image.not_found"
	sampleFailure       = "Error response from daemon: manifest for ghcr.io/crashlab/web:1.4.3 not found: manifest unknown: manifest unknown\n" +
		"GET https://ghcr.io/v2/crashlab/web/manifests/1.4.3 returned 404 Not Found with an HTML body instead of a manifest:\n" +
		`<!DOCTYPE html><html lang="en"><head><title>Not Found</title><script>&"use strict";window.location.replace("/login?return_to=%2Fcrashlab%2Fweb")</script></head><body><main><h1>404</h1><p>This package does not exist, or you cannot read it.</p></main></body></html>` + "\n" +
		"Tried 3 pulls over 6s, waiting 1s, 2s and 3s between them. Check that the tag was pushed, that the package is public or that relay's registry credentials can read it, and that the image reference in the release is spelled correctly. The last attempt ended at 14:31:54 CET."
)

func main() {
	os.Exit(run())
}

func run() int {
	logger := logging.New("notify-preview")
	out := flag.String("out", "previews", "directory the previews are written to")
	flag.Parse()

	location, err := time.LoadLocation("Europe/Skopje")
	if err != nil {
		logger.Error("Cannot load the time zone", "error", err)
		return 1
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		logger.Error("Cannot create the output directory", "error", err)
		return 1
	}

	composer := email.NewComposer(location)
	names := app.Names{App: "web", Project: "crashlab"}
	at := time.Date(2026, time.November, 23, 14, 32, 0, 0, location)
	for _, sample := range samples(at) {
		rendered, err := composer.Compose(sample, names)
		if err != nil {
			logger.Error("Cannot compose the preview", "kind", sample.Kind, "error", err)
			return 1
		}
		base := filepath.Join(*out, string(sample.Kind))
		if err := write(base+".html", rendered.HTML); err != nil {
			logger.Error("Cannot write the preview", "kind", sample.Kind, "error", err)
			return 1
		}
		if err := write(base+".txt", rendered.Text); err != nil {
			logger.Error("Cannot write the preview", "kind", sample.Kind, "error", err)
			return 1
		}
		logger.Info("Wrote the preview", "kind", sample.Kind, "subject", rendered.Subject, "path", base+".html")
	}
	return 0
}

func samples(at time.Time) []domain.Email {
	firstCrash := &domain.CrashAlert{
		AppID:               sampleAppID,
		ProjectID:           sampleProjectID,
		ReleaseID:           sampleReleaseID,
		OpenedAt:            at,
		LastCrashAt:         at,
		CrashCount:          1,
		LastExitCode:        137,
		RanOutOfMemory:      true,
		OpeningRestartCount: 2,
	}
	loop := &domain.CrashAlert{
		AppID:          sampleAppID,
		ProjectID:      sampleProjectID,
		ReleaseID:      sampleReleaseID,
		OpenedAt:       at.Add(-30*time.Minute - 12*time.Second),
		LastCrashAt:    at.Add(-25 * time.Second),
		CrashCount:     42,
		LastExitCode:   137,
		RanOutOfMemory: true,
	}
	recoveredAt := at
	recovery := &domain.CrashAlert{
		AppID:          sampleAppID,
		ProjectID:      sampleProjectID,
		ReleaseID:      sampleReleaseID,
		OpenedAt:       at.Add(-4*time.Minute - 18*time.Second),
		LastCrashAt:    at.Add(-40 * time.Second),
		CrashCount:     6,
		LastExitCode:   137,
		RanOutOfMemory: true,
		RecoveredAt:    &recoveredAt,
	}
	return []domain.Email{
		domain.DeployedEmail(sampleAppID, sampleProjectID, domain.Facts{
			ReleaseID:      sampleReleaseID,
			ImageReference: sampleImage,
			ImageDigest:    sampleDigest,
			Duration:       14*time.Second + 300*time.Millisecond,
		}, at),
		domain.DeployFailedEmail(sampleAppID, sampleProjectID, domain.Facts{
			ReleaseID:      sampleNextReleaseID,
			ImageReference: sampleNextImage,
			FailureCode:    sampleFailureCode,
			FailureMessage: sampleFailure,
			Duration:       6*time.Second + 800*time.Millisecond,
		}, at),
		domain.CrashedEmail(firstCrash),
		domain.StillCrashingEmail(loop, at),
		domain.RecoveredEmail(recovery),
	}
}

func write(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
