package service

import (
	"strings"
	"testing"
)

func TestTemplateValidatorAllowsDocumentedPlaceholdersOnly(t *testing.T) {
	validator := NewTemplateValidator()

	tmpl := "{{.SubscriptionName}} {{.BillingDate}} {{.Amount}} {{.Currency}} {{.DaysUntil}} {{.Category}} {{.PaymentMethod}} {{.URL}} {{.Remark}} {{.UserEmail}}"
	if err := validator.ValidateTemplate(tmpl); err != nil {
		t.Fatalf("ValidateTemplate() error = %v, want nil", err)
	}
}

func TestTemplateValidatorRejectsUnsupportedDirectiveAndPlaceholder(t *testing.T) {
	validator := NewTemplateValidator()

	tests := []struct {
		name    string
		tmpl    string
		wantErr string
	}{
		{
			name:    "reject function call directive",
			tmpl:    `{{printf "%s" .SubscriptionName}}`,
			wantErr: "unsupported directive",
		},
		{
			name:    "reject control directive",
			tmpl:    `{{if .SubscriptionName}}ok{{end}}`,
			wantErr: "unsupported directive",
		},
		{
			name:    "reject unsupported placeholder",
			tmpl:    `{{.UnknownField}}`,
			wantErr: "unsupported placeholder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateTemplate(tt.tmpl)
			if err == nil {
				t.Fatalf("ValidateTemplate() error = nil, want %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ValidateTemplate() error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestTemplateRendererRendersOnlyApprovedPlaceholders(t *testing.T) {
	renderer := NewTemplateRenderer(NewTemplateValidator())
	data := TemplateData{
		SubscriptionName: "Netflix",
		BillingDate:      "2026-03-15",
		Amount:           15.99,
		Currency:         "USD",
		DaysUntil:        3,
		Category:         "Entertainment",
		PaymentMethod:    "Card",
		URL:              "https://example.com",
		Remark:           "Family",
		UserEmail:        "user@example.com",
	}

	out, err := renderer.RenderTemplate("{{.SubscriptionName}}|{{.Amount}}|{{.DaysUntil}}|{{.UserEmail}}", data)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v, want nil", err)
	}

	const want = "Netflix|15.99|3|user@example.com"
	if out != want {
		t.Fatalf("RenderTemplate() = %q, want %q", out, want)
	}
}

func TestTemplateRendererRejectsDirectiveAndLargeOutput(t *testing.T) {
	renderer := NewTemplateRenderer(NewTemplateValidator())

	if _, err := renderer.RenderTemplate(`{{call .SubscriptionName}}`, TemplateData{}); err == nil {
		t.Fatal("RenderTemplate() error = nil, want directive validation error")
	}

	tooLongRemark := strings.Repeat("a", MaxRenderedLength+1)
	_, err := renderer.RenderTemplate("{{.Remark}}", TemplateData{Remark: tooLongRemark})
	if err == nil {
		t.Fatal("RenderTemplate() error = nil, want max length error")
	}
	if !strings.Contains(err.Error(), "maximum length") {
		t.Fatalf("RenderTemplate() error = %q, want maximum length error", err.Error())
	}
}
