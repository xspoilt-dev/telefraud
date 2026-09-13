package handlers

import (
	"context"
	"fmt"
	"html"
	"log"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"telefraud/internal/i18n"
	"telefraud/internal/models"
	"telefraud/internal/services/identity"
)

var scamKeywords = []string{
	"scammer", "scam", "scams", "scammed", "fraud", "fraudster", "cheat", "cheater",
	"batpar", "batparr", "butpar", "batpari", "বাটপার", "বাটপারি",
	"chor", "chhor", "চোর", "চুরি", "चोर", "चोरी",
	"dhokebaaz", "dhokhebaaz", "dhoka", "dhokha", "ধোঁকাবাজ", "ধোকা", "धोखेबाज", "धोखा",
	"protarok", "protarona", "প্রতারক", "প্রতারণা",
	"thief", "lootera", "lutera", "लुटेरा", "thag", "thaggi", "ठग", "ठगी", "জালিয়াত", "জালিয়াতি",
	"ripper", "imposter", "phishing", "fake", "420",
}

// containsScamKeyword checks if text contains any scammer/fraud accusation terms.
func containsScamKeyword(text string) bool {
	lower := strings.ToLower(text)
	words := strings.FieldsFunc(lower, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 0x0980 && r <= 0x09FF) || (r >= 0x0900 && r <= 0x097F) || r == '_')
	})
	for _, w := range words {
		for _, kw := range scamKeywords {
			if w == kw {
				return true
			}
		}
	}
	for _, kw := range scamKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// handleJoinGate checks newly added members against the blacklist.
func (h *Handlers) handleJoinGate(ctx context.Context, chat telego.Chat, members []telego.User) {
	group, err := h.Store.GetGroup(ctx, chat.ID)
	if err != nil {
		group = &models.Group{
			GroupID:           chat.ID,
			Title:             chat.Title,
			Username:          chat.Username,
			Language:          i18n.LangEN,
			AutoBanEnabled:    true,
			AutoDeleteEnabled: true,
			ScanOnJoinEnabled: true,
			DailyScanEnabled:  true,
			WarnOnDetected:    true,
		}
		_ = h.Store.UpsertGroup(ctx, *group)
	}

	if !group.ScanOnJoinEnabled {
		return
	}

	globalMode, _ := h.Store.GetSetting(ctx, "ENFORCEMENT_MODE", "BAN")

	for _, m := range members {
		if m.IsBot {
			continue
		}

		cand := identity.Candidates{UserID: m.ID}
		if u, ok := models.NormalizeUsername(m.Username); ok {
			cand.Username = u
		}

		scam, err := h.Identity.Resolve(ctx, cand)
		if err != nil {
			if err == identity.ErrNoMatch {
				continue
			}
			log.Printf("join gate: resolve member %d: %v", m.ID, err)
			continue
		}
		if scam.Status != models.StatusVerified {
			continue
		}

		h.enforceFraudster(ctx, chat.ID, m, scam, globalMode, group, "Detected on group join")
	}
}

// handleGroupMessage intercepts group chat messages for real-time security inspection.
func (h *Handlers) handleGroupMessage(ctx context.Context, msg telego.Message) {
	chat := msg.Chat
	sender := msg.From
	if sender == nil || sender.IsBot {
		return
	}

	// Register / update group in DB
	_ = h.Store.UpsertGroup(ctx, models.Group{
		GroupID:           chat.ID,
		Title:             chat.Title,
		Username:          chat.Username,
		Language:          i18n.LangEN,
		AutoBanEnabled:    true,
		AutoDeleteEnabled: true,
		ScanOnJoinEnabled: true,
		DailyScanEnabled:  true,
		WarnOnDetected:    true,
	})

	group, _ := h.Store.GetGroup(ctx, chat.ID)
	groupLang := i18n.LangEN
	if group != nil && group.Language != "" {
		groupLang = group.Language
	}

	globalMode, _ := h.Store.GetSetting(ctx, "ENFORCEMENT_MODE", "BAN")

	// 1. Check sender against fraud database
	cand := identity.Candidates{UserID: sender.ID}
	if u, ok := models.NormalizeUsername(sender.Username); ok {
		cand.Username = u
	}

	scam, err := h.Identity.Resolve(ctx, cand)
	if err == nil && scam != nil && scam.Status == models.StatusVerified {
		// Delete scam message immediately
		if group == nil || group.AutoDeleteEnabled {
			_ = h.Bot.DeleteMessage(&telego.DeleteMessageParams{
				ChatID:    tu.ID(chat.ID),
				MessageID: msg.MessageID,
			})
		}

		h.enforceFraudster(ctx, chat.ID, *sender, scam, globalMode, group, "Sent message in protected group")
		return
	}

	// 2. Check for mentioned scam handles or phones in text
	text := msg.Text
	if text != "" {
		words := strings.Fields(text)
		for _, w := range words {
			if strings.HasPrefix(w, "@") || strings.HasPrefix(w, "+") {
				if matchCand, ok := parseCandidates(w); ok {
					if mentionedScam, mErr := h.Identity.Resolve(ctx, matchCand); mErr == nil && mentionedScam != nil && mentionedScam.Status == models.StatusVerified {
						if group == nil || group.WarnOnDetected {
							warnMsg := i18n.Format(groupLang, "group_fraud_mention", html.EscapeString(w), html.EscapeString(mentionedScam.ThreatLevel), html.EscapeString(mentionedScam.Reason))
							h.sendHTML(ctx, chat.ID, warnMsg)
						}
						break
					}
				}
			}
		}

		// 3. Check for scam accusation words (batpar, scammer, chor, fraud, etc.)
		if containsScamKeyword(text) {
			h.handleScamAccusationTrigger(ctx, msg, group)
		}
	}
}

