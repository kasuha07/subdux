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
		{
			name:        "valid pushover minimal config",
			channelType: "pushover",
			config:      `{"token":"a1b2c3d4e5f6","user":"u1v2w3x4y5z6"}`,
		},
		{
			name:        "valid pushover with endpoint",
			channelType: "pushover",
			config:      `{"token":"a1b2c3d4e5f6","user":"u1v2w3x4y5z6","endpoint":"https://api.pushover.net/1/messages.json"}`,
		},
		{
			name:        "reject pushover missing token",
			channelType: "pushover",
			config:      `{"user":"u1v2w3x4y5z6"}`,
			wantErr:     "pushover channel requires token",
		},
		{
			name:        "reject pushover missing user",
			channelType: "pushover",
			config:      `{"token":"a1b2c3d4e5f6"}`,
			wantErr:     "pushover channel requires user",
		},
		{
			name:        "reject pushover invalid endpoint",
			channelType: "pushover",
			config:      `{"token":"a1b2c3d4e5f6","user":"u1v2w3x4y5z6","endpoint":"ftp://example.com"}`,
			wantErr:     "pushover endpoint must start with http:// or https://",
		},
		{
			name:        "valid napcat private message",
			channelType: "napcat",
			config:      `{"url":"http://127.0.0.1:3000","message_type":"private","user_id":"123456789"}`,
		},
		{
			name:        "valid napcat group message",
			channelType: "napcat",
			config:      `{"url":"http://127.0.0.1:3000","message_type":"group","group_id":"987654321"}`,
		},
		{
			name:        "valid napcat with access token",
			channelType: "napcat",
			config:      `{"url":"https://napcat.example.com","access_token":"mytoken","message_type":"private","user_id":"123"}`,
		},
		{
			name:        "valid napcat defaults to private",
			channelType: "napcat",
			config:      `{"url":"http://127.0.0.1:3000","user_id":"123456789"}`,
		},
		{
			name:        "reject napcat missing url",
			channelType: "napcat",
			config:      `{"message_type":"private","user_id":"123"}`,
			wantErr:     "napcat channel requires url",
		},
		{
			name:        "reject napcat invalid url scheme",
			channelType: "napcat",
			config:      `{"url":"ftp://example.com","user_id":"123"}`,
			wantErr:     "napcat url must start with http:// or https://",
		},
		{
			name:        "reject napcat invalid message type",
			channelType: "napcat",
			config:      `{"url":"http://127.0.0.1:3000","message_type":"channel","user_id":"123"}`,
			wantErr:     "napcat message_type must be private or group",
		},
		{
			name:        "reject napcat private missing user_id",
			channelType: "napcat",
			config:      `{"url":"http://127.0.0.1:3000","message_type":"private"}`,
			wantErr:     "napcat channel requires user_id for private messages",
		},
		{
			name:        "reject napcat group missing group_id",
			channelType: "napcat",
			config:      `{"url":"http://127.0.0.1:3000","message_type":"group"}`,
			wantErr:     "napcat channel requires group_id for group messages",
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
