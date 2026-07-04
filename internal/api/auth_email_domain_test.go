package api

import (
	"net/http"
	"testing"

	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
)

// The former authServiceErrorStatus switch has been replaced by the typed Kind
// carried on each sentinel plus the single central status mapper. This locks
// that an email-domain-not-allowed error still resolves to HTTP 403.
func TestEmailDomainNotAllowedMapsToForbidden(t *testing.T) {
	kind, ok := serviceerr.KindOf(serviceauth.ErrEmailDomainNotAllowed)
	if !ok {
		t.Fatal("ErrEmailDomainNotAllowed is not a typed service error")
	}
	if got := statusForServiceError(kind); got != http.StatusForbidden {
		t.Fatalf("statusForServiceError(%v) = %d, want %d", kind, got, http.StatusForbidden)
	}
}
