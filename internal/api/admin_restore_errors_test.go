package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	servicebackup "github.com/kasuha07/subdux/internal/service/backup"
	"github.com/labstack/echo/v4"
)

func TestWriteRestoreBackupErrorKeepsEndpointInternalMessage(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/restore", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := writeRestoreBackupError(c, errors.New("replace database failed"))
	if err != nil {
		t.Fatalf("writeRestoreBackupError() error = %v, want nil", err)
	}

	if got, want := rec.Code, http.StatusInternalServerError; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	body := rec.Body.String()
	if !hasErrorCodeForMessage(body, "failed to restore backup") {
		t.Fatalf("body = %q, want failed_to_restore_backup", body)
	}
	if strings.Contains(body, "replace database failed") {
		t.Fatalf("body = %q, should keep endpoint message without internal details", body)
	}
}

func TestWriteRestoreBackupErrorLeavesTypedBackupErrorsToCentralHandler(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/restore", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := writeRestoreBackupError(c, servicebackup.ErrBackupInvalidPassword)
	if err == nil {
		t.Fatal("writeRestoreBackupError() error = nil, want typed service error")
	}
	APIErrorHandler(e.HTTPErrorHandler)(err, c)

	if got, want := rec.Code, http.StatusBadRequest; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	body := rec.Body.String()
	if !hasErrorCodeForMessage(body, "invalid backup password") {
		t.Fatalf("body = %q, want invalid_backup_password", body)
	}
	if hasErrorCodeForMessage(body, "failed to restore backup") {
		t.Fatalf("body = %q, should not replace client-correctable backup error", body)
	}
}
