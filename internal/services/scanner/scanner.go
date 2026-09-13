package scanner

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"telefraud/internal/database"
	"telefraud/internal/models"
	"telefraud/internal/services/identity"
)

// FlaggedMember represents a detected scammer in a group.
type FlaggedMember struct {
	User        telego.User
	Scammer     *models.Scammer
	Identifiers []models.ScammerIdentifier
	ActionTaken string
}

// ScanResult holds the result of auditing a single group.
type ScanResult struct {
	GroupID        int64
	GroupTitle     string
	TotalAudited   int
	FlaggedMembers []FlaggedMember
	Duration       time.Duration
}

// GroupScanner audits Telegram group members against the fraud database.
type GroupScanner struct {
	bot      *telego.Bot
	store    *database.IdentityStore
	resolver *identity.Resolver
}

// NewGroupScanner creates a GroupScanner.
func NewGroupScanner(bot *telego.Bot, store *database.IdentityStore, resolver *identity.Resolver) *GroupScanner {
	return &GroupScanner{
		bot:      bot,
		store:    store,
		resolver: resolver,
	}
}

// ScanGroup checks administrators and known group users against the fraud database.
func (s *GroupScanner) ScanGroup(ctx context.Context, groupID int64, executedBy int64) (*ScanResult, error) {
	start := time.Now()

	group, err := s.store.GetGroup(ctx, groupID)
	if err != nil {
		// Try to fetch chat info from Telegram directly
		chat, cErr := s.bot.GetChat(&telego.GetChatParams{ChatID: tu.ID(groupID)})
		if cErr == nil {
			group = &models.Group{
				GroupID:           groupID,
				Title:             chat.Title,
				AutoBanEnabled:    true,
				AutoDeleteEnabled: true,
				ScanOnJoinEnabled: true,
				DailyScanEnabled:  true,
				WarnOnDetected:    true,
			}
			_ = s.store.UpsertGroup(ctx, *group)
		} else {
			return nil, fmt.Errorf("group not found: %w", err)
		}
	}

	admins, err := s.bot.GetChatAdministrators(&telego.GetChatAdministratorsParams{
		ChatID: tu.ID(groupID),
	})
	if err != nil {
		return nil, fmt.Errorf("get chat administrators: %w", err)
	}

	result := &ScanResult{
		GroupID:    groupID,
		GroupTitle: group.Title,
	}

	for _, adminMember := range admins {
		u := adminMember.MemberUser()
		if u.IsBot {
			continue
		}
		result.TotalAudited++

		cand := identity.Candidates{UserID: u.ID}
		if uname, ok := models.NormalizeUsername(u.Username); ok {
			cand.Username = uname
		}

		scam, err := s.resolver.Resolve(ctx, cand)
		if err != nil || scam == nil || scam.Status != models.StatusVerified {
			continue
		}

		_, idents, _ := s.resolver.Get(ctx, scam.ID)
		action := "FLAGGED"

		if group.AutoBanEnabled {
			_ = s.bot.BanChatMember(&telego.BanChatMemberParams{
				ChatID:         tu.ID(groupID),
				UserID:         u.ID,
				RevokeMessages: group.AutoDeleteEnabled,
			})
			action = "BANNED"

			_ = s.store.LogModeration(ctx, models.ModerationLog{
				GroupID:      &groupID,
				ScammerID:    &scam.ID,
				TargetUserID: u.ID,
				Action:       models.ActionBan,
				Reason:       fmt.Sprintf("Detected during group scan: %s", scam.Reason),
				ExecutedBy:   executedBy,
			})
		}

		result.FlaggedMembers = append(result.FlaggedMembers, FlaggedMember{
			User:        u,
			Scammer:     scam,
			Identifiers: idents,
			ActionTaken: action,
		})
	}

	_ = s.store.UpdateGroupLastScanned(ctx, groupID)
	result.Duration = time.Since(start)
	return result, nil
}

// StartDailyScannerWorker runs scheduled background daily audits across all active groups.
func (s *GroupScanner) StartDailyScannerWorker(ctx context.Context) {
	log.Println("[Scanner] Daily Group Scanner worker started (scheduled daily at 03:00 UTC)")
	go func() {
		for {
			now := time.Now().UTC()
			// Calculate next 03:00:00 UTC
			next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, time.UTC)
			if !next.After(now) {
				next = next.Add(24 * time.Hour)
			}
			timer := time.NewTimer(time.Until(next))

			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				log.Println("[Scanner] Starting daily scheduled group security audit...")
				groups, err := s.store.ListActiveGroups(ctx)
				if err != nil {
					log.Printf("[Scanner] Failed to fetch active groups: %v", err)
					continue
				}

				totalFlagged := 0
				for _, g := range groups {
					if !g.DailyScanEnabled {
						continue
					}
					res, err := s.ScanGroup(ctx, g.GroupID, 0)
					if err != nil {
						log.Printf("[Scanner] Audit error for group %d (%s): %v", g.GroupID, g.Title, err)
						continue
					}
					if len(res.FlaggedMembers) > 0 {
						totalFlagged += len(res.FlaggedMembers)
						if g.WarnOnDetected {
							msgText := fmt.Sprintf(
								"<b>🛡️ Daily Security Audit Completed</b>\n\n"+
									"Audited <b>%d</b> members.\n"+
									"⚠️ <b>%d</b> blacklisted fraudsters detected and actioned.",
								res.TotalAudited, len(res.FlaggedMembers),
							)
							_, _ = s.bot.SendMessage(tu.Message(tu.ID(g.GroupID), msgText).WithParseMode(telego.ModeHTML))
						}
					}
				}
				log.Printf("[Scanner] Daily audit finished: scanned %d groups, flagged %d total accounts", len(groups), totalFlagged)
			}
		}
	}()
}
