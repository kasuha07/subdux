package service

import "github.com/shiroha/subdux/internal/model"

// loadSubscriptionsByIDs loads the given user's subscriptions keyed by ID and
// normalized for response, collapsing what would otherwise be a per-row lookup
// (an N+1 pattern) into a single IN query. IDs with no matching row — for
// example a subscription deleted after the referencing event was recorded — are
// simply absent from the returned map, so callers should test for presence.
func (s *SubscriptionService) loadSubscriptionsByIDs(userID uint, ids []uint) (map[uint]model.Subscription, error) {
	result := make(map[uint]model.Subscription, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	var subs []model.Subscription
	if err := s.DB.Where("user_id = ? AND id IN ?", userID, ids).Find(&subs).Error; err != nil {
		return nil, err
	}

	for i := range subs {
		normalizeSubscriptionForResponse(&subs[i])
		result[subs[i].ID] = subs[i]
	}
	return result, nil
}
