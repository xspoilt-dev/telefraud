package handlers

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"telefraud/config"
	"telefraud/internal/database"
	"telefraud/internal/i18n"
	"telefraud/internal/models"
	"telefraud/internal/services/backup"
	"telefraud/internal/services/identity"
	"telefraud/internal/services/scanner"
)

// Handlers holds the bot client, configuration and services shared across handlers.
type Handlers struct {
	Bot         *telego.Bot
	BotUsername string
	Cfg         *config.Config
	Identity    *identity.Resolver
	Store       *database.IdentityStore
	Sessions    *SessionManager
	Scanner     *scanner.GroupScanner
	Backup      *backup.BackupManager
	StartTime   time.Time
}

// New builds the handler stack.
func New(
	bot *telego.Bot,
	cfg *config.Config,
	resolver *identity.Resolver,
	store *database.IdentityStore,
	groupScanner *scanner.GroupScanner,
	backupManager *backup.BackupManager,
) *Handlers {
	botUser := "telefraudbot"
	me, err := bot.GetMe()
	if err == nil && me != nil && me.Username != "" {
		botUser = me.Username
	}

	return &Handlers{
		Bot:         bot,
		BotUsername: botUser,
		Cfg:         cfg,
		Identity:    resolver,
		Store:       store,
		Sessions:    NewSessionManager(),
		Scanner:     groupScanner,
		Backup:      backupManager,
		StartTime:   time.Now(),
	}
}

func (h *Handlers) mainMenu(lang string) *telego.ReplyKeyboardMarkup {
	return tu.Keyboard(
		tu.KeyboardRow(
			tu.KeyboardButton(i18n.Get(lang, "btn_report")),
			tu.KeyboardButton(i18n.Get(lang, "btn_check")),
		),
		tu.KeyboardRow(
			tu.KeyboardButton(i18n.Get(lang, "btn_myreports")),
			tu.KeyboardButton(i18n.Get(lang, "btn_stats")),
		),
		tu.KeyboardRow(
			tu.KeyboardButton(i18n.Get(lang, "btn_help")),
			tu.KeyboardButton(i18n.Get(lang, "btn_language")),
		),
		tu.KeyboardRow(
			tu.KeyboardButton(i18n.Get(lang, "btn_dev")),
		),
	).WithResizeKeyboard()
}

func (h *Handlers) getUserLang(ctx context.Context, userID int64) string {
	u, err := h.Store.GetUser(ctx, userID)
	if err == nil && u != nil && u.Language != "" {
		return i18n.NormalizeLanguage(u.Language)
	}
	return i18n.LangEN
}

func (h *Handlers) getGroupLang(ctx context.Context, groupID int64) string {
	g, err := h.Store.GetGroup(ctx, groupID)
	if err == nil && g != nil && g.Language != "" {
		return i18n.NormalizeLanguage(g.Language)
	}
	return i18n.LangEN
}

