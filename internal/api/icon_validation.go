package api

import "github.com/kasuha07/subdux/internal/api/contract"

func validateIcon(icon string) bool {
	return contract.ValidateIcon(icon)
}

func validateSubscriptionIcon(icon string) bool {
	return contract.ValidateSubscriptionIcon(icon)
}
