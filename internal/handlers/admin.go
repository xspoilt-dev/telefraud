package handlers

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"telefraud/internal/models"
	"telefraud/internal/services/broadcast"
	"telefraud/internal/services/exporter"
	"telefraud/internal/services/identity"
)

type namedBytesReader struct {
	*bytes.Reader
	name string
}

func (n namedBytesReader) Name() string {
	return n.name
}

// isAdmin checks whether a user has administrative privileges.
func (h *Handlers) isAdmin(ctx context.Context, userID int64) bool {
	if h.Cfg.IsAdmin(userID) {
		return true
	}
	u, err := h.Store.GetUser(ctx, userID)
	if err == nil && u != nil {
		return u.Role == models.RoleAdmin || u.Role == models.RoleSuperAdmin
	}
	return false
}

// handleAdminDashboard opens the interactive Admin Control Panel.
func (h *Handlers) handleAdminDashboard(ctx context.Context, chatID, userID int64) {
	if !h.isAdmin(ctx, userID) {
		h.sendHTML(ctx, chatID, "<b>⛔ Access Denied:</b> This command is restricted to bot administrators.")
		return
	}

	stats, _ := h.Store.GetGlobalStats(ctx)
	mode, _ := h.Store.GetSetting(ctx, "ENFORCEMENT_MODE", "BAN")

	text := fmt.Sprintf(
		"<b>👑 TELEFRAUD BOT ADMIN DASHBOARD</b>\n\n"+
			"<b>🛡️ Verified Scammers:</b> <code>%d</code>\n"+
			"<b>📥 Pending Reports:</b> <code>%d</code>\n"+
			"<b>👥 Total Users:</b> <code>%d</code>\n"+
			"<b>🛡️ Protected Groups:</b> <code>%d</code>\n"+
			"<b>⚙️ Global Mode:</b> <code>%s</code>\n\n"+
			"<i>Select an administrative module from the control panel below:</i>",
		stats.VerifiedScammers, stats.PendingReports, stats.TotalUsers, stats.ProtectedGroups, mode,
	)

	markup := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(fmt.Sprintf("📥 Pending Reports (%d)", stats.PendingReports)).WithCallbackData("adm_pending:0"),
			tu.InlineKeyboardButton("🚫 Blacklist Manager").WithCallbackData("adm_bl_menu"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("👥 Admin Management").WithCallbackData("adm_mgmt_menu"),
			tu.InlineKeyboardButton("📢 Global Broadcast").WithCallbackData("adm_bcast_menu"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("📂 Export CSV").WithCallbackData("adm_exp:csv"),
			tu.InlineKeyboardButton("📂 Export JSON").WithCallbackData("adm_exp:json"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("⚙️ Enforcement Mode").WithCallbackData("adm_mode_menu"),
			tu.InlineKeyboardButton("💾 Trigger DB Backup").WithCallbackData("adm_backup_now"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🔄 Refresh Dashboard").WithCallbackData("adm_dash_refresh"),
		),
	)

	msg := tu.Message(tu.ID(chatID), text).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(markup)
	_, _ = h.Bot.SendMessage(msg)
}

// handlePendingReports displays pending reports review queue with pagination.
func (h *Handlers) handlePendingReports(ctx context.Context, chatID, userID int64, offset int) {
	if !h.isAdmin(ctx, userID) {
		h.sendHTML(ctx, chatID, "<b>⛔ Access Denied.</b>")
		return
	}

	reports, total, err := h.Store.GetPendingReports(ctx, 1, offset)
	if err != nil || len(reports) == 0 {
		h.sendHTML(ctx, chatID, "<b>✅ No Pending Reports!</b>\n\nThe review queue is currently empty.")
		return
	}

	rep := reports[0]
	_, proofs, _ := h.Store.GetReport(ctx, rep.ID)

	var target string
	if rep.TargetUsername != "" {
		target = "@" + rep.TargetUsername
	} else if rep.TargetUserID != nil && *rep.TargetUserID > 0 {
		target = fmt.Sprintf("ID: %d", *rep.TargetUserID)
	} else if rep.TargetPhone != "" {
		target = rep.TargetPhone
	} else {
		target = "Unknown"
	}

	text := fmt.Sprintf(
		"<b>📥 PENDING REPORT [%d of %d]</b>\n\n"+
			"<b>Report ID:</b> <code>#%d</code>\n"+
			"<b>Reporter ID:</b> <code>%d</code>\n"+
			"<b>Target:</b> <code>%s</code>\n"+
			"<b>Category:</b> %s\n"+
			"<b>Proofs:</b> <code>%d file(s)</code>\n"+
			"<b>Submitted:</b> %s\n\n"+
			"<b>Description:</b>\n%s",
		offset+1, total, rep.ID, rep.ReporterID, html.EscapeString(target),
		html.EscapeString(rep.Category), len(proofs), rep.CreatedAt.Format("Jan 02, 2006 15:04 UTC"),
		html.EscapeString(rep.Description),
	)

	var navRow []telego.InlineKeyboardButton
	if offset > 0 {
		navRow = append(navRow, tu.InlineKeyboardButton("⬅️ Prev").WithCallbackData(fmt.Sprintf("adm_pending:%d", offset-1)))
	}
	if int64(offset+1) < total {
		navRow = append(navRow, tu.InlineKeyboardButton("Next ➡️").WithCallbackData(fmt.Sprintf("adm_pending:%d", offset+1)))
	}

	var markupRows [][]telego.InlineKeyboardButton
	if len(proofs) > 0 {
		markupRows = append(markupRows, []telego.InlineKeyboardButton{
			tu.InlineKeyboardButton(fmt.Sprintf("🖼️ View Attached Proofs (%d)", len(proofs))).WithCallbackData(fmt.Sprintf("adm_view_proofs:%d", rep.ID)),
		})
	}
	markupRows = append(markupRows,
		[]telego.InlineKeyboardButton{
			tu.InlineKeyboardButton("✅ Approve").WithCallbackData(fmt.Sprintf("rep_act:approve:%d", rep.ID)),
			tu.InlineKeyboardButton("❌ Reject").WithCallbackData(fmt.Sprintf("rep_act:reject:%d", rep.ID)),
		},
		[]telego.InlineKeyboardButton{
			tu.InlineKeyboardButton("❓ Request Proof").WithCallbackData(fmt.Sprintf("rep_act:req_proof:%d", rep.ID)),
			tu.InlineKeyboardButton("🚫 Ban Reporter").WithCallbackData(fmt.Sprintf("rep_act:ban_rep:%d", rep.ID)),
		},
	)
	if len(navRow) > 0 {
		markupRows = append(markupRows, navRow)
	}
	markupRows = append(markupRows, []telego.InlineKeyboardButton{
		tu.InlineKeyboardButton("🔙 Back to Admin").WithCallbackData("adm_dash_refresh"),
	})

	msg := tu.Message(tu.ID(chatID), text).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(tu.InlineKeyboard(markupRows...))
	_, _ = h.Bot.SendMessage(msg)
}

// handleReportActionCallback processes moderator actions on reports.
func (h *Handlers) handleReportActionCallback(ctx context.Context, query telego.CallbackQuery, payload string) {
	reviewerID := query.From.ID
	if !h.isAdmin(ctx, reviewerID) {
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "⛔ Admin privilege required.",
			ShowAlert:       true,
		})
		return
	}

	parts := strings.Split(payload, ":")
	if len(parts) < 2 {
		return
	}

	action := parts[0]
	reportID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}

	rep, _, err := h.Store.GetReport(ctx, reportID)
	if err != nil || rep == nil {
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "⚠️ Report not found.",
			ShowAlert:       true,
		})
		return
	}

	switch action {
	case "approve":
		_ = h.Store.UpdateReportStatus(ctx, reportID, models.StatusReportApproved, &reviewerID, "Approved by Admin")

		cand := identity.Candidates{
			Username: rep.TargetUsername,
			Phone:    rep.TargetPhone,
		}
		if rep.TargetUserID != nil {
			cand.UserID = *rep.TargetUserID
		}

		if !cand.Empty() {
			scam, regErr := h.Identity.Register(ctx, cand, rep.Description, rep.Category, &reviewerID)
			if regErr == nil && scam != nil {
				scam.Status = models.StatusVerified
				scam.ThreatLevel = models.ThreatHigh
				_ = h.Store.Update(ctx, scam)
			}
		}

		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            fmt.Sprintf("✅ Report #%d APPROVED!", reportID),
		})

		reporterMsg := fmt.Sprintf(
			"<b>✅ Report Approved!</b>\n\n"+
				"Your report <b>#%d</b> has been verified and approved by the moderation team.\n"+
				"The target account has been permanently added to the global blacklist.",
			reportID,
		)
		h.sendHTML(ctx, rep.ReporterID, reporterMsg)

	case "reject":
		_ = h.Store.UpdateReportStatus(ctx, reportID, models.StatusReportRejected, &reviewerID, "Rejected by Admin")

		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            fmt.Sprintf("❌ Report #%d REJECTED.", reportID),
		})

		reporterMsg := fmt.Sprintf(
			"<b>❌ Report Update</b>\n\n"+
				"Your report <b>#%d</b> was reviewed by our moderation team and was not approved at this time due to insufficient verifiable evidence.",
			reportID,
		)
		h.sendHTML(ctx, rep.ReporterID, reporterMsg)

	case "req_proof":
		_ = h.Store.UpdateReportStatus(ctx, reportID, models.StatusReportInfoRequested, &reviewerID, "Additional proof requested")

		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            fmt.Sprintf("❓ Proof requested for Report #%d.", reportID),
		})

		reporterMsg := fmt.Sprintf(
			"<b>❓ Additional Proof Requested</b>\n\n"+
				"Regarding your report <b>#%d</b>: our team requires additional screenshots or transaction details before taking enforcement action. Please submit an updated report with further proof.",
			reportID,
		)
		h.sendHTML(ctx, rep.ReporterID, reporterMsg)

	case "ban_rep":
		_ = h.Store.UpdateReportStatus(ctx, reportID, models.StatusReportRejected, &reviewerID, "Reporter banned for false report")
		_ = h.Store.SetUserBanned(ctx, rep.ReporterID, true)

		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            fmt.Sprintf("🚫 Reporter %d BANNED.", rep.ReporterID),
			ShowAlert:       true,
		})

		reporterMsg := "<b>⛔ Account Suspended</b>\n\nYou have been banned from using @telefraudbot due to submission of abusive or false reports."
		h.sendHTML(ctx, rep.ReporterID, reporterMsg)
	}

	if query.Message != nil {
		h.handlePendingReports(ctx, query.Message.GetChat().ID, reviewerID, 0)
	}
}

