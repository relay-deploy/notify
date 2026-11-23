package domain

import "time"

type Crash struct {
	AppID        string
	ProjectID    string
	ReleaseID    string
	ExitCode     int
	OutOfMemory  bool
	RestartCount int
	At           time.Time
}
