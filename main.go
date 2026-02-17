package main

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/lexandro/rest-api-mcp/client"
	"github.com/lexandro/rest-api-mcp/register"
	"github.com/lexandro/rest-api-mcp/server"
	"github.com/lexandro/rest-api-mcp/tools"
)

type repeatedFlag []string

func (f *repeatedFlag) String() string { return strings.Join(*f, ", ") }
func (f *repeatedFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "register" {
		register.Run(register.ServerInfo{Name: "rest-api"}, os.Args[2:])
		return
	}

	var (
		baseURL         string
		defaultHeaders  repeatedFlag
		timeout         time.Duration
		maxResponseSize int64
		proxy           string
		retry           int
		retryDelay      time.Duration
		insecure        bool
		logEnabled      bool
		logFile         string
		logLevel        string
	)

	flag.StringVar(&baseURL, "base-url", "", "Base URL prepended to relative URLs")
	flag.Var(&defaultHeaders, "default-header", "Default header (repeatable, format: \"Key: Value\")")
	flag.DurationVar(&timeout, "timeout", 30*time.Second, "Request timeout")
	flag.Int64Var(&maxResponseSize, "max-response-size", 51200, "Maximum response body size in bytes")
	flag.StringVar(&proxy, "proxy", "", "HTTP/HTTPS proxy URL")
	flag.IntVar(&retry, "retry", 0, "Number of retries for failed requests")
	flag.DurationVar(&retryDelay, "retry-delay", 1000*time.Millisecond, "Delay between retries")
	flag.BoolVar(&insecure, "insecure", false, "Skip TLS certificate verification")
	flag.BoolVar(&logEnabled, "log-enabled", false, "Enable logging")
	flag.StringVar(&logFile, "log-file", "", "Log file path (stderr if empty)")
	flag.StringVar(&logLevel, "log-level", "info", "Log level (debug/info/warn/error)")

	flag.Parse()

	_ = logEnabled
	_ = logFile
	_ = logLevel

	config := client.Config{
		BaseURL:         baseURL,
		DefaultHeaders:  client.ParseHeaders(defaultHeaders),
		Timeout:         timeout,
		MaxResponseSize: maxResponseSize,
		ProxyURL:        proxy,
		RetryCount:      retry,
		RetryDelay:      retryDelay,
		InsecureTLS:     insecure,
	}

	httpClient := client.NewClient(config)
	mcpServer := server.New()
	tools.Register(mcpServer, httpClient)

	if err := server.Run(mcpServer); err != nil {
		log.Fatal(err)
	}
}
