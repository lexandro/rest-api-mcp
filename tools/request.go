package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/lexandro/rest-api-mcp/client"
)

type HttpRequestInput struct {
	Method                 string            `json:"method" jsonschema:"HTTP method: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS"`
	URL                    string            `json:"url" jsonschema:"Full URL or relative path (if base_url configured)"`
	Headers                map[string]string `json:"headers,omitempty" jsonschema:"Request headers as key-value pairs"`
	Body                   string            `json:"body,omitempty" jsonschema:"Request body (typically JSON)"`
	QueryParams            map[string]string `json:"queryParams,omitempty" jsonschema:"Query parameters as key-value pairs"`
	Timeout                string            `json:"timeout,omitempty" jsonschema:"Per-request timeout (e.g. 10s, 500ms)"`
	FollowRedirects        *bool             `json:"followRedirects,omitempty" jsonschema:"Follow HTTP redirects (default: true)"`
	IncludeResponseHeaders *bool             `json:"includeResponseHeaders,omitempty" jsonschema:"Include response headers in output (default: false)"`
}

var validMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "DELETE": true,
	"PATCH": true, "HEAD": true, "OPTIONS": true,
}

func Register(mcpServer *mcp.Server, httpClient *client.Client, cfg client.Config) {
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "http_request",
		Description: buildToolDescription(cfg),
	}, makeHandler(httpClient))
}

// sensitiveHeaderNames contains lowercase header names whose values must be censored in the tool description.
var sensitiveHeaderNames = map[string]bool{
	"authorization":       true,
	"proxy-authorization": true,
	"x-api-key":           true,
	"x-auth-token":        true,
}

func censorHeaderValue(name, value string) string {
	if sensitiveHeaderNames[strings.ToLower(name)] {
		return "***"
	}
	return value
}

func buildToolDescription(cfg client.Config) string {
	desc := "Make HTTP requests. Use instead of curl for reliable cross-platform HTTP calls. Supports all methods, headers, body, query params, redirects, and timeout."

	if cfg.BaseURL != "" {
		desc += fmt.Sprintf(" Base URL: %s — use relative paths like /api/endpoint.", cfg.BaseURL)
	}

	if len(cfg.DefaultHeaders) > 0 {
		keys := make([]string, 0, len(cfg.DefaultHeaders))
		for k := range cfg.DefaultHeaders {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		headerParts := make([]string, 0, len(keys))
		for _, k := range keys {
			headerParts = append(headerParts, fmt.Sprintf("%s: %s", k, censorHeaderValue(k, cfg.DefaultHeaders[k])))
		}
		desc += fmt.Sprintf(" Default headers: %s.", strings.Join(headerParts, ", "))
	}

	return desc
}

func makeHandler(httpClient *client.Client) func(context.Context, *mcp.CallToolRequest, HttpRequestInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input HttpRequestInput) (*mcp.CallToolResult, any, error) {
		if input.Method == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "method is required"}},
				IsError: true,
			}, nil, nil
		}
		upperMethod := strings.ToUpper(input.Method)
		if !validMethods[upperMethod] {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("unsupported method: %s", input.Method)}},
				IsError: true,
			}, nil, nil
		}

		if input.URL == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "url is required"}},
				IsError: true,
			}, nil, nil
		}

		var timeout time.Duration
		if input.Timeout != "" {
			var err error
			timeout, err = time.ParseDuration(input.Timeout)
			if err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("invalid timeout: %s", err)}},
					IsError: true,
				}, nil, nil
			}
		}

		followRedirects := true
		if input.FollowRedirects != nil {
			followRedirects = *input.FollowRedirects
		}
		includeHeaders := false
		if input.IncludeResponseHeaders != nil {
			includeHeaders = *input.IncludeResponseHeaders
		}

		params := client.RequestParams{
			Method:          upperMethod,
			URL:             input.URL,
			Headers:         input.Headers,
			Body:            input.Body,
			QueryParams:     input.QueryParams,
			Timeout:         timeout,
			FollowRedirects: followRedirects,
			IncludeHeaders:  includeHeaders,
		}

		resp, err := httpClient.ExecuteRequest(ctx, params)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Request failed: %s", err)}},
				IsError: true,
			}, nil, nil
		}

		formatted := FormatResponse(resp, includeHeaders)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: formatted}},
		}, nil, nil
	}
}
