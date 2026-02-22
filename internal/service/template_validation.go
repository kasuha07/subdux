package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"text/template"
)

// Constants for template validation limits
const (
	MaxTemplateLength = 2000
)

// TemplateValidator provides security-focused template validation
// to prevent DoS attacks from malicious user-provided templates.
type TemplateValidator struct{}

// NewTemplateValidator creates a new TemplateValidator instance.
func NewTemplateValidator() *TemplateValidator {
	return &TemplateValidator{}
}

// ValidateTemplate parses and validates a template string.
// It checks length limits, parses the template, and verifies it can be
// rendered with sample data within a timeout.
func (v *TemplateValidator) ValidateTemplate(tmplStr string) error {
	// Check template length
	if len(tmplStr) == 0 {
		return errors.New("template cannot be empty")
	}

	if len(tmplStr) > MaxTemplateLength {
		return fmt.Errorf("template length %d exceeds maximum %d", len(tmplStr), MaxTemplateLength)
	}

	// Parse template with no custom functions (security: empty FuncMap)
	tmpl, err := template.New("validation").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("template parse error: %w", err)
	}

	// Test render with sample data using timeout
	sampleData := map[string]interface{}{
		"name":   "test",
		"value":  123,
		"items":  []string{"a", "b", "c"},
		"active": true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), RenderTimeout)
	defer cancel()

	if err := renderWithTimeout(ctx, tmpl, sampleData); err != nil {
		return fmt.Errorf("template render error: %w", err)
	}

	return nil
}

// ValidateFormat checks if the format is one of the supported values.
// Supported formats: "plaintext", "markdown", "html"
func (v *TemplateValidator) ValidateFormat(format string) error {
	switch format {
	case "plaintext", "markdown", "html":
		return nil
	default:
		return fmt.Errorf("invalid format %q: must be 'plaintext', 'markdown', or 'html'", format)
	}
}

// renderWithTimeout renders a template with sample data within a context timeout.
// Uses goroutine + channel pattern to handle timeout safely.
func renderWithTimeout(ctx context.Context, tmpl *template.Template, data interface{}) error {
	type renderResult struct {
		output string
		err    error
	}

	resultChan := make(chan renderResult, 1)

	go func() {
		var buf bytes.Buffer
		err := tmpl.Execute(&buf, data)
		resultChan <- renderResult{output: buf.String(), err: err}
	}()

	select {
	case <-ctx.Done():
		return errors.New("template render timeout: possible infinite loop or expensive operation")
	case result := <-resultChan:
		if result.err != nil {
			return result.err
		}
		// Check rendered output length
		if len(result.output) > MaxRenderedLength {
			return fmt.Errorf("rendered output length %d exceeds maximum %d", len(result.output), MaxRenderedLength)
		}
		return nil
	}
}
