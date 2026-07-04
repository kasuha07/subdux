package api

import (
	"errors"

	"github.com/kasuha07/subdux/internal/service"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
)

func isIconUploadBadRequestError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, serviceutil.ErrIconUploadUnsupportedType) ||
		errors.Is(err, serviceutil.ErrIconUploadSizeLimit) ||
		errors.Is(err, serviceutil.ErrIconUploadContentMismatch) ||
		errors.Is(err, serviceutil.ErrIconUploadInvalidICO) {
		return true
	}

	switch err.Error() {
	case "subscription not found", "payment method not found":
		return true
	default:
		return false
	}
}

func isIconUploadForbiddenError(err error) bool {
	return errors.Is(err, service.ErrImageUploadDisabled)
}
