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

	"telefraud/internal/i18n"
	"telefraud/internal/models"
	"telefraud/internal/services/identity"
)

func (h *Handlers) handleStart(ctx context.Context, userID, chatID int64, text, userLang string) {
	if strings.HasPrefix(text, "/start rep_") {
		target := strings.TrimPrefix(text, "/start rep_")
		h.startReportWizardWithTarget(ctx, userID, chatID, target, userLang)
		return
	}
	if text == "/start report" {
		h.startReportWizard(ctx, userID, chatID, userLang)
		return
	}

	welcomeText := i18n.Format(userLang, "start_welcome",
		html.EscapeString(h.Cfg.DeveloperName), html.EscapeString(h.Cfg.DeveloperUsername),
	)

	msg := tu.Message(tu.ID(chatID), welcomeText).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(h.mainMenu(userLang))
	_, _ = h.Bot.SendMessage(msg)
}

func (h *Handlers) handleDev(ctx context.Context, chatID int64, userLang string) {
	uptime := formatUptime(time.Since(h.StartTime))
	text := i18n.Format(userLang, "dev_info",
		html.EscapeString(h.Cfg.DeveloperName), html.EscapeString(h.Cfg.DeveloperUsername),
		uptime,
	)
	h.sendHTML(ctx, chatID, text)
}

func (h *Handlers) handleLanguageSelect(ctx context.Context, chatID, userID int64, userLang string) {
	_ = userID
	text := i18n.Get(userLang, "lang_select_title")
	markup := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🇺🇸 English").WithCallbackData("usr_lang:en"),
			tu.InlineKeyboardButton("🇧🇩 বাংলা (Bengali)").WithCallbackData("usr_lang:bn"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🇮🇳 हिन्दी (Hindi)").WithCallbackData("usr_lang:hi"),
		),
	)

	msg := tu.Message(tu.ID(chatID), text).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(markup)
	_, _ = h.Bot.SendMessage(msg)
}

func (h *Handlers) handleUserLanguageCallback(ctx context.Context, query telego.CallbackQuery, newLang string) {
	userID := query.From.ID
	chatID := int64(0)
	if query.Message != nil {
		chatID = query.Message.GetChat().ID
	}

	normLang := i18n.NormalizeLanguage(newLang)
	_ = h.Store.SetUserLanguage(ctx, userID, normLang)

	langName := i18n.LanguageDisplayName(normLang)
	_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		Text:            "Language updated to " + langName,
	})

	confirmText := i18n.Format(normLang, "lang_updated_user", langName)
	if query.Message != nil {
		_, _ = h.Bot.EditMessageText(&telego.EditMessageTextParams{
			ChatID:    tu.ID(chatID),
			MessageID: query.Message.GetMessageID(),
			Text:      confirmText,
			ParseMode: telego.ModeHTML,
		})
	}

	// Refresh persistent reply keyboard in new language
	menuMsg := tu.Message(tu.ID(chatID), i18n.Format(normLang, "start_welcome", html.EscapeString(h.Cfg.DeveloperName), html.EscapeString(h.Cfg.DeveloperUsername))).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(h.mainMenu(normLang))
	_, _ = h.Bot.SendMessage(menuMsg)
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

func (h *Handlers) handleHelp(ctx context.Context, chatID int64, userLang string) {
	h.sendHTML(ctx, chatID, i18n.Get(userLang, "help_text"))
}

func (h *Handlers) handleMyReports(ctx context.Context, userID, chatID int64, userLang string) {
	reports, err := h.Store.GetUserReports(ctx, userID)
	if err != nil {
		h.sendHTML(ctx, chatID, "<b>⚠️ Failed to fetch submissions. Try again later.</b>")
		return
	}

	if len(reports) == 0 {
		h.sendHTML(ctx, chatID, i18n.Get(userLang, "myreports_empty"))
		return
	}

	var b strings.Builder
	b.WriteString(i18n.Get(userLang, "myreports_header"))
	for _, r := range reports {
		statusEmoji := "⏳"
		switch r.Status {
		case "APPROVED":
			statusEmoji = "✅"
		case "REJECTED":
			statusEmoji = "❌"
		case "INFO_REQUESTED":
			statusEmoji = "❓"
		}
		fmt.Fprintf(&b, "• <b>Report #%d</b>\n", r.ID)
		fmt.Fprintf(&b, "  <b>Target:</b> <code>%s</code>\n", html.EscapeString(r.Target))
		fmt.Fprintf(&b, "  <b>Category:</b> %s\n", html.EscapeString(r.Category))
		fmt.Fprintf(&b, "  <b>Status:</b> %s %s\n", statusEmoji, html.EscapeString(r.Status))
		fmt.Fprintf(&b, "  <b>Submitted:</b> %s\n\n", r.CreatedAt.Format("Jan 02, 2006 15:04 UTC"))
	}
	h.sendHTML(ctx, chatID, b.String())
}

