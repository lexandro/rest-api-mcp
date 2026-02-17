# Development Prompt: rest-api-mcp

Use this prompt to implement the rest-api-mcp MCP server. Read CLAUDE.md first for coding principles.

## What to build

A Go MCP server that exposes a single `http_request` tool over stdio. AI agents use this tool instead of curl to make HTTP requests — solving Windows quoting/escaping issues and providing token-efficient output.

## Architecture overview

```
main.go                 CLI flags, wiring, starts MCP server
client/client.go        HTTP client wrapper (retry, proxy, TLS, defaults)
tools/request.go        MCP tool handler for http_request
tools/format.go         Response formatting (compact, token-efficient)
server/server.go        MCP server setup, tool registration
```

## Step 1: main.go — CLI flags and wiring

Parse these CLI flags using the standard `flag` package:

```go
// Connection
baseURL       string   // --base-url (prepended to relative URLs)
defaultHeaders []string // --default-header (repeatable, format: "Key: Value")

// Timeouts & limits
timeout         time.Duration // --timeout (default: 30s)
maxResponseSize int64         // --max-response-size (default: 51200 = 50KB)

// Network
proxy    string // --proxy (HTTP/HTTPS proxy URL)
retry    int    // --retry (default: 0)
retryDelay time.Duration // --retry-delay (default: 1000ms)
insecure bool   // --insecure (skip TLS verification)

// Logging
logEnabled bool   // --log-enabled (default: false)
logFile    string // --log-file (path, stderr if empty)
logLevel   string // --log-level (debug/info/warn/error, default: info)
```

For repeatable flags (`--default-header`), use a custom `flag.Value` implementation:

```go
type repeatedFlag []string

func (f *repeatedFlag) String() string { return strings.Join(*f, ", ") }
func (f *repeatedFlag) Set(value string) error {
    *f = append(*f, value)
    return nil
}
```

Wiring order in main():
1. Parse flags
2. Set up logging (if enabled)
3. Create client.Config from flags
4. Create client.Client from config
5. Create MCP server with server.New()
6. Register tool with tools.Register(server, client)
7. Start stdio transport with server.Run()

## Step 2: client/client.go — HTTP client wrapper

Define a Config struct and a Client struct:

```go
type Config struct {
    BaseURL         string
    DefaultHeaders  map[string]string  // parsed from "Key: Value" strings
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
```

NewClient(config Config) creates the *http.Client with:
- Custom Transport for proxy and TLS settings
- Timeout from config

ExecuteRequest method:

```go
type RequestParams struct {
    Method           string
    URL              string
    Headers          map[string]string
    Body             string
    QueryParams      map[string]string
    Timeout          time.Duration      // per-request override, 0 = use default
    FollowRedirects  bool
    IncludeHeaders   bool
}

type Response struct {
    StatusCode     int
    StatusText     string
    Headers        http.Header
    Body           []byte
    Duration       time.Duration
    Truncated      bool
    OriginalSize   int64
}
```

ExecuteRequest(ctx context.Context, params RequestParams) (*Response, error):
1. Build full URL (prepend baseURL if URL is relative — starts with `/`)
2. Add query parameters to URL
3. Create http.Request with method, URL, body
4. Apply default headers first, then per-request headers (per-request wins)
5. Configure redirect policy based on FollowRedirects
6. Apply per-request timeout if set (use context.WithTimeout)
7. Execute with retry logic:
   - Retry only on network errors and 5xx responses
   - Wait retryDelay between attempts
   - Do NOT retry on 4xx responses
8. Read response body up to maxResponseSize
   - If body exceeds limit, set Truncated=true and OriginalSize
9. Return Response

Important implementation details:
- Parse defaultHeaders in NewClient by splitting on first `: ` (colon-space)
- For non-redirect mode, set `httpClient.CheckRedirect` to return `http.ErrUseLastResponse`
- Use `io.LimitReader(resp.Body, maxResponseSize+1)` to detect truncation — if you read maxResponseSize+1 bytes, it was truncated
- Measure duration with `time.Now()` before request and `time.Since()` after

## Step 3: tools/format.go — Response formatting

FormatResponse(resp *client.Response, includeHeaders bool) string:

Format:
```
HTTP {statusCode} {statusText} ({duration})

{headers if requested, one per line: "Key: Value"}

{body}
{truncation notice if truncated}
```

Rules:
- Duration format: milliseconds for <1s (e.g., "154ms"), seconds with 1 decimal for >=1s (e.g., "2.3s")
- Headers: only if includeHeaders is true, skip common noise headers (Date, Server, Connection, etc.)
  - Keep: Content-Type, Content-Length, Location, X-* headers, Authorization-related
- Body: as-is (the raw response body string)
- Truncation notice: `[truncated, showing {shown} of {total} bytes]`
- Empty body: show `(empty body)` instead of blank space
- Error responses (non-2xx): same format, no special treatment — let the AI agent decide what to do

## Step 4: tools/request.go — MCP tool handler

Define the input struct for the tool:

