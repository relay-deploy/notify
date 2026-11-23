package app

import "github.com/relay-deploy/notify/internal/domain"

type Names struct {
	App     string
	Project string
}

type Rendered struct {
	Subject string
	HTML    string
	Text    string
}

type Composer interface {
	Compose(email domain.Email, names Names) (Rendered, error)
}
