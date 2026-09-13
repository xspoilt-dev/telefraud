package handlers

import (
	"context"
	"fmt"
	"html"
	"log"
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"telefraud/internal/i18n"
	"telefraud/internal/models"
)

// Category options for report wizard
var reportCategories = []struct {
	Label string
	Code  string
}{
	{"💸 Financial Fraud", models.CategoryFinancial},
	{"👤 Impersonation", models.CategoryImpersonation},
	{"🔐 Crypto / Phishing", models.CategoryCrypto},
	{"📦 Fake Store / Goods", models.CategoryFakeStore},
	{"⚠️ Other Abuse", models.CategoryOther},
}

// handleReportCommand handles direct quick submission via /report <target> <reason>.
func (h *Handlers) handleReportCommand(ctx context.Context, reporterID, chatID int64, args, userLang string) {
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		h.startReportWizard(ctx, reporterID, chatID, userLang)
		return
	}

	target := strings.TrimSpace(parts[0])
	reason := strings.TrimSpace(parts[1])

	cand, _ := parseCandidates(target)
	var targetUser *int64
	var targetUsername, targetPhone string
	if cand.UserID > 0 {
		targetUser = &cand.UserID
	}
	if cand.Username != "" {
		targetUsername = cand.Username
	}
	if cand.Phone != "" {
		targetPhone = cand.Phone
	}

	rep := &models.Report{
		ReporterID:     reporterID,
		TargetUserID:   targetUser,
		TargetUsername: targetUsername,
		TargetPhone:    targetPhone,
		Category:       models.CategoryFinancial,
		Description:    reason,
		Status:         models.StatusReportPending,
	}

	reportID, err := h.Store.CreateReportWithProofs(ctx, rep, nil)
	if err != nil {
		h.sendHTML(ctx, chatID, "<b>⚠️ Failed to submit report. Please try again later.</b>")
		return
	}

	text := fmt.Sprintf(
		"<b>✅ Report Submitted Successfully!</b>\n\n"+
			"<b>Report ID:</b> <code>#%d</code>\n"+
			"<b>Target:</b> <code>%s</code>\n"+
			"<b>Reason:</b> %s\n\n"+
			"<i>Our moderation team will review your submission shortly.</i>",
		reportID, html.EscapeString(target), html.EscapeString(reason),
	)
	h.sendHTML(ctx, chatID, text)

	h.notifyAdminsNewReport(ctx, reportID, reporterID, target, models.CategoryFinancial, reason, 0, nil)
}

// startReportWizard initiates the multi-step report flow.
func (h *Handlers) startReportWizard(ctx context.Context, userID, chatID int64, userLang string) {
	_ = ctx
	h.Sessions.Update(userID, func(s *UserSession) {
		*s = UserSession{
			State: StateReportWaitTarget,
		}
	})

	text := i18n.Get(userLang, "wiz_step1")

	inlineMarkup := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(i18n.Get(userLang, "btn_cancel")).WithCallbackData("wiz_cancel"),
		),
	)

	msg := tu.Message(tu.ID(chatID), text).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(inlineMarkup)
	_, _ = h.Bot.SendMessage(msg)
}

// startReportWizardWithTarget starts the wizard with the target already pre-populated from group link.
func (h *Handlers) startReportWizardWithTarget(ctx context.Context, userID, chatID int64, targetInput, userLang string) {
	cand, ok := parseCandidates(targetInput)
	if !ok {
		h.startReportWizard(ctx, userID, chatID, userLang)
		return
	}

	var targetUser *int64
	var targetUsername, targetPhone string
	if cand.UserID > 0 {
		targetUser = &cand.UserID
	}
	if cand.Username != "" {
		targetUsername = cand.Username
	}
	if cand.Phone != "" {
		targetPhone = cand.Phone
	}

	h.Sessions.Update(userID, func(s *UserSession) {
		*s = UserSession{
			State:            StateReportWaitCategory,
			ReportTargetRaw:  targetInput,
			ReportCandidates: cand,
			ReportTargetUser: targetUser,
			ReportUsername:   targetUsername,
			ReportPhone:      targetPhone,
		}
	})

	text := i18n.Format(userLang, "wiz_step2", html.EscapeString(targetInput))

	var rows [][]telego.InlineKeyboardButton
	for _, cat := range reportCategories {
		rows = append(rows, tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(cat.Label).WithCallbackData("wiz_cat:"+cat.Code),
		))
	}
	rows = append(rows, tu.InlineKeyboardRow(
		tu.InlineKeyboardButton(i18n.Get(userLang, "btn_cancel")).WithCallbackData("wiz_cancel"),
	))

	msg := tu.Message(tu.ID(chatID), text).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(tu.InlineKeyboard(rows...))
	_, _ = h.Bot.SendMessage(msg)
}

