package handlers

import (
	"context"
	"fmt"
	"html"
	"log"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"telefraud/internal/i18n"
	"telefraud/internal/models"
)

// handleGroupSettings displays the interactive moderation settings dashboard for a group.
func (h *Handlers) handleGroupSettings(ctx context.Context, chat telego.Chat, userID int64) {
	if chat.Type == telego.ChatTypePrivate {
		h.sendHTML(ctx, chat.ID, "<b>⚠️ This command can only be used in Telegram Groups.</b>")
		return
	}

	isAdmin, err := h.isGroupAdmin(ctx, chat.ID, userID)
	if err != nil {
		h.sendHTML(ctx, chat.ID, "<b>⚠️ Permission Check Failed:</b> Please ensure <b>@telefraudbot</b> is promoted to <b>Administrator</b> in this group so it can verify admin permissions.")
		return
	}
	if !isAdmin {
		h.sendHTML(ctx, chat.ID, "<b>⛔ Access Denied:</b> Only group administrators and the group owner can configure TelefFraud protection.")
		return
	}

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

	text := formatGroupSettingsText(group)
	markup := buildGroupSettingsMarkup(group)

	msg := tu.Message(tu.ID(chat.ID), text).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(markup)
	_, _ = h.Bot.SendMessage(msg)
}

func formatGroupSettingsText(g *models.Group) string {
	lastScan := "Never"
	if g.LastScannedAt != nil {
		lastScan = g.LastScannedAt.Format("Jan 02, 2006 15:04 UTC")
	}

	langName := i18n.LanguageDisplayName(g.Language)
	return i18n.Format(g.Language, "group_settings_title",
		html.EscapeString(g.Title), g.GroupID, langName, lastScan,
	)
}

func toggleIcon(enabled bool) string {
	if enabled {
		return "🟢 ON"
	}
	return "🔴 OFF"
}

func buildGroupSettingsMarkup(g *models.Group) *telego.InlineKeyboardMarkup {
	langName := i18n.LanguageDisplayName(g.Language)
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(fmt.Sprintf("🛡️ Auto-Ban: %s", toggleIcon(g.AutoBanEnabled))).
				WithCallbackData(fmt.Sprintf("grp_tgl:autoban:%d", g.GroupID)),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(fmt.Sprintf("🗑️ Auto-Delete: %s", toggleIcon(g.AutoDeleteEnabled))).
				WithCallbackData(fmt.Sprintf("grp_tgl:autodel:%d", g.GroupID)),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(fmt.Sprintf("⚡ Scan on Join: %s", toggleIcon(g.ScanOnJoinEnabled))).
				WithCallbackData(fmt.Sprintf("grp_tgl:joinscan:%d", g.GroupID)),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(fmt.Sprintf("⏰ Daily Scan: %s", toggleIcon(g.DailyScanEnabled))).
				WithCallbackData(fmt.Sprintf("grp_tgl:dailyscan:%d", g.GroupID)),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(fmt.Sprintf("⚠️ Warn Banner: %s", toggleIcon(g.WarnOnDetected))).
				WithCallbackData(fmt.Sprintf("grp_tgl:warn:%d", g.GroupID)),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(fmt.Sprintf("🌐 Language: %s", langName)).
				WithCallbackData(fmt.Sprintf("grp_tgl:lang:%d", g.GroupID)),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🔄 Refresh").WithCallbackData(fmt.Sprintf("grp_tgl:refresh:%d", g.GroupID)),
			tu.InlineKeyboardButton("❌ Close").WithCallbackData(fmt.Sprintf("grp_tgl:close:%d", g.GroupID)),
		),
	)
}

