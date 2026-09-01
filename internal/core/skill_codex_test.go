package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexMCP_InjectUpsertRemove(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		t.Fatal(err)
	}

	initial := `model = "gpt-5"
mcp_oauth_credentials_store = "auto"

[mcp_servers.other_server]
command = "other-bin"
args = ["serve"]
enabled = true
`
	if err := os.WriteFile(cfgPath, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	bin := "/Users/usuario/.local/bin/cogni"
	entry := map[string]any{
		"command": bin,
		"args":    []string{"mcp"},
		"type":    "stdio",
	}

	if err := injectMCPServer(cfgPath, "cogni", entry); err != nil {
		t.Fatalf("primera inyección falló: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, `[mcp_servers.cogni]`) {
		t.Errorf("Falta la sección [mcp_servers.cogni]:\n%s", body)
	}
	if !strings.Contains(body, `[mcp_servers.other_server]`) {
		t.Errorf("Se perdió other_server en la inyección:\n%s", body)
	}
	if !strings.Contains(body, `model = "gpt-5"`) {
		t.Errorf("Se perdió clave raíz model en la inyección:\n%s", body)
	}
	if !strings.Contains(body, `command = "/Users/usuario/.local/bin/cogni"`) {
		t.Errorf("Comando mal escrito:\n%s", body)
	}

	// Segunda inyección: debe seguir conteniendo las mismas claves
	if err := injectMCPServer(cfgPath, "cogni", entry); err != nil {
		t.Fatalf("segunda inyección falló: %v", err)
	}

	data2, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data2) != body {
		t.Errorf("La segunda inyección debería ser idempotente:\nAntes:\n%s\nDespués:\n%s", body, string(data2))
	}

	// Remover la entrada
	if err := RemoveCodexMCPServer(cfgPath, "cogni"); err != nil {
		t.Fatalf("remove falló: %v", err)
	}

	data3, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	body3 := string(data3)
	if strings.Contains(body3, `[mcp_servers.cogni]`) {
		t.Errorf("cogni no fue eliminado:\n%s", body3)
	}
	if !strings.Contains(body3, `[mcp_servers.other_server]`) {
		t.Errorf("other_server no debió haberse eliminado:\n%s", body3)
	}
}

func TestCodexMCP_InjectOnEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		t.Fatal(err)
	}

	if err := injectMCPServer(cfgPath, "cogni", map[string]any{
		"command": "/usr/local/bin/cogni",
		"args":    []string{"mcp"},
	}); err != nil {
		t.Fatalf("inyección sobre archivo inexistente falló: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, `[mcp_servers.cogni]`) {
		t.Errorf("Falta [mcp_servers.cogni] tras crear archivo:\n%s", body)
	}
	if !strings.Contains(body, `command = "/usr/local/bin/cogni"`) {
		t.Errorf("Comando mal escrito:\n%s", body)
	}
}

func TestCodexMCP_RemoveOnNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, ".codex", "config.toml")

	if err := RemoveCodexMCPServer(cfgPath, "cogni"); err != nil {
		t.Errorf("RemoveCodexMCPServer debería ser no-op sobre archivo inexistente, got: %v", err)
	}
}
