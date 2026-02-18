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

	if !strings.HasPrefix(result, "200 OK") {
		t.Errorf("unexpected status line, got: %s", result)
	}
	if strings.Contains(result, "HTTP ") {
		t.Errorf("output should not contain 'HTTP ' prefix, got: %s", result)
	}
	if strings.Contains(result, "154ms") {
		t.Errorf("output should not contain duration, got: %s", result)
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

	if result != "204 No Content" {
		t.Errorf("expected '204 No Content', got: %q", result)
	}
	if strings.Contains(result, "empty body") {
		t.Errorf("output should not contain body placeholder, got: %s", result)
	}
}

func Test_FormatResponse_WithHeaders(t *testing.T) {
	resp := &client.Response{
		StatusCode: 200,
		StatusText: "OK",
		Headers: http.Header{
			"Content-Type": {"application/json"},
			"X-Request-Id": {"abc123"},
			"Date":         {"Mon, 01 Jan 2024 00:00:00 GMT"},
			"Server":       {"nginx"},
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

	if !strings.Contains(result, "[truncated: 100/5000 bytes]") {
		t.Errorf("expected truncation notice, got: %s", result)
	}
}

func Test_FormatResponse_TruncatedUnknownSize(t *testing.T) {
	resp := &client.Response{
		StatusCode:   200,
		StatusText:   "OK",
		Body:         []byte(strings.Repeat("x", 100)),
		Duration:     50 * time.Millisecond,
		Truncated:    true,
		OriginalSize: 0,
	}

	result := FormatResponse(resp, false)

	if !strings.Contains(result, "[truncated: 100 bytes shown]") {
		t.Errorf("expected truncation notice, got: %s", result)
	}
}
