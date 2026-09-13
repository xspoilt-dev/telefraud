package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"telefraud/config"
	"telefraud/internal/bot"
	"telefraud/internal/database"
	"telefraud/internal/services/identity"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if cfg.BotToken == "YOUR_TELEGRAM_BOT_TOKEN" {
		log.Fatal("BOT_TOKEN is unset; configure the environment (see .env.example)")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	store := database.NewIdentityStore(pool)
	resolver := identity.NewResolver(store)

	b, err := bot.New(cfg, resolver, store)
	if err != nil {
		log.Fatalf("bot: %v", err)
	}

	log.Printf("Starting @telefraudbot | Developer: %s (%s)", cfg.DeveloperName, cfg.DeveloperUsername)
	if err := b.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("run: %v", err)
	}
	log.Println("Shutdown complete")
}
