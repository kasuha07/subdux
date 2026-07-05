package msgcode

import "strings"

// FromMessage derives a stable snake_case contract code from a human-readable
// backend message. It preserves slash-delimited alternatives such as
// "username/email" as "_or_" while collapsing other punctuation to
// underscores.
func FromMessage(message string) string {
	message = strings.TrimSpace(strings.ToLower(message))
	if message == "" {
		return "message"
	}

	runes := []rune(message)
	var b strings.Builder
	prevSeparator := true

	writeSeparator := func() {
		if prevSeparator || b.Len() == 0 {
			return
		}
		b.WriteByte('_')
		prevSeparator = true
	}

	for i, r := range runes {
		if isAlphaNumeric(r) {
			b.WriteRune(r)
			prevSeparator = false
			continue
		}

		prevAlnum := i > 0 && isAlphaNumeric(runes[i-1])
		nextAlnum := i+1 < len(runes) && isAlphaNumeric(runes[i+1])
		if r == '/' && prevAlnum && nextAlnum {
			writeSeparator()
			b.WriteString("or")
			prevSeparator = false
			writeSeparator()
			continue
		}

		writeSeparator()
	}

	code := strings.Trim(b.String(), "_")
	if code == "" {
		return "message"
	}
	return code
}

func isAlphaNumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}
