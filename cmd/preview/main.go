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
	crashingReleaseID   = "019ab3c4-1a2b-7c3d-8e4f-5a6b3e9d27c1"
	deployingReleaseID  = "019ab3d9-5c6e-7a1b-9d2f-4e6a8c0b51f7"
	liveImage           = "ghcr.io/crashlab/web:1.4.2"
	missingImage        = "ghcr.io/crashlab/web:1.4.3"
	liveDigest          = "sha256:4f1c9a2e7b309c1d4e7a2b3f6a8d0e5c7b9a1f3d5e7c9b2a4f6d8e0c2a4b6d8f"
	missingImageCode    = "image.not_found"
	missingImageFailure = "manifest for ghcr.io/crashlab/web:1.4.3 not found: manifest unknown\n" +
		"GET https://ghcr.io/v2/crashlab/web/manifests/1.4.3 returned 404 Not Found with an HTML body instead of a manifest:\n" +
		`<!DOCTYPE html><html lang="en"><head><title>Not Found</title><script>&"use strict";window.location.replace("/login?return_to=%2Fcrashlab%2Fweb")</script></head><body><main><h1>404</h1><p>This package does not exist, or you cannot read it.</p></main></body></html>` + "\n" +
		"Tried 3 pulls over 6s, waiting 1s, 2s and 3s between them. Check that the tag was pushed, that the package is public or that relay's registry credentials can read it, and that the image reference in the release is spelled correctly. The last attempt ended at 14:36:04 CET."
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
	for _, sample := range samples(location) {
		kind := sample.email.Kind
		rendered, err := composer.Compose(sample.email, names, sample.sentAt)
		if err != nil {
			logger.Error("Cannot compose the preview", "kind", kind, "error", err)
			return 1
		}
		base := filepath.Join(*out, string(kind))
		if err := write(base+".html", rendered.HTML); err != nil {
			logger.Error("Cannot write the preview", "kind", kind, "error", err)
			return 1
		}
		if err := write(base+".txt", rendered.Text); err != nil {
			logger.Error("Cannot write the preview", "kind", kind, "error", err)
			return 1
		}
		logger.Info("Wrote the preview", "kind", kind, "subject", rendered.Subject, "path", base+".html")
	}
	return 0
}

type sample struct {
	email  domain.Email
	sentAt time.Time
}

func samples(location *time.Location) []sample {
	clock := func(hour, minute int) time.Time {
		return time.Date(2026, time.November, 23, hour, minute, 0, 0, location)
	}
	firstCrash := &domain.CrashAlert{
		AppID:               sampleAppID,
		ProjectID:           sampleProjectID,
		ReleaseID:           crashingReleaseID,
		OpenedAt:            clock(14, 27),
		LastCrashAt:         clock(14, 27),
		CrashCount:          1,
		LastExitCode:        137,
		RanOutOfMemory:      true,
		OpeningRestartCount: 1,
	}
	loop := &domain.CrashAlert{
		AppID:          sampleAppID,
		ProjectID:      sampleProjectID,
		ReleaseID:      crashingReleaseID,
		OpenedAt:       clock(14, 27),
		LastCrashAt:    clock(14, 56).Add(35 * time.Second),
		CrashCount:     42,
		LastExitCode:   137,
		RanOutOfMemory: true,
	}
	recoveredAt := clock(14, 31)
	recovery := &domain.CrashAlert{
		AppID:          sampleAppID,
		ProjectID:      sampleProjectID,
		ReleaseID:      crashingReleaseID,
		OpenedAt:       clock(14, 27),
		LastCrashAt:    clock(14, 30).Add(40 * time.Second),
		CrashCount:     6,
		LastExitCode:   137,
		RanOutOfMemory: true,
		RecoveredAt:    &recoveredAt,
	}
	return []sample{
		{email: domain.DeployedEmail(sampleAppID, sampleProjectID, domain.Facts{
			ReleaseID:      deployingReleaseID,
			ImageReference: liveImage,
			ImageDigest:    liveDigest,
			Duration:       14*time.Second + 300*time.Millisecond,
		}, clock(14, 36)), sentAt: clock(14, 36)},
		{email: domain.DeployFailedEmail(sampleAppID, sampleProjectID, domain.Facts{
			ReleaseID:      deployingReleaseID,
			ImageReference: missingImage,
			FailureCode:    missingImageCode,
			FailureMessage: missingImageFailure,
			Duration:       6*time.Second + 800*time.Millisecond,
		}, clock(14, 36)), sentAt: clock(14, 36)},
		{email: domain.CrashedEmail(firstCrash), sentAt: clock(14, 27)},
		{email: domain.StillCrashingEmail(loop, clock(14, 57)), sentAt: clock(14, 57)},
		{email: domain.RecoveredEmail(recovery), sentAt: clock(14, 36)},
	}
}

func write(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