// handleScamAccusationTrigger prompts the user to submit an official report when fraud words are detected.
func (h *Handlers) handleScamAccusationTrigger(ctx context.Context, msg telego.Message, group *models.Group) {
	groupLang := i18n.LangEN
	if group != nil && group.Language != "" {
		groupLang = group.Language
	}

	botUser := h.BotUsername
	if botUser == "" {
		botUser = "telefraudbot"
	}

	var promptText string
	var startParam string

	if msg.ReplyToMessage != nil && msg.ReplyToMessage.From != nil && !msg.ReplyToMessage.From.IsBot {
		targetUser := *msg.ReplyToMessage.From
		targetName := displayName(targetUser)
		promptText = i18n.Format(groupLang, "trigger_scam_with_target", html.EscapeString(targetName))
		if targetUser.Username != "" {
			startParam = "rep_" + targetUser.Username
		} else {
			startParam = fmt.Sprintf("rep_%d", targetUser.ID)
		}
	} else {
		promptText = i18n.Get(groupLang, "trigger_scam_detected")
		startParam = "report"
	}

	deepLink := fmt.Sprintf("https://t.me/%s?start=%s", botUser, startParam)

	markup := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(i18n.Get(groupLang, "btn_submit_report")).WithURL(deepLink),
		),
	)

	replyMsg := tu.Message(tu.ID(msg.Chat.ID), promptText).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(markup).
		WithReplyParameters(&telego.ReplyParameters{
			MessageID: msg.MessageID,
		})

	_, _ = h.Bot.SendMessage(replyMsg)
}

// handleGroupLanguageCommand displays the language selector in a group chat for admins.
func (h *Handlers) handleGroupLanguageCommand(ctx context.Context, chat telego.Chat, senderID int64) {
	if chat.Type == telego.ChatTypePrivate {
		return
	}

	isAdmin, err := h.isGroupAdmin(ctx, chat.ID, senderID)
	if err != nil {
		h.sendHTML(ctx, chat.ID, "<b>⚠️ Permission Check Failed:</b> Please ensure <b>@telefraudbot</b> is promoted to <b>Administrator</b> in this group.")
		return
	}
	if !isAdmin {
		h.sendHTML(ctx, chat.ID, "<b>⛔ Access Denied:</b> Only group administrators and the group owner can change group language.")
		return
	}

	text := i18n.Get(h.getGroupLang(ctx, chat.ID), "lang_select_title")
	markup := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🇺🇸 English").WithCallbackData(fmt.Sprintf("grp_lang:en:%d", chat.ID)),
			tu.InlineKeyboardButton("🇧🇩 বাংলা (Bengali)").WithCallbackData(fmt.Sprintf("grp_lang:bn:%d", chat.ID)),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🇮🇳 हिन्दी (Hindi)").WithCallbackData(fmt.Sprintf("grp_lang:hi:%d", chat.ID)),
			tu.InlineKeyboardButton("❌ Close").WithCallbackData(fmt.Sprintf("grp_tgl:close:%d", chat.ID)),
		),
	)

	msg := tu.Message(tu.ID(chat.ID), text).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(markup)
	_, _ = h.Bot.SendMessage(msg)
}

