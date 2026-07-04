package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
	"github.com/kasuha07/subdux/internal/model"
	auditservice "github.com/kasuha07/subdux/internal/service/audit"
	"github.com/labstack/echo/v4"
)

type AuditHandler struct {
	Service *auditservice.Service
}

func NewAuditHandler(s *auditservice.Service) *AuditHandler {
	return &AuditHandler{Service: s}
}

type auditEventResponse struct {
	EventID             string          `json:"event_id"`
	OccurredAt          time.Time       `json:"occurred_at"`
	UserID              uint            `json:"user_id"`
	KeyID               uint            `json:"key_id"`
	KeyKind             string          `json:"key_kind"`
	ScopeUsed           string          `json:"scope_used"`
	Transport           string          `json:"transport"`
	ToolName            string          `json:"tool_name"`
	ResourceType        string          `json:"resource_type"`
	ResourceID          string          `json:"resource_id"`
	Action              string          `json:"action"`
	Status              string          `json:"status"`
	Error               string          `json:"error"`
	LatencyMS           int64           `json:"latency_ms"`
	ClientName          string          `json:"client_name"`
	ClientVersion       string          `json:"client_version"`
	RequestID           string          `json:"request_id"`
	RequestArgsRedacted json.RawMessage `json:"request_args_redacted,omitempty"`
	BeforeSnapshot      json.RawMessage `json:"before_snapshot,omitempty"`
	AfterSnapshot       json.RawMessage `json:"after_snapshot,omitempty"`
}

func (h *AuditHandler) ListUserEvents(c echo.Context) error {
	userID := apimw.From(c).UserID
	events, err := h.Service.WithContext(c.Request().Context()).List(parseAuditEventFilter(c, &userID))
	if err != nil {
		return httpx.WriteError(c, http.StatusInternalServerError, "failed to list audit events")
	}
	return c.JSON(http.StatusOK, mapAuditEventResponses(events))
}

func (h *AuditHandler) ListAdminEvents(c echo.Context) error {
	events, err := h.Service.WithContext(c.Request().Context()).List(parseAuditEventFilter(c, nil))
	if err != nil {
		return httpx.WriteError(c, http.StatusInternalServerError, "failed to list audit events")
	}
	return c.JSON(http.StatusOK, mapAuditEventResponses(events))
}

func parseAuditEventFilter(c echo.Context, userID *uint) auditservice.EventFilter {
	limit, _ := strconv.Atoi(strings.TrimSpace(c.QueryParam("limit")))
	var before *time.Time
	if rawBefore := strings.TrimSpace(c.QueryParam("before")); rawBefore != "" {
		if parsed, err := time.Parse(time.RFC3339, rawBefore); err == nil {
			before = &parsed
		}
	}
	return auditservice.EventFilter{
		UserID:       userID,
		Limit:        limit,
		Before:       before,
		Status:       strings.TrimSpace(c.QueryParam("status")),
		ResourceType: strings.TrimSpace(c.QueryParam("resource_type")),
	}
}

func mapAuditEventResponses(events []model.AuditEvent) []auditEventResponse {
	responses := make([]auditEventResponse, len(events))
	for i, event := range events {
		responses[i] = mapAuditEventResponse(event)
	}
	return responses
}

func mapAuditEventResponse(event model.AuditEvent) auditEventResponse {
	return auditEventResponse{
		EventID:             event.EventID,
		OccurredAt:          event.OccurredAt,
		UserID:              event.UserID,
		KeyID:               event.KeyID,
		KeyKind:             event.KeyKind,
		ScopeUsed:           event.ScopeUsed,
		Transport:           event.Transport,
		ToolName:            event.ToolName,
		ResourceType:        event.ResourceType,
		ResourceID:          event.ResourceID,
		Action:              event.Action,
		Status:              event.Status,
		Error:               event.Error,
		LatencyMS:           event.LatencyMS,
		ClientName:          event.ClientName,
		ClientVersion:       event.ClientVersion,
		RequestID:           event.RequestID,
		RequestArgsRedacted: auditJSON(event.RequestArgsRedacted),
		BeforeSnapshot:      auditJSON(event.BeforeSnapshot),
		AfterSnapshot:       auditJSON(event.AfterSnapshot),
	}
}

func auditJSON(value string) json.RawMessage {
	value = strings.TrimSpace(value)
	if value == "" || !json.Valid([]byte(value)) {
		return nil
	}
	return json.RawMessage(value)
}
