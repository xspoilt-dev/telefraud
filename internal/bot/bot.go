// Package bot wires the telego client to the handler layer and drives the
// update loop.
package bot

import (
	"context"
	"fmt"
	"log"

	"github.com/mymmrac/telego"

	"telefraud/config"
	"telefraud/internal/handlers"
	"telefraud/internal/services/identity"
)

// Bot connects to Telegram and dispatches updates to the handler stack.
type Bot struct {
	tg *telego.Bot
	h  *handlers.Handlers
}

// New connects to Telegram and builds the handler stack.
func New(cfg *config.Config, resolver *identity.Resolver) (*Bot, error) {
	tg, err := telego.NewBot(cfg.BotToken, telego.WithDefaultLogger(cfg.Debug, true))
	if err != nil {
		return nil, fmt.Errorf("create bot: %w", err)
	}
	return &Bot{
		tg: tg,
		h:  handlers.New(tg, cfg, resolver),
	}, nil
}

// Run starts long polling and blocks until ctx is cancelled.
func (b *Bot) Run(ctx context.Context) error {
	updates, err := b.tg.UpdatesViaLongPolling(nil)
	if err != nil {
		return fmt.Errorf("open updates: %w", err)
	}
	defer b.tg.StopLongPolling()

	log.Println("Listening for Telegram updates...")
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
		}
	}
}
