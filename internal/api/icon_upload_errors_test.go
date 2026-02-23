package api

import (
	"errors"
	"testing"

	"github.com/shiroha/subdux/internal/service"
)

func TestIsIconUploadBadRequestError(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{err: errors.New("subscription not found"), want: true},
		{err: errors.New("payment method not found"), want: true},
		{err: service.ErrIconUploadSizeLimit, want: true},
		{err: service.ErrIconUploadUnsupportedType, want: true},
		{err: service.ErrIconUploadContentMismatch, want: true},
		{err: service.ErrIconUploadInvalidICO, want: true},
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
	if isIconUploadForbiddenError(service.ErrIconUploadUnsupportedType) {
		t.Fatalf("isIconUploadForbiddenError(%v) = true, want false", service.ErrIconUploadUnsupportedType)
	}
}
