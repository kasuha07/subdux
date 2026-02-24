package service

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// MaxRenderedLength is the maximum allowed length for rendered template output
	MaxRenderedLength = 4000
)

// TemplateData holds all notification variables for template rendering
type TemplateData struct {
	SubscriptionName string
	BillingDate      string // Formatted as 2006-01-02
	Amount           float64
	Currency         string
	DaysUntil        int
	Category         string
	PaymentMethod    string
	URL              string
	Remark           string
	UserEmail        string
}

// TemplateRenderer renders templates with subscription data safely
type TemplateRenderer struct {
	validator *TemplateValidator
}

// NewTemplateRenderer creates a new TemplateRenderer with the given validator
func NewTemplateRenderer(validator *TemplateValidator) *TemplateRenderer {
	return &TemplateRenderer{
		validator: validator,
	}
}

// RenderTemplate renders a template string with the provided data.
// It allows placeholder-only actions (e.g. {{.SubscriptionName}}) and
// rejects unsupported directives/functions.
func (tr *TemplateRenderer) RenderTemplate(tmplStr string, data TemplateData) (string, error) {
	tokens, err := parseTemplateActions(tmplStr)
	if err != nil {
		return "", err
	}

	values := map[string]string{
		"SubscriptionName": data.SubscriptionName,
		"BillingDate":      data.BillingDate,
		"Amount":           strconv.FormatFloat(data.Amount, 'f', -1, 64),
		"Currency":         data.Currency,
		"DaysUntil":        strconv.Itoa(data.DaysUntil),
		"Category":         data.Category,
		"PaymentMethod":    data.PaymentMethod,
		"URL":              data.URL,
		"Remark":           data.Remark,
		"UserEmail":        data.UserEmail,
	}

	var builder strings.Builder
	last := 0
	for _, token := range tokens {
		builder.WriteString(tmplStr[last:token.start])
		builder.WriteString(values[token.variable])
		last = token.end
	}
	builder.WriteString(tmplStr[last:])
	output := builder.String()

	// Check rendered length to prevent expansion attacks
	if len(output) > MaxRenderedLength {
		return "", fmt.Errorf("rendered template exceeds maximum length of %d characters", MaxRenderedLength)
	}

	return output, nil
}