// handleAdminAdd grants administrator privileges to a user.
func (h *Handlers) handleAdminAdd(ctx context.Context, chatID, senderID int64, targetArg string) {
	if !h.isAdmin(ctx, senderID) {
		h.sendHTML(ctx, chatID, "<b>⛔ Access Denied.</b>")
		return
	}

	targetID, err := strconv.ParseInt(strings.TrimSpace(targetArg), 10, 64)
	if err != nil || targetID <= 0 {
		h.sendHTML(ctx, chatID, "<b>Usage:</b> <code>/admin_add &lt;user_id&gt;</code>")
		return
	}

	_ = h.Store.UpsertUser(ctx, models.User{
		UserID:    targetID,
		FirstName: "Admin User",
		Role:      models.RoleAdmin,
	})
	_ = h.Store.SetUserRole(ctx, targetID, models.RoleAdmin)

	h.sendHTML(ctx, chatID, fmt.Sprintf("<b>✅ Granted Admin role to User ID:</b> <code>%d</code>", targetID))
}

// handleAdminRemove revokes administrator privileges from a user.
func (h *Handlers) handleAdminRemove(ctx context.Context, chatID, senderID int64, targetArg string) {
	if !h.isAdmin(ctx, senderID) {
		h.sendHTML(ctx, chatID, "<b>⛔ Access Denied.</b>")
		return
	}

	targetID, err := strconv.ParseInt(strings.TrimSpace(targetArg), 10, 64)
	if err != nil || targetID <= 0 {
		h.sendHTML(ctx, chatID, "<b>Usage:</b> <code>/admin_remove &lt;user_id&gt;</code>")
		return
	}

	_ = h.Store.SetUserRole(ctx, targetID, models.RoleUser)
	h.sendHTML(ctx, chatID, fmt.Sprintf("<b>✅ Revoked Admin role from User ID:</b> <code>%d</code>", targetID))
}

