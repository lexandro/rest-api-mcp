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

	result := FormatResponse(resp, FormatOptions{})

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

	result := FormatResponse(resp, FormatOptions{})

	if result != "204 No Content" {
		t.Errorf("expected '204 No Content', got: %q", result)
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

	result := FormatResponse(resp, FormatOptions{IncludeHeaders: true})

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

	result := FormatResponse(resp, FormatOptions{})

	if !strings.Contains(result, "[truncated: 100/5000 bytes") {
		t.Errorf("expected truncation notice, got: %s", result)
	}
	if !strings.Contains(result, "saveTo") {
		t.Errorf("expected saveTo hint in truncation notice, got: %s", result)
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

	result := FormatResponse(resp, FormatOptions{})

	if !strings.Contains(result, "[truncated: 100 bytes shown") {
		t.Errorf("expected truncation notice, got: %s", result)
	}
}

func Test_FormatResponse_MinifiesPrettyJSON(t *testing.T) {
	resp := &client.Response{
		StatusCode:  200,
		StatusText:  "OK",
		ContentType: "application/json",
		Body:        []byte("{\n  \"name\": \"test\",\n  \"items\": [\n    1,\n    2\n  ]\n}"),
	}

	result := FormatResponse(resp, FormatOptions{})

	if !strings.Contains(result, `{"name":"test","items":[1,2]}`) {
		t.Errorf("expected minified JSON, got: %s", result)
	}
}

func Test_FormatResponse_NonJSONBodyUnchanged(t *testing.T) {
	body := "line one\n  indented line two"
	resp := &client.Response{
		StatusCode:  200,
		StatusText:  "OK",
		ContentType: "text/plain",
		Body:        []byte(body),
	}

	result := FormatResponse(resp, FormatOptions{})

	if !strings.Contains(result, body) {
		t.Errorf("expected plain text body unchanged, got: %s", result)
	}
}

func Test_FormatResponse_TruncatedJSONNotMinified(t *testing.T) {
	// Truncated JSON is invalid and must fall through unchanged.
	body := "{\n  \"name\": \"test\",\n  \"value"
	resp := &client.Response{
		StatusCode: 200,
		StatusText: "OK",
		Body:       []byte(body),
		Truncated:  true,
	}

	result := FormatResponse(resp, FormatOptions{})

	if !strings.Contains(result, body) {
		t.Errorf("expected invalid JSON body unchanged, got: %s", result)
	}
}

func Test_FormatResponse_JSONFilterMatch(t *testing.T) {
	resp := &client.Response{
		StatusCode:  200,
		StatusText:  "OK",
		ContentType: "application/json",
		Body:        []byte(`{"name":"widget","price":42,"internal":{"big":"blob"}}`),
	}

	result := FormatResponse(resp, FormatOptions{JSONFilter: "name"})

	if !strings.Contains(result, `"widget"`) {
		t.Errorf("expected filtered value, got: %s", result)
	}
	if strings.Contains(result, "blob") {
		t.Errorf("expected unfiltered fields to be omitted, got: %s", result)
	}
}

func Test_FormatResponse_JSONFilterMultipath(t *testing.T) {
	resp := &client.Response{
		StatusCode:  200,
		StatusText:  "OK",
		ContentType: "application/json",
		Body:        []byte(`{"name":"widget","price":42,"junk":"x"}`),
	}

	result := FormatResponse(resp, FormatOptions{JSONFilter: "{name,price}"})

	if !strings.Contains(result, `"name":"widget"`) || !strings.Contains(result, `"price":42`) {
		t.Errorf("expected multipath result, got: %s", result)
	}
	if strings.Contains(result, "junk") {
		t.Errorf("expected junk field to be omitted, got: %s", result)
	}
}

func Test_FormatResponse_JSONFilterNoMatch(t *testing.T) {
	resp := &client.Response{
		StatusCode:  200,
		StatusText:  "OK",
		ContentType: "application/json",
		Body:        []byte(`{"name":"widget"}`),
	}

	result := FormatResponse(resp, FormatOptions{JSONFilter: "nonexistent"})

	if !strings.Contains(result, "matched nothing") {
		t.Errorf("expected no-match notice, got: %s", result)
	}
	if strings.Contains(result, "widget") {
		t.Errorf("expected body to be suppressed on no-match, got: %s", result)
	}
}

func Test_FormatResponse_BinaryBody(t *testing.T) {
	resp := &client.Response{
		StatusCode:  200,
		StatusText:  "OK",
		ContentType: "image/png",
		Body:        []byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0x00, 0x01, 0xFF, 0xFE},
	}

	result := FormatResponse(resp, FormatOptions{})

	if !strings.Contains(result, "[binary: image/png, 9 bytes") {
		t.Errorf("expected binary summary, got: %s", result)
	}
	if strings.Contains(result, "\x89PNG") {
		t.Errorf("expected raw bytes to be suppressed, got: %s", result)
	}
}

func Test_FormatResponse_BinaryTruncatedShowsTotalSize(t *testing.T) {
	resp := &client.Response{
		StatusCode:   200,
		StatusText:   "OK",
		ContentType:  "application/octet-stream",
		Body:         []byte{0x00, 0x01, 0x02},
		Truncated:    true,
		OriginalSize: 99999,
	}

	result := FormatResponse(resp, FormatOptions{})

	if !strings.Contains(result, "99999 bytes") {
		t.Errorf("expected total size in binary summary, got: %s", result)
	}
	if strings.Contains(result, "[truncated") {
		t.Errorf("expected no separate truncation notice for binary, got: %s", result)
	}
}

func Test_FormatResponse_MislabeledJSONStillShown(t *testing.T) {
	// APIs sometimes serve JSON as application/octet-stream — byte sniffing must rescue it.
	resp := &client.Response{
		StatusCode:  200,
		StatusText:  "OK",
		ContentType: "application/octet-stream",
		Body:        []byte(`{"ok":true}`),
	}

	result := FormatResponse(resp, FormatOptions{})

	if !strings.Contains(result, `{"ok":true}`) {
		t.Errorf("expected sniffed text body to be shown, got: %s", result)
	}
}

func Test_FormatResponse_SavedToFile(t *testing.T) {
	resp := &client.Response{
		StatusCode:  200,
		StatusText:  "OK",
		ContentType: "application/json; charset=utf-8",
		SavedPath:   "C:\\temp\\out.json",
		SavedSize:   245891,
	}

	result := FormatResponse(resp, FormatOptions{})

	if !strings.Contains(result, "[saved to C:\\temp\\out.json: 245891 bytes, application/json]") {
		t.Errorf("expected saved-file summary, got: %s", result)
	}
}