// HandleMessage dispatches an incoming message to private or group routing.
func (h *Handlers) HandleMessage(ctx context.Context, msg telego.Message) {
	chat := msg.Chat
	sender := msg.From
	senderID := int64(0)
	if sender != nil {
		senderID = sender.ID
		// Automatically register/update user in database
		_ = h.Store.UpsertUser(ctx, models.User{
			UserID:    sender.ID,
			Username:  sender.Username,
			FirstName: sender.FirstName,
			LastName:  sender.LastName,
			Role:      models.RoleUser,
		})
	}

	// 1. Group Message Routing
	if chat.Type == telego.ChatTypeGroup || chat.Type == telego.ChatTypeSupergroup {
		if len(msg.NewChatMembers) > 0 {
			h.handleJoinGate(ctx, chat, msg.NewChatMembers)
			return
		}

		// Handle anonymous admins and linked channels posting on behalf of the group/channel
		if msg.SenderChat != nil {
			senderID = 1087968824 // Telegram GroupAnonymousBot ID
		}

		text := strings.TrimSpace(msg.Text)
		switch {
		case text == "/telefraud_settings" || strings.HasPrefix(text, "/telefraud_settings@"):
			h.handleGroupSettings(ctx, chat, senderID)
			return
		case text == "/setlang" || strings.HasPrefix(text, "/setlang@") || text == "/language" || strings.HasPrefix(text, "/language@"):
			h.handleGroupLanguageCommand(ctx, chat, senderID)
			return
		case text == "/scangroup" || strings.HasPrefix(text, "/scangroup@"):
			h.handleScanGroupCommand(ctx, chat, senderID)
			return
		case strings.HasPrefix(text, "/checkuser"):
			h.handleCheckUserCommand(ctx, msg)
			return
		default:
			h.handleGroupMessage(ctx, msg)
			return
		}
	}

	// 2. Private Message Routing
	if chat.Type != telego.ChatTypePrivate {
		return
	}

	userLang := h.getUserLang(ctx, senderID)

	// Check if user is banned
	userRecord, _ := h.Store.GetUser(ctx, senderID)
	if userRecord != nil && userRecord.IsBanned {
		h.sendHTML(ctx, chat.ID, "<b>⛔ Account Suspended:</b> You are prohibited from using @telefraudbot.")
		return
	}

	// Enforce mandatory channel subscription before allowing private bot interactions
	joined, err := h.checkChannelMembership(ctx, senderID)
	if err != nil {
		log.Printf("check subscription for user %d: %v", senderID, err)
	} else if !joined {
		h.promptMustJoin(ctx, chat.ID, userLang)
		return
	}

	text := strings.TrimSpace(msg.Text)

	// Check for document import
	if msg.Document != nil && (strings.HasPrefix(msg.Caption, "/import_blacklist") || h.isAdmin(ctx, senderID)) {
		if strings.HasSuffix(strings.ToLower(msg.Document.FileName), ".csv") || strings.HasSuffix(strings.ToLower(msg.Document.FileName), ".json") {
			h.handleImportDocument(ctx, msg)
			return
		}
	}

	// Check for direct photo broadcast with /broadcast caption
	if len(msg.Photo) > 0 && (msg.Caption == "/broadcast" || strings.HasPrefix(msg.Caption, "/broadcast ")) && h.isAdmin(ctx, senderID) {
		highest := msg.Photo[len(msg.Photo)-1]
		caption := strings.TrimSpace(strings.TrimPrefix(msg.Caption, "/broadcast"))
		h.promptBroadcastPhotoConfirm(ctx, chat.ID, senderID, caption, highest.FileID)
		return
	}

	// Active Wizard Sessions Check
	sess := h.Sessions.Get(senderID)
	if text == "/cancel" {
		h.cancelWizard(ctx, senderID, chat.ID, userLang)
		return
	}

	switch sess.State {
	case StateCheckWaitInput:
		h.Sessions.Clear(senderID)
		h.handleCheck(ctx, chat.ID, text, userLang)
		return
	case StateReportWaitTarget:
		h.handleReportTargetInput(ctx, senderID, chat.ID, text, userLang)
		return
	case StateReportWaitDescription:
		h.handleReportDescriptionInput(ctx, senderID, chat.ID, text, userLang)
		return
	case StateReportWaitProof:
		if msg.Photo != nil || msg.Document != nil || msg.Video != nil {
			h.handleReportProofMedia(ctx, msg, userLang)
			return
		}
		if text == "/submit" || text == "/done" {
			h.finishReportWizard(ctx, senderID, chat.ID, userLang)
			return
		}
	case StateBroadcastWaitText:
		if len(msg.Photo) > 0 {
			highest := msg.Photo[len(msg.Photo)-1]
			h.promptBroadcastPhotoConfirm(ctx, chat.ID, senderID, msg.Caption, highest.FileID)
			return
		}
		h.promptBroadcastConfirm(ctx, chat.ID, senderID, text)
		return
	}

	// Standard Commands & Menu buttons
	switch {
	case strings.HasPrefix(text, "/start"):
		h.handleStart(ctx, senderID, chat.ID, text, userLang)
	case text == "/language" || text == "/setlang" || text == "🌐 Language" || text == "🌐 ভাষা পরিবর্তন" || text == "🌐 भाषा बदलें":
		h.handleLanguageSelect(ctx, chat.ID, senderID, userLang)
	case text == "/dev" || text == "👨‍💻 Developer Info" || text == "👨‍💻 ডেভেলপার তথ্য" || text == "👨‍💻 डेवलपर जानकारी":
		h.handleDev(ctx, chat.ID, userLang)
	case text == "/check" || text == "🔍 Check Identifier" || text == "🔍 আইডি পরীক্ষা করুন" || text == "🔍 पहचान जांचें":
		h.promptCheck(ctx, senderID, chat.ID, userLang)
	case text == "/help" || text == "❓ Help & FAQ" || text == "❓ সাহায্য ও নিয়মাবলী" || text == "❓ सहायता और नियम":
		h.handleHelp(ctx, chat.ID, userLang)
	case text == "/myreports" || text == "📋 My Submissions" || text == "📋 আমার রিপোর্টসমূহ" || text == "📋 मेरी रिपोर्ट्स":
		h.handleMyReports(ctx, senderID, chat.ID, userLang)
	case text == "/stats" || text == "📊 Global Statistics" || text == "📊 সামগ্রিক পরিসংখ্যান" || text == "📊 वैश्विक आँकड़े":
		h.handleStats(ctx, chat.ID, userLang)
	case text == "/report" || text == "🛡️ Report Fraudster" || text == "🛡️ প্রতারক রিপোর্ট করুন" || text == "🛡️ धोखेबाज़ रिपोर्ट करें":
		h.startReportWizard(ctx, senderID, chat.ID, userLang)
	case strings.HasPrefix(text, "/check "):
		h.handleCheck(ctx, chat.ID, strings.TrimSpace(strings.TrimPrefix(text, "/check ")), userLang)
	case strings.HasPrefix(text, "/report "):
		h.handleReportCommand(ctx, senderID, chat.ID, strings.TrimSpace(strings.TrimPrefix(text, "/report ")), userLang)

	// Admin Control Commands
	case text == "/admin":
		h.handleAdminDashboard(ctx, chat.ID, senderID)
	case text == "/pending":
		h.handlePendingReports(ctx, chat.ID, senderID, 0)
	case strings.HasPrefix(text, "/admin_add "):
		h.handleAdminAdd(ctx, chat.ID, senderID, strings.TrimPrefix(text, "/admin_add "))
	case strings.HasPrefix(text, "/admin_remove "):
		h.handleAdminRemove(ctx, chat.ID, senderID, strings.TrimPrefix(text, "/admin_remove "))
	case text == "/admin_list":
		h.handleAdminList(ctx, chat.ID, senderID)
	case strings.HasPrefix(text, "/blacklist add "):
		h.handleBlacklistAddCommand(ctx, chat.ID, senderID, strings.TrimPrefix(text, "/blacklist add "))
	case strings.HasPrefix(text, "/blacklist remove "):
		h.handleBlacklistRemoveCommand(ctx, chat.ID, senderID, strings.TrimPrefix(text, "/blacklist remove "))
	case strings.HasPrefix(text, "/broadcast "):
		h.handleBroadcastCommand(ctx, chat.ID, senderID, strings.TrimPrefix(text, "/broadcast "))
	case text == "/broadcast":
		h.handleBroadcastCommand(ctx, chat.ID, senderID, "")
	case strings.HasPrefix(text, "/export_blacklist"):
		format := strings.TrimSpace(strings.TrimPrefix(text, "/export_blacklist"))
		h.handleExportCommand(ctx, chat.ID, senderID, format)
	case text == "/import_blacklist":
		h.sendHTML(ctx, chat.ID, "<b>📂 Blacklist Import:</b> Please upload a <code>.csv</code> or <code>.json</code> file with caption <code>/import_blacklist</code>.")
	case strings.HasPrefix(text, "/mode"):
		arg := strings.TrimSpace(strings.TrimPrefix(text, "/mode"))
		h.handleModeCommand(ctx, chat.ID, senderID, arg)
	case text == "/backup":
		h.handleBackupCommand(ctx, chat.ID, senderID)
	case strings.HasPrefix(text, "/admin_scangroup "):
		h.handleAdminScanGroup(ctx, chat.ID, senderID, strings.TrimPrefix(text, "/admin_scangroup "))
	}
}

