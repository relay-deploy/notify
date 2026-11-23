package domain

import (
	"encoding/json"
	"time"
)

type EmailKind string

const (
	Deployed      EmailKind = "deployed"
	DeployFailed  EmailKind = "deploy_failed"
	Crashed       EmailKind = "crashed"
	StillCrashing EmailKind = "still_crashing"
	Recovered     EmailKind = "recovered"
)

type Facts struct {
	ReleaseID      string
	ImageReference string
	ImageDigest    string
	FailureCode    string
	FailureMessage string
	Duration       time.Duration
	ExitCode       int
	RestartCount   int
	CrashCount     int
	OutOfMemory    bool
	Since          time.Time
	Until          time.Time
}

type factsJSON struct {
	ReleaseID      string     `json:"releaseId,omitempty"`
	ImageReference string     `json:"imageReference,omitempty"`
	ImageDigest    string     `json:"imageDigest,omitempty"`
	FailureCode    string     `json:"failureCode,omitempty"`
	FailureMessage string     `json:"failureMessage,omitempty"`
	DurationMillis int64      `json:"durationMs,omitempty"`
	ExitCode       int        `json:"exitCode,omitempty"`
	RestartCount   int        `json:"restartCount,omitempty"`
	CrashCount     int        `json:"crashCount,omitempty"`
	OutOfMemory    bool       `json:"outOfMemory,omitempty"`
	Since          *time.Time `json:"since,omitempty"`
	Until          *time.Time `json:"until,omitempty"`
}

func (f Facts) MarshalJSON() ([]byte, error) {
	return json.Marshal(factsJSON{
		ReleaseID:      f.ReleaseID,
		ImageReference: f.ImageReference,
		ImageDigest:    f.ImageDigest,
		FailureCode:    f.FailureCode,
		FailureMessage: f.FailureMessage,
		DurationMillis: f.Duration.Milliseconds(),
		ExitCode:       f.ExitCode,
		RestartCount:   f.RestartCount,
		CrashCount:     f.CrashCount,
		OutOfMemory:    f.OutOfMemory,
		Since:          timeOrNil(f.Since),
		Until:          timeOrNil(f.Until),
	})
}

func (f *Facts) UnmarshalJSON(data []byte) error {
	var raw factsJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*f = Facts{
		ReleaseID:      raw.ReleaseID,
		ImageReference: raw.ImageReference,
		ImageDigest:    raw.ImageDigest,
		FailureCode:    raw.FailureCode,
		FailureMessage: raw.FailureMessage,
		Duration:       time.Duration(raw.DurationMillis) * time.Millisecond,
		ExitCode:       raw.ExitCode,
		RestartCount:   raw.RestartCount,
		CrashCount:     raw.CrashCount,
		OutOfMemory:    raw.OutOfMemory,
	}
	if raw.Since != nil {
		f.Since = *raw.Since
	}
	if raw.Until != nil {
		f.Until = *raw.Until
	}
	return nil
}

func timeOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

type Email struct {
	Kind      EmailKind
	AppID     string
	ProjectID string
	Facts     Facts
	At        time.Time
}

func DeployedEmail(appID, projectID string, facts Facts, at time.Time) Email {
	return Email{Kind: Deployed, AppID: appID, ProjectID: projectID, Facts: facts, At: at}
}

func DeployFailedEmail(appID, projectID string, facts Facts, at time.Time) Email {
	return Email{Kind: DeployFailed, AppID: appID, ProjectID: projectID, Facts: facts, At: at}
}

func CrashedEmail(a *CrashAlert) Email {
	return Email{
		Kind:      Crashed,
		AppID:     a.AppID,
		ProjectID: a.ProjectID,
		At:        a.OpenedAt,
		Facts: Facts{
			ReleaseID:    a.ReleaseID,
			ExitCode:     a.LastExitCode,
			OutOfMemory:  a.RanOutOfMemory,
			RestartCount: a.OpeningRestartCount,
			Since:        a.OpenedAt,
		},
	}
}

func StillCrashingEmail(a *CrashAlert, now time.Time) Email {
	return Email{
		Kind:      StillCrashing,
		AppID:     a.AppID,
		ProjectID: a.ProjectID,
		At:        now,
		Facts: Facts{
			ReleaseID:   a.ReleaseID,
			CrashCount:  a.CrashCount,
			ExitCode:    a.LastExitCode,
			OutOfMemory: a.RanOutOfMemory,
			Since:       a.OpenedAt,
			Until:       now,
		},
	}
}

func RecoveredEmail(a *CrashAlert) Email {
	until := a.LastCrashAt
	if a.RecoveredAt != nil {
		until = *a.RecoveredAt
	}
	return Email{
		Kind:      Recovered,
		AppID:     a.AppID,
		ProjectID: a.ProjectID,
		At:        until,
		Facts: Facts{
			ReleaseID:   a.ReleaseID,
			CrashCount:  a.CrashCount,
			ExitCode:    a.LastExitCode,
			OutOfMemory: a.RanOutOfMemory,
			Since:       a.OpenedAt,
			Until:       until,
		},
	}
}
