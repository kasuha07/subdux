package service

import (
	"strings"
	"testing"
)

func TestBuildServerChanEndpoint(t *testing.T) {
	tests := []struct {
		name    string
		sendKey string
		want    string
	}{
		{
			name:    "uid based endpoint for sc3 sendkey",
			sendKey: "sctp12345tabcdef",
			want:    "https://12345.push.ft07.com/send/sctp12345tabcdef.send",
		},
		{
			name:    "fallback endpoint for non uid key",
			sendKey: "my-send-key",
			want:    "https://sctapi.ftqq.com/my-send-key.send",
		},
		{
			name:    "trims spaces",
			sendKey: "  sctp42txyz  ",
			want:    "https://42.push.ft07.com/send/sctp42txyz.send",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildServerChanEndpoint(tt.sendKey)
			if got != tt.want {
				t.Fatalf("buildServerChanEndpoint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateServerChanBusinessResponse(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			name: "success code 0",
			body: `{"code":0,"message":"success"}`,
		},
		{
			name:    "non zero code from message",
			body:    `{"code":40001,"message":"invalid sendkey"}`,
			wantErr: "serverchan business error code 40001: invalid sendkey",
		},
		{
			name:    "non zero code from info fallback",
			body:    `{"code":40002,"info":"quota exceeded"}`,
			wantErr: "serverchan business error code 40002: quota exceeded",
		},
		{
			name:    "invalid json",
			body:    `not json`,
			wantErr: "invalid serverchan response",
		},
		{
			name:    "non zero code from error fallback",
			body:    `{"code":10003,"error":"sendkey not found"}`,
			wantErr: "serverchan business error code 10003: sendkey not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServerChanBusinessResponse([]byte(tt.body))
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateServerChanBusinessResponse() error = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("validateServerChanBusinessResponse() error = nil, want %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validateServerChanBusinessResponse() error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}
