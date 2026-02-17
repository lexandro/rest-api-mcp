# rest-api-mcp

MCP server providing a structured HTTP request tool for AI agents. Replaces fragile curl commands with a cross-platform, token-efficient `http_request` tool.

## Tech Stack
- Language: Go 1.22+
- MCP SDK: github.com/modelcontextprotocol/go-sdk
- HTTP: net/http standard library (no external HTTP client dependencies)

## Build & Test
- Build: `go build -o rest-api-mcp.exe .`
- Test all: `go test ./...`
- Test one package: `go test ./client/...`
- Run: `./rest-api-mcp.exe --base-url http://localhost:8080`

## Architecture
- `main.go` - Entry point, CLI flag parsing, subcommand dispatch, component wiring
- `client/` - HTTP client wrapper (retry, proxy, TLS, default headers, timeout)
- `server/` - MCP server setup, tool registration (stdio transport)
- `tools/` - MCP tool handler for `http_request` + response formatting
- `register/` - `register` subcommand for auto-registering in Claude Code config

## AI-Optimized Coding Principles

These principles override traditional SOLID/Clean Code/DRY when they conflict.
The goal: any AI agent can read, understand, and correctly modify any file in isolation.

### 1. Explicit Over Implicit (overrides DRY)
- NO init() functions - all initialization in explicit function calls from main.go
- NO interface-based dependency injection unless there are 2+ real implementations
- NO reflection, struct tags only for JSON/schema serialization
- Duplicate 3-5 lines rather than create a shared helper that obscures the logic flow
- Every function signature tells its full story - no hidden state, no package-level vars mutated as side effects

### 2. Flat Over Deep (overrides SOLID's abstraction layers)
- Maximum 1 level of function call depth for business logic (handler -> client operation)
- No "manager calls service calls repository calls adapter" chains
- The tool handler in tools/ directly calls client/ methods - no intermediate layers
- Prefer switch statements over polymorphism when there are <5 cases
- No abstract factory, strategy pattern, or visitor pattern - use plain functions

### 3. Co-located Over Separated (overrides separation of concerns when it hurts readability)
- Each file is self-contained: reading one file gives the full picture of one feature
- Types used only in one file are defined in that file, not in a separate types.go
- Error types specific to a package are defined in the file that returns them
- Test files mirror source files 1:1 (client.go -> client_test.go)

### 4. Predictable Patterns (the most important principle)
- The MCP tool handler follows the exact same structure:
  1. Parse input struct
  2. Validate parameters
  3. Call client method
  4. Format response output
  5. Return MCP result
- Consistent error handling: wrap with context, return early, never panic

### 5. Small Context Window Files (overrides "one class per file" dogma)
- Target: each .go file under 300 lines
- If a file grows beyond 300 lines, split by FUNCTIONALITY not by type
- Good split: format_headers.go, format_body.go
- Bad split: types.go, interfaces.go, impl.go

### 6. Self-Documenting Names (overrides brevity)
- Functions: verb + noun, describe what they do: `ExecuteRequest()`, `FormatResponse()`, `BuildRequestURL()`
- Variables: full words, no abbreviations: `responseBody` not `rb`, `headerCount` not `hc`
- Constants: SCREAMING_SNAKE for true constants, descriptive: `MaxResponseSizeBytes`, `DefaultTimeoutSeconds`
- Package names: single word, lowercase, obvious: `client`, `server`, `tools`

### 7. Error Handling (explicit, not clever)
- Always return errors, never log-and-continue silently
- Wrap errors with context: `fmt.Errorf("executing %s %s: %w", method, url, err)`
- No custom error types unless the caller needs to match on error type
- Log at the boundary (main.go, tool handlers), not deep in library code

### 8. Concurrency (explicit, simple patterns only)
- This server is mostly single-threaded (one request at a time from MCP)
- No goroutines needed for HTTP requests (sequential by nature)
- If concurrency is ever needed, use sync.WaitGroup for fan-out/fan-in

### 9. Testing (pragmatic, not dogmatic)
- Test public API of each package, not internal functions
- Table-driven tests for input/output variations
- Use httptest.NewServer for HTTP client tests - no mocks
- Test names: Test_FunctionName_Scenario (e.g., Test_ExecuteRequest_PostWithBody)

### 10. When Traditional Principles DO Apply
- DRY: Apply for true business logic duplication (same algorithm in 2+ places)
- KISS: Always - never add complexity without immediate need
- Single Responsibility: Each PACKAGE has one responsibility; within a package, files can do related things
- Interface Segregation: MCP tool input/output types should be minimal
