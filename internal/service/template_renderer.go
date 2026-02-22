package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"text/template"
	"time"
)

const (
	// MaxRenderedLength is the maximum allowed length for rendered template output
	MaxRenderedLength = 4000
	// RenderTimeout is the maximum time allowed for template rendering
	RenderTimeout = 100 * time.Millisecond
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
// It enforces a timeout to prevent DoS attacks and checks output length.
func (tr *TemplateRenderer) RenderTemplate(tmplStr string, data TemplateData) (string, error) {
	// Parse the template
	tmpl, err := template.New("notification").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), RenderTimeout)
	defer cancel()

	// Use goroutine + channel pattern for timeout enforcement
	type result struct {
		output string
		err    error
	}

	resultChan := make(chan result, 1)

	go func() {
		var buf bytes.Buffer
		err := tmpl.Execute(&buf, data)
		if err != nil {
			resultChan <- result{"", fmt.Errorf("failed to execute template: %w", err)}
			return
		}
		resultChan <- result{buf.String(), nil}
	}()

	select {
	case <-ctx.Done():
		return "", errors.New("template rendering timeout exceeded")
	case res := <-resultChan:
		if res.err != nil {
			return "", res.err
		}

		// Check rendered length to prevent expansion attacks
		if len(res.output) > MaxRenderedLength {
			return "", fmt.Errorf("rendered template exceeds maximum length of %d characters", MaxRenderedLength)
		}

		return res.output, nil
	}
}