// handleAdminList lists all active system administrators.
func (h *Handlers) handleAdminList(ctx context.Context, chatID, senderID int64) {
	if !h.isAdmin(ctx, senderID) {
		h.sendHTML(ctx, chatID, "<b>⛔ Access Denied.</b>")
		return
	}

	admins, err := h.Store.ListAdmins(ctx)
	if err != nil || len(admins) == 0 {
		var b strings.Builder
		b.WriteString("<b>👥 System Administrators</b>\n\n")
		for _, aid := range h.Cfg.AdminIDs {
			fmt.Fprintf(&b, "• <code>%d</code> (Config Superadmin)\n", aid)
		}
		h.sendHTML(ctx, chatID, b.String())
		return
	}

	var b strings.Builder
	b.WriteString("<b>👥 Active System Administrators</b>\n\n")
	for _, a := range admins {
		uname := a.Username
		if uname != "" {
			uname = "@" + uname
		} else {
			uname = a.FirstName
		}
		fmt.Fprintf(&b, "• <b>%s</b> (ID: <code>%d</code>) — <i>%s</i>\n", html.EscapeString(uname), a.UserID, a.Role)
	}
	h.sendHTML(ctx, chatID, b.String())
}

// handleBlacklistAddCommand directly inserts a target into the verified fraud database.
func (h *Handlers) handleBlacklistAddCommand(ctx context.Context, chatID, senderID int64, args string) {
	if !h.isAdmin(ctx, senderID) {
		h.sendHTML(ctx, chatID, "<b>⛔ Access Denied.</b>")
		return
	}

	parts := strings.Fields(args)
	if len(parts) < 2 {
		h.sendHTML(ctx, chatID,
			"<b>Usage:</b> <code>/blacklist add &lt;target&gt; [threat: LOW|MEDIUM|HIGH|CRITICAL] &lt;reason&gt;</code>\n\n"+
				"<b>Example:</b> <code>/blacklist add @scammer HIGH Escrow fraud in crypto group</code>",
		)
		return
	}

	target := parts[0]
	var threatLevel = models.ThreatHigh
	var reasonParts []string

	if len(parts) >= 3 {
		upper := strings.ToUpper(parts[1])
		switch upper {
		case models.ThreatLow, models.ThreatMedium, models.ThreatHigh, models.ThreatCritical:
			threatLevel = upper
			reasonParts = parts[2:]
		default:
			reasonParts = parts[1:]
		}
	} else {
		reasonParts = parts[1:]
	}

	reason := strings.Join(reasonParts, " ")
	cand, ok := parseCandidates(target)
	if !ok {
		h.sendHTML(ctx, chatID, "<b>⚠️ Invalid target format.</b> Provide @username, numeric ID, or phone.")
		return
	}

	scam, err := h.Identity.Register(ctx, cand, reason, models.CategoryOther, &senderID)
	if err != nil {
		h.sendHTML(ctx, chatID, fmt.Sprintf("<b>⚠️ Failed to add scammer:</b> %v", err))
		return
	}

	scam.Status = models.StatusVerified
	scam.ThreatLevel = threatLevel
	_ = h.Store.Update(ctx, scam)

	h.sendHTML(ctx, chatID, fmt.Sprintf(
		"<b>✅ Blacklisted Scammer Added!</b>\n\n"+
			"<b>Scammer ID:</b> <code>#%d</code>\n"+
			"<b>Target:</b> <code>%s</code>\n"+
			"<b>Threat Level:</b> <code>%s</code>\n"+
			"<b>Reason:</b> %s",
		scam.ID, html.EscapeString(target), threatLevel, html.EscapeString(reason),
	))
}

