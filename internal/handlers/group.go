package handlers

import (
	"context"
	"fmt"
	"html"
	"log"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"telefraud/internal/models"
	"telefraud/internal/services/identity"
)

// handleJoinGate checks newly added members against the blacklist. Verified
// fraudsters are banned and a warning is posted to the group. Each member is
// matched by user id first, then by username, so a username change does not
// evade detection.
func (h *Handlers) handleJoinGate(ctx context.Context, chat telego.Chat, members []telego.User) {
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
			continue // only verified records trigger enforcement
		}

		h.banMember(ctx, chat.ID, m.ID)

		warn := fmt.Sprintf(
			"<b>⚠️ Flagged fraudster banned on join</b>\n\n"+
				"<b>User:</b> %s (ID: <code>%d</code>)\n"+
				"<b>Threat:</b> %s",
			html.EscapeString(displayName(m)), m.ID, html.EscapeString(scam.ThreatLevel),
		)
		h.sendHTML(ctx, chat.ID, warn)
	}
}

func (h *Handlers) banMember(ctx context.Context, chatID, userID int64) {
	err := h.Bot.BanChatMember(&telego.BanChatMemberParams{
		ChatID:         tu.ID(chatID),
		UserID:         userID,
		RevokeMessages: true,
	})
	if err != nil {
		log.Printf("join gate: ban member %d in chat %d: %v", userID, chatID, err)
	}
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
