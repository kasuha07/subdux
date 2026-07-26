package reauth

import (
	"fmt"
)

// TicketBinding narrows a reauth ticket to one mutable backup destination
// revision. It is intentionally separate from the operation name so the
// operation and resource checks cannot drift apart at a call site.
type TicketBinding struct {
	DestinationID       uint   `json:"destination_id"`
	DestinationRevision uint64 `json:"destination_revision"`
}

// ValidateTicketBinding enforces which reauth operations may carry a resource
// binding. Destination update/delete tickets must name both the destination
// and the revision that the user reviewed; all other operations remain
// unbound.
func ValidateTicketBinding(operation string, binding *TicketBinding) error {
	if !IsValidReauthOperation(operation) {
		return ErrInvalidReauthOperation
	}

	requiresBinding := operation == ReauthOperationBackupDestinationUpdate ||
		operation == ReauthOperationBackupDestinationDelete
	if requiresBinding {
		if binding == nil || binding.DestinationID == 0 || binding.DestinationRevision == 0 {
			return ErrReauthRequired
		}
		return nil
	}

	if binding != nil {
		return ErrReauthRequired
	}
	return nil
}

// scopedOperation is an opaque operation value used only by the passkey and
// OIDC challenge stores. Those stores already bind an in-progress challenge to
// an operation; including the same target tuple there prevents a challenge
// started for one destination revision from being finished for another.
func scopedOperation(operation string, binding *TicketBinding) (string, error) {
	if err := ValidateTicketBinding(operation, binding); err != nil {
		return "", err
	}
	if binding == nil {
		return operation, nil
	}
	return fmt.Sprintf("%s#destination=%d#revision=%d", operation, binding.DestinationID, binding.DestinationRevision), nil
}

func cloneTicketBinding(binding *TicketBinding) *TicketBinding {
	if binding == nil {
		return nil
	}
	copy := *binding
	return &copy
}

func sameTicketBinding(left, right *TicketBinding) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.DestinationID == right.DestinationID &&
		left.DestinationRevision == right.DestinationRevision
}
