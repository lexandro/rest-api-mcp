package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/lexandro/rest-api-mcp/client"
)

func newTestClient(serverURL string) *client.Client {
	return client.NewClient(client.Config{
		Timeout:         5 * time.Second,
		MaxResponseSize: 1024,
	})
}

func Test_HttpRequestHandler_ValidGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	handler := makeHandler(c)

	result, _, err := handler(context.Background(), nil, HttpRequestInput{
		Method: "GET",
		URL:    server.URL,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %+v", result.Content)
	}

	text := extractText(result)
	if !strings.Contains(text, "200 OK") {
		t.Errorf("expected 200 OK in output, got: %s", text)
	}
	if !strings.Contains(text, `{"status":"ok"}`) {
		t.Errorf("expected body in output, got: %s", text)
	}
}

func Test_HttpRequestHandler_MissingMethod(t *testing.T) {
	c := client.NewClient(client.Config{
		Timeout:         5 * time.Second,
		MaxResponseSize: 1024,
	})
	handler := makeHandler(c)

	result, _, err := handler(context.Background(), nil, HttpRequestInput{
		URL: "http://example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError to be true for missing method")
	}
	text := extractText(result)
	if !strings.Contains(text, "method is required") {
		t.Errorf("expected 'method is required' error, got: %s", text)
	}
}