// handleBlacklistRemoveCommand clears an entity from the active blacklist.
func (h *Handlers) handleBlacklistRemoveCommand(ctx context.Context, chatID, senderID int64, target string) {
	if !h.isAdmin(ctx, senderID) {
		h.sendHTML(ctx, chatID, "<b>⛔ Access Denied.</b>")
		return
	}

	cand, ok := parseCandidates(target)
	if !ok {
		h.sendHTML(ctx, chatID, "<b>Usage:</b> <code>/blacklist remove &lt;@username | id | phone&gt;</code>")
		return
	}

	scam, err := h.Identity.Resolve(ctx, cand)
	if err != nil || scam == nil {
		h.sendHTML(ctx, chatID, "<b>⚠️ No matching fraudster found.</b>")
		return
	}

	_ = h.Store.ClearScammer(ctx, scam.ID)
	h.sendHTML(ctx, chatID, fmt.Sprintf("<b>✅ Cleared Record:</b> Scammer #%d is no longer blacklisted.", scam.ID))
}

// handleExportCommand generates and sends CSV or JSON blacklist export.
func (h *Handlers) handleExportCommand(ctx context.Context, chatID, senderID int64, format string) {
	if !h.isAdmin(ctx, senderID) {
		h.sendHTML(ctx, chatID, "<b>⛔ Access Denied.</b>")
		return
	}

	format = strings.ToLower(strings.TrimSpace(format))
	if format != "json" {
		format = "csv"
	}

	records, err := h.Store.GetAllVerifiedScammers(ctx)
	if err != nil {
		h.sendHTML(ctx, chatID, "<b>⚠️ Failed to retrieve records.</b>")
		return
	}

	var data []byte
	var filename string

	if format == "json" {
		data, err = exporter.ExportJSON(records)
		filename = fmt.Sprintf("telefraud_blacklist_%s.json", time.Now().Format("20060102_150405"))
	} else {
		data, err = exporter.ExportCSV(records)
		filename = fmt.Sprintf("telefraud_blacklist_%s.csv", time.Now().Format("20060102_150405"))
	}

	if err != nil {
		h.sendHTML(ctx, chatID, "<b>⚠️ Export encoding failed.</b>")
		return
	}

	caption := fmt.Sprintf(
		"<b>📂 Verified Fraud Blacklist Export</b>\n\n"+
			"<b>Format:</b> <code>%s</code>\n"+
			"<b>Total Records:</b> <code>%d</code>\n"+
			"<b>Exported:</b> %s",
		strings.ToUpper(format), len(records), time.Now().UTC().Format("Jan 02, 2006 15:04 UTC"),
	)

	docMsg := tu.Document(
		tu.ID(chatID),
		tu.File(namedBytesReader{Reader: bytes.NewReader(data), name: filename}),
	).WithCaption(caption).WithParseMode(telego.ModeHTML)

	_, _ = h.Bot.SendDocument(docMsg)
}

