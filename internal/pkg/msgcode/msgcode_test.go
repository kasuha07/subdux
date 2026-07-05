package msgcode

import (
	"encoding/json"
	"os"
	"testing"
)

type derivationCase struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

func TestFromMessageFixtures(t *testing.T) {
	raw, err := os.ReadFile("testdata/from_message_cases.json")
	if err != nil {
		t.Fatalf("ReadFile(testdata/from_message_cases.json) error = %v", err)
	}

	var cases []derivationCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("json.Unmarshal(fixtures) error = %v", err)
	}

	for _, tc := range cases {
		t.Run(tc.Code, func(t *testing.T) {
			if got := FromMessage(tc.Message); got != tc.Code {
				t.Fatalf("FromMessage(%q) = %q, want %q", tc.Message, got, tc.Code)
			}
		})
	}
}
