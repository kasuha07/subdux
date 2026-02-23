package api

import (
	"errors"

	"github.com/shiroha/subdux/internal/service"
)

func isIconUploadBadRequestError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, service.ErrIconUploadUnsupportedType) ||
		errors.Is(err, service.ErrIconUploadSizeLimit) ||
		errors.Is(err, service.ErrIconUploadContentMismatch) ||
		errors.Is(err, service.ErrIconUploadInvalidICO) {
		return true
	}

	switch err.Error() {
	case "subscription not found", "payment method not found":
		return true
	default:
		return false
	}
}
