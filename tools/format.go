package tools

import (
	"fmt"
	"sort"
	"strings"

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

func FormatResponse(resp *client.Response, includeHeaders bool) string {
	var builder strings.Builder

	fmt.Fprintf(&builder, "%d %s", resp.StatusCode, resp.StatusText)

	if includeHeaders && len(resp.Headers) > 0 {
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

	if len(resp.Body) > 0 {
		builder.WriteString("\n\n")
		builder.Write(resp.Body)
	}

	if resp.Truncated {
		shown := len(resp.Body)
		if resp.OriginalSize > 0 {
			fmt.Fprintf(&builder, "\n[truncated: %d/%d bytes]", shown, resp.OriginalSize)
		} else {
			fmt.Fprintf(&builder, "\n[truncated: %d bytes shown]", shown)
		}
	}

	return builder.String()
}