// handleReportTargetInput processes the target provided in Step 1.
func (h *Handlers) handleReportTargetInput(ctx context.Context, userID, chatID int64, input, userLang string) {
	cand, ok := parseCandidates(input)
	if !ok {
		h.sendHTML(ctx, chatID, i18n.Get(userLang, "check_invalid"))
		return
	}

	var targetUser *int64
	var targetUsername, targetPhone string
	if cand.UserID > 0 {
		targetUser = &cand.UserID
	}
	if cand.Username != "" {
		targetUsername = cand.Username
	}
	if cand.Phone != "" {
		targetPhone = cand.Phone
	}

	h.Sessions.Update(userID, func(s *UserSession) {
		s.State = StateReportWaitCategory
		s.ReportTargetRaw = input
		s.ReportCandidates = cand
		s.ReportTargetUser = targetUser
		s.ReportUsername = targetUsername
		s.ReportPhone = targetPhone
	})

	text := i18n.Format(userLang, "wiz_step2", html.EscapeString(input))

	var rows [][]telego.InlineKeyboardButton
	for _, cat := range reportCategories {
		rows = append(rows, tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(cat.Label).WithCallbackData("wiz_cat:"+cat.Code),
		))
	}
	rows = append(rows, tu.InlineKeyboardRow(
		tu.InlineKeyboardButton(i18n.Get(userLang, "btn_cancel")).WithCallbackData("wiz_cancel"),
	))

	msg := tu.Message(tu.ID(chatID), text).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(tu.InlineKeyboard(rows...))
	_, _ = h.Bot.SendMessage(msg)
}

// handleReportCategoryCallback handles category selection from inline keyboard.
func (h *Handlers) handleReportCategoryCallback(ctx context.Context, query telego.CallbackQuery, category, userLang string) {
	userID := query.From.ID
	chatID := query.Message.GetChat().ID

	h.Sessions.Update(userID, func(s *UserSession) {
		s.State = StateReportWaitDescription
		s.ReportCategory = category
	})

	_ = h.Bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		Text:            "Category selected: " + category,
	})

	text := i18n.Format(userLang, "wiz_step3", html.EscapeString(category))

	inlineMarkup := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(i18n.Get(userLang, "btn_cancel")).WithCallbackData("wiz_cancel"),
		),
	)

	msg := tu.Message(tu.ID(chatID), text).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(inlineMarkup)
	_, _ = h.Bot.SendMessage(msg)
}

// handleReportDescriptionInput processes the description input in Step 3.
func (h *Handlers) handleReportDescriptionInput(ctx context.Context, userID, chatID int64, description, userLang string) {
	if len(strings.TrimSpace(description)) < 5 {
		h.sendHTML(ctx, chatID, i18n.Get(userLang, "wiz_short_desc"))
		return
	}

	h.Sessions.Update(userID, func(s *UserSession) {
		s.State = StateReportWaitProof
		s.ReportDescription = strings.TrimSpace(description)
	})

	text := i18n.Get(userLang, "wiz_step4")

	inlineMarkup := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(i18n.Get(userLang, "btn_cancel")).WithCallbackData("wiz_cancel"),
		),
	)

	msg := tu.Message(tu.ID(chatID), text).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(inlineMarkup)
	_, _ = h.Bot.SendMessage(msg)
}

// handleReportProofMedia collects uploaded media proofs during Step 4.
func (h *Handlers) handleReportProofMedia(ctx context.Context, msg telego.Message, userLang string) {
	userID := msg.From.ID
	chatID := msg.Chat.ID

	var proof models.ReportProof

	switch {
	case len(msg.Photo) > 0:
		highest := msg.Photo[len(msg.Photo)-1]
		proof = models.ReportProof{
			FileID:       highest.FileID,
			FileUniqueID: highest.FileUniqueID,
			FileType:     "PHOTO",
			Caption:      msg.Caption,
		}
	case msg.Document != nil:
		proof = models.ReportProof{
			FileID:       msg.Document.FileID,
			FileUniqueID: msg.Document.FileUniqueID,
			FileType:     "DOCUMENT",
			Caption:      msg.Caption,
		}
	case msg.Video != nil:
		proof = models.ReportProof{
			FileID:       msg.Video.FileID,
			FileUniqueID: msg.Video.FileUniqueID,
			FileType:     "VIDEO",
			Caption:      msg.Caption,
		}
	default:
		return
	}

	proofCount := 0
	h.Sessions.Update(userID, func(s *UserSession) {
		s.ReportProofs = append(s.ReportProofs, proof)
		proofCount = len(s.ReportProofs)
	})

	inlineMarkup := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(fmt.Sprintf("%s (%d)", i18n.Get(userLang, "btn_submit"), proofCount)).WithCallbackData("wiz_submit"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(i18n.Get(userLang, "btn_cancel")).WithCallbackData("wiz_cancel"),
		),
	)

	ack := fmt.Sprintf("✅ <b>Proof #%d received!</b> Send more files or tap Submit below.", proofCount)
	msgOut := tu.Message(tu.ID(chatID), ack).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(inlineMarkup)
	_, _ = h.Bot.SendMessage(msgOut)
}

