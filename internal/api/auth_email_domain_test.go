package api

import (
	"net/http"
	"testing"

	"github.com/kasuha07/subdux/internal/service"
	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
)

func TestAuthServiceErrorStatusEmailDomainNotAllowed(t *testing.T) {
	for name, err := range map[string]error{
		"compat": service.ErrEmailDomainNotAllowed,
		"auth":   serviceauth.ErrEmailDomainNotAllowed,
	} {
		t.Run(name, func(t *testing.T) {
			status := authServiceErrorStatus(err)
			if status != http.StatusForbidden {
				t.Fatalf("authServiceErrorStatus() = %d, want %d", status, http.StatusForbidden)
			}
		})
	}
}
