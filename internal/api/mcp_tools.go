package api

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/service"
	"gorm.io/gorm"
)

func (h *MCPHandler) callListSubscriptions(userID uint) (*mcpToolResult, *mcpError) {
	subs, err := h.subscriptions.List(userID)
	if err != nil {
		return nil, internalMCPError(err)
	}
	return mcpStructuredResult(map[string]interface{}{
		"subscriptions": mapSubscriptionResponses(subs),
	}), nil
}

func (h *MCPHandler) callSearchSubscriptions(userID uint, args map[string]interface{}) (*mcpToolResult, *mcpError) {
	filters, err := readMCPSubscriptionSearchFilters(args)
	if err != nil {
		return nil, invalidMCPParams(err)
	}

	subs, err := h.subscriptions.List(userID)
	if err != nil {
		return nil, internalMCPError(err)
	}

	categoryLabels, err := h.mcpCategoryLabels(userID, filters)
	if err != nil {
		return nil, internalMCPError(err)
	}

	matches := make([]model.Subscription, 0, len(subs))
	for _, sub := range subs {
		if matchesMCPSubscriptionSearch(sub, filters, categoryLabels) {
			matches = append(matches, sub)
		}
	}

	totalMatches := len(matches)
	if filters.Limit > 0 && totalMatches > filters.Limit {
		matches = matches[:filters.Limit]
	}

	return mcpStructuredResult(map[string]interface{}{
		"subscriptions": mapSubscriptionResponses(matches),
		"count":         len(matches),
		"total_matches": totalMatches,
		"limit":         filters.Limit,
	}), nil
}

func (h *MCPHandler) mcpCategoryLabels(userID uint, filters mcpSubscriptionSearchFilters) (map[uint]string, error) {
	if filters.Query == "" && filters.Category == "" {
		return nil, nil
	}

	categories, err := h.categories.List(userID)
	if err != nil {
		return nil, err
	}

	labels := make(map[uint]string, len(categories))
	for _, category := range categories {
		labels[category.ID] = category.Name
	}
	return labels, nil
}

func (h *MCPHandler) callGetSubscription(userID uint, args map[string]interface{}) (*mcpToolResult, *mcpError) {
	id, err := readRequiredIDArg(args, "id")
	if err != nil {
		return nil, invalidMCPParams(err)
	}
	sub, err := h.subscriptions.GetByID(userID, id)
	if err != nil {
		return mcpToolExecutionError("subscription not found"), nil
	}
	return mcpStructuredResult(mapSubscriptionResponse(*sub)), nil
}

func (h *MCPHandler) callCreateSubscription(principal *mcpPrincipal, args map[string]interface{}) (*mcpToolResult, *mcpError) {
	userID := principal.UserID
	if err := validateSubscriptionWriteArgTypes(args); err != nil {
		return nil, invalidMCPParams(err)
	}
	input := createSubscriptionInputFromMCPArgs(args)
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return nil, invalidMCPParams(errors.New("name is required"))
	}
	if input.Amount < 0 {
		return nil, invalidMCPParams(errors.New("amount must not be negative"))
	}
	if !validateSubscriptionIcon(input.Icon) {
		return nil, invalidMCPParams(errors.New("invalid icon value"))
	}

	auditEnabled, err := h.audit.IsEnabled()
	if err != nil {
		return nil, internalMCPError(err)
	}
	if !auditEnabled {
		sub, err := h.subscriptions.Create(userID, input)
		if err != nil {
			if isSubscriptionBadRequestError(err.Error()) {
				return mcpToolExecutionError(err.Error()), nil
			}
			return nil, internalMCPError(err)
		}
		return mcpStructuredResult(mapSubscriptionResponse(*sub)), nil
	}

	start := time.Now()
	var sub model.Subscription
	err = h.subscriptions.DB.Transaction(func(tx *gorm.DB) error {
		created, err := service.NewSubscriptionService(tx).Create(userID, input)
		if err != nil {
			return err
		}
		sub = *created
		_, err = service.NewAuditService(tx).Create(service.CreateAuditEventInput{
			UserID:              userID,
			KeyID:               principal.KeyID,
			KeyKind:             principal.KeyKind,
			ScopeUsed:           service.APIKeyScopeWrite,
			Transport:           service.AuditTransportMCP,
			ToolName:            "create_subscription",
			ResourceType:        service.AuditResourceSubscription,
			ResourceID:          fmt.Sprint(sub.ID),
			Action:              "create",
			Status:              service.AuditStatusSuccess,
			LatencyMS:           time.Since(start).Milliseconds(),
			ClientName:          principal.Request.ClientName,
			ClientVersion:       principal.Request.ClientVersion,
			RequestID:           principal.Request.RequestID,
			RequestArgsRedacted: args,
			AfterSnapshot:       auditSubscriptionSnapshot(sub, nil),
		})
		return err
	})
	if err != nil {
		if isSubscriptionBadRequestError(err.Error()) {
			return mcpToolExecutionError(err.Error()), nil
		}
		return nil, internalMCPError(err)
	}
	return mcpStructuredResult(mapSubscriptionResponse(sub)), nil
}

