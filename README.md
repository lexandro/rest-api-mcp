# rest-api-mcp

A lightweight MCP (Model Context Protocol) server that provides a structured `http_request` tool for AI agents. Eliminates the need for fragile `curl` commands that break on Windows due to quoting/escaping issues.

## Why?

AI agents (like Claude Code) frequently need to make HTTP requests during API development. Using `curl` via Bash is unreliable on Windows — quoting JSON bodies, escaping special characters, and handling headers all break differently across shells. This MCP server provides a single, structured `http_request` tool that works identically on all platforms.

## Installation

### Build from source

```bash
git clone https://github.com/lexandro/rest-api-mcp.git
cd rest-api-mcp
go build -o rest-api-mcp.exe .
```

### Download binary

Pre-built binaries are available on the [Releases](https://github.com/lexandro/rest-api-mcp/releases) page.

## MCP Configuration

Add to your MCP config (e.g., `~/.claude/mcp.json` or `.mcp.json`):

```json
{
  "mcpServers": {
    "rest-api": {
      "command": "C:\\path\\to\\rest-api-mcp.exe",
      "args": [
        "--base-url", "http://localhost:8080",
        "--default-header", "Authorization: Bearer ${API_TOKEN}",
        "--default-header", "Content-Type: application/json"
      ]
    }
  }
}
```

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--base-url` | _(none)_ | Base URL prepended to relative paths |
| `--default-header` | _(none)_ | Default header (repeatable), format: `Key: Value` |
| `--timeout` | `30s` | Default request timeout |
| `--max-response-size` | `50KB` | Maximum response body size before truncation |
| `--proxy` | _(none)_ | HTTP/HTTPS proxy URL |
| `--retry` | `0` | Number of retry attempts for failed requests |
| `--retry-delay` | `1000ms` | Delay between retries |
| `--insecure` | `false` | Skip TLS certificate verification |
| `--log-enabled` | `false` | Enable request/response logging |
| `--log-file` | _(none)_ | Log file path (stderr if not set) |
| `--log-level` | `info` | Log level: debug, info, warn, error |

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
| `timeout` | string | no | Per-request timeout override (e.g., "10s", "500ms") |
| `followRedirects` | boolean | no | Follow HTTP redirects (default: true) |
| `includeResponseHeaders` | boolean | no | Include response headers in output (default: false) |

### Response Format

Compact, token-efficient output:

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
HTTP 200 OK (1204ms)

{"data": [...first 50KB...]}
[truncated, showing 51200 of 245891 bytes]
```

## Examples

### Simple GET
```
Use the http_request tool:
- method: GET
- url: /api/users
```

### POST with JSON body
```
Use the http_request tool:
- method: POST
- url: /api/users
- headers: {"Content-Type": "application/json"}
- body: {"name": "John", "email": "john@example.com"}
```

### PUT with query parameters
```
Use the http_request tool:
- method: PUT
- url: /api/users/123
- queryParams: {"fields": "name,email"}
- body: {"name": "Updated Name"}
```

## License

MIT