func (h *Handlers) handleStats(ctx context.Context, chatID int64, userLang string) {
	_ = userLang
	stats, err := h.Store.GetGlobalStats(ctx)
	if err != nil {
		h.sendHTML(ctx, chatID, "<b>⚠️ Failed to fetch statistics. Try again later.</b>")
		return
	}

	text := fmt.Sprintf(
		"<b>📊 Global Fraud Statistics</b>\n\n"+
			"<b>🛡️ Verified Blacklisted Scammers:</b> <code>%d</code>\n"+
			"<b>📋 Total Reports Submitted:</b> <code>%d</code>\n"+
			"<b>⏳ Pending Admin Reviews:</b> <code>%d</code>\n"+
			"<b>👥 Protected Groups:</b> <code>%d</code>\n"+
			"<b>👤 Registered Users:</b> <code>%d</code>\n\n"+
			"<i>Database updated continuously in real-time.</i>",
		stats.VerifiedScammers, stats.TotalReports, stats.PendingReports, stats.ProtectedGroups, stats.TotalUsers,
	)
	h.sendHTML(ctx, chatID, text)
}

func (h *Handlers) promptCheck(ctx context.Context, userID, chatID int64, userLang string) {
	h.Sessions.SetState(userID, StateCheckWaitInput)
	h.sendHTML(ctx, chatID, i18n.Get(userLang, "check_prompt"))
}

func (h *Handlers) handleCheck(ctx context.Context, chatID int64, raw string, userLang string) {
	cand, ok := parseCandidates(raw)
	if !ok {
		h.sendHTML(ctx, chatID, i18n.Get(userLang, "check_invalid"))
		return
	}

	scam, err := h.Identity.Resolve(ctx, cand)
	if err != nil && !errors.Is(err, identity.ErrNoMatch) {
		h.sendHTML(ctx, chatID, "<b>⚠️ Lookup failed, try again later.</b>")
		return
	}
	if errors.Is(err, identity.ErrNoMatch) || scam == nil || scam.Status != models.StatusVerified {
		h.sendHTML(ctx, chatID, i18n.Get(userLang, "check_clean"))
		return
	}

	h.renderScammer(ctx, chatID, scam)
}

// renderScammer shows a resolved scammer with every linked identifier.
func (h *Handlers) renderScammer(ctx context.Context, chatID int64, scam *models.Scammer) {
	_, idents, err := h.Identity.Get(ctx, scam.ID)
	if err != nil {
		h.sendHTML(ctx, chatID, "<b>⚠️ Lookup failed, try again later.</b>")
		return
	}

	var userIDs, usernames, phones []string
	for _, i := range idents {
		switch i.Kind {
		case models.KindUserID:
			userIDs = append(userIDs, i.Value)
		case models.KindUsername:
			usernames = append(usernames, "@"+i.Value)
		case models.KindPhone:
			phones = append(phones, i.Value)
		}
	}

	var b strings.Builder
	b.WriteString("<b>⚠️ VERIFIED FRAUD RECORD FOUND</b>\n\n")
	if len(userIDs) > 0 {
		fmt.Fprintf(&b, "<b>User ID(s):</b> <code>%s</code>\n", html.EscapeString(strings.Join(userIDs, ", ")))
	}
	if len(usernames) > 0 {
		fmt.Fprintf(&b, "<b>Username(s):</b> <code>%s</code>\n", html.EscapeString(strings.Join(usernames, ", ")))
	}
	if len(phones) > 0 {
		fmt.Fprintf(&b, "<b>Phone(s):</b> <code>%s</code>\n", html.EscapeString(strings.Join(phones, ", ")))
	}
	fmt.Fprintf(&b, "<b>Status:</b> %s 🚫\n", html.EscapeString(scam.Status))
	fmt.Fprintf(&b, "<b>Threat Level:</b> %s\n", html.EscapeString(scam.ThreatLevel))
	if scam.Category != "" {
		fmt.Fprintf(&b, "<b>Category:</b> %s\n", html.EscapeString(scam.Category))
	}
	fmt.Fprintf(&b, "<b>Verified Reports:</b> %d\n", scam.ReportCount)
	if scam.Reason != "" {
		fmt.Fprintf(&b, "<b>Reason:</b> %s\n", html.EscapeString(scam.Reason))
	}
	b.WriteString("\n<i>Exercise extreme caution. Do not send funds or engage in transactions with this entity.</i>")
	h.sendHTML(ctx, chatID, b.String())
}

// parseCandidates classifies a raw user input as a username, user id, or phone.
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

func (h *Handlers) checkChannelMembership(ctx context.Context, userID int64) (bool, error) {
	_ = ctx
	if h.Cfg.RequiredChannelID == 0 {
		return true, nil
	}

	member, err := h.Bot.GetChatMember(&telego.GetChatMemberParams{
		ChatID: tu.ID(h.Cfg.RequiredChannelID),
		UserID: userID,
	})
	if err != nil {
		return false, err
	}

	status := member.MemberStatus()
	switch status {
	case "creator", "administrator", "member", "restricted":
		return true, nil
	default:
		return false, nil
	}
}

func (h *Handlers) promptMustJoin(ctx context.Context, chatID int64, userLang string) {
	_ = ctx
	text := i18n.Format(userLang, "channel_required", html.EscapeString(h.Cfg.RequiredChannelLink))

	inlineMarkup := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(i18n.Get(userLang, "btn_join_channel")).WithURL(h.Cfg.RequiredChannelLink),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(i18n.Get(userLang, "btn_i_have_joined")).WithCallbackData("check_subscription"),
		),
	)

	msg := tu.Message(tu.ID(chatID), text).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(inlineMarkup)
	_, _ = h.Bot.SendMessage(msg)
}
