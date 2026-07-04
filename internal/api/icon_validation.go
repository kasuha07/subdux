package api

import "github.com/kasuha07/subdux/internal/api/apicontract"

func validateIcon(icon string) bool {
	return apicontract.ValidateIcon(icon)
}

func validateSubscriptionIcon(icon string) bool {
	return apicontract.ValidateSubscriptionIcon(icon)
}