// HandleCallbackQuery routes inline button clicks.
func (h *Handlers) HandleCallbackQuery(ctx context.Context, query telego.CallbackQuery) {
	data := query.Data
	userID := query.From.ID
	chatID := int64(0)
	if query.Message != nil {
		chatID = query.Message.GetChat().ID
	}
	userLang := h.getUserLang(ctx, userID)

	switch {
	case data == "check_subscription":
		joined, err := h.checkChannelMembership(ctx, userID)
		if err != nil || !joined {
			_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
				CallbackQueryID: query.ID,
				Text:            "❌ You have not joined @telefraud_info yet! Please join first.",
				ShowAlert:       true,
			})
			return
		}
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "✅ Subscription verified! You can now use @telefraudbot.",
		})
		h.handleStart(ctx, userID, chatID, "/start", userLang)

	// User Language Selection callbacks
	case strings.HasPrefix(data, "usr_lang:"):
		newLang := strings.TrimPrefix(data, "usr_lang:")
		h.handleUserLanguageCallback(ctx, query, newLang)

	// Group Language Selection callbacks
	case strings.HasPrefix(data, "grp_lang:"):
		parts := strings.Split(strings.TrimPrefix(data, "grp_lang:"), ":")
		if len(parts) >= 2 {
			newLang := parts[0]
			groupID, _ := strconv.ParseInt(parts[1], 10, 64)
			h.handleGroupLanguageCallback(ctx, query, groupID, newLang)
		}

	// Report Wizard callbacks
	case strings.HasPrefix(data, "wiz_cat:"):
		category := strings.TrimPrefix(data, "wiz_cat:")
		h.handleReportCategoryCallback(ctx, query, category, userLang)
	case data == "wiz_submit":
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{CallbackQueryID: query.ID})
		h.finishReportWizard(ctx, userID, chatID, userLang)
	case data == "wiz_cancel":
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{CallbackQueryID: query.ID, Text: "Report cancelled."})
		h.cancelWizard(ctx, userID, chatID, userLang)

	// Report Review Queue callbacks
	case strings.HasPrefix(data, "rep_act:"):
		payload := strings.TrimPrefix(data, "rep_act:")
		h.handleReportActionCallback(ctx, query, payload)

	// Group Settings callbacks
	case strings.HasPrefix(data, "grp_tgl:"):
		payload := strings.TrimPrefix(data, "grp_tgl:")
		h.handleGroupSettingsToggle(ctx, query, payload)

	// Admin Panel callbacks
	case strings.HasPrefix(data, "adm_view_proofs:"):
		reportIDStr := strings.TrimPrefix(data, "adm_view_proofs:")
		h.handleViewProofsCallback(ctx, query, reportIDStr)

	case strings.HasPrefix(data, "adm_pending:"):
		offsetStr := strings.TrimPrefix(data, "adm_pending:")
		offset, _ := strconv.Atoi(offsetStr)
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{CallbackQueryID: query.ID})
		h.handlePendingReports(ctx, chatID, userID, offset)

	case data == "adm_dash_refresh":
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{CallbackQueryID: query.ID, Text: "Dashboard updated."})
		h.handleAdminDashboard(ctx, chatID, userID)

	case data == "adm_bl_menu":
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{CallbackQueryID: query.ID})
		h.sendHTML(ctx, chatID,
			"<b>🚫 Blacklist Manager Commands</b>\n\n"+
				"• Add Scammer: <code>/blacklist add &lt;target&gt; [threat] &lt;reason&gt;</code>\n"+
				"• Remove Scammer: <code>/blacklist remove &lt;target&gt;</code>\n"+
				"• Export: <code>/export_blacklist csv</code> or <code>json</code>\n"+
				"• Import: Upload file with <code>/import_blacklist</code>",
		)

	case data == "adm_mgmt_menu":
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{CallbackQueryID: query.ID})
		h.handleAdminList(ctx, chatID, userID)

	case data == "adm_bcast_menu":
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{CallbackQueryID: query.ID})
		h.handleBroadcastCommand(ctx, chatID, userID, "")

	case strings.HasPrefix(data, "adm_exp:"):
		format := strings.TrimPrefix(data, "adm_exp:")
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{CallbackQueryID: query.ID, Text: "Exporting blacklist..."})
		h.handleExportCommand(ctx, chatID, userID, format)

	case data == "adm_backup_now":
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{CallbackQueryID: query.ID, Text: "Triggering database backup..."})
		h.handleBackupCommand(ctx, chatID, userID)

	case data == "adm_mode_menu":
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{CallbackQueryID: query.ID})
		h.handleModeCommand(ctx, chatID, userID, "")

	case strings.HasPrefix(data, "adm_setmode:"):
		newMode := strings.TrimPrefix(data, "adm_setmode:")
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{CallbackQueryID: query.ID, Text: "Mode set to " + newMode})
		h.handleModeCommand(ctx, chatID, userID, newMode)

	case data == "bcast_act:send":
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{CallbackQueryID: query.ID, Text: "Starting broadcast..."})
		h.executeBroadcast(ctx, chatID, userID)

	case data == "bcast_act:cancel":
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{CallbackQueryID: query.ID, Text: "Broadcast cancelled."})
		h.Sessions.Clear(userID)
		h.sendHTML(ctx, chatID, "<b>❌ Broadcast cancelled.</b>")
	}
}

// HandleChatMember dispatches chat member status transitions.
func (h *Handlers) HandleChatMember(ctx context.Context, u telego.ChatMemberUpdated) {
	h.handleChatMemberUpdate(ctx, u)
}

// HandleMyChatMember dispatches bot membership updates.
func (h *Handlers) HandleMyChatMember(ctx context.Context, u telego.ChatMemberUpdated) {
	h.handleMyChatMemberUpdate(ctx, u)
}

func (h *Handlers) sendHTML(ctx context.Context, chatID int64, text string) {
	_ = ctx
	_, _ = h.Bot.SendMessage(tu.Message(tu.ID(chatID), text).WithParseMode(telego.ModeHTML))
}
