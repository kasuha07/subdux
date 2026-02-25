package service

import "github.com/shiroha/subdux/internal/model"

func (s *AdminService) GetStats() (*AdminStats, error) {
	var stats AdminStats

	s.DB.Model(&model.User{}).Count(&stats.TotalUsers)
	s.DB.Model(&model.Subscription{}).Count(&stats.TotalSubscriptions)

	var subs []model.Subscription
	if err := s.DB.Where("enabled = ?", true).Find(&subs).Error; err != nil {
		return nil, err
	}

	for _, sub := range subs {
		factor := subscriptionMonthlyFactor(sub)
		if factor > 0 {
			stats.TotalMonthlySpend += sub.Amount * factor
		}
	}

	return &stats, nil
}