func Test_HttpRequestHandler_MissingURL(t *testing.T) {
	c := client.NewClient(client.Config{
		Timeout:         5 * time.Second,
		MaxResponseSize: 1024,
	})
	handler := makeHandler(c)

	result, _, err := handler(context.Background(), nil, HttpRequestInput{
		Method: "GET",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError to be true for missing URL")
	}
	text := extractText(result)
	if !strings.Contains(text, "url is required") {
		t.Errorf("expected 'url is required' error, got: %s", text)
	}
}

func Test_HttpRequestHandler_InvalidMethod(t *testing.T) {
	c := client.NewClient(client.Config{
		Timeout:         5 * time.Second,
		MaxResponseSize: 1024,
	})
	handler := makeHandler(c)

	result, _, err := handler(context.Background(), nil, HttpRequestInput{
		Method: "INVALID",
		URL:    "http://example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError to be true for invalid method")
	}
	text := extractText(result)
	if !strings.Contains(text, "unsupported method") {
		t.Errorf("expected 'unsupported method' error, got: %s", text)
	}
}

func Test_HttpRequestHandler_PostWithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		contentType := r.Header.Get("Content-Type")
		w.WriteHeader(201)
		fmt.Fprintf(w, "body=%s ct=%s", string(bodyBytes), contentType)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	handler := makeHandler(c)

	result, _, err := handler(context.Background(), nil, HttpRequestInput{
		Method:  "POST",
		URL:     server.URL,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    `{"key":"value"}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %+v", result.Content)
	}

	text := extractText(result)
	if !strings.Contains(text, "201 Created") {
		t.Errorf("expected 201 Created, got: %s", text)
	}
	if !strings.Contains(text, `body={"key":"value"}`) {
		t.Errorf("expected body echo, got: %s", text)
	}
}

func Test_BuildToolDescription_NoConfig(t *testing.T) {
	desc := buildToolDescription(client.Config{})
	if !strings.Contains(desc, "Make HTTP requests") {
		t.Errorf("expected base description, got: %s", desc)
	}
	if strings.Contains(desc, "Base URL") {
		t.Errorf("expected no Base URL section, got: %s", desc)
	}
	if strings.Contains(desc, "Default headers") {
		t.Errorf("expected no Default headers section, got: %s", desc)
	}
}

func Test_BuildToolDescription_WithBaseURL(t *testing.T) {
	desc := buildToolDescription(client.Config{
		BaseURL: "http://localhost:8080",
	})
	if !strings.Contains(desc, "Base URL: http://localhost:8080") {
		t.Errorf("expected base URL in description, got: %s", desc)
	}
	if !strings.Contains(desc, "relative paths") {
		t.Errorf("expected relative paths hint, got: %s", desc)
	}
}

func Test_BuildToolDescription_WithDefaultHeaders(t *testing.T) {
	desc := buildToolDescription(client.Config{
		DefaultHeaders: map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		},
	})
	if !strings.Contains(desc, "Default headers:") {
		t.Errorf("expected Default headers section, got: %s", desc)
	}
	if !strings.Contains(desc, "Content-Type: application/json") {
		t.Errorf("expected Content-Type header in description, got: %s", desc)
	}
	if !strings.Contains(desc, "Accept: application/json") {
		t.Errorf("expected Accept header in description, got: %s", desc)
	}
}

func Test_BuildToolDescription_CensorsSensitiveHeaders(t *testing.T) {
	desc := buildToolDescription(client.Config{
		DefaultHeaders: map[string]string{
			"Authorization": "Bearer secret-token-123",
			"X-Api-Key":     "sk-my-secret-key",
			"Content-Type":  "application/json",
		},
	})
	if strings.Contains(desc, "secret-token-123") {
		t.Errorf("expected Authorization value to be censored, got: %s", desc)
	}
	if strings.Contains(desc, "sk-my-secret-key") {
		t.Errorf("expected X-Api-Key value to be censored, got: %s", desc)
	}
	if !strings.Contains(desc, "Authorization: ***") {
		t.Errorf("expected censored Authorization header, got: %s", desc)
	}
	if !strings.Contains(desc, "X-Api-Key: ***") {
		t.Errorf("expected censored X-Api-Key header, got: %s", desc)
	}
	if !strings.Contains(desc, "Content-Type: application/json") {
		t.Errorf("expected non-sensitive header to be shown, got: %s", desc)
	}
}

func Test_BuildToolDescription_FullConfig(t *testing.T) {
	desc := buildToolDescription(client.Config{
		BaseURL: "https://api.example.com",
		DefaultHeaders: map[string]string{
			"Authorization": "Bearer token",
			"Content-Type":  "application/json",
		},
	})
	if !strings.Contains(desc, "Base URL: https://api.example.com") {
		t.Errorf("expected base URL, got: %s", desc)
	}
	if !strings.Contains(desc, "Authorization: ***") {
		t.Errorf("expected censored auth header, got: %s", desc)
	}
	if !strings.Contains(desc, "Content-Type: application/json") {
		t.Errorf("expected content-type header, got: %s", desc)
	}
}

func Test_HttpRequestHandler_BodyAndFilesMutuallyExclusive(t *testing.T) {
	c := newTestClient("")
	handler := makeHandler(c)

	result, _, err := handler(context.Background(), nil, HttpRequestInput{
		Method: "POST",
		URL:    "http://example.com",
		Body:   `{"a":1}`,
		Files:  map[string]string{"file": "C:\\temp\\x.txt"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for body+files")
	}
	if !strings.Contains(extractText(result), "mutually exclusive") {
		t.Errorf("expected mutual exclusion error, got: %s", extractText(result))
	}
}

func Test_HttpRequestHandler_JSONFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"widget","huge":"` + strings.Repeat("x", 500) + `"}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	handler := makeHandler(c)

	result, _, err := handler(context.Background(), nil, HttpRequestInput{
		Method:     "GET",
		URL:        server.URL,
		JSONFilter: "name",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := extractText(result)
	if !strings.Contains(text, `"widget"`) {
		t.Errorf("expected filtered field, got: %s", text)
	}
	if strings.Contains(text, "xxxx") {
		t.Errorf("expected huge field to be filtered out, got: %s", text)
	}
}

func Test_HttpRequestHandler_SaveTo(t *testing.T) {
	payload := strings.Repeat("data", 1000)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte(payload))
	}))
	defer server.Close()

	savePath := filepath.Join(t.TempDir(), "download.bin")
	c := newTestClient(server.URL)
	handler := makeHandler(c)

	result, _, err := handler(context.Background(), nil, HttpRequestInput{
		Method: "GET",
		URL:    server.URL,
		SaveTo: savePath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := extractText(result)
	if !strings.Contains(text, "[saved to "+savePath) {
		t.Errorf("expected saved-file summary, got: %s", text)
	}
	if strings.Contains(text, "datadata") {
		t.Errorf("expected body to be omitted from output, got: %s", text)
	}

	saved, readErr := os.ReadFile(savePath)
	if readErr != nil {
		t.Fatalf("reading saved file: %v", readErr)
	}
	if string(saved) != payload {
		t.Errorf("saved file content mismatch: got %d bytes, want %d", len(saved), len(payload))
	}
}

func Test_HttpRequestHandler_SaveToErrorResponseStaysInline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer server.Close()

	savePath := filepath.Join(t.TempDir(), "should-not-exist.bin")
	c := newTestClient(server.URL)
	handler := makeHandler(c)

	result, _, err := handler(context.Background(), nil, HttpRequestInput{
		Method: "GET",
		URL:    server.URL,
		SaveTo: savePath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := extractText(result)
	if !strings.Contains(text, "404 Not Found") || !strings.Contains(text, "not found") {
		t.Errorf("expected inline error body, got: %s", text)
	}
	if _, statErr := os.Stat(savePath); statErr == nil {
		t.Error("error response must not be written to the save file")
	}
}

func Test_HttpRequestHandler_MultipartUpload(t *testing.T) {
	uploadPath := filepath.Join(t.TempDir(), "upload.txt")
	if err := os.WriteFile(uploadPath, []byte("file-content"), 0o644); err != nil {
		t.Fatalf("writing upload fixture: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			w.WriteHeader(400)
			fmt.Fprintf(w, "parse error: %s", err)
			return
		}
		file, header, err := r.FormFile("document")
		if err != nil {
			w.WriteHeader(400)
			fmt.Fprintf(w, "form file error: %s", err)
			return
		}
		defer file.Close()
		content, _ := io.ReadAll(file)
		fmt.Fprintf(w, "field=%s filename=%s content=%s", r.FormValue("note"), header.Filename, content)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	handler := makeHandler(c)

	result, _, err := handler(context.Background(), nil, HttpRequestInput{
		Method:     "POST",
		URL:        server.URL,
		Files:      map[string]string{"document": uploadPath},
		FormFields: map[string]string{"note": "hello"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %+v", result.Content)
	}
	text := extractText(result)
	if !strings.Contains(text, "field=hello") {
		t.Errorf("expected form field echo, got: %s", text)
	}
	if !strings.Contains(text, "filename=upload.txt") {
		t.Errorf("expected filename echo, got: %s", text)
	}
	if !strings.Contains(text, "content=file-content") {
		t.Errorf("expected file content echo, got: %s", text)
	}
}

func Test_HttpRequestHandler_MaxResponseBytesOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("y", 500)))
	}))
	defer server.Close()

	// Client default is 1024 bytes; the per-request override shrinks it to 100.
	c := newTestClient(server.URL)
	handler := makeHandler(c)

	result, _, err := handler(context.Background(), nil, HttpRequestInput{
		Method:           "GET",
		URL:              server.URL,
		MaxResponseBytes: 100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := extractText(result)
	if !strings.Contains(text, "[truncated:") {
		t.Errorf("expected truncation with per-request limit, got: %s", text)
	}
	if strings.Contains(text, strings.Repeat("y", 200)) {
		t.Errorf("expected body capped at 100 bytes, got %d-char output", len(text))
	}
}

func extractText(result *mcp.CallToolResult) string {
	var texts []string
	for _, c := range result.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			texts = append(texts, tc.Text)
		}
	}
	return strings.Join(texts, "\n")
}