// handleImportDocument imports records from an uploaded CSV or JSON file.
func (h *Handlers) handleImportDocument(ctx context.Context, msg telego.Message) {
	senderID := msg.From.ID
	chatID := msg.Chat.ID

	if !h.isAdmin(ctx, senderID) {
		h.sendHTML(ctx, chatID, "<b>⛔ Access Denied.</b>")
		return
	}

	doc := msg.Document
	if doc == nil {
		h.sendHTML(ctx, chatID, "<b>⚠️ Please attach a CSV or JSON file to import.</b>")
		return
	}

	file, err := h.Bot.GetFile(&telego.GetFileParams{FileID: doc.FileID})
	if err != nil {
		h.sendHTML(ctx, chatID, "<b>⚠️ Failed to retrieve file from Telegram servers.</b>")
		return
	}

	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", h.Cfg.BotToken, file.FilePath)
	resp, err := http.Get(fileURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		h.sendHTML(ctx, chatID, "<b>⚠️ Failed to download file content.</b>")
		return
	}
	defer resp.Body.Close()

	var items []models.ScammerImportItem
	if strings.HasSuffix(strings.ToLower(doc.FileName), ".json") {
		items, err = exporter.ImportJSON(resp.Body)
	} else {
		items, err = exporter.ImportCSV(resp.Body)
	}

	if err != nil {
		h.sendHTML(ctx, chatID, fmt.Sprintf("<b>⚠️ Import parsing error:</b> %v", err))
		return
	}

	importedCount := 0
	for _, it := range items {
		cand := identity.Candidates{
			UserID:   it.UserID,
			Username: it.Username,
			Phone:    it.Phone,
		}
		if cand.Empty() {
			continue
		}

		scam, rErr := h.Identity.Register(ctx, cand, it.Reason, it.Category, &senderID)
		if rErr == nil && scam != nil {
			scam.Status = models.StatusVerified
			if it.ThreatLevel != "" {
				scam.ThreatLevel = it.ThreatLevel
			}
			_ = h.Store.Update(ctx, scam)
			importedCount++
		}
	}

	h.sendHTML(ctx, chatID, fmt.Sprintf(
		"<b>✅ Blacklist Import Completed!</b>\n\n"+
			"<b>File:</b> <code>%s</code>\n"+
			"<b>Parsed Records:</b> <code>%d</code>\n"+
			"<b>Successfully Imported/Linked:</b> <code>%d</code>",
		html.EscapeString(doc.FileName), len(items), importedCount,
	))
}

// handleBroadcastCommand initiates global broadcast.
func (h *Handlers) handleBroadcastCommand(ctx context.Context, chatID, senderID int64, text string) {
	if !h.isAdmin(ctx, senderID) {
		h.sendHTML(ctx, chatID, "<b>⛔ Access Denied.</b>")
		return
	}

	if strings.TrimSpace(text) == "" {
		h.Sessions.SetState(senderID, StateBroadcastWaitText)
		h.sendHTML(ctx, chatID,
			"<b>📢 Global Broadcast Engine</b>\n\n"+
				"Please send your announcement message:\n"+
				"• Send <b>HTML formatted text</b>, OR\n"+
				"• Upload/send a <b>Photo with caption</b>!\n\n"+
				"<i>All broadcasts will automatically include a <b>📢 Join TeleFraud</b> button pointing to your official channel.</i>\n\n"+
				"<i>Send /cancel to exit.</i>",
		)
		return
	}

	h.promptBroadcastConfirm(ctx, chatID, senderID, text)
}

