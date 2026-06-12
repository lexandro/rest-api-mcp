package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/tidwall/gjson"

	"github.com/lexandro/rest-api-mcp/client"
)

var noiseHeaders = map[string]bool{
	"Date":              true,
	"Server":            true,
	"Connection":        true,
	"Keep-Alive":        true,
	"Transfer-Encoding": true,
	"Accept-Ranges":     true,
	"Vary":              true,
	"Etag":              true,
	"Cache-Control":     true,
	"Pragma":            true,
	"Expires":           true,
	"Age":               true,
	"Via":               true,
	"X-Cache":           true,
}

// FormatOptions controls how a response is rendered for the model.
type FormatOptions struct {
	IncludeHeaders bool
	JSONFilter     string
}

func FormatResponse(resp *client.Response, opts FormatOptions) string {
	var builder strings.Builder

	fmt.Fprintf(&builder, "%d %s", resp.StatusCode, resp.StatusText)

	if opts.IncludeHeaders && len(resp.Headers) > 0 {
		builder.WriteString("\n")
		keys := make([]string, 0, len(resp.Headers))
		for key := range resp.Headers {
			if !noiseHeaders[key] {
				keys = append(keys, key)
			}
		}
		sort.Strings(keys)
		for _, key := range keys {
			for _, value := range resp.Headers[key] {
				fmt.Fprintf(&builder, "\n%s: %s", key, value)
			}
		}
	}

	if resp.SavedPath != "" {
		fmt.Fprintf(&builder, "\n\n[saved to %s: %d bytes, %s]", resp.SavedPath, resp.SavedSize, displayContentType(resp.ContentType))
		return builder.String()
	}

	if len(resp.Body) > 0 {
		if !isTextContent(resp.ContentType, resp.Body) {
			fmt.Fprintf(&builder, "\n\n[binary: %s, %d bytes — pass saveTo to write it to a file]", displayContentType(resp.ContentType), totalBodySize(resp))
			return builder.String()
		}
		builder.WriteString("\n\n")
		builder.WriteString(renderTextBody(resp.Body, opts.JSONFilter))
	}

	if resp.Truncated {
		shown := len(resp.Body)
		if resp.OriginalSize > 0 {
			fmt.Fprintf(&builder, "\n[truncated: %d/%d bytes — pass saveTo to fetch the full body to a file]", shown, resp.OriginalSize)
		} else {
			fmt.Fprintf(&builder, "\n[truncated: %d bytes shown — pass saveTo to fetch the full body to a file]", shown)
		}
	}

	return builder.String()
}

func renderTextBody(body []byte, jsonFilter string) string {
	if jsonFilter != "" {
		result := gjson.GetBytes(body, jsonFilter)
		if !result.Exists() {
			return fmt.Sprintf("[jsonFilter %q matched nothing — retry without jsonFilter to inspect the body]", jsonFilter)
		}
		body = []byte(result.Raw)
	}
	return string(minifyJSON(body))
}

// totalBodySize reports the full body size, using the server-reported size
// when the body was truncated during reading.
func totalBodySize(resp *client.Response) int64 {
	if resp.Truncated && resp.OriginalSize > 0 {
		return resp.OriginalSize
	}
	return int64(len(resp.Body))
}

// minifyJSON compacts the body if it is valid JSON; anything else (including
// truncated JSON, which fails to parse) is returned unchanged.
func minifyJSON(body []byte) []byte {
	trimmed := bytes.TrimLeft(body, " \t\r\n")
	if len(trimmed) == 0 || (trimmed[0] != '{' && trimmed[0] != '[') {
		return body
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, body); err != nil {
		return body
	}
	return compact.Bytes()
}

var textContentTypes = map[string]bool{
	"application/json":                  true,
	"application/xml":                   true,
	"application/javascript":            true,
	"application/x-www-form-urlencoded": true,
	"application/x-ndjson":              true,
	"application/yaml":                  true,
	"application/x-yaml":                true,
	"application/toml":                  true,
	"application/graphql":               true,
	"application/sql":                   true,
	"image/svg+xml":                     true,
}

func isTextContent(contentType string, body []byte) bool {
	mediaType := mediaTypeOf(contentType)
	if strings.HasPrefix(mediaType, "text/") || textContentTypes[mediaType] ||
		strings.HasSuffix(mediaType, "+json") || strings.HasSuffix(mediaType, "+xml") {
		return true
	}
	return looksLikeText(body)
}

// looksLikeText sniffs up to 512 bytes: NUL bytes or invalid UTF-8 mean binary.
func looksLikeText(body []byte) bool {
	sample := body
	if len(sample) > 512 {
		sample = sample[:512]
	}
	if bytes.IndexByte(sample, 0) >= 0 {
		return false
	}
	if utf8.Valid(sample) {
		return true
	}
	// The 512-byte cut may have split a multi-byte rune; trim up to 3 trailing bytes.
	for i := 1; i <= 3 && i < len(sample); i++ {
		if utf8.Valid(sample[:len(sample)-i]) {
			return true
		}
	}
	return false
}

func mediaTypeOf(contentType string) string {
	return strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
}

func displayContentType(contentType string) string {
	mediaType := mediaTypeOf(contentType)
	if mediaType == "" {
		return "unknown content type"
	}
	return mediaType
}
