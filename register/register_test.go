package register

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func Test_DeriveServerName(t *testing.T) {
	tests := []struct {
		name       string
		binaryPath string
		want       string
	}{
		{"strip mcp suffix", "/usr/local/bin/rest-api-mcp", "rest-api"},
		{"strip exe and mcp", `C:\bin\rest-api-mcp.exe`, "rest-api"},
		{"no mcp suffix", "/usr/local/bin/myserver", "myserver"},
		{"only exe suffix", `C:\bin\myserver.exe`, "myserver"},
		{"codeindex-mcp", "/bin/codeindex-mcp", "codeindex"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveServerName(tt.binaryPath)
			if got != tt.want {
				t.Errorf("DeriveServerName(%q) = %q, want %q", tt.binaryPath, got, tt.want)
			}
		})
	}
}

func Test_parseProjectArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantDir  string
		wantArgs []string
	}{
		{"no args", nil, ".", nil},
		{"directory only", []string{"./mydir"}, "./mydir", nil},
		{"directory with server args", []string{".", "--", "--base-url", "http://localhost"}, ".", []string{"--base-url", "http://localhost"}},
		{"just dash-dash", []string{"--", "--base-url", "http://localhost"}, ".", []string{"--base-url", "http://localhost"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDir, gotArgs := parseProjectArgs(tt.args)
			if gotDir != tt.wantDir {
				t.Errorf("dir = %q, want %q", gotDir, tt.wantDir)
			}
			if !sliceEqual(gotArgs, tt.wantArgs) {
				t.Errorf("args = %v, want %v", gotArgs, tt.wantArgs)
			}
		})
	}
}

func Test_parseUserArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantArgs []string
	}{
		{"no args", nil, nil},
		{"with server args", []string{"--", "--timeout", "60s"}, []string{"--timeout", "60s"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotArgs := parseUserArgs(tt.args)
			if !sliceEqual(gotArgs, tt.wantArgs) {
				t.Errorf("args = %v, want %v", gotArgs, tt.wantArgs)
			}
		})
	}
}

func Test_writeConfig_CreatesNewFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".mcp.json")

	entry := mcpServerEntry{Command: "/usr/bin/myserver", Args: []string{"--port", "8080"}}
	if err := writeConfig(configPath, "test-server", entry); err != nil {
		t.Fatalf("writeConfig: %s", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile: %s", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("Unmarshal: %s", err)
	}

	servers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("mcpServers not found or wrong type")
	}
	if _, ok := servers["test-server"]; !ok {
		t.Fatal("test-server entry not found")
	}
}

func Test_writeConfig_UpdatesExistingEntry(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".mcp.json")

	initialConfig := `{
  "mcpServers": {
    "existing": {"command": "old", "args": []},
    "test-server": {"command": "old-cmd", "args": ["--old"]}
  }
}
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("WriteFile: %s", err)
	}

	entry := mcpServerEntry{Command: "new-cmd", Args: []string{"--new"}}
	if err := writeConfig(configPath, "test-server", entry); err != nil {
		t.Fatalf("writeConfig: %s", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile: %s", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("Unmarshal: %s", err)
	}

	servers := config["mcpServers"].(map[string]interface{})

	// Existing entry preserved
	if _, ok := servers["existing"]; !ok {
		t.Error("existing entry was removed")
	}

	// Updated entry has new values
	updated := servers["test-server"].(map[string]interface{})
	if updated["command"] != "new-cmd" {
		t.Errorf("command = %v, want new-cmd", updated["command"])
	}
}

func Test_writeConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".mcp.json")

	if err := os.WriteFile(configPath, []byte("not json{{{"), 0644); err != nil {
		t.Fatalf("WriteFile: %s", err)
	}

	entry := mcpServerEntry{Command: "cmd", Args: []string{}}
	err := writeConfig(configPath, "test", entry)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func Test_buildEntry_DirectBinaryCommand(t *testing.T) {
	entry := buildEntry("/usr/bin/rest-api-mcp", []string{"--base-url", "http://localhost"})

	if entry.Command != "/usr/bin/rest-api-mcp" {
		t.Errorf("command = %q, want binary path", entry.Command)
	}
	wantArgs := []string{"--base-url", "http://localhost"}
	if !sliceEqual(entry.Args, wantArgs) {
		t.Errorf("args = %v, want %v", entry.Args, wantArgs)
	}
}

func Test_buildEntry_EmptyArgs(t *testing.T) {
	entry := buildEntry("/bin/myserver", nil)

	if entry.Command != "/bin/myserver" {
		t.Errorf("command = %q, want /bin/myserver", entry.Command)
	}
	if len(entry.Args) != 0 {
		t.Errorf("args = %v, want empty", entry.Args)
	}
}

func Test_buildEntry_DoesNotMutateInput(t *testing.T) {
	original := []string{"--flag", "value"}
	originalCopy := make([]string, len(original))
	copy(originalCopy, original)

	entry := buildEntry("/bin/server", original)
	entry.Args[0] = "mutated"

	if !sliceEqual(original, originalCopy) {
		t.Errorf("buildEntry mutated input slice: got %v, want %v", original, originalCopy)
	}
}

func Test_resolveConfigPath_Project(t *testing.T) {
	tmpDir := t.TempDir()
	got, err := resolveConfigPath("project", tmpDir)
	if err != nil {
		t.Fatalf("resolveConfigPath: %s", err)
	}
	want := filepath.Join(tmpDir, ".mcp.json")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func Test_resolveConfigPath_User(t *testing.T) {
	got, err := resolveConfigPath("user", "")
	if err != nil {
		t.Fatalf("resolveConfigPath: %s", err)
	}
	homeDir, _ := os.UserHomeDir()
	want := filepath.Join(homeDir, ".claude.json")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func sliceEqual(a, b []string) bool {
	if a == nil && b == nil {
		return true
	}
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
