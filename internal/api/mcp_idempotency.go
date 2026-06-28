package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/service"
	"gorm.io/gorm"
)

const (
	idempotencyKeyArg       = "idempotency_key"
	maxIdempotencyKeyLength = 255
)

// errIdempotencyMismatch is returned when a stored idempotency key is replayed
// with different request arguments. It signals client misuse (the same key was
// reused for a genuinely different operation) rather than a transient failure.
var errIdempotencyMismatch = errors.New("idempotency_key was reused with different request arguments")

// mcpWriteOutcome is what a write mutation produces inside the idempotent
// transaction: the success result returned to the caller plus the metadata
// needed to record an audit event and to run a post-commit side effect such as
// deleting a managed icon file once the mutation is durable.
type mcpWriteOutcome struct {
	Result         *mcpToolResult
	Action         string
	ResourceID     string
	BeforeSnapshot interface{}
	AfterSnapshot  interface{}
	PostCommit     func()
}

// mcpWriteSpec describes a single MCP write tool for the shared idempotent
// runner. mutate performs the write within the transaction; returning an error
// rolls back the transaction so that no audit event and no idempotency record
// are persisted, and the error is mapped to a tool or RPC error by the runner.
type mcpWriteSpec struct {
	ToolName     string
	ResourceType string
	mutate       func(tx *gorm.DB) (*mcpWriteOutcome, error)
}

// runIdempotentWrite executes a write tool exactly once per (user, idempotency
// key). A first call runs the mutation, audit event, and idempotency record in
// a single transaction. A repeat call with the same key and matching request
// fingerprint replays the stored result without re-executing the mutation; a
// repeat call with a different fingerprint is rejected. SQLite serializes
// writes (a single open connection) and the unique (user_id, idempotency_key)
// index makes the lookup-then-insert race-safe across concurrent retries.
func (h *MCPHandler) runIdempotentWrite(
	ctx context.Context,
	principal *mcpPrincipal,
	args map[string]interface{},
	spec mcpWriteSpec,
) (*mcpToolResult, *mcpError) {
	key, err := readWriteIdempotencyKey(args)
	if err != nil {
		return nil, invalidMCPParams(err)
	}
	if err := rejectUnknownMCPArgs(spec.ToolName, args); err != nil {
		return nil, invalidMCPParams(err)
	}
	fingerprint, err := mcpRequestFingerprint(spec.ToolName, args)
	if err != nil {
		return nil, internalMCPError(err)
	}

	auditEnabled, err := h.audit.WithContext(ctx).IsEnabled()
	if err != nil {
		return nil, internalMCPError(err)
	}

	var (
		result     *mcpToolResult
		postCommit func()
	)

	txErr := h.subscriptions.WithContext(ctx).DB.Transaction(func(tx *gorm.DB) error {
		existing, err := service.NewIdempotencyService(tx).Lookup(principal.UserID, key)
		if err != nil {
			return err
		}
		if existing != nil {
			if existing.RequestHash != fingerprint {
				return errIdempotencyMismatch
			}
			stored, err := decodeStoredMCPResult(existing.Response)
			if err != nil {
				return err
			}
			result = stored
			return nil
		}

		start := time.Now()
		outcome, err := spec.mutate(tx)
		if err != nil {
			return err
		}

		if auditEnabled {
			if _, err := service.NewAuditService(tx).Create(service.CreateAuditEventInput{
				UserID:              principal.UserID,
				KeyID:               principal.KeyID,
				KeyKind:             principal.KeyKind,
				ScopeUsed:           service.APIKeyScopeWrite,
				Transport:           service.AuditTransportMCP,
				ToolName:            spec.ToolName,
				ResourceType:        spec.ResourceType,
				ResourceID:          outcome.ResourceID,
				Action:              outcome.Action,
				Status:              service.AuditStatusSuccess,
				LatencyMS:           time.Since(start).Milliseconds(),
				ClientName:          principal.Request.ClientName,
				ClientVersion:       principal.Request.ClientVersion,
				RequestID:           principal.Request.RequestID,
				RequestArgsRedacted: args,
				BeforeSnapshot:      outcome.BeforeSnapshot,
				AfterSnapshot:       outcome.AfterSnapshot,
			}); err != nil {
				return err
			}
		}

		encoded, err := encodeStoredMCPResult(outcome.Result)
		if err != nil {
			return err
		}
		if err := service.NewIdempotencyService(tx).Save(&model.MCPIdempotencyKey{
			UserID:         principal.UserID,
			IdempotencyKey: key,
			KeyID:          principal.KeyID,
			ToolName:       spec.ToolName,
			RequestHash:    fingerprint,
			ResourceType:   spec.ResourceType,
			ResourceID:     outcome.ResourceID,
			Response:       encoded,
		}); err != nil {
			return err
		}

		result = outcome.Result
		postCommit = outcome.PostCommit
		return nil
	})

	if txErr != nil {
		return mapMCPWriteError(txErr)
	}

	// Post-commit side effects (for example deleting a managed icon file) run
	// only after the mutation is durable and only for the call that actually
	// performed the write, never on a replay.
	if postCommit != nil {
		postCommit()
	}
	return result, nil
}

