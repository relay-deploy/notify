package email

import (
	"fmt"
	"strconv"
	"time"

	"github.com/relay-deploy/notify/internal/domain"
)

const (
	tickLimit           = 24
	failureMessageLimit = 600
	imageNotFound       = "image.not_found"
	imageAdvice         = "Check that the tag was pushed and that the registry is reachable, then deploy again."
)

func (c *Composer) deployed(f domain.Facts, about naming) page {
	took := "The deploy took " + spell(f.Duration) + "."
	body := []segment{plain("The new release of "), bold(about.app), plain(" is live. " + took)}
	summary := fmt.Sprintf("%s was deployed. %s", about.inProject(), took)
	if f.ImageReference != "" {
		body = []segment{bold(f.ImageReference), plain(" is live. " + took)}
		summary = fmt.Sprintf("%s was deployed: %s is live. %s", about.inProject(), f.ImageReference, took)
	}
	return page{
		Subject:  fmt.Sprintf("✓ %s deployed · %s", about.app, humanize(f.Duration)),
		Headline: about.app + " is deployed",
		Calm:     true,
		Summary:  summary,
		Body:     body,
		Rows: present(
			monoRow("Image", f.ImageReference),
			monoRow("Digest", shortDigest(f.ImageDigest)),
			monoRow("Release", domain.ShortID(f.ReleaseID)),
		),
	}
}

func (c *Composer) deployFailed(f domain.Facts, about naming) page {
	lead := "Deploying " + about.inProject() + " failed: "
	target := f.ImageReference
	if target == "" {
		target = about.app
	}
	body := []segment{plain("relay couldn't deploy "), bold(target), plain(parenthesized(f.FailureCode) + ".")}
	summary := lead + "relay couldn't deploy " + target + parenthesized(f.FailureCode) + "."
	if f.FailureCode == imageNotFound && f.ImageReference != "" {
		body = []segment{plain("relay couldn't find the image "), bold(f.ImageReference), plain(". " + imageAdvice)}
		summary = lead + "the image " + f.ImageReference + " was not found. " + imageAdvice
	}
	return page{
		Subject:  fmt.Sprintf("✗ %s deploy failed%s", about.app, detail(f.FailureCode)),
		Headline: about.app + " deploy failed",
		Summary:  summary,
		Body:     body,
		Failure:  failureOf(f),
		Rows: present(
			monoRow("Image", f.ImageReference),
			monoRow("Release", domain.ShortID(f.ReleaseID)),
			textRow("Took", tookOf(f.Duration)),
		),
	}
}

func (c *Composer) crashed(e domain.Email, about naming) page {
	f := e.Facts
	first := f.Since
	if first.IsZero() {
		first = e.At
	}
	at := c.clock(first)
	return page{
		Subject:  fmt.Sprintf("● %s crashed · exit %d%s", about.app, f.ExitCode, outOfMemory(f, ", out of memory")),
		Headline: about.app + " crashed and is restarting",
		Summary:  fmt.Sprintf("%s crashed at %s: it %s. Docker restarted it automatically.", about.inProject(), at, exitCause(f)),
		Body: []segment{
			plain(fmt.Sprintf("It %s at %s. Docker restarted it straight away. You will hear again when ", exitCause(f), at)),
			bold(about.app),
			plain(fmt.Sprintf(" has stayed up for %s, or in %s if it keeps crashing.", spell(c.quietPeriod), spell(c.reminderInterval))),
		},
		Timeline: c.timeline(f.CrashCount, first, time.Time{}, false),
		Rows: present(
			textRow("Exit code", exitCode(f)),
			textRow("Restarts", positive(f.RestartCount)),
			monoRow("Release", domain.ShortID(f.ReleaseID)),
		),
	}
}