// finishReportWizard submits the report to the database and alerts moderators.
func (h *Handlers) finishReportWizard(ctx context.Context, userID, chatID int64, userLang string) {
	sess := h.Sessions.Get(userID)
	if sess.State != StateReportWaitProof && sess.State != StateReportWaitDescription {
		h.sendHTML(ctx, chatID, "<b>⚠️ No active report submission found.</b>")
		return
	}

	if len(sess.ReportProofs) == 0 {
		h.sendHTML(ctx, chatID, i18n.Get(userLang, "wiz_proof_required"))
		return
	}

	rep := &models.Report{
		ReporterID:     userID,
		TargetUserID:   sess.ReportTargetUser,
		TargetUsername: sess.ReportUsername,
		TargetPhone:    sess.ReportPhone,
		Category:       sess.ReportCategory,
		Description:    sess.ReportDescription,
		Status:         models.StatusReportPending,
	}

	reportID, err := h.Store.CreateReportWithProofs(ctx, rep, sess.ReportProofs)
	if err != nil {
		log.Printf("create report wizard error: %v", err)
		h.sendHTML(ctx, chatID, "<b>⚠️ Failed to submit report. Please try again later.</b>")
		return
	}

	proofCount := len(sess.ReportProofs)
	targetDisplay := sess.ReportTargetRaw
	category := sess.ReportCategory
	description := sess.ReportDescription

	// Clear session
	h.Sessions.Clear(userID)

	confirmText := i18n.Format(userLang, "wiz_submitted", reportID, html.EscapeString(targetDisplay), html.EscapeString(category), proofCount)
	h.sendHTML(ctx, chatID, confirmText)

	// Forward notification to Admin Log Chat / Admins with inline review queue buttons
	h.notifyAdminsNewReport(ctx, reportID, userID, targetDisplay, category, description, proofCount, sess.ReportProofs)
}

// notifyAdminsNewReport sends an alert with review buttons to the admin log channel.
func (h *Handlers) notifyAdminsNewReport(ctx context.Context, reportID, reporterID int64, target, category, description string, proofCount int, proofs []models.ReportProof) {
	adminChatID := h.Cfg.AdminLogChatID
	if adminChatID == 0 {
		if len(h.Cfg.AdminIDs) > 0 {
			adminChatID = h.Cfg.AdminIDs[0]
		} else {
			return
		}
	}

	text := fmt.Sprintf(
		"<b>📥 NEW FRAUD REPORT PENDING REVIEW</b>\n\n"+
			"<b>Report ID:</b> <code>#%d</code>\n"+
			"<b>Reporter ID:</b> <code>%d</code>\n"+
			"<b>Target:</b> <code>%s</code>\n"+
			"<b>Category:</b> %s\n"+
			"<b>Proofs Attached:</b> %d file(s)\n\n"+
			"<b>Description:</b>\n%s",
		reportID, reporterID, html.EscapeString(target),
		html.EscapeString(category), proofCount, html.EscapeString(description),
	)

	var reviewRows [][]telego.InlineKeyboardButton
	if proofCount > 0 {
		reviewRows = append(reviewRows, []telego.InlineKeyboardButton{
			tu.InlineKeyboardButton(fmt.Sprintf("🖼️ View Attached Proofs (%d)", proofCount)).WithCallbackData(fmt.Sprintf("adm_view_proofs:%d", reportID)),
		})
	}
	reviewRows = append(reviewRows,
		[]telego.InlineKeyboardButton{
			tu.InlineKeyboardButton("✅ Approve").WithCallbackData(fmt.Sprintf("rep_act:approve:%d", reportID)),
			tu.InlineKeyboardButton("❌ Reject").WithCallbackData(fmt.Sprintf("rep_act:reject:%d", reportID)),
		},
		[]telego.InlineKeyboardButton{
			tu.InlineKeyboardButton("❓ Request Proof").WithCallbackData(fmt.Sprintf("rep_act:req_proof:%d", reportID)),
			tu.InlineKeyboardButton("🚫 Ban Reporter").WithCallbackData(fmt.Sprintf("rep_act:ban_rep:%d", reportID)),
		},
	)

	msg := tu.Message(tu.ID(adminChatID), text).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(tu.InlineKeyboard(reviewRows...))
	_, _ = h.Bot.SendMessage(msg)

	// Also forward proof files to admin log chat
	for idx, p := range proofs {
		caption := fmt.Sprintf("📎 Proof #%d for Report #%d", idx+1, reportID)
		switch p.FileType {
		case "PHOTO":
			_, _ = h.Bot.SendPhoto(tu.Photo(tu.ID(adminChatID), tu.FileFromID(p.FileID)).WithCaption(caption))
		case "DOCUMENT":
			_, _ = h.Bot.SendDocument(tu.Document(tu.ID(adminChatID), tu.FileFromID(p.FileID)).WithCaption(caption))
		case "VIDEO":
			_, _ = h.Bot.SendVideo(tu.Video(tu.ID(adminChatID), tu.FileFromID(p.FileID)).WithCaption(caption))
		}
	}
}

// cancelWizard cancels the current wizard.
func (h *Handlers) cancelWizard(ctx context.Context, userID, chatID int64, userLang string) {
	h.Sessions.Clear(userID)
	h.sendHTML(ctx, chatID, i18n.Get(userLang, "wiz_cancelled"))
}
