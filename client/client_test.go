package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func Test_ExecuteRequest_Methods(t *testing.T) {
	tests := []struct {
		method     string
		wantStatus int
	}{
		{"GET", 200},
		{"POST", 201},
		{"PUT", 200},
		{"DELETE", 204},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case "POST":
					w.WriteHeader(201)
				case "DELETE":
					w.WriteHeader(204)
				default:
					w.WriteHeader(200)
				}
			}))
			defer server.Close()

			c := NewClient(Config{
				Timeout:         5 * time.Second,
				MaxResponseSize: 1024,
			})

			resp, err := c.ExecuteRequest(context.Background(), RequestParams{
				Method:          tt.method,
				URL:             server.URL,
				FollowRedirects: true,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("got status %d, want %d", resp.StatusCode, tt.wantStatus)
			}
		})
	}
}

func Test_ExecuteRequest_Headers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "X-Default=%s X-Custom=%s", r.Header.Get("X-Default"), r.Header.Get("X-Custom"))
	}))
	defer server.Close()

	c := NewClient(Config{
		DefaultHeaders:  map[string]string{"X-Default": "default-value", "X-Custom": "will-be-overridden"},
		Timeout:         5 * time.Second,
		MaxResponseSize: 1024,
	})

	resp, err := c.ExecuteRequest(context.Background(), RequestParams{
		Method:          "GET",
		URL:             server.URL,
		Headers:         map[string]string{"X-Custom": "override-value"},
		FollowRedirects: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := string(resp.Body)
	if !strings.Contains(body, "X-Default=default-value") {
		t.Errorf("default header not applied, got: %s", body)
	}
	if !strings.Contains(body, "X-Custom=override-value") {
		t.Errorf("per-request header did not override default, got: %s", body)
	}
}

func Test_ExecuteRequest_QueryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "foo=%s bar=%s", r.URL.Query().Get("foo"), r.URL.Query().Get("bar"))
	}))
	defer server.Close()

	c := NewClient(Config{
		Timeout:         5 * time.Second,
		MaxResponseSize: 1024,
	})

	resp, err := c.ExecuteRequest(context.Background(), RequestParams{
		Method:          "GET",
		URL:             server.URL,
		QueryParams:     map[string]string{"foo": "hello", "bar": "world"},
		FollowRedirects: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := string(resp.Body)
	if !strings.Contains(body, "foo=hello") || !strings.Contains(body, "bar=world") {
		t.Errorf("query params not applied correctly, got: %s", body)
	}
}

func Test_ExecuteRequest_Body(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		w.WriteHeader(200)
		w.Write(bodyBytes)
	}))
	defer server.Close()

	c := NewClient(Config{
		Timeout:         5 * time.Second,
		MaxResponseSize: 1024,
	})

	sentBody := `{"key":"value"}`
	resp, err := c.ExecuteRequest(context.Background(), RequestParams{
		Method:          "POST",
		URL:             server.URL,
		Body:            sentBody,
		FollowRedirects: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(resp.Body) != sentBody {
		t.Errorf("body not sent correctly, got: %s, want: %s", string(resp.Body), sentBody)
	}
}

func Test_ExecuteRequest_RelativeURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "path=%s", r.URL.Path)
	}))
	defer server.Close()

	c := NewClient(Config{
		BaseURL:         server.URL,
		Timeout:         5 * time.Second,
		MaxResponseSize: 1024,
	})

	resp, err := c.ExecuteRequest(context.Background(), RequestParams{
		Method:          "GET",
		URL:             "/api/test",
		FollowRedirects: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(resp.Body), "path=/api/test") {
		t.Errorf("relative URL not resolved correctly, got: %s", string(resp.Body))
	}
}

