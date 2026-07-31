package notification

import (
	"errors"
	"math"
	"testing"

	"github.com/kasuha07/subdux/internal/service/money"
)

func TestTemplateRendererFormatsAmountWithCurrencyMinorUnit(t *testing.T) {
	renderer := NewTemplateRenderer(NewTemplateValidator())

	tests := []struct {
		name     string
		amount   float64
		currency string
		want     string
	}{
		{name: "two decimal currency keeps both decimals", amount: 9.99, currency: "USD", want: "9.99"},
		{name: "two decimal currency pads a whole amount", amount: 15, currency: "EUR", want: "15.00"},
		{name: "zero decimal currency renders an integer", amount: 1234.6, currency: "JPY", want: "1235"},
		{name: "three decimal currency keeps three decimals", amount: 1.2, currency: "KWD", want: "1.200"},
		{name: "unknown currency falls back to two decimals", amount: 5, currency: "", want: "5.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := renderer.RenderTemplate("{{.Amount}}", TemplateData{
				Amount:   tt.amount,
				Currency: tt.currency,
			})
			if err != nil {
				t.Fatalf("RenderTemplate() error = %v, want nil", err)
			}
			if out != tt.want {
				t.Fatalf("RenderTemplate() = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestTemplateRendererRejectsUnsafeAmount(t *testing.T) {
	renderer := NewTemplateRenderer(NewTemplateValidator())
	for _, amount := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), math.MaxFloat64} {
		out, err := renderer.RenderTemplate("{{.Amount}}", TemplateData{Amount: amount, Currency: "USD"})
		if !errors.Is(err, money.ErrUnsafeFormat) || out != "" {
			t.Fatalf("RenderTemplate(amount=%v) = %q, %v; want empty ErrUnsafeFormat", amount, out, err)
		}
	}
}
