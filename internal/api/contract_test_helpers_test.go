package api

import (
	"strings"

	"github.com/kasuha07/subdux/internal/pkg/msgcode"
)

func hasErrorCode(body string, code string) bool {
	return strings.Contains(body, `"`+code+`"`)
}

func hasErrorCodeForMessage(body string, message string) bool {
	return hasErrorCode(body, msgcode.FromMessage(message))
}
