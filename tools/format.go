package tools

import (
	"fmt"
	"sort"
	"strings"
	"time"

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

	durationStr := formatDuration(resp.Duration)
	fmt.Fprintf(&builder, "HTTP %d %s (%s)", resp.StatusCode, resp.StatusText, durationStr)

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

	builder.WriteString("\n\n")
	if len(resp.Body) == 0 {
		builder.WriteString("(empty body)")
	} else {
		builder.Write(resp.Body)
	}

	if resp.Truncated {
		shown := len(resp.Body)
		if resp.OriginalSize > 0 {
			fmt.Fprintf(&builder, "\n[truncated, showing %d of %d bytes]", shown, resp.OriginalSize)
		} else {
			fmt.Fprintf(&builder, "\n[truncated, showing first %d bytes]", shown)
		}
	}

	return builder.String()
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
