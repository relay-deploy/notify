package email

import (
	_ "embed"
	"fmt"
	htmltemplate "html/template"
	"strconv"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/relay-deploy/notify/internal/app"
	"github.com/relay-deploy/notify/internal/domain"
)

const (
	alarmAccent         = "#d92d20"
	calmAccent          = "#079455"
	timeLayout          = "2 Jan 2006, 15:04 MST"
	failureMessageLimit = 600
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
	location *time.Location
}

func NewComposer(location *time.Location) *Composer {
	return &Composer{location: location}
}

type page struct {
	Subject string
	Accent  string
	App     string
	Project string
	Verb    string
	Summary string
	Rows    []row
	Failure *failure
	SentAt  string
}

type row struct {
	Label string
	Value string
	Note  string
	Mono  bool
}

type failure struct {
	Code    string
	Message string
}

func (c *Composer) Compose(e domain.Email, names app.Names) (app.Rendered, error) {
	content, err := c.page(e, names)
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

func (c *Composer) page(e domain.Email, names app.Names) (page, error) {
	facts := e.Facts
	appName := nameOrShortID(names.App, e.AppID)
	content := page{
		Accent:  alarmAccent,
		App:     appName,
		Project: nameOrShortID(names.Project, e.ProjectID),
		Rows:    c.rows(e),
		SentAt:  c.format(e.At),
	}
	span := humanize(facts.Until.Sub(facts.Since))
	switch e.Kind {
	case domain.Deployed:
		content.Accent = calmAccent
		content.Verb = "deployed"
		content.Subject = fmt.Sprintf("✓ %s deployed · %s", appName, humanize(facts.Duration))
	case domain.DeployFailed:
		content.Verb = "deploy failed"
		content.Subject = fmt.Sprintf("✗ %s deploy failed%s", appName, detail(facts.FailureCode))
		content.Failure = failureOf(facts)
	case domain.Crashed:
		content.Verb = "crashed"
		content.Subject = fmt.Sprintf("● %s crashed · exit %d%s", appName, facts.ExitCode, outOfMemory(facts, ", out of memory"))
	case domain.StillCrashing:
		content.Verb = "is still crashing"
		content.Subject = fmt.Sprintf("● %s is still crashing · %d crashes in %s", appName, facts.CrashCount, span)
	case domain.Recovered:
		content.Accent = calmAccent
		content.Verb = "recovered"
		content.Subject = fmt.Sprintf("● %s recovered · crashed %s over %s", appName, times(facts.CrashCount), span)
		content.Summary = fmt.Sprintf("Crashed %s over %s. Last exit %d%s.", times(facts.CrashCount), span, facts.ExitCode, outOfMemory(facts, " (out of memory)"))
	default:
		return page{}, fmt.Errorf("Cannot render an email of kind %q", e.Kind)
	}
	return content, nil
}

func (c *Composer) rows(e domain.Email) []row {
	facts := e.Facts
	var rows []row
	if facts.CrashCount > 0 {
		rows = append(rows, row{Label: "Crashes", Value: strconv.Itoa(facts.CrashCount)})
	}
	if isCrash(e.Kind) {
		rows = append(rows, row{Label: exitLabel(facts), Value: strconv.Itoa(facts.ExitCode), Note: outOfMemory(facts, "out of memory"), Mono: true})
	}
	if facts.RestartCount > 0 {
		rows = append(rows, row{Label: "Restarts", Value: strconv.Itoa(facts.RestartCount)})
	}
	if !facts.Since.IsZero() && !facts.Since.Equal(e.At) {
		rows = append(rows, row{Label: sinceLabel(e.Kind), Value: c.format(facts.Since)})
	}
	if facts.ReleaseID != "" {
		rows = append(rows, row{Label: "Release", Value: domain.ShortID(facts.ReleaseID), Mono: true})
	}
	if facts.ImageReference != "" {
		rows = append(rows, row{Label: "Image", Value: facts.ImageReference, Mono: true})
	}
	if facts.ImageDigest != "" {
		rows = append(rows, row{Label: "Digest", Value: shortDigest(facts.ImageDigest), Mono: true})
	}
	if facts.Duration > 0 {
		rows = append(rows, row{Label: "Duration", Value: humanize(facts.Duration)})
	}
	return rows
}

func (c *Composer) format(t time.Time) string {
	return t.In(c.location).Format(timeLayout)
}

func isCrash(kind domain.EmailKind) bool {
	return kind == domain.Crashed || kind == domain.StillCrashing || kind == domain.Recovered
}

func exitLabel(facts domain.Facts) string {
	if facts.CrashCount > 1 {
		return "Last exit code"
	}
	return "Exit code"
}

func sinceLabel(kind domain.EmailKind) string {
	if kind == domain.Recovered {
		return "First crash"
	}
	return "Since"
}

func outOfMemory(facts domain.Facts, text string) string {
	if facts.OutOfMemory {
		return text
	}
	return ""
}

func detail(text string) string {
	if text == "" {
		return ""
	}
	return " · " + text
}

func failureOf(facts domain.Facts) *failure {
	message := truncate(facts.FailureMessage, failureMessageLimit)
	if facts.FailureCode == "" && message == "" {
		return nil
	}
	return &failure{Code: facts.FailureCode, Message: message}
}

func breakable(value string) htmltemplate.HTML {
	segments := strings.SplitAfter(value, "/")
	for index, segment := range segments {
		segments[index] = htmltemplate.HTMLEscapeString(segment)
	}
	return htmltemplate.HTML(strings.Join(segments, "<wbr>"))
}

func nameOrShortID(name, id string) string {
	if name != "" {
		return name
	}
	return domain.ShortID(id)
}
