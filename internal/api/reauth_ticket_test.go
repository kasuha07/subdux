package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestReauthTicketFromRequestReadsCanonicalHeader(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{
			name:   "trims header ticket",
			header: "  header-ticket  ",
			want:   "header-ticket",
		},
		{
			name: "empty when header is missing",
			want: "",
		},
	}

	e := echo.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tt.header != "" {
				req.Header.Set(reauthTicketHeader, tt.header)
			}
			c := e.NewContext(req, httptest.NewRecorder())

			if got := reauthTicketFromRequest(c); got != tt.want {
				t.Fatalf("reauthTicketFromRequest() = %q, want %q", got, tt.want)
			}
		})
	}
}