func (h *MCPHandler) callUpdateSubscription(principal *mcpPrincipal, args map[string]interface{}) (*mcpToolResult, *mcpError) {
	userID := principal.UserID
	id, err := readRequiredIDArg(args, "id")
	if err != nil {
		return nil, invalidMCPParams(err)
	}
	input, err := updateSubscriptionInputFromMCPArgs(args)
	if err != nil {
		return nil, invalidMCPParams(err)
	}
	if input.Amount != nil && *input.Amount < 0 {
		return nil, invalidMCPParams(errors.New("amount must not be negative"))
	}
	if input.Icon != nil && !validateSubscriptionIcon(*input.Icon) {
		return nil, invalidMCPParams(errors.New("invalid icon value"))
	}

	auditEnabled, err := h.audit.IsEnabled()
	if err != nil {
		return nil, internalMCPError(err)
	}
	if !auditEnabled {
		sub, err := h.subscriptions.Update(userID, id, input)
		if err != nil {
			if isSubscriptionBadRequestError(err.Error()) || errors.Is(err, gorm.ErrRecordNotFound) {
				return mcpToolExecutionError(err.Error()), nil
			}
			return nil, internalMCPError(err)
		}
		return mcpStructuredResult(mapSubscriptionResponse(*sub)), nil
	}

	start := time.Now()
	var before model.Subscription
	var sub model.Subscription
	err = h.subscriptions.DB.Transaction(func(tx *gorm.DB) error {
		txService := service.NewSubscriptionService(tx)
		existing, err := txService.GetByID(userID, id)
		if err != nil {
			return err
		}
		before = *existing
		updated, err := txService.Update(userID, id, input)
		if err != nil {
			return err
		}
		sub = *updated
		changedFields := auditChangedSubscriptionFields(before, sub)
		_, err = service.NewAuditService(tx).Create(service.CreateAuditEventInput{
			UserID:              userID,
			KeyID:               principal.KeyID,
			KeyKind:             principal.KeyKind,
			ScopeUsed:           service.APIKeyScopeWrite,
			Transport:           service.AuditTransportMCP,
			ToolName:            "update_subscription",
			ResourceType:        service.AuditResourceSubscription,
			ResourceID:          fmt.Sprint(id),
			Action:              "update",
			Status:              service.AuditStatusSuccess,
			LatencyMS:           time.Since(start).Milliseconds(),
			ClientName:          principal.Request.ClientName,
			ClientVersion:       principal.Request.ClientVersion,
			RequestID:           principal.Request.RequestID,
			RequestArgsRedacted: args,
			BeforeSnapshot:      auditSubscriptionSnapshot(before, nil),
			AfterSnapshot:       auditSubscriptionSnapshot(sub, changedFields),
		})
		return err
	})
	if err != nil {
		if isSubscriptionBadRequestError(err.Error()) || errors.Is(err, gorm.ErrRecordNotFound) {
			return mcpToolExecutionError(err.Error()), nil
		}
		return nil, internalMCPError(err)
	}
	return mcpStructuredResult(mapSubscriptionResponse(sub)), nil
}

