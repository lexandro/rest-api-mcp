# rest-api-mcp

[![CI](https://github.com/lexandro/rest-api-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/lexandro/rest-api-mcp/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/lexandro/rest-api-mcp)](https://goreportcard.com/report/github.com/lexandro/rest-api-mcp)
[![Go Reference](https://pkg.go.dev/badge/github.com/lexandro/rest-api-mcp.svg)](https://pkg.go.dev/github.com/lexandro/rest-api-mcp)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![MCP Compatible](https://img.shields.io/badge/MCP-compatible-blue)](https://modelcontextprotocol.io)
[![Claude Code](https://img.shields.io/badge/Claude_Code-Extension-orange)](https://claude.ai/claude-code)

A lightweight MCP (Model Context Protocol) server that provides a structured `http_request` tool for AI agents. Eliminates the need for fragile `curl` commands that break on Windows due to quoting/escaping issues.

## Why?

AI agents (like Claude Code) frequently need to make HTTP requests during API development. Using `curl` via Bash is unreliable on Windows — quoting JSON bodies, escaping special characters, and handling headers all break differently across shells. This MCP server provides a single, structured `http_request` tool that works identically on all platforms — and formats responses to burn as few context tokens as possible.

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

# Session-based API (login + cookie flows)
rest-api-mcp register project . -- \
  --base-url http://localhost:8080 \
  --cookie-jar

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
        "--retry-delay", "500ms",
        "--cookie-jar"
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

Requires Go 1.25+.

```bash
git clone https://github.com/lexandro/rest-api-mcp.git
cd rest-api-mcp
go build -o rest-api-mcp.exe .
```

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--base-url` | _(none)_ | Base URL prepended to relative paths (with or without leading slash) |
| `--default-header` | _(none)_ | Default header (repeatable), format: `Key: Value` |
| `--timeout` | `30s` | Default request timeout |
| `--max-response-size` | `51200` | Maximum response body size in bytes (default 50KB) |
| `--proxy` | _(none)_ | HTTP/HTTPS proxy URL |
| `--retry` | `0` | Number of retry attempts for failed requests |
| `--retry-delay` | `1s` | Delay between retries |
| `--insecure` | `false` | Skip TLS certificate verification |
| `--cookie-jar` | `false` | In-memory cookie jar — persists cookies across requests for session/login flows |

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
| `jsonFilter` | string | no | [GJSON path](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) to extract fields from a JSON response, e.g. `name`, `items.#.id`, `{name,id}` |
| `saveTo` | string | no | Write the response body to this file path instead of returning it inline |
| `maxResponseBytes` | number | no | Per-request response size limit (overrides `--max-response-size`) |
| `files` | object | no | multipart/form-data upload: form field name → local file path (mutually exclusive with `body`) |
| `formFields` | object | no | Text fields for multipart/form-data |

### Response Format

Compact, token-efficient output designed to minimize context window usage:

```
200 OK

{"id":1,"name":"example"}
```

Pretty-printed JSON responses are **minified automatically** (saves 20–40% tokens on indented APIs). With `includeResponseHeaders: true`:

```
200 OK

Content-Type: application/json
X-Request-Id: abc123

{"id":1,"name":"example"}
```

With `jsonFilter` only the requested fields are returned:

```
200 OK

{"name":"example","price":42}
```

Binary responses are summarized instead of dumped into context:

```
200 OK

[binary: image/png, 245891 bytes — pass saveTo to write it to a file]
```

With `saveTo` the body is streamed to disk (no size limit) and only a summary is returned — the agent can then read or grep the file:

```
200 OK

[saved to C:\temp\response.json: 245891 bytes, application/json]
```

Large inline responses are automatically truncated:

```
200 OK

{"data": [...first 50KB...]}
[truncated: 51200/245891 bytes — pass saveTo to fetch the full body to a file]
```

### Token Efficiency

- **Automatic JSON minification** — pretty-printed API responses are compacted before entering context
- **`jsonFilter` field extraction** — return only the fields the agent needs from large payloads (GJSON path syntax)
- **Binary detection** — binary bodies become a one-line summary, never raw bytes in context
- **`saveTo` file offload** — large/binary responses go to disk; the full body is available without burning tokens
- **No response headers by default** — saves ~200-500 tokens per request
- **50KB response limit** — prevents dumping huge payloads into context (per-request override via `maxResponseBytes`)
- **Minimal status line** — `200 OK` instead of verbose curl output, no duration overhead
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

### Extract only needed fields from a large response
```json
{
  "method": "GET",
  "url": "https://api.github.com/repos/lexandro/rest-api-mcp",
  "jsonFilter": "{name,stargazers_count,topics}"
}
```

### Get one field from every array item
```json
{
  "method": "GET",
  "url": "/api/users",
  "jsonFilter": "#.email"
}
```

### Download a large or binary response to a file
```json
{
  "method": "GET",
  "url": "/api/export/report.pdf",
  "saveTo": "C:\\temp\\report.pdf"
}
```

### Upload a file (multipart/form-data)
```json
{
  "method": "POST",
  "url": "/api/upload",
  "files": { "document": "C:\\temp\\invoice.pdf" },
  "formFields": { "category": "invoices" }
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
