# rest-api-mcp

A lightweight MCP (Model Context Protocol) server that provides a structured `http_request` tool for AI agents. Eliminates the need for fragile `curl` commands that break on Windows due to quoting/escaping issues.

## Why?

AI agents (like Claude Code) frequently need to make HTTP requests during API development. Using `curl` via Bash is unreliable on Windows — quoting JSON bodies, escaping special characters, and handling headers all break differently across shells. This MCP server provides a single, structured `http_request` tool that works identically on all platforms.

## Quick Start

### Register in Claude Code

The `register` subcommand automatically adds the server to Claude Code's config — no manual JSON editing needed.

```bash
# Register for current project (creates .mcp.json)
rest-api-mcp register project

# Register with a base URL for API development
rest-api-mcp register project . -- --base-url http://localhost:8080

# Register globally for all projects (writes to ~/.claude.json)
rest-api-mcp register user
```

Arguments after `--` are forwarded to the MCP server on every startup.

### More examples

```bash
# Authenticated API with default headers
rest-api-mcp register project . -- \
  --base-url https://api.example.com \
  --default-header "Authorization: Bearer YOUR_TOKEN_HERE" \
  --default-header "Content-Type: application/json"

# Corporate proxy / self-signed certs
rest-api-mcp register project . -- \
  --base-url https://internal-api.corp.local \
  --proxy http://proxy.corp.local:8080 \
  --insecure
```

### Manual configuration

You can also edit the config files directly. The `register` command generates entries like this in `.mcp.json` or `~/.claude.json`:

```json
{
  "mcpServers": {
    "rest-api": {
      "command": "/path/to/rest-api-mcp",
      "args": ["--base-url", "http://localhost:8080"]
    }
  }
}
```

<details>
<summary>Full configuration example with all options</summary>

```json
{
  "mcpServers": {
    "rest-api": {
      "command": "/path/to/rest-api-mcp",
      "args": [
        "--base-url", "http://localhost:3000",
        "--default-header", "Authorization: Bearer YOUR_TOKEN_HERE",
        "--default-header", "Content-Type: application/json",
        "--default-header", "Accept: application/json",
        "--timeout", "15s",
        "--max-response-size", "102400",
        "--retry", "2",
        "--retry-delay", "500ms"
      ]
    }
  }
}
```

</details>

## Installation

### Download binary

Pre-built binaries for Windows, macOS, and Linux are available on the [Releases](https://github.com/lexandro/rest-api-mcp/releases) page.

### Build from source

Requires Go 1.22+.

```bash
git clone https://github.com/lexandro/rest-api-mcp.git
cd rest-api-mcp
go build -o rest-api-mcp.exe .
```

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--base-url` | _(none)_ | Base URL prepended to relative paths |
| `--default-header` | _(none)_ | Default header (repeatable), format: `Key: Value` |
| `--timeout` | `30s` | Default request timeout |
| `--max-response-size` | `51200` | Maximum response body size in bytes (default 50KB) |
| `--proxy` | _(none)_ | HTTP/HTTPS proxy URL |
| `--retry` | `0` | Number of retry attempts for failed requests |
| `--retry-delay` | `1s` | Delay between retries |
| `--insecure` | `false` | Skip TLS certificate verification |

## Tool: `http_request`

A single, versatile tool for making HTTP requests.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `method` | string | yes | HTTP method: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS |
| `url` | string | yes | Full URL or relative path (if `--base-url` is set) |
| `headers` | object | no | Request headers as key-value pairs |
| `body` | string | no | Request body (typically JSON) |
| `queryParams` | object | no | Query parameters as key-value pairs |
| `timeout` | string | no | Per-request timeout override (e.g., `10s`, `500ms`) |
| `followRedirects` | boolean | no | Follow HTTP redirects (default: true) |
| `includeResponseHeaders` | boolean | no | Include response headers in output (default: false) |

### Response Format

Compact, token-efficient output designed to minimize context window usage:

```
HTTP 200 OK (154ms)

{"id": 1, "name": "example"}
```

With `includeResponseHeaders: true`:

```
HTTP 200 OK (154ms)

Content-Type: application/json
X-Request-Id: abc123

{"id": 1, "name": "example"}
```

Large responses are automatically truncated:

```
HTTP 200 OK (1.2s)

{"data": [...first 50KB...]}
[truncated, showing 51200 of 245891 bytes]
```

### Token Efficiency

- **No response headers by default** — saves ~200-500 tokens per request
- **50KB response limit** — prevents dumping huge payloads into context
- **Compact status line** — `HTTP 200 OK (154ms)` instead of verbose curl output
- **No request echo** — the agent already knows what it sent
- **Error as text** — `Request failed: connection refused` not a stack trace

## Examples

### Simple GET
```json
{ "method": "GET", "url": "/api/users" }
```

### POST with JSON body
```json
{
  "method": "POST",
  "url": "/api/users",
  "headers": { "Content-Type": "application/json" },
  "body": "{\"name\": \"John\", \"email\": \"john@example.com\"}"
}
```

### GET with query parameters
```json
{
  "method": "GET",
  "url": "/api/search",
  "queryParams": { "q": "hello", "limit": "10" }
}
```

### PUT with timeout override
```json
{
  "method": "PUT",
  "url": "/api/users/123",
  "body": "{\"name\": \"Updated\"}",
  "timeout": "5s"
}
```

### GET without following redirects
```json
{
  "method": "GET",
  "url": "/api/short-link",
  "followRedirects": false,
  "includeResponseHeaders": true
}
```

## Development

### Build

```bash
go build -o rest-api-mcp.exe .
```

### Test

```bash
go test ./...
```

### Vet

```bash
go vet ./...
```

## License

MIT
