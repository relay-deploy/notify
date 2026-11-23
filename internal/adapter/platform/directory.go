package platform

import (
	"context"
	"time"

	"github.com/relay-deploy/chassis/go/fault"
	"github.com/relay-deploy/chassis/go/grpcclient"
	platformv1 "github.com/relay-deploy/proto/go/relay/platform/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/relay-deploy/notify/internal/app"
)

const callTimeout = 5 * time.Second

type Directory struct {
	connection *grpc.ClientConn
	apps       platformv1.AppServiceClient
	projects   platformv1.ProjectServiceClient
}

func NewDirectory(address string) (*Directory, error) {
	connection, err := grpcclient.Dial(address)
	if err != nil {
		return nil, err
	}
	return &Directory{
		connection: connection,
		apps:       platformv1.NewAppServiceClient(connection),
		projects:   platformv1.NewProjectServiceClient(connection),
	}, nil
}

func (d *Directory) Close() error {
	return d.connection.Close()
}

func (d *Directory) Names(ctx context.Context, appID, projectID string) (app.Names, error) {
	appName, err := d.appName(ctx, appID)
	if err != nil {
		return app.Names{}, err
	}
	projectName, err := d.projectName(ctx, projectID)
	if err != nil {
		return app.Names{}, err
	}
	return app.Names{App: appName, Project: projectName}, nil
}

func (d *Directory) appName(ctx context.Context, appID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	response, err := d.apps.GetApp(ctx, &platformv1.GetAppRequest{AppId: appID})
	if err != nil {
		return "", lookupFailure("app", appID, err)
	}
	return response.GetApp().GetName(), nil
}

func (d *Directory) projectName(ctx context.Context, projectID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	response, err := d.projects.GetProject(ctx, &platformv1.GetProjectRequest{ProjectId: projectID})
	if err != nil {
		return "", lookupFailure("project", projectID, err)
	}
	return response.GetProject().GetName(), nil
}

func lookupFailure(subject, id string, err error) error {
	message := "Platform cannot name " + subject + " " + id + ": " + err.Error()
	if status.Code(err) == codes.NotFound {
		return fault.New(fault.NotFound, "platform."+subject+"_not_found", message)
	}
	return fault.New(fault.Unavailable, "platform.unavailable", message)
}
