package api

import (
	"net/http"
	"testing"

	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
)

func TestAuthServiceErrorStatusEmailDomainNotAllowed(t *testing.T) {
	status := authServiceErrorStatus(serviceauth.ErrEmailDomainNotAllowed)
	if status != http.StatusForbidden {
		t.Fatalf("authServiceErrorStatus() = %d, want %d", status, http.StatusForbidden)
	}
}
