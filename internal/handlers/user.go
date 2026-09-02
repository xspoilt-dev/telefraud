package handlers

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"telefraud/internal/models"
	"telefraud/internal/services/identity"
)

func (h *Handlers) handleStart(ctx context.Context, chatID int64) {
	text := fmt.Sprintf(
		"<b>🛡️ Welcome to @telefraudbot</b>\n\n"+
			"Your community defense network against Telegram fraud.\n\n"+
			"<b>Developer:</b> %s (%s)\n"+
			"Use the menu below to report scammers, check identifiers, or view statistics.",
		html.EscapeString(h.Cfg.DeveloperName), html.EscapeString(h.Cfg.DeveloperUsername),
	)
	msg := tu.Message(tu.ID(chatID), text).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(h.Menu)
	_, _ = h.Bot.SendMessage(msg)
}

func (h *Handlers) handleDev(ctx context.Context, chatID int64) {
	uptime := formatUptime(time.Since(h.StartTime))
	text := fmt.Sprintf(
		"<b>🤖 @telefraudbot System Info</b>\n\n"+
			"<b>Lead Developer:</b> %s (<code>%s</code>)\n"+
			"<b>Uptime:</b> %s\n\n"+
			"<i>Designed &amp; Built for Telegram Group Safety.</i>",
		html.EscapeString(h.Cfg.DeveloperName), html.EscapeString(h.Cfg.DeveloperUsername),
		uptime,
	)
	h.sendHTML(ctx, chatID, text)
}

func formatUptime(d time.Duration) string {
	d = d.Round(time.Second)
	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute
	d -= minutes * time.Minute
	seconds := d / time.Second

	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	parts = append(parts, fmt.Sprintf("%ds", seconds))
	return strings.Join(parts, " ")
}

func (h *Handlers) handleHelp(ctx context.Context, chatID int64) {
	h.sendHTML(ctx, chatID,
		"<b>❓ Help &amp; FAQ</b>\n\n"+
			"Use <b>🔍 Check Identifier</b> to look up an account by username, user ID, or phone number.\n\n"+
			"Use <b>🛡️ Report Fraudster</b> to report a scam account with proof.",
	)
}

func (h *Handlers) promptCheck(ctx context.Context, chatID int64) {
	h.sendHTML(ctx, chatID,
		"<b>🔍 Check Identifier</b>\n\n"+
			"Send an identifier to look up:\n"+
			"• Username: <code>@handle</code>\n"+
			"• User ID: <code>123456789</code>\n"+
			"• Phone: <code>+1234567890</code>\n\n"+
			"Example: <code>/check @handle</code>",
	)
}

func (h *Handlers) handleCheck(ctx context.Context, chatID int64, raw string) {
	cand, ok := parseCandidates(raw)
	if !ok {
		h.sendHTML(ctx, chatID,
			"<b>⚠️ Unrecognized identifier.</b>\n\n"+
				"Send a username (<code>@handle</code>), user ID, or phone number (<code>+…</code>).",
		)
		return
	}

	scam, err := h.Identity.Resolve(ctx, cand)
	if err != nil && !errors.Is(err, identity.ErrNoMatch) {
		h.sendHTML(ctx, chatID, "<b>⚠️ Lookup failed, try again later.</b>")
		return
	}
	if errors.Is(err, identity.ErrNoMatch) || scam == nil {
		h.sendHTML(ctx, chatID, "<b>✅ No fraud record found.</b>\n\nThe identifier is not in our blacklist.")
		return
	}

	h.renderScammer(ctx, chatID, scam)
}

// renderScammer shows a resolved scammer with every linked identifier, so a
// user can see the current username as well as the immutable user id.
func (h *Handlers) renderScammer(ctx context.Context, chatID int64, scam *models.Scammer) {
	_, idents, err := h.Identity.Get(ctx, scam.ID)
	if err != nil {
		h.sendHTML(ctx, chatID, "<b>⚠️ Lookup failed, try again later.</b>")
		return
	}

	var userID, username, phone string
	for _, i := range idents {
		switch i.Kind {
		case models.KindUserID:
			userID = i.Value
		case models.KindUsername:
			username = "@" + i.Value
		case models.KindPhone:
			phone = i.Value
		}
	}

	var b strings.Builder
	b.WriteString("<b>⚠️ Fraud Record Found</b>\n\n")
	if userID != "" {
		fmt.Fprintf(&b, "<b>User ID:</b> <code>%s</code>\n", html.EscapeString(userID))
	}
	if username != "" {
		fmt.Fprintf(&b, "<b>Username:</b> <code>%s</code>\n", html.EscapeString(username))
	}
	if phone != "" {
		fmt.Fprintf(&b, "<b>Phone:</b> <code>%s</code>\n", html.EscapeString(phone))
	}
	fmt.Fprintf(&b, "<b>Status:</b> %s\n", html.EscapeString(scam.Status))
	fmt.Fprintf(&b, "<b>Threat Level:</b> %s\n", html.EscapeString(scam.ThreatLevel))
	fmt.Fprintf(&b, "<b>Reports:</b> %d\n", scam.ReportCount)
	if scam.Reason != "" {
		fmt.Fprintf(&b, "<b>Reason:</b> %s\n", html.EscapeString(scam.Reason))
	}
	h.sendHTML(ctx, chatID, b.String())
}

// parseCandidates classifies a raw user input as a username, user id, or phone
// number, returning normalized candidates. Precedence: '@' prefix → username,
// '+' prefix → phone, contains a letter/underscore → username, pure digits →
// user id (phone numbers should be sent with a '+' or country code).
func parseCandidates(raw string) (identity.Candidates, bool) {
	raw = strings.TrimSpace(raw)
	switch {
	case raw == "":
		return identity.Candidates{}, false
	case strings.HasPrefix(raw, "@"):
		u, ok := models.NormalizeUsername(raw)
		if !ok {
			return identity.Candidates{}, false
		}
		return identity.Candidates{Username: u}, true
	case strings.HasPrefix(raw, "+"):
		p, ok := models.NormalizePhone(raw)
		if !ok {
			return identity.Candidates{}, false
		}
		return identity.Candidates{Phone: p}, true
	case hasLetterOrUnderscore(raw):
		u, ok := models.NormalizeUsername(raw)
		if !ok {
			return identity.Candidates{}, false
		}
		return identity.Candidates{Username: u}, true
	default:
		id, ok := models.NormalizeUserID(raw)
		if !ok {
			return identity.Candidates{}, false
		}
		return identity.Candidates{UserID: id}, true
	}
}

func hasLetterOrUnderscore(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' {
			return true
		}
	}
	return false
}
