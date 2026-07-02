package api

import (
	"net/http"
	"testing"

	"github.com/kasuha07/subdux/internal/service"
)

func TestAuthServiceErrorStatusEmailDomainNotAllowed(t *testing.T) {
	status := authServiceErrorStatus(service.ErrEmailDomainNotAllowed)
	if status != http.StatusForbidden {
		t.Fatalf("authServiceErrorStatus() = %d, want %d", status, http.StatusForbidden)
	}
}