// mapMCPWriteError translates a rolled-back transaction error into the response
// shape the MCP layer expects: recoverable business errors surface as tool
// execution errors (isError) while everything else is an RPC-level error.
func mapMCPWriteError(err error) (*mcpToolResult, *mcpError) {
	switch {
	case errors.Is(err, errIdempotencyMismatch):
		return nil, invalidMCPParams(err)
	case isSubscriptionBadRequestError(err.Error()) || errors.Is(err, gorm.ErrRecordNotFound):
		return mcpToolExecutionError(err.Error()), nil
	default:
		return nil, internalMCPError(err)
	}
}

// readWriteIdempotencyKey extracts and validates the required idempotency key
// from a write tool's arguments. The key must be a genuine JSON string: the
// lenient coercion used elsewhere is rejected here so a numeric or boolean key
// cannot masquerade as a distinct string.
func readWriteIdempotencyKey(args map[string]interface{}) (string, error) {
	raw, exists := args[idempotencyKeyArg]
	if !exists || raw == nil {
		return "", errors.New("idempotency_key is required")
	}
	value, ok := raw.(string)
	if !ok {
		return "", errors.New("idempotency_key must be a string")
	}
	key := strings.TrimSpace(value)
	if key == "" {
		return "", errors.New("idempotency_key is required")
	}
	if len(key) > maxIdempotencyKeyLength {
		return "", errors.New("idempotency_key must be 255 characters or less")
	}
	return key, nil
}

// rejectUnknownMCPArgs enforces the tool's declared input schema by rejecting
// any argument key the schema does not list. The low-level SDK dispatch path
// does not validate arguments against the schema, so "additionalProperties:
// false" would otherwise be advisory only. Beyond tightening the input surface,
// this keeps the idempotency fingerprint stable: a stray or mistyped field
// cannot silently perturb the hash and turn a legitimate retry into a new
// operation.
func rejectUnknownMCPArgs(toolName string, args map[string]interface{}) error {
	definition, ok := mcpToolDefinitionByName(toolName)
	if !ok {
		return nil
	}
	schema := definition.InputSchema()
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return nil
	}
	for key := range args {
		if _, allowed := properties[key]; !allowed {
			return fmt.Errorf("unexpected argument: %s", key)
		}
	}
	return nil
}

// mcpRequestFingerprint produces a deterministic hash of the tool name and its
// arguments excluding the idempotency key itself. encoding/json sorts map keys,
// so the same logical request always yields the same fingerprint regardless of
// argument ordering. A marshal failure is returned rather than swallowed: a
// degraded fingerprint could let two genuinely different requests collide and
// wrongly replay each other's result.
func mcpRequestFingerprint(toolName string, args map[string]interface{}) (string, error) {
	filtered := make(map[string]interface{}, len(args))
	for key, value := range args {
		if key == idempotencyKeyArg {
			continue
		}
		filtered[key] = value
	}

	payload := map[string]interface{}{
		"tool": toolName,
		"args": filtered,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func encodeStoredMCPResult(result *mcpToolResult) (string, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func decodeStoredMCPResult(stored string) (*mcpToolResult, error) {
	var result mcpToolResult
	if err := json.Unmarshal([]byte(stored), &result); err != nil {
		return nil, err
	}
	return &result, nil
}
