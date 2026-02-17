package tools

import (
	"context"
	"fmt"
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

func Register(mcpServer *mcp.Server, httpClient *client.Client) {
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "http_request",
		Description: "Make HTTP requests. Use instead of curl for reliable cross-platform HTTP calls. Supports all methods, headers, body, query params, redirects, and timeout.",
	}, makeHandler(httpClient))
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
