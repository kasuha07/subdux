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

// testDestinationConfigRequest probes a config the admin has not saved yet.
// DestinationID names the destination the edit form was opened from, which is
// what lets secrets left blank be inherited from storage; cleared_secret_fields
// carries the same "drop this secret" intent as an update.
type testDestinationConfigRequest struct {
	Type                string   `json:"type"`
	Config              string   `json:"config"`
	DestinationID       uint     `json:"destination_id"`
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

// RunBackupDestination fires one destination's backup plan immediately, using
// that destination's own archive settings. It is gated by the same step-up
// operation the old global run-now used: a run ships a full database snapshot
// to the configured target, so it is proven-present-admin work.
//
// The ticket is intentionally unbound. Unlike create/update/delete, this changes
// no configuration — the destination it delivers to was already reviewed behind
// a bound ticket when it was saved — so there is no revision for the admin to
// confirm here.
func (h *AdminHandler) RunBackupDestination(c echo.Context) error {
	id, ok := httpx.ParseUintParam(c, "id", "invalid_backup_destination_id")
	if !ok {
		return nil
	}

	if err := h.consumeBackupDestinationReauth(c, servicereauth.ReauthOperationBackupRun, nil); err != nil {
		return apimw.WriteReauthError(c, err)
	}

	result, err := h.Backup.WithContext(c.Request().Context()).RunDestinationBackup(c.Request().Context(), id)
	if err != nil {
		if _, ok := serviceerr.KindOf(err); ok {
			return err
		}
		return httpx.WriteError(c, http.StatusInternalServerError, "backup_failed")
	}

	return httpx.WriteMessageFields(c, http.StatusOK, "backup_created", map[string]any{
		"file":                      result.ArchiveName,
		"run_id":                    result.RunID,
		"status":                    result.Status,
		"delivery_status":           result.DeliveryStatus,
		"retention_status":          result.RetentionStatus,
		"bookkeeping_status":        result.BookkeepingStatus,
		"global_bookkeeping_status": result.GlobalBookkeepingStatus,
		"error":                     result.Error,
		"results":                   result.Results,
	})
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
		return writeBackupDestinationProbeError(c, err)
	}
	return httpx.WriteMessageCodeFields(
		c,
		http.StatusOK,
		"backup_destination_reachable",
		map[string]any{"backup_count": count},
		nil,
	)
}

// TestBackupDestinationConfig runs the same read-only connectivity probe as
// TestBackupDestination against a config the admin has not saved yet, so a
// destination can be validated from the add/edit dialog before it is committed.
//
// It writes nothing and carries no reauth ticket, matching the saved-config probe
// above. Be precise about what that costs, because it is easy to overstate the
// precedent: POST /admin/settings/ssrf/test does NOT reach out — it resolves DNS
// and evaluates policy (see outbound.TestSSRF) — so it is not the thing that
// makes this safe. What does hold is that UpdateSettings carries no step-up
// ticket either, and the OIDC issuer and exchange-rate endpoint it writes are
// themselves exempt from the SSRF filter, so a session-only admin already has
// SSRF-exempt outbound reach. This endpoint makes that reach more direct rather
// than newly possible.
//
// What it must not become is a way around the step-up gate on where credentials
// are sent, so the service refuses to pair a stored secret with an endpoint
// changed in the same request.
//
// For a local destination the exposure is a filesystem rather than a network one:
// an unticketed probe reveals whether an arbitrary absolute path is readable and
// how many backup archives sit in it. The yield is a count of subdux-backup-*.zip
// files. It is deliberately not confined to the data directory, because the save
// path accepts any absolute path (resolveBackupDir) and a probe that rejected
// configs which save fine would be worse than the disclosure it prevents.
func (h *AdminHandler) TestBackupDestinationConfig(c echo.Context) error {
	var input testDestinationConfigRequest
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}

	count, err := h.Backup.WithContext(c.Request().Context()).TestDestinationConfig(
		c.Request().Context(),
		servicebackup.TestDestinationConfigInput{
			Type:                input.Type,
			Config:              input.Config,
			DestinationID:       input.DestinationID,
			ClearedSecretFields: input.ClearedSecretFields,
		},
	)
	if err != nil {
		return writeBackupDestinationProbeError(c, err)
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

// writeBackupDestinationProbeError is the connectivity-probe counterpart. The
// probes need their own fallback because their most common failure — a wrong
// endpoint, a rejected credential, a refused connection, an unverifiable
// certificate — arrives as an untyped error from minio-go, net/http, or os, and
// the mutation fallback above would report it as a 500 reading "failed to save
// backup destination" for a request that saves nothing.
//
// 502 is the honest status: the request was well formed and the server worked;
// the destination beyond it did not answer.
func writeBackupDestinationProbeError(c echo.Context, err error) error {
	if _, ok := serviceerr.KindOf(err); ok {
		return err
	}
	return httpx.WriteError(c, http.StatusBadGateway, "backup_destination_unreachable")
}
