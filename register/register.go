package register

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ServerInfo holds the identity of the MCP server being registered.
type ServerInfo struct {
	Name string // e.g. "rest-api" (without -mcp suffix)
}

// Run executes the register subcommand.
// args is os.Args[2:] (everything after "register").
func Run(info ServerInfo, args []string) {
	if len(args) == 0 {
		printUsageAndExit()
	}

	scope := args[0]
	if scope != "project" && scope != "user" {
		fmt.Fprintf(os.Stderr, "Error: unknown scope %q (expected \"project\" or \"user\")\n", scope)
		os.Exit(1)
	}

	directory := "."
	var serverArgs []string

	remaining := args[1:]
	if scope == "project" {
		directory, serverArgs = parseProjectArgs(remaining)
	} else {
		serverArgs = parseUserArgs(remaining)
	}

	binaryPath, err := detectBinaryPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: detecting binary path: %s\n", err)
		os.Exit(1)
	}

	configPath, err := resolveConfigPath(scope, directory)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: resolving config path: %s\n", err)
		os.Exit(1)
	}

	entry := buildEntry(binaryPath, serverArgs)

	if err := writeConfig(configPath, info.Name, entry); err != nil {
		fmt.Fprintf(os.Stderr, "Error: writing config: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Registered %q in %s\n", info.Name, configPath)
}

// parseProjectArgs splits remaining args into directory and server args.
// Format: [directory] [-- args...]
func parseProjectArgs(args []string) (string, []string) {
	directory := "."
	var serverArgs []string

	dashIdx := findStringIndex(args, "--")
	if dashIdx == -1 {
		if len(args) > 0 {
			directory = args[0]
		}
	} else {
		if dashIdx > 0 {
			directory = args[0]
		}
		serverArgs = args[dashIdx+1:]
	}
	return directory, serverArgs
}

// parseUserArgs splits remaining args after "user" into server args.
// Format: [-- args...]
func parseUserArgs(args []string) []string {
	dashIdx := findStringIndex(args, "--")
	if dashIdx == -1 {
		return nil
	}
	return args[dashIdx+1:]
}

func findStringIndex(slice []string, target string) int {
	for i, v := range slice {
		if v == target {
			return i
		}
	}
	return -1
}

func detectBinaryPath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("os.Executable: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", fmt.Errorf("EvalSymlinks: %w", err)
	}
	return resolved, nil
}

// DeriveServerName strips the -mcp suffix and .exe extension from a binary name.
func DeriveServerName(binaryPath string) string {
	name := filepath.Base(binaryPath)
	name = strings.TrimSuffix(name, ".exe")
	name = strings.TrimSuffix(name, "-mcp")
	return name
}

func resolveConfigPath(scope string, directory string) (string, error) {
	if scope == "user" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("UserHomeDir: %w", err)
		}
		return filepath.Join(homeDir, ".claude.json"), nil
	}
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return "", fmt.Errorf("Abs(%s): %w", directory, err)
	}
	return filepath.Join(absDir, ".mcp.json"), nil
}

type mcpServerEntry struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

func buildEntry(binaryPath string, serverArgs []string) mcpServerEntry {
	args := make([]string, len(serverArgs))
	copy(args, serverArgs)

	return mcpServerEntry{
		Command: binaryPath,
		Args:    args,
	}
}

func writeConfig(configPath string, serverName string, entry mcpServerEntry) error {
	config := make(map[string]interface{})

	data, err := os.ReadFile(configPath)
	if err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("parsing %s: %w", configPath, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", configPath, err)
	}

	servers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		servers = make(map[string]interface{})
	}
	servers[serverName] = entry
	config["mcpServers"] = servers

	output, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	// Atomic write: write to temp file then rename to avoid data loss on crash
	tmpFile, err := os.CreateTemp(filepath.Dir(configPath), ".mcp-register-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(append(output, '\n')); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Rename(tmpPath, configPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming %s to %s: %w", tmpPath, configPath, err)
	}
	return nil
}

func printUsageAndExit() {
	bin := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, `Usage:
  %s register project [directory]                          # → <directory>/.mcp.json
  %s register user                                         # → ~/.claude.json
  %s register project . -- --base-url http://localhost:8080 # with forwarded args
`, bin, bin, bin)
	os.Exit(1)
}