func (c *Composer) stillCrashing(f domain.Facts, about naming) page {
	elapsed := f.Until.Sub(f.Since)
	since := c.clock(f.Since)
	return page{
		Subject:  fmt.Sprintf("● %s is still crashing · %d crashes in %s", about.app, f.CrashCount, humanize(elapsed)),
		Headline: about.app + " is still crashing",
		Summary: fmt.Sprintf("%s is still crashing: %s since %s, last with exit code %d%s.",
			about.inProject(), quantity(f.CrashCount, "crash", "crashes"), since, f.ExitCode, outOfMemory(f, " (out of memory)")),
		Body: []segment{
			bold(about.app),
			plain(fmt.Sprintf(" has crashed %s in the last %s, most recently with exit code %d%s.%s "+
				"Docker keeps restarting it, with longer pauses each time. The next reminder comes in %s if nothing changes.",
				times(f.CrashCount), spell(elapsed), f.ExitCode, outOfMemory(f, " after running out of memory"), earlierOutOfMemory(f),
				spell(c.reminderInterval))),
		},
		Timeline: c.timeline(f.CrashCount, f.Since, f.Until, false),
		Rows: present(
			textRow("Crashes", fmt.Sprintf("%d since %s", f.CrashCount, since)),
			textRow("Last exit code", exitCode(f)),
			monoRow("Release", domain.ShortID(f.ReleaseID)),
		),
	}
}

func (c *Composer) recovered(f domain.Facts, about naming) page {
	elapsed := humanize(f.Until.Sub(f.Since))
	window := c.crashWindow(f)
	return page{
		Subject:  fmt.Sprintf("● %s recovered · crashed %s over %s", about.app, times(f.CrashCount), elapsed),
		Headline: about.app + " is running again",
		Calm:     true,
		Summary:  fmt.Sprintf("%s recovered: it crashed %s and has run without crashing since.", about.inProject(), window),
		Body: []segment{
			plain(fmt.Sprintf("It crashed %s, %s exit code %d%s, and has stayed up for %s since.%s No action is needed.",
				window, lastExitLead(f.CrashCount), f.ExitCode, outOfMemory(f, " after running out of memory"), spell(c.quietPeriod),
				earlierOutOfMemory(f))),
		},
		Timeline: c.timeline(f.CrashCount, f.Since, f.Until, true),
		Rows: present(
			textRow("Crashes", fmt.Sprintf("%d over %s", f.CrashCount, elapsed)),
			textRow("Last exit code", exitCode(f)),
			monoRow("Release", domain.ShortID(f.ReleaseID)),
		),
	}
}

func (c *Composer) crashWindow(f domain.Facts) string {
	start, end := c.clock(f.Since), c.clock(f.Until)
	switch {
	case f.CrashCount == 1:
		return "once at " + start
	case start == end:
		return times(f.CrashCount) + " at " + start
	}
	return fmt.Sprintf("%s between %s and %s", times(f.CrashCount), start, end)
}

func lastExitLead(crashes int) string {
	if crashes == 1 {
		return "with"
	}
	return "last with"
}

func exitCause(f domain.Facts) string {
	if f.LastOutOfMemory {
		return fmt.Sprintf("ran out of memory and exited with code %d", f.ExitCode)
	}
	return fmt.Sprintf("exited with code %d", f.ExitCode)
}

func exitCode(f domain.Facts) string {
	return strconv.Itoa(f.ExitCode) + outOfMemory(f, ", out of memory")
}

func outOfMemory(f domain.Facts, text string) string {
	if f.LastOutOfMemory {
		return text
	}
	return ""
}

func earlierOutOfMemory(f domain.Facts) string {
	if f.RanOutOfMemory && !f.LastOutOfMemory {
		return " It ran out of memory at least once."
	}
	return ""
}

func positive(count int) string {
	if count <= 0 {
		return ""
	}
	return strconv.Itoa(count)
}

func tookOf(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	return humanize(d)
}

func detail(text string) string {
	if text == "" {
		return ""
	}
	return " · " + text
}

func parenthesized(text string) string {
	if text == "" {
		return ""
	}
	return " (" + text + ")"
}

func failureOf(f domain.Facts) *failure {
	message := truncate(f.FailureMessage, failureMessageLimit)
	if f.FailureCode == "" && message == "" {
		return nil
	}
	return &failure{Code: f.FailureCode, Message: message}
}