// handleGroupLanguageCallback updates the language setting for a group chat.
func (h *Handlers) handleGroupLanguageCallback(ctx context.Context, query telego.CallbackQuery, groupID int64, newLang string) {
	userID := query.From.ID
	isAdmin, err := h.isGroupAdmin(ctx, groupID, userID)
	if err != nil || !isAdmin {
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "⛔ Only group administrators can change language.",
			ShowAlert:       true,
		})
		return
	}

	normLang := i18n.NormalizeLanguage(newLang)
	_ = h.Store.SetGroupLanguage(ctx, groupID, normLang)

	langName := i18n.LanguageDisplayName(normLang)
	_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		Text:            "Language updated to " + langName,
	})

	confirmText := i18n.Format(normLang, "lang_updated_group", langName)
	if query.Message != nil {
		_, _ = h.Bot.EditMessageText(&telego.EditMessageTextParams{
			ChatID:    tu.ID(query.Message.GetChat().ID),
			MessageID: query.Message.GetMessageID(),
			Text:      confirmText,
			ParseMode: telego.ModeHTML,
		})
	}
}

// enforceFraudster executes the configured enforcement action (BAN, KICK, WARN, SILENT).
func (h *Handlers) enforceFraudster(ctx context.Context, chatID int64, target telego.User, scam *models.Scammer, mode string, group *models.Group, reason string) {
	actionTaken := models.ActionBan

	switch mode {
	case "KICK":
		actionTaken = models.ActionKick
		_ = h.Bot.BanChatMember(&telego.BanChatMemberParams{
			ChatID:         tu.ID(chatID),
			UserID:         target.ID,
			RevokeMessages: true,
		})
		_ = h.Bot.UnbanChatMember(&telego.UnbanChatMemberParams{
			ChatID: tu.ID(chatID),
			UserID: target.ID,
		})
	case "WARN":
		actionTaken = models.ActionWarn
	case "SILENT":
		actionTaken = models.ActionSilent
		_ = h.Bot.BanChatMember(&telego.BanChatMemberParams{
			ChatID:         tu.ID(chatID),
			UserID:         target.ID,
			RevokeMessages: true,
		})
	default: // "BAN"
		actionTaken = models.ActionBan
		_ = h.Bot.BanChatMember(&telego.BanChatMemberParams{
			ChatID:         tu.ID(chatID),
			UserID:         target.ID,
			RevokeMessages: true,
		})
	}

	// Log in moderation_logs
	_ = h.Store.LogModeration(ctx, models.ModerationLog{
		GroupID:      &chatID,
		ScammerID:    &scam.ID,
		TargetUserID: target.ID,
		Action:       actionTaken,
		Reason:       reason,
		ExecutedBy:   0,
	})

	groupLang := i18n.LangEN
	if group != nil && group.Language != "" {
		groupLang = group.Language
	}

	// Post public alert unless SILENT or group disabled warnings
	if mode != "SILENT" && (group == nil || group.WarnOnDetected) {
		actionLabel := "Banned"
		switch mode {
		case "KICK":
			actionLabel = "Kicked"
		case "WARN":
			actionLabel = "Flagged / Warned"
		}

		warn := i18n.Format(groupLang, "group_fraud_action", actionLabel, html.EscapeString(displayName(target)), target.ID, html.EscapeString(scam.ThreatLevel), html.EscapeString(scam.Reason))
		h.sendHTML(ctx, chatID, warn)
	}
}

// handleChatMemberUpdate processes real-time Telegram ChatMemberUpdated events.
func (h *Handlers) handleChatMemberUpdate(ctx context.Context, u telego.ChatMemberUpdated) {
	chat := u.Chat
	newMember := u.NewChatMember
	if newMember == nil {
		return
	}

	user := newMember.MemberUser()
	if user.IsBot {
		return
	}

	status := newMember.MemberStatus()
	if status == "member" || status == "restricted" {
		h.handleJoinGate(ctx, chat, []telego.User{user})
	}
}

// handleMyChatMemberUpdate processes when the bot is added or updated in a group.
func (h *Handlers) handleMyChatMemberUpdate(ctx context.Context, u telego.ChatMemberUpdated) {
	chat := u.Chat
	newStatus := u.NewChatMember.MemberStatus()

	if newStatus == "member" || newStatus == "administrator" {
		_ = h.Store.UpsertGroup(ctx, models.Group{
			GroupID:           chat.ID,
			Title:             chat.Title,
			Username:          chat.Username,
			Language:          i18n.LangEN,
			AutoBanEnabled:    true,
			AutoDeleteEnabled: true,
			ScanOnJoinEnabled: true,
			DailyScanEnabled:  true,
			WarnOnDetected:    true,
		})

		group, _ := h.Store.GetGroup(ctx, chat.ID)
		groupLang := i18n.LangEN
		if group != nil && group.Language != "" {
			groupLang = group.Language
		}

		welcomeText := i18n.Format(groupLang, "group_activated",
			html.EscapeString(h.Cfg.DeveloperName), html.EscapeString(h.Cfg.DeveloperUsername),
			i18n.LanguageDisplayName(groupLang),
		)
		h.sendHTML(ctx, chat.ID, welcomeText)
	}
}

