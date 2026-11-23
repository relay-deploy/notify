package domain

import "time"

type CrashAlert struct {
	AppID                  string
	ProjectID              string
	ReleaseID              string
	OpenedAt               time.Time
	LastCrashAt            time.Time
	LastNotifiedAt         time.Time
	CrashCount             int
	LastExitCode           int
	LastNotifiedCrashCount int
	RanOutOfMemory         bool
	RecoveredAt            *time.Time
	OpeningRestartCount    int
}

func OpenAlert(c Crash) *CrashAlert {
	return &CrashAlert{
		AppID:                  c.AppID,
		ProjectID:              c.ProjectID,
		ReleaseID:              c.ReleaseID,
		OpenedAt:               c.At,
		LastCrashAt:            c.At,
		LastNotifiedAt:         c.At,
		CrashCount:             1,
		LastExitCode:           c.ExitCode,
		LastNotifiedCrashCount: 1,
		RanOutOfMemory:         c.OutOfMemory,
		OpeningRestartCount:    c.RestartCount,
	}
}

func (a *CrashAlert) IsStale(c Crash, newLoopAfter time.Duration) bool {
	return a.RecoveredAt == nil && c.At.Sub(a.LastCrashAt) > newLoopAfter
}

func (a *CrashAlert) Add(c Crash) {
	a.CrashCount++
	a.LastExitCode = c.ExitCode
	a.ReleaseID = c.ReleaseID
	a.LastCrashAt = c.At
	a.RanOutOfMemory = a.RanOutOfMemory || c.OutOfMemory
	a.RecoveredAt = nil
}

func (a *CrashAlert) Recover(at time.Time) {
	if a.RecoveredAt == nil {
		a.RecoveredAt = &at
	}
}

func (a *CrashAlert) RecoveryDue(now time.Time, quiet time.Duration) bool {
	return a.RecoveredAt != nil && !now.Before(a.RecoveredAt.Add(quiet))
}

func (a *CrashAlert) ReminderDue(now time.Time, interval time.Duration) bool {
	return a.RecoveredAt == nil &&
		now.Sub(a.LastNotifiedAt) >= interval &&
		a.CrashCount > a.LastNotifiedCrashCount
}

func (a *CrashAlert) Remind(now time.Time) {
	a.LastNotifiedAt = now
	a.LastNotifiedCrashCount = a.CrashCount
}

func (a *CrashAlert) Expired(now time.Time, expiry time.Duration) bool {
	return now.Sub(a.LastCrashAt) > expiry
}