func (h *Handlers) promptBroadcastConfirm(ctx context.Context, chatID, senderID int64, text string) {
	_ = ctx
	h.Sessions.Update(senderID, func(s *UserSession) {
		s.BroadcastText = text
		s.BroadcastPhotoID = ""
	})

	channelLink := h.Cfg.RequiredChannelLink
	if channelLink == "" {
		channelLink = "https://t.me/telefraud_info"
	}

	preview := fmt.Sprintf(
		"<b>📢 BROADCAST PREVIEW:</b>\n\n"+
			"%s\n\n"+
			"➖➖➖➖➖➖➖➖➖➖\n"+
			"<b>Attached Button:</b> [ 📢 Join TeleFraud ]\n\n"+
			"<i>Tap <b>🚀 Send Now</b> to broadcast this announcement.</i>",
		text,
	)

	markup := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("📢 Join TeleFraud").WithURL(channelLink),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🚀 Send Now").WithCallbackData("bcast_act:send"),
			tu.InlineKeyboardButton("❌ Cancel").WithCallbackData("bcast_act:cancel"),
		),
	)

	msg := tu.Message(tu.ID(chatID), preview).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(markup)
	_, _ = h.Bot.SendMessage(msg)
}

func (h *Handlers) promptBroadcastPhotoConfirm(ctx context.Context, chatID, senderID int64, caption, photoID string) {
	_ = ctx
	h.Sessions.Update(senderID, func(s *UserSession) {
		s.BroadcastText = caption
		s.BroadcastPhotoID = photoID
	})

	channelLink := h.Cfg.RequiredChannelLink
	if channelLink == "" {
		channelLink = "https://t.me/telefraud_info"
	}

	captionText := caption
	if captionText == "" {
		captionText = "<i>(Photo announcement without text caption)</i>"
	}

	previewCaption := fmt.Sprintf(
		"<b>📢 PHOTO BROADCAST PREVIEW:</b>\n\n"+
			"%s\n\n"+
			"➖➖➖➖➖➖➖➖➖➖\n"+
			"<b>Attached Button:</b> [ 📢 Join TeleFraud ]\n\n"+
			"<i>Tap <b>🚀 Send Now</b> to broadcast this photo announcement.</i>",
		captionText,
	)

	markup := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("📢 Join TeleFraud").WithURL(channelLink),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🚀 Send Now").WithCallbackData("bcast_act:send"),
			tu.InlineKeyboardButton("❌ Cancel").WithCallbackData("bcast_act:cancel"),
		),
	)

	msg := tu.Photo(tu.ID(chatID), tu.FileFromID(photoID)).
		WithCaption(previewCaption).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(markup)
	_, _ = h.Bot.SendPhoto(msg)
}

