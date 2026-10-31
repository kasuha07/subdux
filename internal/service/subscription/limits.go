package subscription

import (
	"unicode/utf8"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"gorm.io/gorm"
)

// Retained-data bounds. Request bodies are already size-limited; these bound
// what one account can accumulate across requests so a single user, API key,
// or MCP client cannot grow the shared database (and every full-collection
// list, report, calendar, and backup built from it) without limit.
const (
	MaxSubscriptionsPerUser      = 10000
	MaxSubscriptionNameRunes     = 255
	MaxSubscriptionCategoryRunes = 100
	MaxSubscriptionNotesRunes    = 10000
	MaxSubscriptionURLBytes      = 2048
)

// ErrSubscriptionLimitReached rejects creating a subscription beyond
// MaxSubscriptionsPerUser.
var ErrSubscriptionLimitReached = serviceerr.NewCode(
	serviceerr.KindInvalid,
	"subscription_limit_reached",
	"subscription limit reached",
	map[string]any{"limit": MaxSubscriptionsPerUser},
)

func errSubscriptionFieldTooLong(field string, max int) error {
	return serviceerr.NewCode(
		serviceerr.KindInvalid,
		"subscription_field_too_long",
		"subscription field is too long",
		map[string]any{"field": field, "max": max},
	)
}

// ValidateSubscriptionText enforces the stored-length bounds for a
// subscription's free-text fields. Empty values are always accepted.
func ValidateSubscriptionText(name, category, notes, url string) error {
	if utf8.RuneCountInString(name) > MaxSubscriptionNameRunes {
		return errSubscriptionFieldTooLong("name", MaxSubscriptionNameRunes)
	}
	if utf8.RuneCountInString(category) > MaxSubscriptionCategoryRunes {
		return errSubscriptionFieldTooLong("category", MaxSubscriptionCategoryRunes)
	}
	if utf8.RuneCountInString(notes) > MaxSubscriptionNotesRunes {
		return errSubscriptionFieldTooLong("notes", MaxSubscriptionNotesRunes)
	}
	if len(url) > MaxSubscriptionURLBytes {
		return errSubscriptionFieldTooLong("url", MaxSubscriptionURLBytes)
	}
	return nil
}

func validateOptionalSubscriptionText(name, category, notes, url *string) error {
	value := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	return ValidateSubscriptionText(value(name), value(category), value(notes), value(url))
}

// CapacityTracker hands out the subscriptions a user may still create inside
// one transaction, so bulk paths (imports) count once instead of per row.
type CapacityTracker struct {
	remaining int64
}

// NewCapacityTracker counts the user's subscriptions in tx. Run it inside the
// transaction that performs the inserts.
func NewCapacityTracker(tx *gorm.DB, userID uint) (*CapacityTracker, error) {
	var count int64
	if err := tx.Model(&model.Subscription{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return nil, err
	}
	return &CapacityTracker{remaining: MaxSubscriptionsPerUser - count}, nil
}

// Reserve claims room for one more subscription or returns
// ErrSubscriptionLimitReached.
func (c *CapacityTracker) Reserve() error {
	if c.remaining <= 0 {
		return ErrSubscriptionLimitReached
	}
	c.remaining--
	return nil
}
