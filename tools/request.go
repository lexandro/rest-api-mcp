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
	JSONFilter             string            `json:"jsonFilter,omitempty" jsonschema:"GJSON path to extract from a JSON response body, e.g. name, items.#.id, or {name,id} for multiple fields — use on large payloads to save tokens"`
	SaveTo                 string            `json:"saveTo,omitempty" jsonschema:"Write the response body to this file path instead of returning it inline — use for binary or large responses"`
	MaxResponseBytes       int64             `json:"maxResponseBytes,omitempty" jsonschema:"Per-request response size limit in bytes (overrides server default)"`
	Files                  map[string]string `json:"files,omitempty" jsonschema:"Send multipart/form-data: form field name -> local file path (mutually exclusive with body)"`
	FormFields             map[string]string `json:"formFields,omitempty" jsonschema:"Text fields for multipart/form-data (mutually exclusive with body)"`
}

var validMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "DELETE": true,
	"PATCH": true, "HEAD": true, "OPTIONS": true,
}

func Register(mcpServer *mcp.Server, httpClient *client.Client, cfg client.Config) {
	openWorld := true
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "http_request",
		Description: buildToolDescription(cfg),
		Annotations: &mcp.ToolAnnotations{
			OpenWorldHint: &openWorld,
		},
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
	desc := "Make HTTP requests. Use instead of curl for reliable cross-platform HTTP calls. " +
		"Supports all methods, headers, body, query params, redirects, timeout, and multipart file upload (files/formFields). " +
		"JSON responses are minified automatically. " +
		"Token savers: jsonFilter extracts only the fields you need from JSON; saveTo writes large or binary bodies to a file instead of returning them."

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

func errorResult(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: true,
	}
}

// validateInput checks the request input and returns the normalized method and
// parsed per-request timeout. A non-empty error message means invalid input.
func validateInput(input HttpRequestInput) (string, time.Duration, string) {
	if input.Method == "" {
		return "", 0, "method is required"
	}
	upperMethod := strings.ToUpper(input.Method)
	if !validMethods[upperMethod] {
		return "", 0, fmt.Sprintf("unsupported method: %s", input.Method)
	}
	if input.URL == "" {
		return "", 0, "url is required"
	}
	if input.Body != "" && (len(input.Files) > 0 || len(input.FormFields) > 0) {
		return "", 0, "body and files/formFields are mutually exclusive"
	}

	var timeout time.Duration
	if input.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(input.Timeout)
		if err != nil {
			return "", 0, fmt.Sprintf("invalid timeout: %s", err)
		}
	}
	return upperMethod, timeout, ""
}

func makeHandler(httpClient *client.Client) func(context.Context, *mcp.CallToolRequest, HttpRequestInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input HttpRequestInput) (*mcp.CallToolResult, any, error) {
		method, timeout, validationError := validateInput(input)
		if validationError != "" {
			return errorResult(validationError), nil, nil
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
			Method:          method,
			URL:             input.URL,
			Headers:         input.Headers,
			Body:            input.Body,
			QueryParams:     input.QueryParams,
			Timeout:         timeout,
			FollowRedirects: followRedirects,
			IncludeHeaders:  includeHeaders,
			SaveTo:          input.SaveTo,
			MaxResponseSize: input.MaxResponseBytes,
			Files:           input.Files,
			FormFields:      input.FormFields,
		}

		resp, err := httpClient.ExecuteRequest(ctx, params)
		if err != nil {
			return errorResult(fmt.Sprintf("Request failed: %s", err)), nil, nil
		}

		formatted := FormatResponse(resp, FormatOptions{
			IncludeHeaders: includeHeaders,
			JSONFilter:     input.JSONFilter,
		})
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: formatted}},
		}, nil, nil
	}
}