func (h *Handlers) executeBroadcast(ctx context.Context, chatID, senderID int64) {
	sess := h.Sessions.Get(senderID)
	text := sess.BroadcastText
	photoID := sess.BroadcastPhotoID
	if text == "" && photoID == "" {
		h.sendHTML(ctx, chatID, "<b>⚠️ No broadcast message draft found.</b>")
		return
	}
	h.Sessions.Clear(senderID)

	channelLink := h.Cfg.RequiredChannelLink
	if channelLink == "" {
		channelLink = "https://t.me/telefraud_info"
	}

	statusMsg, _ := h.Bot.SendMessage(tu.Message(tu.ID(chatID), "<b>🚀 Initiating Global Broadcast...</b>").WithParseMode(telego.ModeHTML))

	broadcaster := broadcast.NewBroadcaster(h.Bot, h.Store)

	go func() {
		bgCtx := context.Background()
		var lastEdit time.Time

		res, err := broadcaster.Broadcast(bgCtx, senderID, text, photoID, channelLink, func(curr, total, succ, fail int) {
			if statusMsg != nil && time.Since(lastEdit) > 2*time.Second {
				lastEdit = time.Now()
				pct := (curr * 100) / total
				progressText := fmt.Sprintf(
					"<b>📢 Broadcasting in progress...</b>\n\n"+
						"<b>Progress:</b> <code>%d%% (%d/%d)</code>\n"+
						"<b>Delivered:</b> <code>%d</code> | <b>Failed:</b> <code>%d</code>",
					pct, curr, total, succ, fail,
				)
				_, _ = h.Bot.EditMessageText(&telego.EditMessageTextParams{
					ChatID:    tu.ID(chatID),
					MessageID: statusMsg.MessageID,
					Text:      progressText,
					ParseMode: telego.ModeHTML,
				})
			}
		})

		if err != nil {
			log.Printf("broadcast failed: %v", err)
			return
		}

		finalText := fmt.Sprintf(
			"<b>✅ Broadcast Completed!</b>\n\n"+
				"<b>Total Targets:</b> <code>%d</code>\n"+
				"<b>Delivered:</b> <code>%d</code>\n"+
				"<b>Failed / Blocked:</b> <code>%d</code>\n"+
				"<b>Elapsed Time:</b> <code>%v</code>",
			res.TotalRecipients, res.SuccessCount, res.FailedCount, res.Duration.Round(time.Second),
		)
		if statusMsg != nil {
			_, _ = h.Bot.EditMessageText(&telego.EditMessageTextParams{
				ChatID:    tu.ID(chatID),
				MessageID: statusMsg.MessageID,
				Text:      finalText,
				ParseMode: telego.ModeHTML,
			})
		} else {
			h.sendHTML(bgCtx, chatID, finalText)
		}
	}()
}

// handleModeCommand configures the global group moderation strictness level.
func (h *Handlers) handleModeCommand(ctx context.Context, chatID, senderID int64, mode string) {
	if !h.isAdmin(ctx, senderID) {
		h.sendHTML(ctx, chatID, "<b>⛔ Access Denied.</b>")
		return
	}

	mode = strings.ToUpper(strings.TrimSpace(mode))
	switch mode {
	case "BAN", "KICK", "WARN", "SILENT":
		_ = h.Store.SetSetting(ctx, "ENFORCEMENT_MODE", mode)
		h.sendHTML(ctx, chatID, fmt.Sprintf("<b>✅ Global Enforcement Mode updated to:</b> <code>%s</code>", mode))
	default:
		current, _ := h.Store.GetSetting(ctx, "ENFORCEMENT_MODE", "BAN")
		text := fmt.Sprintf(
			"<b>⚙️ Global Enforcement Mode</b>\n\n"+
				"<b>Current Mode:</b> <code>%s</code>\n\n"+
				"• <b>BAN (Default)</b>: Immediately auto-bans scammers on join/message.\n"+
				"• <b>KICK</b>: Kicks scammers (allows rejoining if cleared).\n"+
				"• <b>WARN</b>: Deletes scam message and sends warning without banning.\n"+
				"• <b>SILENT</b>: Silently logs to DB without public group notice.\n\n"+
				"<i>Select a mode below:</i>",
			current,
		)
		markup := tu.InlineKeyboard(
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton("🚫 BAN").WithCallbackData("adm_setmode:BAN"),
				tu.InlineKeyboardButton("👢 KICK").WithCallbackData("adm_setmode:KICK"),
			),
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton("⚠️ WARN").WithCallbackData("adm_setmode:WARN"),
				tu.InlineKeyboardButton("🤫 SILENT").WithCallbackData("adm_setmode:SILENT"),
			),
		)
		msg := tu.Message(tu.ID(chatID), text).
			WithParseMode(telego.ModeHTML).
			WithReplyMarkup(markup)
		_, _ = h.Bot.SendMessage(msg)
	}
}

// handleBackupCommand executes immediate database backup and delivers file.
func (h *Handlers) handleBackupCommand(ctx context.Context, chatID, senderID int64) {
	if !h.isAdmin(ctx, senderID) {
		h.sendHTML(ctx, chatID, "<b>⛔ Access Denied.</b>")
		return
	}

	h.sendHTML(ctx, chatID, "<b>💾 Generating PostgreSQL snapshot...</b>")
	res, err := h.Backup.RunBackup(ctx)
	if err != nil {
		h.sendHTML(ctx, chatID, fmt.Sprintf("<b>⚠️ Backup failed:</b> %v", err))
		return
	}

	_ = h.Backup.SendBackupFile(ctx, chatID, res)
}

