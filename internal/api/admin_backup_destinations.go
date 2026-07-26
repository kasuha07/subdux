package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
	servicebackup "github.com/kasuha07/subdux/internal/service/backup"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"github.com/labstack/echo/v4"
)

// createDestinationRequest and updateDestinationRequest are the API-layer
// payloads for backup destination mutations. Config is the raw JSON object for
// the destination type; on update, empty secret fields are preserved and fields
// named in cleared_secret_fields are wiped, mirroring notification channels.
type createDestinationRequest struct {
	Type      string `json:"type"`
	Enabled   bool   `json:"enabled"`
	Config    string `json:"config"`
	SortOrder int    `json:"sort_order"`
}

type updateDestinationRequest struct {
	Revision            uint64   `json:"revision"`
	Enabled             *bool    `json:"enabled"`
	Config              *string  `json:"config"`
	SortOrder           *int     `json:"sort_order"`
	ClearedSecretFields []string `json:"cleared_secret_fields"`
}

func (h *AdminHandler) ListBackupDestinations(c echo.Context) error {
	destinations, err := h.Backup.WithContext(c.Request().Context()).ListDestinations()
	if err != nil {
		if _, ok := serviceerr.KindOf(err); ok {
			return err
		}
		return httpx.WriteError(c, http.StatusInternalServerError, "failed_to_list_backup_destinations")
	}
	return c.JSON(http.StatusOK, destinations)
}

func (h *AdminHandler) CreateBackupDestination(c echo.Context) error {
	var input createDestinationRequest
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}

	if err := h.consumeBackupDestinationReauth(c, servicereauth.ReauthOperationBackupDestinationCreate, nil); err != nil {
		return apimw.WriteReauthError(c, err)
	}

	destination, err := h.Backup.WithContext(c.Request().Context()).CreateDestination(servicebackup.CreateDestinationInput{
		Type:      input.Type,
		Enabled:   input.Enabled,
		Config:    input.Config,
		SortOrder: input.SortOrder,
	})
	if err != nil {
		return writeBackupDestinationError(c, err)
	}
	return c.JSON(http.StatusCreated, destination)
}

func (h *AdminHandler) UpdateBackupDestination(c echo.Context) error {
	id, ok := httpx.ParseUintParam(c, "id", "invalid_backup_destination_id")
	if !ok {
		return nil
	}

	var input updateDestinationRequest
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}

	if err := h.consumeBackupDestinationReauth(c, servicereauth.ReauthOperationBackupDestinationUpdate, &servicereauth.TicketBinding{
		DestinationID:       id,
		DestinationRevision: input.Revision,
	}); err != nil {
		return apimw.WriteReauthError(c, err)
	}

	destination, err := h.Backup.WithContext(c.Request().Context()).UpdateDestination(id, servicebackup.UpdateDestinationInput{
		Revision:            input.Revision,
		Enabled:             input.Enabled,
		Config:              input.Config,
		SortOrder:           input.SortOrder,
		ClearedSecretFields: input.ClearedSecretFields,
	})
	if err != nil {
		return writeBackupDestinationError(c, err)
	}
	return c.JSON(http.StatusOK, destination)
}

func (h *AdminHandler) DeleteBackupDestination(c echo.Context) error {
	id, ok := httpx.ParseUintParam(c, "id", "invalid_backup_destination_id")
	if !ok {
		return nil
	}

	revision, err := parseBackupDestinationRevision(c)
	if err != nil {
		return writeBackupDestinationError(c, err)
	}

	if err := h.consumeBackupDestinationReauth(c, servicereauth.ReauthOperationBackupDestinationDelete, &servicereauth.TicketBinding{
		DestinationID:       id,
		DestinationRevision: revision,
	}); err != nil {
		return apimw.WriteReauthError(c, err)
	}

	if err := h.Backup.WithContext(c.Request().Context()).DeleteDestination(id, revision); err != nil {
		return writeBackupDestinationError(c, err)
	}
	return httpx.WriteMessage(c, http.StatusOK, "backup_destination_deleted")
}

// TestBackupDestination runs a read-only connectivity probe against a saved
// destination. It writes nothing, so it does not require a reauth ticket: it
// only reveals whether the stored configuration can reach and list the target.
func (h *AdminHandler) TestBackupDestination(c echo.Context) error {
	id, ok := httpx.ParseUintParam(c, "id", "invalid_backup_destination_id")
	if !ok {
		return nil
	}

	count, err := h.Backup.WithContext(c.Request().Context()).TestDestination(c.Request().Context(), id)
	if err != nil {
		return writeBackupDestinationError(c, err)
	}
	return httpx.WriteMessageCodeFields(
		c,
		http.StatusOK,
		"backup_destination_reachable",
		map[string]any{"backup_count": count},
		nil,
	)
}

// consumeBackupDestinationReauth enforces the step-up ticket that guards backup
// destination mutations. Adding, editing, or removing a destination changes
// where sensitive database backups are shipped, so it is gated the same way as
// the backup schedule.
func (h *AdminHandler) consumeBackupDestinationReauth(c echo.Context, operation string, binding *servicereauth.TicketBinding) error {
	if h.Reauth == nil {
		return serviceerr.New(serviceerr.KindInternal, "reauthentication_service_is_not_configured", "reauthentication service is not configured")
	}
	if binding == nil {
		return h.Reauth.WithContext(c.Request().Context()).Consume(
			apimw.From(c).UserID,
			operation,
			apimw.ReauthTicketFromRequest(c),
		)
	}
	return h.Reauth.WithContext(c.Request().Context()).ConsumeWithBinding(
		apimw.From(c).UserID,
		operation,
		binding,
		apimw.ReauthTicketFromRequest(c),
	)
}

// errInvalidBackupDestinationRevision is returned when the delete endpoint's
// required revision query parameter is missing, unparseable, or zero. It is
// deliberately distinct from servicereauth.ErrReauthRequired: a malformed
// request is a client validation error, not a missing step-up ticket, and
// should not make the admin UI prompt for re-authentication.
var errInvalidBackupDestinationRevision = serviceerr.New(
	serviceerr.KindInvalid,
	"invalid_backup_destination_revision",
	"backup destination revision query parameter is required and must be a positive integer",
)

func parseBackupDestinationRevision(c echo.Context) (uint64, error) {
	raw := strings.TrimSpace(c.QueryParam("revision"))
	if raw == "" {
		return 0, errInvalidBackupDestinationRevision
	}
	revision, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || revision == 0 {
		return 0, errInvalidBackupDestinationRevision
	}
	return revision, nil
}

// writeBackupDestinationError maps typed service errors to their HTTP status via
// the shared error handler and falls back to a generic 500 for anything else.
func writeBackupDestinationError(c echo.Context, err error) error {
	if _, ok := serviceerr.KindOf(err); ok {
		return err
	}
	return httpx.WriteError(c, http.StatusInternalServerError, "failed_to_save_backup_destination")
}
