package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInjectAgentDirectives_PreservesExistingContentAndIdempotent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cogni-directives-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	agentFile := filepath.Join(tmpDir, "AGENTS.md")
	initialContent := "# Pre-existing Project Rules\n\n- Rule 1: Always write tests\n- Rule 2: Zero technical debt\n"
	if err := os.WriteFile(agentFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write initial AGENTS.md: %v", err)
	}

	// First injection
	if err := injectDirectivesToFile(agentFile); err != nil {
		t.Fatalf("First injection failed: %v", err)
	}

	data1, err := os.ReadFile(agentFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	s1 := string(data1)

	// Verify existing content is intact
	if !strings.Contains(s1, "# Pre-existing Project Rules") {
		t.Errorf("Existing content header was deleted or lost")
	}
	if !strings.Contains(s1, "- Rule 1: Always write tests") {
		t.Errorf("Existing rule 1 was deleted or lost")
	}
	if !strings.Contains(s1, "- Rule 2: Zero technical debt") {
		t.Errorf("Existing rule 2 was deleted or lost")
	}
	// Verify cogni protocol block was added
	if !strings.Contains(s1, "<!-- cogni:protocol:start -->") || !strings.Contains(s1, "<!-- cogni:protocol:end -->") {
		t.Errorf("Cogni protocol markers missing")
	}
	if strings.Count(s1, "<!-- cogni:protocol:start -->") != 1 {
		t.Errorf("Expected exactly 1 start marker, got %d", strings.Count(s1, "<!-- cogni:protocol:start -->"))
	}

	// Second injection (Idempotency test)
	if err := injectDirectivesToFile(agentFile); err != nil {
		t.Fatalf("Second injection failed: %v", err)
	}

	data2, err := os.ReadFile(agentFile)
	if err != nil {
		t.Fatalf("Failed to read file after second injection: %v", err)
	}
	s2 := string(data2)

	if strings.Count(s2, "<!-- cogni:protocol:start -->") != 1 {
		t.Errorf("Second injection duplicated the block: expected 1 marker, got %d", strings.Count(s2, "<!-- cogni:protocol:start -->"))
	}
	if !strings.Contains(s2, "# Pre-existing Project Rules") {
		t.Errorf("Existing content was lost on second injection")
	}
}

func TestInjectMCPServer_PreservesExistingServers(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cogni-mcp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mcpFile := filepath.Join(tmpDir, "mcp.json")
	initialJSON := `{
  "mcpServers": {
    "postgres": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-postgres", "postgresql://localhost:5432"]
    },
    "pencil": {
      "command": "/Applications/Pencil.app/server",
      "args": ["--app", "desktop"]
    }
  }
}`
	if err := os.WriteFile(mcpFile, []byte(initialJSON), 0644); err != nil {
		t.Fatalf("Failed to write initial mcp.json: %v", err)
	}

	cogniConfig := map[string]any{
		"command": "/Users/adelysalberto/.local/bin/cogni",
		"args":    []string{"mcp"},
	}

	if err := injectMCPServer(mcpFile, "cogni", cogniConfig); err != nil {
		t.Fatalf("injectMCPServer failed: %v", err)
	}

	data, err := os.ReadFile(mcpFile)
	if err != nil {
		t.Fatalf("Failed to read updated mcp.json: %v", err)
	}

	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	servers, ok := root["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("mcpServers is missing or not a map")
	}

	// Verify postgres is intact
	if _, hasPostgres := servers["postgres"]; !hasPostgres {
		t.Errorf("CRITICAL: Existing server 'postgres' was deleted!")
	}
	// Verify pencil is intact
	if _, hasPencil := servers["pencil"]; !hasPencil {
		t.Errorf("CRITICAL: Existing server 'pencil' was deleted!")
	}
	// Verify cogni was added
	if _, hasCogni := servers["cogni"]; !hasCogni {
		t.Errorf("Server 'cogni' was not added")
	}
}
