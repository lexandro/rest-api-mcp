package tools

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/lexandro/rest-api-mcp/client"
)

func Test_FormatResponse_BasicSuccess(t *testing.T) {
	resp := &client.Response{
		StatusCode: 200,
		StatusText: "OK",
		Body:       []byte(`{"status":"ok"}`),
		Duration:   154 * time.Millisecond,
	}

	result := FormatResponse(resp, false)

	if !strings.HasPrefix(result, "HTTP 200 OK (154ms)") {
		t.Errorf("unexpected status line, got: %s", result)
	}
	if !strings.Contains(result, `{"status":"ok"}`) {
		t.Errorf("body not included, got: %s", result)
	}
}

func Test_FormatResponse_EmptyBody(t *testing.T) {
	resp := &client.Response{
		StatusCode: 204,
		StatusText: "No Content",
		Body:       []byte{},
		Duration:   50 * time.Millisecond,
	}

	result := FormatResponse(resp, false)

	if !strings.Contains(result, "(empty body)") {
		t.Errorf("expected '(empty body)', got: %s", result)
	}
}

func Test_FormatResponse_WithHeaders(t *testing.T) {
	resp := &client.Response{
		StatusCode: 200,
		StatusText: "OK",
		Headers: http.Header{
			"Content-Type":   {"application/json"},
			"X-Request-Id":   {"abc123"},
			"Date":           {"Mon, 01 Jan 2024 00:00:00 GMT"},
			"Server":         {"nginx"},
		},
		Body:     []byte("{}"),
		Duration: 100 * time.Millisecond,
	}

	result := FormatResponse(resp, true)

	if !strings.Contains(result, "Content-Type: application/json") {
		t.Errorf("expected Content-Type header, got: %s", result)
	}
	if !strings.Contains(result, "X-Request-Id: abc123") {
		t.Errorf("expected X-Request-Id header, got: %s", result)
	}
	if strings.Contains(result, "Date:") {
		t.Errorf("Date header should be filtered as noise, got: %s", result)
	}
	if strings.Contains(result, "Server:") {
		t.Errorf("Server header should be filtered as noise, got: %s", result)
	}
}

func Test_FormatResponse_Truncated(t *testing.T) {
	resp := &client.Response{
		StatusCode:   200,
		StatusText:   "OK",
		Body:         []byte(strings.Repeat("x", 100)),
		Duration:     50 * time.Millisecond,
		Truncated:    true,
		OriginalSize: 5000,
	}

	result := FormatResponse(resp, false)

	if !strings.Contains(result, "[truncated, showing 100 of 5000 bytes]") {
		t.Errorf("expected truncation notice, got: %s", result)
	}
}

func Test_FormatResponse_DurationFormat(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"milliseconds", 154 * time.Millisecond, "(154ms)"},
		{"seconds", 2300 * time.Millisecond, "(2.3s)"},
		{"one second", 1000 * time.Millisecond, "(1.0s)"},
		{"sub-second", 999 * time.Millisecond, "(999ms)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &client.Response{
				StatusCode: 200,
				StatusText: "OK",
				Body:       []byte("ok"),
				Duration:   tt.duration,
			}
			result := FormatResponse(resp, false)
			if !strings.Contains(result, tt.want) {
				t.Errorf("expected %q in output, got: %s", tt.want, result)
			}
		})
	}
}
