// Package bot wires the telego client to the handler layer, background workers, and drives the update loop.
package bot

import (
	"context"
	"fmt"
	"log"

	"github.com/mymmrac/telego"

	"telefraud/config"
	"telefraud/internal/database"
	"telefraud/internal/handlers"
	"telefraud/internal/services/backup"
	"telefraud/internal/services/identity"
	"telefraud/internal/services/scanner"
)

// Bot connects to Telegram and dispatches updates to the handler stack and background workers.
type Bot struct {
	cfg     *config.Config
	tg      *telego.Bot
	h       *handlers.Handlers
	scanner *scanner.GroupScanner
	backup  *backup.BackupManager
}

// New connects to Telegram and builds the handler stack with services.
func New(cfg *config.Config, resolver *identity.Resolver, store *database.IdentityStore) (*Bot, error) {
	tg, err := telego.NewBot(cfg.BotToken, telego.WithDefaultLogger(cfg.Debug, true))
	if err != nil {
		return nil, fmt.Errorf("create bot: %w", err)
	}

	groupScanner := scanner.NewGroupScanner(tg, store, resolver)
	backupManager := backup.NewBackupManager(cfg, tg)

	h := handlers.New(tg, cfg, resolver, store, groupScanner, backupManager)

	return &Bot{
		cfg:     cfg,
		tg:      tg,
		h:       h,
		scanner: groupScanner,
		backup:  backupManager,
	}, nil
}

// Run starts background workers and long polling with full update type reception until ctx is cancelled.
func (b *Bot) Run(ctx context.Context) error {
	// Start background workers
	b.scanner.StartDailyScannerWorker(ctx)
	b.backup.StartDailyBackupWorker(ctx)

	updates, err := b.tg.UpdatesViaLongPolling(&telego.GetUpdatesParams{
		AllowedUpdates: []string{
			"message",
			"edited_message",
			"callback_query",
			"chat_member",
			"my_chat_member",
		},
	})
	if err != nil {
		return fmt.Errorf("open updates: %w", err)
	}
	defer b.tg.StopLongPolling()

	log.Println("Listening for Telegram updates (messages, callbacks, chat_member events)...")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			if update.Message != nil {
				b.h.HandleMessage(ctx, *update.Message)
			}
			if update.CallbackQuery != nil {
				b.h.HandleCallbackQuery(ctx, *update.CallbackQuery)
			}
			if update.ChatMember != nil {
				b.h.HandleChatMember(ctx, *update.ChatMember)
			}
			if update.MyChatMember != nil {
				b.h.HandleMyChatMember(ctx, *update.MyChatMember)
			}
		}
	}
}
