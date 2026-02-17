package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	BaseURL         string
	DefaultHeaders  map[string]string
	Timeout         time.Duration
	MaxResponseSize int64
	ProxyURL        string
	RetryCount      int
	RetryDelay      time.Duration
	InsecureTLS     bool
}

type Client struct {
	httpClient      *http.Client
	baseURL         string
	defaultHeaders  map[string]string
	maxResponseSize int64
	retryCount      int
	retryDelay      time.Duration
}

type RequestParams struct {
	Method          string
	URL             string
	Headers         map[string]string
	Body            string
	QueryParams     map[string]string
	Timeout         time.Duration
	FollowRedirects bool
	IncludeHeaders  bool
}

type Response struct {
	StatusCode   int
	StatusText   string
	Headers      http.Header
	Body         []byte
	Duration     time.Duration
	Truncated    bool
	OriginalSize int64
}

// ParseHeaders splits raw "Key: Value" strings into a map.
// Values containing colons are handled correctly (split on first ": " only).
func ParseHeaders(raw []string) map[string]string {
	headers := make(map[string]string)
	for _, h := range raw {
		if idx := strings.Index(h, ": "); idx > 0 {
			headers[h[:idx]] = h[idx+2:]
		}
	}
	return headers
}

func NewClient(config Config) *Client {
	transport := &http.Transport{}

	if config.ProxyURL != "" {
		proxyURL, err := url.Parse(config.ProxyURL)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	if config.InsecureTLS {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}

	maxResponseSize := config.MaxResponseSize
	if maxResponseSize <= 0 {
		maxResponseSize = 51200
	}

	return &Client{
		httpClient:      httpClient,
		baseURL:         config.BaseURL,
		defaultHeaders:  config.DefaultHeaders,
		maxResponseSize: maxResponseSize,
		retryCount:      config.RetryCount,
		retryDelay:      config.RetryDelay,
	}
}

func (c *Client) ExecuteRequest(ctx context.Context, params RequestParams) (*Response, error) {
	requestURL := params.URL
	if strings.HasPrefix(requestURL, "/") && c.baseURL != "" {
		requestURL = strings.TrimRight(c.baseURL, "/") + requestURL
	}

	if len(params.QueryParams) > 0 {
		parsedURL, err := url.Parse(requestURL)
		if err != nil {
			return nil, fmt.Errorf("parsing URL %s: %w", requestURL, err)
		}
		query := parsedURL.Query()
		for key, value := range params.QueryParams {
			query.Set(key, value)
		}
		parsedURL.RawQuery = query.Encode()
		requestURL = parsedURL.String()
	}

	requestCtx := ctx
	var cancel context.CancelFunc
	if params.Timeout > 0 {
		requestCtx, cancel = context.WithTimeout(ctx, params.Timeout)
		defer cancel()
	}

	originalCheckRedirect := c.httpClient.CheckRedirect
	if !params.FollowRedirects {
		c.httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else {
		c.httpClient.CheckRedirect = nil
	}
	defer func() { c.httpClient.CheckRedirect = originalCheckRedirect }()

	maxAttempts := c.retryCount + 1
	var lastErr error
	var lastResponse *Response

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryDelay)
		}

		var bodyReader io.Reader
		if params.Body != "" {
			bodyReader = strings.NewReader(params.Body)
		}

		req, err := http.NewRequestWithContext(requestCtx, params.Method, requestURL, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("creating request %s %s: %w", params.Method, requestURL, err)
		}

		for key, value := range c.defaultHeaders {
			req.Header.Set(key, value)
		}
		for key, value := range params.Headers {
			req.Header.Set(key, value)
		}

		start := time.Now()
		resp, err := c.httpClient.Do(req)
		duration := time.Since(start)

		if err != nil {
			lastErr = fmt.Errorf("executing %s %s: %w", params.Method, requestURL, err)
			if attempt < maxAttempts-1 {
				continue
			}
			return nil, lastErr
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, c.maxResponseSize+1))
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("reading response body: %w", readErr)
			if attempt < maxAttempts-1 {
				continue
			}
			return nil, lastErr
		}

		truncated := int64(len(body)) > c.maxResponseSize
		var originalSize int64
		if truncated {
			originalSize = resp.ContentLength
			if originalSize <= 0 {
				originalSize = int64(len(body))
			}
			body = body[:c.maxResponseSize]
		}

		response := &Response{
			StatusCode:   resp.StatusCode,
			StatusText:   http.StatusText(resp.StatusCode),
			Headers:      resp.Header,
			Body:         body,
			Duration:     duration,
			Truncated:    truncated,
			OriginalSize: originalSize,
		}

		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return response, nil
		}

		if resp.StatusCode >= 500 && attempt < maxAttempts-1 {
			lastResponse = response
			continue
		}

		return response, nil
	}

	if lastResponse != nil {
		return lastResponse, nil
	}
	return nil, lastErr
}