```go
type HttpRequestInput struct {
    Method                string            `json:"method" jsonschema:"HTTP method: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS"`
    URL                   string            `json:"url" jsonschema:"Full URL or relative path (if base_url configured)"`
    Headers               map[string]string `json:"headers,omitempty" jsonschema:"Request headers as key-value pairs"`
    Body                  string            `json:"body,omitempty" jsonschema:"Request body (typically JSON)"`
    QueryParams           map[string]string `json:"queryParams,omitempty" jsonschema:"Query parameters as key-value pairs"`
    Timeout               string            `json:"timeout,omitempty" jsonschema:"Per-request timeout (e.g. 10s, 500ms)"`
    FollowRedirects       *bool             `json:"followRedirects,omitempty" jsonschema:"Follow HTTP redirects (default: true)"`
    IncludeResponseHeaders *bool            `json:"includeResponseHeaders,omitempty" jsonschema:"Include response headers in output (default: false)"`
}
```

Note: Use `*bool` for optional booleans so we can distinguish "not set" from "false".

Register function:

```go
func Register(server *mcp.Server, httpClient *client.Client) {
    mcp.AddTool(server, &mcp.Tool{
        Name:        "http_request",
        Description: "Make HTTP requests. Use instead of curl for reliable cross-platform HTTP calls. Supports all methods, headers, body, query params, redirects, and timeout.",
    }, makeHandler(httpClient))
}
```

Handler logic:
1. Parse input (the SDK does this automatically via the typed handler)
2. Validate: method is required and must be one of GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS
3. Validate: url is required and non-empty
4. Parse timeout string if provided (time.ParseDuration)
5. Build client.RequestParams from input
6. Call client.ExecuteRequest
7. Format response with FormatResponse
8. Return mcp.CallToolResult with TextContent

If the request fails (network error, DNS failure, etc.), return the error message as text content with IsError: true:
```go
&mcp.CallToolResult{
    Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Request failed: %s", err)}},
    IsError: true,
}
```

## Step 5: server/server.go — MCP server setup

```go
func New() *mcp.Server {
    return mcp.NewServer(
        &mcp.Implementation{
            Name:    "rest-api-mcp",
            Version: "0.1.0",
        },
        nil,
    )
}

func Run(server *mcp.Server) error {
    return server.Run(context.Background(), &mcp.StdioTransport{})
}
```

## Step 6: Tests

### client/client_test.go

Use `httptest.NewServer` for all tests. Table-driven tests for:

- Test_ExecuteRequest_Methods — GET, POST, PUT, DELETE return expected status
- Test_ExecuteRequest_Headers — default headers applied, per-request headers override
- Test_ExecuteRequest_QueryParams — query params appended to URL correctly
- Test_ExecuteRequest_Body — POST/PUT body sent correctly
- Test_ExecuteRequest_RelativeURL — relative path prepended with baseURL
- Test_ExecuteRequest_AbsoluteURL — absolute URL used as-is even with baseURL set
- Test_ExecuteRequest_Timeout — request with timeout that exceeds server delay returns error
- Test_ExecuteRequest_Truncation — response larger than maxResponseSize is truncated
- Test_ExecuteRequest_Retry — 500 response retried, then succeeds
- Test_ExecuteRequest_NoRetryOn4xx — 400 response NOT retried
- Test_ExecuteRequest_NoFollowRedirect — redirect not followed when disabled
- Test_NewClient_ParseHeaders — "Key: Value" strings parsed correctly, including values with colons

### tools/format_test.go

Table-driven tests for:
- Test_FormatResponse_BasicSuccess — 200 OK with body
- Test_FormatResponse_EmptyBody — shows "(empty body)"
- Test_FormatResponse_WithHeaders — includeHeaders shows relevant headers
- Test_FormatResponse_Truncated — shows truncation notice
- Test_FormatResponse_DurationFormat — <1s shows ms, >=1s shows seconds

### tools/request_test.go

Test the handler via the MCP tool call interface:
- Test_HttpRequestHandler_ValidGet — basic GET request works
- Test_HttpRequestHandler_MissingMethod — returns error for missing method
- Test_HttpRequestHandler_MissingURL — returns error for missing URL
- Test_HttpRequestHandler_InvalidMethod — returns error for unsupported method
- Test_HttpRequestHandler_PostWithBody — POST with body and headers

## Token efficiency design decisions

These are CRITICAL for keeping AI agent context windows small:

1. **No response headers by default** — saves ~200-500 tokens per request. Only include when explicitly requested via `includeResponseHeaders: true`.

2. **Truncate large responses** — 50KB default limit prevents accidentally dumping huge API responses into context. The truncation notice tells the agent exactly how much was cut.

3. **Compact status line** — `HTTP 200 OK (154ms)` is much shorter than curl's verbose output format.

4. **No request echo** — don't repeat back what was sent. The agent already knows what it requested.

5. **Error as text, not stack trace** — `Request failed: connection refused` not a Go panic trace.

## Implementation order

1. `server/server.go` (simplest, needed first)
2. `client/client.go` + `client/client_test.go` (core functionality)
3. `tools/format.go` + `tools/format_test.go` (formatting)
4. `tools/request.go` + `tools/request_test.go` (tool handler, ties it together)
5. `main.go` (wiring)
6. Build and manual test: `go build && echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./rest-api-mcp.exe`

## Final checklist

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds on Windows: `go build -o rest-api-mcp.exe .`
- [ ] `go vet ./...` reports no issues
- [ ] Each .go file is under 300 lines
- [ ] No init() functions anywhere
- [ ] No interface-based DI (only concrete types)
- [ ] Error messages include context (method, URL, etc.)
- [ ] Handler follows the standard pattern: parse -> validate -> execute -> format -> return
