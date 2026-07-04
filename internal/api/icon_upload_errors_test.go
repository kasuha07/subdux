package api

import (
	"errors"
	"testing"

	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
)

// The former isIconUploadBadRequestError/isIconUploadForbiddenError helpers
// classified icon-upload failures by message. That responsibility now lives in
// the typed serviceerr.Kind carried by each sentinel and the single central
// error handler. These tests lock the Kind of each icon-upload sentinel so the
// resulting HTTP status cannot drift.
func TestIconUploadSentinelKinds(t *testing.T) {
	badRequest := []error{
		serviceutil.ErrIconUploadSizeLimit,
		serviceutil.ErrIconUploadUnsupportedType,
		serviceutil.ErrIconUploadContentMismatch,
		serviceutil.ErrIconUploadInvalidICO,
	}
	for _, err := range badRequest {
		kind, ok := serviceerr.KindOf(err)
		if !ok || kind != serviceerr.KindInvalid {
			t.Fatalf("KindOf(%v) = %v, ok=%v; want KindInvalid", err, kind, ok)
		}
	}

	if kind, ok := serviceerr.KindOf(serviceutil.ErrImageUploadDisabled); !ok || kind != serviceerr.KindForbidden {
		t.Fatalf("KindOf(ErrImageUploadDisabled) = %v, ok=%v; want KindForbidden", kind, ok)
	}

	if _, ok := serviceerr.KindOf(errors.New("failed to save icon file")); ok {
		t.Fatal("plain internal error should not classify as a typed service error")
	}
}
