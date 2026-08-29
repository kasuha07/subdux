package subscription

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"gorm.io/gorm"
)

// MaxBatchSubscriptionIDs bounds the number of subscriptions a single batch
// request may address. Selection-driven bulk operations are naturally small;
// the cap keeps a single request from fanning out into unbounded per-row work.
const MaxBatchSubscriptionIDs = 200

// Batch actions. "update" applies whichever optional fields the client
// provides (status, renewal_mode, category_id, payment_method_id), mirroring
// the single-item Update contract so batch edits behave identically to the
// detail form.
const (
	BatchActionDelete      = "delete"
	BatchActionUpdate      = "update"
	BatchActionMarkRenewed = "mark_renewed"
)

// BatchSubscriptionInput describes one bulk operation. IDs are deduplicated
// and bounded by MaxBatchSubscriptionIDs. CategoryIDSet/PaymentMethodIDSet
// distinguish an explicit clear (null) from "not provided", matching
// UpdateSubscriptionInput's null-handling semantics.
type BatchSubscriptionInput struct {
	Action          string  `json:"action"`
	IDs             []uint  `json:"ids"`
	Status          *string `json:"status"`
	RenewalMode     *string `json:"renewal_mode"`
	CategoryID      *uint   `json:"category_id"`
	PaymentMethodID *uint   `json:"payment_method_id"`

	CategoryIDSet      bool `json:"-"`
	PaymentMethodIDSet bool `json:"-"`
}

func (input *BatchSubscriptionInput) UnmarshalJSON(data []byte) error {
	type alias BatchSubscriptionInput
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*input = BatchSubscriptionInput(decoded)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if _, ok := raw["category_id"]; ok {
		input.CategoryIDSet = true
	}
	if _, ok := raw["payment_method_id"]; ok {
		input.PaymentMethodIDSet = true
	}
	return nil
}

// BatchSubscriptionFailure reports why one addressed subscription could not be
// processed. Code is the stable service error code so clients can localize it.
type BatchSubscriptionFailure struct {
	ID      uint   `json:"id"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// BatchSubscriptionResult aggregates a bulk operation. Total counts the
// deduplicated requested IDs; Succeeded/Failed describe per-item outcomes.
// Failures is empty when every item succeeded.
type BatchSubscriptionResult struct {
	Total     int                        `json:"total"`
	Succeeded int                        `json:"succeeded"`
	Failed    int                        `json:"failed"`
	Failures  []BatchSubscriptionFailure `json:"failures"`
}

func batchFailureFor(id uint, err error) BatchSubscriptionFailure {
	failure := BatchSubscriptionFailure{ID: id}
	if err == nil {
		return failure
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		failure.Code = ErrSubscriptionNotFound.Code
		failure.Message = ErrSubscriptionNotFound.Msg
		return failure
	}
	var typed *serviceerr.Error
	if errors.As(err, &typed) && typed != nil {
		failure.Code = typed.Code
		failure.Message = typed.Msg
		return failure
	}
	failure.Message = err.Error()
	return failure
}

func deduplicateIDs(ids []uint) []uint {
	if len(ids) < 2 {
		return ids
	}
	seen := make(map[uint]struct{}, len(ids))
	deduped := make([]uint, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		deduped = append(deduped, id)
	}
	return deduped
}

// validateBatchInput performs request-level validation that is independent of
// any individual subscription: the action, the addressed ID list, the optional
// update fields, and the referenced category/payment-method existence.
func (s *Service) validateBatchInput(userID uint, input BatchSubscriptionInput) error {
	switch input.Action {
	case BatchActionDelete, BatchActionMarkRenewed:
	case BatchActionUpdate:
		if input.Status == nil && input.RenewalMode == nil && input.CategoryID == nil && !input.CategoryIDSet && input.PaymentMethodID == nil && !input.PaymentMethodIDSet {
			return serviceerr.New(serviceerr.KindInvalid, "batch_update_requires_at_least_one_field", "batch update requires at least one of status, renewal_mode, category_id, payment_method_id")
		}
	default:
		return serviceerr.New(serviceerr.KindInvalid, "invalid_batch_action", fmt.Sprintf("invalid batch action: %s", input.Action))
	}

	if len(input.IDs) == 0 {
		return serviceerr.New(serviceerr.KindInvalid, "batch_ids_required", "batch ids must not be empty")
	}
	if len(input.IDs) > MaxBatchSubscriptionIDs {
		return serviceerr.NewCode(
			serviceerr.KindInvalid,
			"batch_too_many_ids",
			fmt.Sprintf("batch ids must not exceed %d", MaxBatchSubscriptionIDs),
			map[string]any{"max": MaxBatchSubscriptionIDs},
		)
	}

	if input.Status != nil {
		if !isValidSubscriptionStatus(normalizeStatus(*input.Status)) {
			return ErrStatusInvalid
		}
	}
	if input.RenewalMode != nil {
		if !isValidRenewalMode(normalizeRenewalMode(*input.RenewalMode)) {
			return ErrRenewalModeInvalid
		}
	}

	if input.CategoryID != nil && *input.CategoryID != 0 {
		if err := s.validateCategory(userID, *input.CategoryID); err != nil {
			return err
		}
	}
	if input.PaymentMethodID != nil && *input.PaymentMethodID != 0 {
		if err := s.validatePaymentMethod(userID, *input.PaymentMethodID); err != nil {
			return err
		}
	}

	return nil
}

// Batch applies one bulk operation to the caller's subscriptions. Each item is
// processed through the same service methods as the single-item endpoints (so
// lifecycle reconciliation, validation, and subscription-event audit records
// behave identically) and committed independently: an item that cannot be
// processed is reported in the result rather than aborting the rest.
func (s *Service) Batch(userID uint, input BatchSubscriptionInput) (*BatchSubscriptionResult, error) {
	if err := s.validateBatchInput(userID, input); err != nil {
		return nil, err
	}

	ids := deduplicateIDs(input.IDs)
	result := &BatchSubscriptionResult{
		Total:    len(ids),
		Failures: make([]BatchSubscriptionFailure, 0),
	}

	for _, id := range ids {
		var err error
		switch input.Action {
		case BatchActionDelete:
			err = s.Delete(userID, id)
		case BatchActionMarkRenewed:
			_, err = s.MarkManualRenewed(userID, id)
		case BatchActionUpdate:
			_, err = s.Update(userID, id, UpdateSubscriptionInput{
				Status:             input.Status,
				RenewalMode:        input.RenewalMode,
				CategoryID:         input.CategoryID,
				PaymentMethodID:    input.PaymentMethodID,
				CategoryIDSet:      input.CategoryIDSet || input.CategoryID != nil,
				PaymentMethodIDSet: input.PaymentMethodIDSet || input.PaymentMethodID != nil,
			})
		}

		if err != nil {
			result.Failed++
			result.Failures = append(result.Failures, batchFailureFor(id, err))
			continue
		}
		result.Succeeded++
	}

	return result, nil
}