// handleScanGroupCommand triggers an instant member audit by a group admin.
func (h *Handlers) handleScanGroupCommand(ctx context.Context, chat telego.Chat, senderID int64) {
	if chat.Type == telego.ChatTypePrivate {
		h.sendHTML(ctx, chat.ID, "<b>⚠️ This command is for Telegram Groups.</b>")
		return
	}

	isAdmin, err := h.isGroupAdmin(ctx, chat.ID, senderID)
	if err != nil {
		h.sendHTML(ctx, chat.ID, "<b>⚠️ Permission Check Failed:</b> Please ensure <b>@telefraudbot</b> is promoted to <b>Administrator</b> in this group.")
		return
	}
	if !isAdmin {
		h.sendHTML(ctx, chat.ID, "<b>⛔ Access Denied:</b> Only group administrators and the group owner can initiate group scans.")
		return
	}

	h.sendHTML(ctx, chat.ID, "<b>🔍 Initiating security audit against global fraud database...</b>")
	res, err := h.Scanner.ScanGroup(ctx, chat.ID, senderID)
	if err != nil {
		h.sendHTML(ctx, chat.ID, fmt.Sprintf("<b>⚠️ Group scan failed:</b> %v", err))
		return
	}

	summary := fmt.Sprintf(
		"<b>🛡️ Group Audit Completed</b>\n\n"+
			"<b>Audited Members:</b> <code>%d</code>\n"+
			"<b>Blacklisted Accounts Detected:</b> <code>%d</code>\n"+
			"<b>Elapsed Time:</b> <code>%v</code>\n\n",
		res.TotalAudited, len(res.FlaggedMembers), res.Duration.Round(time.Millisecond),
	)

	if len(res.FlaggedMembers) > 0 {
		var b strings.Builder
		b.WriteString(summary)
		b.WriteString("<b>Actioned Scammers:</b>\n")
		for _, fm := range res.FlaggedMembers {
			fmt.Fprintf(&b, "• %s (ID: <code>%d</code>) — <b>%s</b>\n", html.EscapeString(displayName(fm.User)), fm.User.ID, fm.ActionTaken)
		}
		h.sendHTML(ctx, chat.ID, b.String())
	} else {
		h.sendHTML(ctx, chat.ID, summary+"<b>✅ Clean!</b> No blacklisted fraudsters detected in group.")
	}
}

// handleCheckUserCommand checks a group member via reply or argument.
func (h *Handlers) handleCheckUserCommand(ctx context.Context, msg telego.Message) {
	chat := msg.Chat
	senderID := msg.From.ID

	var target string
	if msg.ReplyToMessage != nil && msg.ReplyToMessage.From != nil {
		target = fmt.Sprintf("%d", msg.ReplyToMessage.From.ID)
	} else {
		text := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/checkuser"))
		if text == "" {
			h.sendHTML(ctx, chat.ID, "<b>Usage:</b> Reply to a user with <code>/checkuser</code> or send <code>/checkuser @handle</code>")
			return
		}
		target = text
	}

	cand, ok := parseCandidates(target)
	if !ok {
		h.sendHTML(ctx, chat.ID, "<b>⚠️ Invalid user identifier.</b> Provide a @username or user ID.")
		return
	}

	scam, err := h.Identity.Resolve(ctx, cand)
	if err != nil && err != identity.ErrNoMatch {
		h.sendHTML(ctx, chat.ID, "<b>⚠️ Lookup error, try again later.</b>")
		return
	}

	if err == identity.ErrNoMatch || scam == nil {
		h.sendHTML(ctx, chat.ID, fmt.Sprintf("<b>✅ Clean Record:</b> <code>%s</code> is not listed in the fraud blacklist.", html.EscapeString(target)))
		return
	}

	h.renderScammer(ctx, chat.ID, scam)
	_ = senderID
}

func displayName(u telego.User) string {
	if u.FirstName != "" {
		return u.FirstName
	}
	if u.Username != "" {
		return u.Username
	}
	return fmt.Sprintf("%d", u.ID)
}
