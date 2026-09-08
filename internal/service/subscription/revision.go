package subscription

import (
	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	"gorm.io/gorm"
)

// Check the client's snapshot before reconciling lifecycle state. Keeping both
// operations in one transaction avoids rejecting our own lifecycle advance and
// rolls back its changes if validation or the requested mutation fails.
func (s *Service) withRevision(userID, id uint, revision uint64, mutate func(*Service) (*model.Subscription, error)) (*model.Subscription, error) {
	var result *model.Subscription
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var row model.Subscription
		if err := tx.Where("id = ? AND user_id = ?", id, userID).First(&row).Error; err != nil {
			return err
		}
		if err := serviceutil.CheckRevision(revision, row.Revision); err != nil {
			return err
		}
		clone := *s
		clone.DB = tx
		var err error
		result, err = mutate(&clone)
		return err
	})
	return result, err
}
