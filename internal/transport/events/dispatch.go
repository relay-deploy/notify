package events

import (
	"context"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5"
	chassisevents "github.com/relay-deploy/chassis/go/events"
	"github.com/relay-deploy/chassis/go/fault"
	platformv1 "github.com/relay-deploy/proto/go/relay/platform/v1"
	runtimev1 "github.com/relay-deploy/proto/go/relay/runtime/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/relay-deploy/notify/internal/adapter/postgres"
	"github.com/relay-deploy/notify/internal/app"
	"github.com/relay-deploy/notify/internal/domain"
)

const invalidEventCode = "notify.invalid_event"

var (
	decoder     = protojson.UnmarshalOptions{DiscardUnknown: true}
	uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)

func Dispatcher(workload app.WorkloadEvents, releases app.ReleaseEvents) chassisevents.Handler {
	return func(ctx context.Context, tx pgx.Tx, event chassisevents.Event) error {
		unit := postgres.UnitOf(tx)
		switch event.Type {
		case "relay.runtime.v1.WorkloadCrashed":
			return handle(event, &runtimev1.WorkloadCrashed{}, func(m *runtimev1.WorkloadCrashed) error {
				return workload.Crashed(ctx, unit, domain.Crash{
					AppID:        m.GetAppId(),
					ProjectID:    m.GetProjectId(),
					ReleaseID:    m.GetReleaseId(),
					ExitCode:     int(m.GetExitCode()),
					OutOfMemory:  m.GetOomKilled(),
					RestartCount: int(m.GetRestartCount()),
					At:           event.OccurredAt,
				})
			})
		case "relay.runtime.v1.WorkloadRecovered":
			return handle(event, &runtimev1.WorkloadRecovered{}, func(m *runtimev1.WorkloadRecovered) error {
				return workload.Recovered(ctx, unit, m.GetAppId(), m.GetReleaseId(), event.OccurredAt)
			})
		case "relay.runtime.v1.WorkloadStopped":
			return handle(event, &runtimev1.WorkloadStopped{}, func(m *runtimev1.WorkloadStopped) error {
				return workload.Stopped(ctx, unit, m.GetAppId(), m.GetReleaseId())
			})
		case "relay.platform.v1.ReleaseSucceeded":
			return handle(event, &platformv1.ReleaseSucceeded{}, func(m *platformv1.ReleaseSucceeded) error {
				facts := domain.Facts{
					ReleaseID:      m.GetReleaseId(),
					ImageReference: m.GetImageReference(),
					ImageDigest:    m.GetImageDigest(),
					Duration:       m.GetDuration().AsDuration(),
				}
				return releases.Succeeded(ctx, unit, m.GetAppId(), m.GetProjectId(), facts, event.OccurredAt)
			})
		case "relay.platform.v1.ReleaseFailed":
			return handle(event, &platformv1.ReleaseFailed{}, func(m *platformv1.ReleaseFailed) error {
				facts := domain.Facts{
					ReleaseID:      m.GetReleaseId(),
					ImageReference: m.GetImageReference(),
					FailureCode:    m.GetFailure().GetCode(),
					FailureMessage: m.GetFailure().GetMessage(),
					Duration:       m.GetDuration().AsDuration(),
				}
				return releases.Failed(ctx, unit, m.GetAppId(), m.GetProjectId(), facts, event.OccurredAt)
			})
		}
		return nil
	}
}

type subjectMessage interface {
	proto.Message
	GetAppId() string
	GetProjectId() string
	GetReleaseId() string
}

type namedID struct {
	name  string
	value string
}

func handle[M subjectMessage](event chassisevents.Event, message M, use func(M) error) error {
	if err := decoder.Unmarshal(event.Payload, message); err != nil {
		return invalid(event, err.Error())
	}
	for _, id := range []namedID{
		{name: "app", value: message.GetAppId()},
		{name: "project", value: message.GetProjectId()},
		{name: "release", value: message.GetReleaseId()},
	} {
		if !uuidPattern.MatchString(id.value) {
			return invalid(event, fmt.Sprintf("the %s id %q is not a UUID", id.name, id.value))
		}
	}
	return use(message)
}

func invalid(event chassisevents.Event, reason string) error {
	return fault.New(fault.Validation, invalidEventCode, fmt.Sprintf("Cannot handle event %s of type %s: %s", event.ID, event.Type, reason))
}
