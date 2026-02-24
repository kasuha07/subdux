package service

import (
	"errors"
	"fmt"
	"strings"
)

// Constants for template validation limits
const (
	MaxTemplateLength = 2000
)

var allowedTemplateVariables = map[string]struct{}{
	"SubscriptionName": {},
	"BillingDate":      {},
	"Amount":           {},
	"Currency":         {},
	"DaysUntil":        {},
	"Category":         {},
	"PaymentMethod":    {},
	"URL":              {},
	"Remark":           {},
	"UserEmail":        {},
}

type templateActionToken struct {
	start    int
	end      int
	variable string
}

// TemplateValidator provides security-focused template validation
// to prevent DoS attacks from malicious user-provided templates.
type TemplateValidator struct{}

// NewTemplateValidator creates a new TemplateValidator instance.
func NewTemplateValidator() *TemplateValidator {
	return &TemplateValidator{}
}

// ValidateTemplate validates a user-provided template string.
// It enforces length limits and allows only documented placeholders
// in the form {{.VariableName}}.
func (v *TemplateValidator) ValidateTemplate(tmplStr string) error {
	// Check template length
	if len(tmplStr) == 0 {
		return errors.New("template cannot be empty")
	}

	if len(tmplStr) > MaxTemplateLength {
		return fmt.Errorf("template length %d exceeds maximum %d", len(tmplStr), MaxTemplateLength)
	}

	if _, err := parseTemplateActions(tmplStr); err != nil {
		return err
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

func parseTemplateActions(tmplStr string) ([]templateActionToken, error) {
	tokens := make([]templateActionToken, 0)
	for idx := 0; idx < len(tmplStr); {
		openOffset := strings.Index(tmplStr[idx:], "{{")
		closeOffset := strings.Index(tmplStr[idx:], "}}")
		if closeOffset != -1 && (openOffset == -1 || closeOffset < openOffset) {
			return nil, errors.New("template parse error: unexpected closing delimiter \"}}\"")
		}
		if openOffset == -1 {
			break
		}

		open := idx + openOffset
		actionCloseOffset := strings.Index(tmplStr[open+2:], "}}")
		if actionCloseOffset == -1 {
			return nil, errors.New("template parse error: unclosed template action")
		}

		close := open + 2 + actionCloseOffset
		action := strings.TrimSpace(tmplStr[open+2 : close])
		varName, err := parseTemplateVariable(action)
		if err != nil {
			return nil, err
		}

		tokens = append(tokens, templateActionToken{
			start:    open,
			end:      close + 2,
			variable: varName,
		})
		idx = close + 2
	}

	return tokens, nil
}

func parseTemplateVariable(action string) (string, error) {
	if action == "" {
		return "", errors.New("template parse error: empty template action is not allowed")
	}
	if !strings.HasPrefix(action, ".") {
		return "", fmt.Errorf("template parse error: unsupported directive %q", action)
	}
	if strings.ContainsAny(action, " \t\r\n|()") {
		return "", fmt.Errorf("template parse error: unsupported directive %q", action)
	}

	varName := strings.TrimPrefix(action, ".")
	if varName == "" {
		return "", errors.New("template parse error: empty placeholder is not allowed")
	}
	if _, ok := allowedTemplateVariables[varName]; !ok {
		return "", fmt.Errorf("template parse error: unsupported placeholder %q", action)
	}
	return varName, nil
}