// handleAdminScanGroup remotely scans a target group.
func (h *Handlers) handleAdminScanGroup(ctx context.Context, chatID, senderID int64, groupIDStr string) {
	if !h.isAdmin(ctx, senderID) {
		h.sendHTML(ctx, chatID, "<b>⛔ Access Denied.</b>")
		return
	}

	groupID, err := strconv.ParseInt(strings.TrimSpace(groupIDStr), 10, 64)
	if err != nil {
		h.sendHTML(ctx, chatID, "<b>Usage:</b> <code>/admin_scangroup &lt;group_id&gt;</code>")
		return
	}

	h.sendHTML(ctx, chatID, fmt.Sprintf("<b>🔍 Running security audit on group ID:</b> <code>%d</code>...", groupID))
	res, err := h.Scanner.ScanGroup(ctx, groupID, senderID)
	if err != nil {
		h.sendHTML(ctx, chatID, fmt.Sprintf("<b>⚠️ Scan error:</b> %v", err))
		return
	}

	text := fmt.Sprintf(
		"<b>🛡️ Audit Report for Group: %s</b>\n\n"+
			"<b>Group ID:</b> <code>%d</code>\n"+
			"<b>Audited Members:</b> <code>%d</code>\n"+
			"<b>Flagged Accounts:</b> <code>%d</code>\n"+
			"<b>Duration:</b> <code>%v</code>",
		html.EscapeString(res.GroupTitle), res.GroupID, res.TotalAudited, len(res.FlaggedMembers), res.Duration.Round(time.Millisecond),
	)
	h.sendHTML(ctx, chatID, text)
}

// handleViewProofsCallback sends all attached proofs for a report to the admin chat.
func (h *Handlers) handleViewProofsCallback(ctx context.Context, query telego.CallbackQuery, reportIDStr string) {
	reviewerID := query.From.ID
	if !h.isAdmin(ctx, reviewerID) {
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "⛔ Admin privilege required.",
			ShowAlert:       true,
		})
		return
	}

	reportID, err := strconv.ParseInt(reportIDStr, 10, 64)
	if err != nil {
		return
	}

	chatID := query.From.ID
	if query.Message != nil {
		chatID = query.Message.GetChat().ID
	}

	rep, proofs, err := h.Store.GetReport(ctx, reportID)
	if err != nil || rep == nil {
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "⚠️ Report not found.",
			ShowAlert:       true,
		})
		return
	}

	if len(proofs) == 0 {
		_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "No proof files attached to this report.",
			ShowAlert:       true,
		})
		return
	}

	_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		Text:            fmt.Sprintf("Sending %d proof file(s)...", len(proofs)),
	})

	for idx, p := range proofs {
		caption := fmt.Sprintf("📎 <b>Proof #%d/%d</b> for Report #%d\n<b>Category:</b> %s",
			idx+1, len(proofs), reportID, html.EscapeString(rep.Category),
		)
		if rep.TargetUsername != "" {
			caption = fmt.Sprintf("📎 <b>Proof #%d/%d</b> for Report #%d\n<b>Target:</b> @%s\n<b>Category:</b> %s",
				idx+1, len(proofs), reportID, html.EscapeString(rep.TargetUsername), html.EscapeString(rep.Category),
			)
		} else if rep.TargetUserID != nil && *rep.TargetUserID > 0 {
			caption = fmt.Sprintf("📎 <b>Proof #%d/%d</b> for Report #%d\n<b>Target ID:</b> <code>%d</code>\n<b>Category:</b> %s",
				idx+1, len(proofs), reportID, *rep.TargetUserID, html.EscapeString(rep.Category),
			)
		}

		switch p.FileType {
		case "PHOTO":
			_, err = h.Bot.SendPhoto(tu.Photo(tu.ID(chatID), tu.FileFromID(p.FileID)).WithCaption(caption).WithParseMode(telego.ModeHTML))
		case "DOCUMENT":
			_, err = h.Bot.SendDocument(tu.Document(tu.ID(chatID), tu.FileFromID(p.FileID)).WithCaption(caption).WithParseMode(telego.ModeHTML))
		case "VIDEO":
			_, err = h.Bot.SendVideo(tu.Video(tu.ID(chatID), tu.FileFromID(p.FileID)).WithCaption(caption).WithParseMode(telego.ModeHTML))
		default:
			_, err = h.Bot.SendDocument(tu.Document(tu.ID(chatID), tu.FileFromID(p.FileID)).WithCaption(caption).WithParseMode(telego.ModeHTML))
		}
		if err != nil {
			log.Printf("send proof file %s to admin %d: %v", p.FileID, chatID, err)
		}
	}
}