func (h *MCPHandler) callDeleteSubscription(principal *mcpPrincipal, args map[string]interface{}) (*mcpToolResult, *mcpError) {
	userID := principal.UserID
	id, err := readRequiredIDArg(args, "id")
	if err != nil {
		return nil, invalidMCPParams(err)
	}
	auditEnabled, err := h.audit.IsEnabled()
	if err != nil {
		return nil, internalMCPError(err)
	}
	if !auditEnabled {
		if err := h.subscriptions.Delete(userID, id); err != nil {
			if isSubscriptionBadRequestError(err.Error()) || errors.Is(err, gorm.ErrRecordNotFound) {
				return mcpToolExecutionError(err.Error()), nil
			}
			return nil, internalMCPError(err)
		}
		return mcpStructuredResult(map[string]interface{}{"deleted": true, "id": id}), nil
	}

	start := time.Now()
	var before model.Subscription
	var deleted *model.Subscription
	err = h.subscriptions.DB.Transaction(func(tx *gorm.DB) error {
		txService := service.NewSubscriptionService(tx)
		existing, err := txService.GetByID(userID, id)
		if err != nil {
			return err
		}
		before = *existing
		deleted, err = txService.DeleteRecord(userID, id)
		if err != nil {
			return err
		}
		_, err = service.NewAuditService(tx).Create(service.CreateAuditEventInput{
			UserID:              userID,
			KeyID:               principal.KeyID,
			KeyKind:             principal.KeyKind,
			ScopeUsed:           service.APIKeyScopeWrite,
			Transport:           service.AuditTransportMCP,
			ToolName:            "delete_subscription",
			ResourceType:        service.AuditResourceSubscription,
			ResourceID:          fmt.Sprint(id),
			Action:              "delete",
			Status:              service.AuditStatusSuccess,
			LatencyMS:           time.Since(start).Milliseconds(),
			ClientName:          principal.Request.ClientName,
			ClientVersion:       principal.Request.ClientVersion,
			RequestID:           principal.Request.RequestID,
			RequestArgsRedacted: args,
			BeforeSnapshot:      auditSubscriptionSnapshot(before, nil),
		})
		return err
	})
	if err != nil {
		if isSubscriptionBadRequestError(err.Error()) || errors.Is(err, gorm.ErrRecordNotFound) {
			return mcpToolExecutionError(err.Error()), nil
		}
		return nil, internalMCPError(err)
	}
	if deleted != nil {
		h.subscriptions.CleanupDeletedSubscriptionResources(*deleted)
	}
	return mcpStructuredResult(map[string]interface{}{"deleted": true, "id": id}), nil
}

func (h *MCPHandler) callMarkSubscriptionRenewed(principal *mcpPrincipal, args map[string]interface{}) (*mcpToolResult, *mcpError) {
	userID := principal.UserID
	id, err := readRequiredIDArg(args, "id")
	if err != nil {
		return nil, invalidMCPParams(err)
	}
	auditEnabled, err := h.audit.IsEnabled()
	if err != nil {
		return nil, internalMCPError(err)
	}
	if !auditEnabled {
		sub, err := h.subscriptions.MarkManualRenewed(userID, id)
		if err != nil {
			if isSubscriptionBadRequestError(err.Error()) || errors.Is(err, gorm.ErrRecordNotFound) {
				return mcpToolExecutionError(err.Error()), nil
			}
			return nil, internalMCPError(err)
		}
		return mcpStructuredResult(mapSubscriptionResponse(*sub)), nil
	}

	start := time.Now()
	var before model.Subscription
	var sub model.Subscription
	err = h.subscriptions.DB.Transaction(func(tx *gorm.DB) error {
		txService := service.NewSubscriptionService(tx)
		existing, err := txService.GetByID(userID, id)
		if err != nil {
			return err
		}
		before = *existing
		updated, err := txService.MarkManualRenewed(userID, id)
		if err != nil {
			return err
		}
		sub = *updated
		changedFields := auditChangedSubscriptionFields(before, sub)
		_, err = service.NewAuditService(tx).Create(service.CreateAuditEventInput{
			UserID:              userID,
			KeyID:               principal.KeyID,
			KeyKind:             principal.KeyKind,
			ScopeUsed:           service.APIKeyScopeWrite,
			Transport:           service.AuditTransportMCP,
			ToolName:            "mark_subscription_renewed",
			ResourceType:        service.AuditResourceSubscription,
			ResourceID:          fmt.Sprint(id),
			Action:              "mark_renewed",
			Status:              service.AuditStatusSuccess,
			LatencyMS:           time.Since(start).Milliseconds(),
			ClientName:          principal.Request.ClientName,
			ClientVersion:       principal.Request.ClientVersion,
			RequestID:           principal.Request.RequestID,
			RequestArgsRedacted: args,
			BeforeSnapshot:      auditSubscriptionSnapshot(before, nil),
			AfterSnapshot:       auditSubscriptionSnapshot(sub, changedFields),
		})
		return err
	})
	if err != nil {
		if isSubscriptionBadRequestError(err.Error()) || errors.Is(err, gorm.ErrRecordNotFound) {
			return mcpToolExecutionError(err.Error()), nil
		}
		return nil, internalMCPError(err)
	}
	return mcpStructuredResult(mapSubscriptionResponse(sub)), nil
}

