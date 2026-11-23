package logmailer

import (
	"context"
	"log/slog"

	"github.com/relay-deploy/chassis/go/logging"
	"github.com/relay-deploy/notify/internal/app"
)

type Mailer struct {
	logger *slog.Logger
}

func NewMailer() *Mailer {
	return &Mailer{logger: logging.New("notify.log_mailer")}
}

func (m *Mailer) Send(_ context.Context, idempotencyKey string, r app.Rendered) (string, error) {
	m.logger.Info("Email "+r.Subject, "idempotencyKey", idempotencyKey, "text", r.Text)
	return "log:" + idempotencyKey, nil
}
