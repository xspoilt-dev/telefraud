// Package handlers groups the bot's update handling with its dependencies.
package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"telefraud/config"
	"telefraud/internal/services/identity"
)

// Handlers holds the bot client, configuration and identity resolver shared by
// every command and event handler.
type Handlers struct {
	Bot       *telego.Bot
	Cfg       *config.Config
	Identity  *identity.Resolver
	Menu      *telego.ReplyKeyboardMarkup
	StartTime time.Time
}

// New builds the handler stack.
func New(bot *telego.Bot, cfg *config.Config, resolver *identity.Resolver) *Handlers {
	return &Handlers{
		Bot:       bot,
		Cfg:       cfg,
		Identity:  resolver,
		Menu:      mainMenu(),
		StartTime: time.Now(),
	}
}

func mainMenu() *telego.ReplyKeyboardMarkup {
	return tu.Keyboard(
		tu.KeyboardRow(
			tu.KeyboardButton("🛡️ Report Fraudster"),
			tu.KeyboardButton("🔍 Check Identifier"),
		),
		tu.KeyboardRow(
			tu.KeyboardButton("📋 My Submissions"),
			tu.KeyboardButton("📊 Global Statistics"),
		),
		tu.KeyboardRow(
			tu.KeyboardButton("❓ Help & FAQ"),
			tu.KeyboardButton("👨‍💻 Developer Info"),
		),
	).WithResizeKeyboard()
}

// HandleMessage dispatches an incoming message.
func (h *Handlers) HandleMessage(ctx context.Context, msg telego.Message) {
	chat := msg.Chat

	// Join gate: newly added members are checked against the blacklist first.
	if len(msg.NewChatMembers) > 0 {
		h.handleJoinGate(ctx, chat, msg.NewChatMembers)
		return
	}

	if chat.Type != telego.ChatTypePrivate {
		return
	}

	text := strings.TrimSpace(msg.Text)
	switch {
	case text == "/start":
		h.handleStart(ctx, chat.ID)
	case text == "/dev" || text == "👨‍💻 Developer Info":
		h.handleDev(ctx, chat.ID)
	case text == "/check" || text == "🔍 Check Identifier":
		h.promptCheck(ctx, chat.ID)
	case text == "/help" || text == "❓ Help & FAQ":
		h.handleHelp(ctx, chat.ID)
	case strings.HasPrefix(text, "/check "):
		h.handleCheck(ctx, chat.ID, strings.TrimSpace(strings.TrimPrefix(text, "/check ")))
	}
}

func (h *Handlers) sendHTML(ctx context.Context, chatID int64, text string) {
	_, _ = h.Bot.SendMessage(tu.Message(tu.ID(chatID), text).WithParseMode(telego.ModeHTML))
}
