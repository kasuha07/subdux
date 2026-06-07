package api

import (
	"errors"
	"strings"

	"github.com/shiroha/subdux/internal/model"
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

func (h *MCPHandler) callCreateSubscription(userID uint, args map[string]interface{}) (*mcpToolResult, *mcpError) {
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

	sub, err := h.subscriptions.Create(userID, input)
	if err != nil {
		if isSubscriptionBadRequestError(err.Error()) {
			return mcpToolExecutionError(err.Error()), nil
		}
		return nil, internalMCPError(err)
	}
	return mcpStructuredResult(mapSubscriptionResponse(*sub)), nil
}

func (h *MCPHandler) callUpdateSubscription(userID uint, args map[string]interface{}) (*mcpToolResult, *mcpError) {
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

	sub, err := h.subscriptions.Update(userID, id, input)
	if err != nil {
		if isSubscriptionBadRequestError(err.Error()) || errors.Is(err, gorm.ErrRecordNotFound) {
			return mcpToolExecutionError(err.Error()), nil
		}
		return nil, internalMCPError(err)
	}
	return mcpStructuredResult(mapSubscriptionResponse(*sub)), nil
}

func (h *MCPHandler) callDeleteSubscription(userID uint, args map[string]interface{}) (*mcpToolResult, *mcpError) {
	id, err := readRequiredIDArg(args, "id")
	if err != nil {
		return nil, invalidMCPParams(err)
	}
	if err := h.subscriptions.Delete(userID, id); err != nil {
		if isSubscriptionBadRequestError(err.Error()) || errors.Is(err, gorm.ErrRecordNotFound) {
			return mcpToolExecutionError(err.Error()), nil
		}
		return nil, internalMCPError(err)
	}
	return mcpStructuredResult(map[string]interface{}{"deleted": true, "id": id}), nil
}

func (h *MCPHandler) callMarkSubscriptionRenewed(userID uint, args map[string]interface{}) (*mcpToolResult, *mcpError) {
	id, err := readRequiredIDArg(args, "id")
	if err != nil {
		return nil, invalidMCPParams(err)
	}
	sub, err := h.subscriptions.MarkManualRenewed(userID, id)
	if err != nil {
		if isSubscriptionBadRequestError(err.Error()) || errors.Is(err, gorm.ErrRecordNotFound) {
			return mcpToolExecutionError(err.Error()), nil
		}
		return nil, internalMCPError(err)
	}
	return mcpStructuredResult(mapSubscriptionResponse(*sub)), nil
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
