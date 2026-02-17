package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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
	if !strings.Contains(text, "HTTP 200 OK") {
		t.Errorf("expected HTTP 200 OK in output, got: %s", text)
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
	if !strings.Contains(text, "HTTP 201 Created") {
		t.Errorf("expected HTTP 201, got: %s", text)
	}
	if !strings.Contains(text, `body={"key":"value"}`) {
		t.Errorf("expected body echo, got: %s", text)
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