// handleGroupSettingsToggle handles inline callback toggle interactions.
func (h *Handlers) handleGroupSettingsToggle(ctx context.Context, query telego.CallbackQuery, payload string) {
	parts := strings.Split(payload, ":")
	if len(parts) < 2 {
		return
	}

	settingKey := parts[0]
	groupID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}

	userID := query.From.ID
	isAdmin, err := h.isGroupAdmin(ctx, groupID, userID)
	if err != nil || !isAdmin {
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "⛔ Only group administrators can modify settings.",
			ShowAlert:       true,
		})
		return
	}

	if settingKey == "close" {
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "Settings closed.",
		})
		if query.Message != nil {
			_ = h.Bot.DeleteMessage(&telego.DeleteMessageParams{
				ChatID:    tu.ID(query.Message.GetChat().ID),
				MessageID: query.Message.GetMessageID(),
			})
		}
		return
	}

	group, err := h.Store.GetGroup(ctx, groupID)
	if err != nil {
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "⚠️ Failed to load group settings.",
			ShowAlert:       true,
		})
		return
	}

	switch settingKey {
	case "autoban":
		group.AutoBanEnabled = !group.AutoBanEnabled
	case "autodel":
		group.AutoDeleteEnabled = !group.AutoDeleteEnabled
	case "joinscan":
		group.ScanOnJoinEnabled = !group.ScanOnJoinEnabled
	case "dailyscan":
		group.DailyScanEnabled = !group.DailyScanEnabled
	case "warn":
		group.WarnOnDetected = !group.WarnOnDetected
	case "lang":
		switch i18n.NormalizeLanguage(group.Language) {
		case i18n.LangEN:
			group.Language = i18n.LangBN
		case i18n.LangBN:
			group.Language = i18n.LangHI
		case i18n.LangHI:
			group.Language = i18n.LangEN
		default:
			group.Language = i18n.LangEN
		}
		_ = h.Store.SetGroupLanguage(ctx, group.GroupID, group.Language)
	case "refresh":
		// Just re-render
	}

	_ = h.Store.UpdateGroupSettings(ctx, group.GroupID, group.AutoBanEnabled, group.AutoDeleteEnabled, group.ScanOnJoinEnabled, group.DailyScanEnabled, group.WarnOnDetected)

	_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		Text:            "Settings updated!",
	})

	if query.Message != nil {
		text := formatGroupSettingsText(group)
		markup := buildGroupSettingsMarkup(group)
		_, _ = h.Bot.EditMessageText(&telego.EditMessageTextParams{
			ChatID:      tu.ID(query.Message.GetChat().ID),
			MessageID:   query.Message.GetMessageID(),
			Text:        text,
			ParseMode:   telego.ModeHTML,
			ReplyMarkup: markup,
		})
	}
}

func (h *Handlers) isGroupAdmin(ctx context.Context, groupID, userID int64) (bool, error) {
	if userID == 0 {
		return false, nil
	}

	// Telegram Group Anonymous Bot (ID: 1087968824 / @GroupAnonymousBot)
	// When group owners/admins post with "Remain Anonymous" or as a Channel, Telegram routes messages via this bot ID.
	if userID == 1087968824 {
		return true, nil
	}

	// Superadmins always count as group admins
	if h.Cfg.IsAdmin(userID) {
		return true, nil
	}

	// Bot administrators from DB
	if u, err := h.Store.GetUser(ctx, userID); err == nil && u != nil {
		if u.Role == models.RoleAdmin || u.Role == models.RoleSuperAdmin {
			return true, nil
		}
	}

	// 1. Primary check: Query all chat administrators (includes group Creator/Owner and Admins)
	admins, err := h.Bot.GetChatAdministrators(&telego.GetChatAdministratorsParams{
		ChatID: tu.ID(groupID),
	})
	if err == nil {
		for _, admin := range admins {
			if admin.MemberUser().ID == userID {
				return true, nil
			}
		}
		return false, nil
	}

	log.Printf("GetChatAdministrators error (groupID: %d): %v, attempting GetChatMember fallback", groupID, err)

	// 2. Fallback check: Direct GetChatMember lookup
	member, err := h.Bot.GetChatMember(&telego.GetChatMemberParams{
		ChatID: tu.ID(groupID),
		UserID: userID,
	})
	if err != nil {
		log.Printf("GetChatMember error (groupID: %d, userID: %d): %v", groupID, userID, err)
		return false, err
	}

	status := strings.ToLower(member.MemberStatus())
	return status == "creator" || status == "administrator" || status == "owner", nil
}
