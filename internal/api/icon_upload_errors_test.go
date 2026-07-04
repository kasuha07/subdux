package api

import (
	"errors"
	"testing"

	"github.com/kasuha07/subdux/internal/service"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
)

func TestIsIconUploadBadRequestError(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{err: errors.New("subscription not found"), want: true},
		{err: errors.New("payment method not found"), want: true},
		{err: serviceutil.ErrIconUploadSizeLimit, want: true},
		{err: serviceutil.ErrIconUploadUnsupportedType, want: true},
		{err: serviceutil.ErrIconUploadContentMismatch, want: true},
		{err: serviceutil.ErrIconUploadInvalidICO, want: true},
		{err: service.ErrImageUploadDisabled, want: false},
		{err: errors.New("failed to save icon file"), want: false},
	}

	for _, tt := range tests {
		if got := isIconUploadBadRequestError(tt.err); got != tt.want {
			t.Fatalf("isIconUploadBadRequestError(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}

func TestIsIconUploadForbiddenError(t *testing.T) {
	if !isIconUploadForbiddenError(service.ErrImageUploadDisabled) {
		t.Fatalf("isIconUploadForbiddenError(%v) = false, want true", service.ErrImageUploadDisabled)
	}
	if isIconUploadForbiddenError(serviceutil.ErrIconUploadUnsupportedType) {
		t.Fatalf("isIconUploadForbiddenError(%v) = true, want false", serviceutil.ErrIconUploadUnsupportedType)
	}
}
