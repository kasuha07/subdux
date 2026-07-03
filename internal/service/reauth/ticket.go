package reauth

import (
	"context"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	"gorm.io/gorm"
)

const (
	reauthTicketTTL   = 5 * time.Minute
	maxReauthTickets  = 256
	reauthTicketBytes = 32
)

type reauthTicket struct {
	userID    uint
	operation string
	expiresAt time.Time
	createdAt time.Time
}

// Consume validates and atomically spends a ticket. A ticket is valid only for
// the same user and operation it was minted for, and only once.
func (s *Service) Consume(userID uint, operation string, ticket string) error {
	ticket = strings.TrimSpace(ticket)
	if ticket == "" {
		return ErrReauthRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked()

	entry, ok := s.tickets[ticket]
	if !ok {
		return ErrReauthRequired
	}
	// Single-use: remove regardless of whether it matches, so a leaked ticket
	// cannot be probed against multiple users/operations.
	delete(s.tickets, ticket)

	if entry.userID != userID || entry.operation != operation {
		return ErrReauthRequired
	}
	if pkg.NowUTC().After(entry.expiresAt) {
		return ErrReauthRequired
	}
	return nil
}

func (s *Service) mintTicket(userID uint, operation string) (string, error) {
	// generateSecureToken returns URL-safe base64 with no padding.
	ticket, err := serviceutil.GenerateSecureToken(reauthTicketBytes)
	if err != nil {
		return "", err
	}

	now := pkg.NowUTC()
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked()
	s.enforceLimitLocked()

	s.tickets[ticket] = reauthTicket{
		userID:    userID,
		operation: operation,
		expiresAt: now.Add(reauthTicketTTL),
		createdAt: now,
	}
	return ticket, nil
}

func withContext(db *gorm.DB, ctx context.Context) *gorm.DB {
	if db == nil {
		return nil
	}
	return db.WithContext(ctx)
}

func (s *Service) cleanupLocked() {
	now := pkg.NowUTC()
	for ticket, entry := range s.tickets {
		if now.After(entry.expiresAt) {
			delete(s.tickets, ticket)
		}
	}
}

func (s *Service) enforceLimitLocked() {
	overflow := len(s.tickets) - maxReauthTickets + 1
	if overflow <= 0 {
		return
	}
	for i := 0; i < overflow; i++ {
		oldestTicket := ""
		var oldestTime time.Time
		for ticket, entry := range s.tickets {
			if oldestTicket == "" || entry.createdAt.Before(oldestTime) {
				oldestTicket = ticket
				oldestTime = entry.createdAt
			}
		}
		if oldestTicket == "" {
			return
		}
		delete(s.tickets, oldestTicket)
	}
}
