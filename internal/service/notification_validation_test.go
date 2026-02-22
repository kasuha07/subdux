package service

import (
	"strings"
	"testing"
)

func TestValidateChannelConfigWebhook(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
		{
			name:   "valid default method",
			config: `{"url":"https://example.com/webhook"}`,
		},
		{
			name:   "valid get method with custom headers",
			config: `{"url":"https://example.com/webhook","method":"GET","headers":{"X-Token":"abc123","X-Env":"prod"}}`,
		},
		{
			name:    "reject get method with secret",
			config:  `{"url":"https://example.com/webhook","method":"GET","secret":"my-secret"}`,
			wantErr: "webhook secret is not supported when method is GET",
		},
		{
			name:    "reject unsupported method",
			config:  `{"url":"https://example.com/webhook","method":"PATCH"}`,
			wantErr: "webhook method must be one of: GET, POST, PUT",
		},
		{
			name:    "reject invalid header json type",
			config:  `{"url":"https://example.com/webhook","headers":{"X-Token":123}}`,
			wantErr: "invalid webhook config format",
		},
		{
			name:    "reject header with leading space",
			config:  `{"url":"https://example.com/webhook","headers":{" X-Token":"abc123"}}`,
			wantErr: "webhook header name must not contain leading or trailing spaces",
		},
		{
			name:    "reject header with invalid characters",
			config:  `{"url":"https://example.com/webhook","headers":{"X:Token":"abc123"}}`,
			wantErr: "webhook header name contains invalid characters",
		},
		{
			name:    "reject header name with newline",
			config:  `{"url":"https://example.com/webhook","headers":{"X-Token\nTest":"abc123"}}`,
			wantErr: "webhook header name contains invalid characters",
		},
		{
			name:    "reject header value with newline",
			config:  `{"url":"https://example.com/webhook","headers":{"X-Token":"abc123\n"}}`,
			wantErr: "webhook header value contains invalid newline characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateChannelConfig("webhook", tt.config)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateChannelConfig() error = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("validateChannelConfig() error = nil, want %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validateChannelConfig() error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestValidateChannelConfigPushChannels(t *testing.T) {
	tests := []struct {
		name        string
		channelType string
		config      string
		wantErr     string
	}{
		{
			name:        "valid pushdeer minimal config",
			channelType: "pushdeer",
			config:      `{"push_key":"PDU1TTnEZKlRVODU9GkFmtHwaIraV5twUPQbA"}`,
		},
		{
			name:        "valid pushdeer with custom server",
			channelType: "pushdeer",
			config:      `{"push_key":"PDU1TTnEZKlRVODU9GkFmtHwaIraV5twUPQbA","server_url":"https://api2.pushdeer.com"}`,
		},
		{
			name:        "reject pushdeer missing push key",
			channelType: "pushdeer",
			config:      `{"server_url":"https://api2.pushdeer.com"}`,
			wantErr:     "pushdeer channel requires push_key",
		},
		{
			name:        "reject pushdeer invalid server url",
			channelType: "pushdeer",
			config:      `{"push_key":"k","server_url":"ftp://example.com"}`,
			wantErr:     "pushdeer server_url must start with http:// or https://",
		},
		{
			name:        "valid pushplus minimal config",
			channelType: "pushplus",
			config:      `{"token":"5709c7da5c1c4a8b9d2f8e1a3b5c9d2f"}`,
		},
		{
			name:        "valid pushplus with endpoint",
			channelType: "pushplus",
			config:      `{"token":"5709c7da5c1c4a8b9d2f8e1a3b5c9d2f","endpoint":"https://www.pushplus.plus/send"}`,
		},
		{
			name:        "reject pushplus missing token",
			channelType: "pushplus",
			config:      `{"endpoint":"https://www.pushplus.plus/send"}`,
			wantErr:     "pushplus channel requires token",
		},
		{
			name:        "reject pushplus invalid endpoint",
			channelType: "pushplus",
			config:      `{"token":"t","endpoint":"ftp://example.com"}`,
			wantErr:     "pushplus endpoint must start with http:// or https://",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateChannelConfig(tt.channelType, tt.config)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateChannelConfig() error = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("validateChannelConfig() error = nil, want %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validateChannelConfig() error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}