func Test_ExecuteRequest_AbsoluteURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ok")
	}))
	defer server.Close()

	c := NewClient(Config{
		BaseURL:         "http://should-not-be-used:9999",
		Timeout:         5 * time.Second,
		MaxResponseSize: 1024,
	})

	resp, err := c.ExecuteRequest(context.Background(), RequestParams{
		Method:          "GET",
		URL:             server.URL + "/test",
		FollowRedirects: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(resp.Body) != "ok" {
		t.Errorf("absolute URL should be used as-is, got: %s", string(resp.Body))
	}
}

func Test_ExecuteRequest_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(200)
	}))
	defer server.Close()

	c := NewClient(Config{
		Timeout:         5 * time.Second,
		MaxResponseSize: 1024,
	})

	_, err := c.ExecuteRequest(context.Background(), RequestParams{
		Method:          "GET",
		URL:             server.URL,
		Timeout:         100 * time.Millisecond,
		FollowRedirects: true,
	})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func Test_ExecuteRequest_Truncation(t *testing.T) {
	largeBody := strings.Repeat("x", 200)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(largeBody)))
		w.WriteHeader(200)
		w.Write([]byte(largeBody))
	}))
	defer server.Close()

	c := NewClient(Config{
		Timeout:         5 * time.Second,
		MaxResponseSize: 100,
	})

	resp, err := c.ExecuteRequest(context.Background(), RequestParams{
		Method:          "GET",
		URL:             server.URL,
		FollowRedirects: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Truncated {
		t.Error("expected response to be truncated")
	}
	if len(resp.Body) != 100 {
		t.Errorf("expected body length 100, got %d", len(resp.Body))
	}
	if resp.OriginalSize != 200 {
		t.Errorf("expected original size 200, got %d", resp.OriginalSize)
	}
}

func Test_ExecuteRequest_Retry(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := callCount.Add(1)
		if count == 1 {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	c := NewClient(Config{
		Timeout:         5 * time.Second,
		MaxResponseSize: 1024,
		RetryCount:      1,
		RetryDelay:      10 * time.Millisecond,
	})

	resp, err := c.ExecuteRequest(context.Background(), RequestParams{
		Method:          "GET",
		URL:             server.URL,
		FollowRedirects: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 after retry, got %d", resp.StatusCode)
	}
	if callCount.Load() != 2 {
		t.Errorf("expected 2 calls (1 fail + 1 retry), got %d", callCount.Load())
	}
}

func Test_ExecuteRequest_NoRetryOn4xx(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(400)
	}))
	defer server.Close()

	c := NewClient(Config{
		Timeout:         5 * time.Second,
		MaxResponseSize: 1024,
		RetryCount:      2,
		RetryDelay:      10 * time.Millisecond,
	})

	resp, err := c.ExecuteRequest(context.Background(), RequestParams{
		Method:          "GET",
		URL:             server.URL,
		FollowRedirects: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	if callCount.Load() != 1 {
		t.Errorf("expected exactly 1 call (no retry on 4xx), got %d", callCount.Load())
	}
}

func Test_ExecuteRequest_NoFollowRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/redirected", http.StatusFound)
			return
		}
		w.WriteHeader(200)
		w.Write([]byte("redirected"))
	}))
	defer server.Close()

	c := NewClient(Config{
		Timeout:         5 * time.Second,
		MaxResponseSize: 1024,
	})

	resp, err := c.ExecuteRequest(context.Background(), RequestParams{
		Method:          "GET",
		URL:             server.URL,
		FollowRedirects: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 302 {
		t.Errorf("expected 302 (redirect not followed), got %d", resp.StatusCode)
	}
}

func Test_NewClient_ParseHeaders(t *testing.T) {
	tests := []struct {
		name    string
		raw     []string
		wantKey string
		wantVal string
	}{
		{
			name:    "simple header",
			raw:     []string{"Content-Type: application/json"},
			wantKey: "Content-Type",
			wantVal: "application/json",
		},
		{
			name:    "value with colon",
			raw:     []string{"Authorization: Bearer token:with:colons"},
			wantKey: "Authorization",
			wantVal: "Bearer token:with:colons",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := ParseHeaders(tt.raw)
			if got := headers[tt.wantKey]; got != tt.wantVal {
				t.Errorf("ParseHeaders(%v)[%s] = %q, want %q", tt.raw, tt.wantKey, got, tt.wantVal)
			}
		})
	}
}
