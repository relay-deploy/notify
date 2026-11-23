package email

import (
	_ "embed"
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/relay-deploy/notify/internal/app"
	"github.com/relay-deploy/notify/internal/domain"
)

const (
	defaultQuietPeriod      = 5 * time.Minute
	defaultReminderInterval = 30 * time.Minute
	clockLayout             = "15:04"
)

var (
	//go:embed templates/layout.html.tmpl
	htmlLayout string

	//go:embed templates/layout.txt.tmpl
	textLayout string

	htmlPage = htmltemplate.Must(htmltemplate.New("layout.html").Funcs(htmltemplate.FuncMap{"breakable": breakable}).Parse(htmlLayout))
	textPage = texttemplate.Must(texttemplate.New("layout.txt").Parse(textLayout))
)

type Composer struct {
	location         *time.Location
	quietPeriod      time.Duration
	reminderInterval time.Duration
}

func NewComposer(location *time.Location) *Composer {
	return &Composer{location: location, quietPeriod: defaultQuietPeriod, reminderInterval: defaultReminderInterval}
}

func (c *Composer) WithCadence(quietPeriod, reminderInterval time.Duration) *Composer {
	return &Composer{location: c.location, quietPeriod: quietPeriod, reminderInterval: reminderInterval}
}

type page struct {
	Subject    string
	Headline   string
	Calm       bool
	Project    string
	Summary    string
	LinkedData htmltemplate.JS
	Body       []segment
	Timeline   *timeline
	Failure    *failure
	Rows       []row
	Footer     string
}

type segment struct {
	Text string
	Bold bool
}

type timeline struct {
	Ticks     int
	Hidden    int
	Recovered bool
	Start     string
	End       string
}

type failure struct {
	Code    string
	Message string
}

type row struct {
	Label string
	Value string
	Mono  bool
}

type naming struct {
	app     string
	project string
}

type emailMessage struct {
	Context     string       `json:"@context"`
	Type        string       `json:"@type"`
	Description string       `json:"description"`
	Publisher   organization `json:"publisher"`
}

type organization struct {
	Type string `json:"@type"`
	Name string `json:"name"`
}

func (c *Composer) Compose(e domain.Email, names app.Names, renderedAt time.Time) (app.Rendered, error) {
	content, err := c.page(e, names, renderedAt)
	if err != nil {
		return app.Rendered{}, err
	}
	var html, text strings.Builder
	if err := htmlPage.Execute(&html, content); err != nil {
		return app.Rendered{}, fmt.Errorf("Cannot render the %s email as HTML: %w", e.Kind, err)
	}
	if err := textPage.Execute(&text, content); err != nil {
		return app.Rendered{}, fmt.Errorf("Cannot render the %s email as text: %w", e.Kind, err)
	}
	return app.Rendered{Subject: content.Subject, HTML: html.String(), Text: text.String()}, nil
}

func (c *Composer) page(e domain.Email, names app.Names, renderedAt time.Time) (page, error) {
	about := naming{app: nameOrShortID(names.App, e.AppID), project: nameOrShortID(names.Project, e.ProjectID)}
	var content page
	switch e.Kind {
	case domain.Deployed:
		content = c.deployed(e.Facts, about)
	case domain.DeployFailed:
		content = c.deployFailed(e.Facts, about)
	case domain.Crashed:
		content = c.crashed(e, about)
	case domain.StillCrashing:
		content = c.stillCrashing(e.Facts, about)
	case domain.Recovered:
		content = c.recovered(e.Facts, about)
	default:
		return page{}, fmt.Errorf("Cannot render an email of kind %q", e.Kind)
	}
	linked, err := linkedData(content.Summary)
	if err != nil {
		return page{}, fmt.Errorf("Cannot describe the %s email as JSON-LD: %w", e.Kind, err)
	}
	content.Project = about.project
	content.LinkedData = linked
	content.Footer = c.footer(renderedAt)
	return content, nil
}

func (c *Composer) timeline(crashes int, start, end time.Time, recovered bool) *timeline {
	shown := min(max(crashes, 1), tickLimit)
	strip := &timeline{Ticks: shown, Hidden: max(crashes-tickLimit, 0), Recovered: recovered, Start: c.clock(start)}
	if !end.IsZero() {
		strip.End = c.clock(end)
	}
	return strip
}

func (c *Composer) clock(t time.Time) string {
	return t.In(c.location).Format(clockLayout)
}

func (c *Composer) footer(at time.Time) string {
	local := at.In(c.location)
	return fmt.Sprintf("Sent by relay at %s on %s.", local.Format("15:04 MST"), local.Format("2 January 2006"))
}

func (n naming) inProject() string {
	if n.project == "" {
		return n.app
	}
	return n.app + " in " + n.project
}

func linkedData(summary string) (htmltemplate.JS, error) {
	encoded, err := json.Marshal(emailMessage{
		Context:     "http://schema.org",
		Type:        "EmailMessage",
		Description: summary,
		Publisher:   organization{Type: "Organization", Name: "relay"},
	})
	if err != nil {
		return "", err
	}
	return htmltemplate.JS(encoded), nil
}

func plain(text string) segment {
	return segment{Text: text}
}

func bold(text string) segment {
	return segment{Text: text, Bold: true}
}

func textRow(label, value string) row {
	return row{Label: label, Value: value}
}

func monoRow(label, value string) row {
	return row{Label: label, Value: value, Mono: true}
}

func present(rows ...row) []row {
	var kept []row
	for _, candidate := range rows {
		if candidate.Value != "" {
			kept = append(kept, candidate)
		}
	}
	return kept
}

func breakable(value string) htmltemplate.HTML {
	parts := strings.SplitAfter(value, "/")
	for index, part := range parts {
		parts[index] = htmltemplate.HTMLEscapeString(part)
	}
	return htmltemplate.HTML(strings.Join(parts, "<wbr>"))
}

func nameOrShortID(name, id string) string {
	if name != "" {
		return name
	}
	return domain.ShortID(id)
}