func auditSubscriptionSnapshot(sub model.Subscription, changedFields []string) map[string]interface{} {
	snapshot := map[string]interface{}{
		"id":                sub.ID,
		"name":              sub.Name,
		"amount":            sub.Amount,
		"currency":          sub.Currency,
		"status":            sub.Status,
		"renewal_mode":      sub.RenewalMode,
		"next_billing_date": formatDateOnly(sub.NextBillingDate),
		"category_id":       sub.CategoryID,
		"payment_method_id": sub.PaymentMethodID,
	}
	if changedFields != nil {
		snapshot["changed_fields"] = changedFields
	}
	return snapshot
}

func auditChangedSubscriptionFields(before, after model.Subscription) []string {
	changed := make([]string, 0, 8)
	if before.Name != after.Name {
		changed = append(changed, "name")
	}
	if before.Amount != after.Amount {
		changed = append(changed, "amount")
	}
	if before.Currency != after.Currency {
		changed = append(changed, "currency")
	}
	if before.Status != after.Status {
		changed = append(changed, "status")
	}
	if before.RenewalMode != after.RenewalMode {
		changed = append(changed, "renewal_mode")
	}
	if !dateStringPtrEqual(formatDateOnly(before.NextBillingDate), formatDateOnly(after.NextBillingDate)) {
		changed = append(changed, "next_billing_date")
	}
	if !uintPtrEqualAPI(before.CategoryID, after.CategoryID) {
		changed = append(changed, "category_id")
	}
	if !uintPtrEqualAPI(before.PaymentMethodID, after.PaymentMethodID) {
		changed = append(changed, "payment_method_id")
	}
	return changed
}

func dateStringPtrEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func uintPtrEqualAPI(a, b *uint) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func (h *MCPHandler) callDashboardSummary(userID uint, args map[string]interface{}) (*mcpToolResult, *mcpError) {
	if err := validateMCPArgTypes(args, []mcpArgSpec{{Key: "currency", Type: "string"}}); err != nil {
		return nil, invalidMCPParams(err)
	}
	currency, _ := readStringArg(args, "currency")
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency != "" {
		if err := h.validateUserCurrency(userID, currency); err != nil {
			return nil, invalidMCPParams(err)
		}
	} else {
		pref, _ := h.exchangeRates.GetUserPreference(userID)
		currency = pref.PreferredCurrency
	}

	summary, err := h.subscriptions.GetDashboardSummary(userID, currency, h.exchangeRates)
	if err != nil {
		return nil, internalMCPError(err)
	}
	return mcpStructuredResult(summary), nil
}

func (h *MCPHandler) validateUserCurrency(userID uint, code string) error {
	currencies, err := h.currencies.List(userID)
	if err != nil {
		return err
	}
	for _, currency := range currencies {
		if strings.EqualFold(currency.Code, code) {
			return nil
		}
	}
	return errors.New("currency not found")
}

func (h *MCPHandler) callListCategories(userID uint) (*mcpToolResult, *mcpError) {
	categories, err := h.categories.List(userID)
	if err != nil {
		return nil, internalMCPError(err)
	}
	return mcpStructuredResult(map[string]interface{}{
		"categories": mapCategoryResponses(categories),
	}), nil
}

func (h *MCPHandler) callListPaymentMethods(userID uint) (*mcpToolResult, *mcpError) {
	methods, err := h.paymentMethods.List(userID)
	if err != nil {
		return nil, internalMCPError(err)
	}
	return mcpStructuredResult(map[string]interface{}{
		"payment_methods": mapPaymentMethodResponses(methods),
	}), nil
}
