package broadcast

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"telefraud/internal/database"
	"telefraud/internal/models"
)

// BroadcastResult contains metrics from a broadcast run.
type BroadcastResult struct {
	TotalRecipients int
	SuccessCount    int
	FailedCount     int
	Duration        time.Duration
}

// ProgressCallback is invoked periodically with progress updates.
type ProgressCallback func(current, total, success, failed int)

// Broadcaster manages mass message dispatch across users and groups.
type Broadcaster struct {
	bot   *telego.Bot
	store *database.IdentityStore
}

// NewBroadcaster creates a Broadcaster.
func NewBroadcaster(bot *telego.Bot, store *database.IdentityStore) *Broadcaster {
	return &Broadcaster{
		bot:   bot,
		store: store,
	}
}

// Broadcast sends HTML-formatted message or photo to all registered users and groups with rate limiting and channel join button.
func (b *Broadcaster) Broadcast(ctx context.Context, senderID int64, text, photoID, channelLink string, cb ProgressCallback) (*BroadcastResult, error) {
	start := time.Now()

	userIDs, err := b.store.GetAllUserIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch users: %w", err)
	}

	groupIDs, err := b.store.GetAllGroupIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch groups: %w", err)
	}

	// Merge targets
	targets := make([]int64, 0, len(userIDs)+len(groupIDs))
	targets = append(targets, userIDs...)
	targets = append(targets, groupIDs...)

	total := len(targets)
	success := 0
	failed := 0

	// Build inline keyboard with "📢 Join TeleFraud" button
	var inlineMarkup *telego.InlineKeyboardMarkup
	if channelLink != "" {
		inlineMarkup = tu.InlineKeyboard(
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton("📢 Join TeleFraud").WithURL(channelLink),
			),
		)
	}

	// Rate limiting: 25 msgs/sec (~40ms delay)
	ticker := time.NewTicker(40 * time.Millisecond)
	defer ticker.Stop()

	for idx, chatID := range targets {
		select {
		case <-ctx.Done():
			break
		case <-ticker.C:
		}

		var sendErr error
		if photoID != "" {
			photoMsg := tu.Photo(tu.ID(chatID), tu.FileFromID(photoID)).
				WithCaption(text).
				WithParseMode(telego.ModeHTML)
			if inlineMarkup != nil {
				photoMsg = photoMsg.WithReplyMarkup(inlineMarkup)
			}
			_, sendErr = b.bot.SendPhoto(photoMsg)
		} else {
			textMsg := tu.Message(tu.ID(chatID), text).
				WithParseMode(telego.ModeHTML)
			if inlineMarkup != nil {
				textMsg = textMsg.WithReplyMarkup(inlineMarkup)
			}
			_, sendErr = b.bot.SendMessage(textMsg)
		}

		if sendErr != nil {
			failed++
		} else {
			success++
		}

		// Progress report every 25 messages or at completion
		if cb != nil && ((idx+1)%25 == 0 || idx+1 == total) {
			cb(idx+1, total, success, failed)
		}
	}

	// Record in broadcast_logs
	logText := text
	if photoID != "" {
		logText = "[Photo Attachment] " + text
	}
	_, logErr := b.store.LogBroadcast(ctx, models.BroadcastLog{
		InitiatedBy:    senderID,
		MessageText:    logText,
		RecipientCount: total,
		SuccessCount:   success,
		FailedCount:    failed,
	})
	if logErr != nil {
		log.Printf("broadcast: log failed: %v", logErr)
	}

	return &BroadcastResult{
		TotalRecipients: total,
		SuccessCount:    success,
		FailedCount:     failed,
		Duration:        time.Since(start),
	}, nil
}
