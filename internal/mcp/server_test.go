package mcp

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestMCPServerTools(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cogni-mcp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldPwd, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer os.Chdir(oldPwd)

	server := NewServer("test-v1")
	tools := server.getToolsList()
	if len(tools) != 7 {
		t.Errorf("Expected 7 tools, got %d", len(tools))
	}

	// 1. Test cogni_save with structured fields (what, why, where, learned)
	saveStructuredArgs := map[string]any{
		"title":     "JWT Refresh Flow",
		"topic_key": "arch/auth/jwt",
		"category":  "architecture",
		"tags":      "auth,jwt,security",
		"what":      "Implemented refresh token rotation",
		"why":       "Security audit",
		"where":     "auth/jwt.go",
		"learned":   "Rotation prevents replay",
		"project":   "test-project",
	}
	saveBytes, _ := json.Marshal(saveStructuredArgs)
	res, isErr := server.executeTool("cogni_save", saveBytes)
	if isErr {
		t.Fatalf("cogni_save failed: %s", res)
	}
	if !strings.Contains(res, "Memoria guardada") {
		t.Errorf("Unexpected cogni_save output: %s", res)
	}

	// 2. Test cogni_search (2-step protocol, step 1)
	searchArgs := map[string]any{
		"query":   "refresh",
		"project": "test-project",
	}
	searchBytes, _ := json.Marshal(searchArgs)
	searchRes, isErr := server.executeTool("cogni_search", searchBytes)
	if isErr {
		t.Fatalf("cogni_search failed: %s", searchRes)
	}
	if !strings.Contains(searchRes, "JWT Refresh Flow") {
		t.Errorf("Expected search result to contain 'JWT Refresh Flow', got: %s", searchRes)
	}

	// 3. Test cogni_get (2-step protocol, step 2) by topic_key
	getArgs := map[string]any{
		"topic_key": "arch/auth/jwt",
		"project":   "test-project",
	}
	getBytes, _ := json.Marshal(getArgs)
	getRes, isErr := server.executeTool("cogni_get", getBytes)
	if isErr {
		t.Fatalf("cogni_get failed: %s", getRes)
	}
	if !strings.Contains(getRes, "What: Implemented refresh token rotation") {
		t.Errorf("Expected get result to contain full structured summary, got: %s", getRes)
	}

	// 4. Test cogni_save upsert behavior with same topic_key
	updateSaveArgs := map[string]any{
		"title":     "JWT Refresh Flow V2",
		"topic_key": "arch/auth/jwt",
		"category":  "architecture",
		"tags":      "auth,jwt,security,v2",
		"summary":   "What: Updated refresh token rotation with Redis blacklist | Why: Scale | Where: auth/jwt.go | Learned: Redis TTL auto cleans",
		"project":   "test-project",
	}
	updateBytes, _ := json.Marshal(updateSaveArgs)
	updateRes, isErr := server.executeTool("cogni_save", updateBytes)
	if isErr {
		t.Fatalf("cogni_save upsert failed: %s", updateRes)
	}

	// Verify get returns updated data
	getRes2, _ := server.executeTool("cogni_get", getBytes)
	if !strings.Contains(getRes2, "JWT Refresh Flow V2") {
		t.Errorf("Expected upserted title in get result, got: %s", getRes2)
	}

	// 5. Test cogni_session_summary
	sessionArgs := map[string]any{
		"goal":          "Integrar autenticación JWT y optimización de contexto",
		"accomplished":  "Endpoints de auth creados, migraciones aplicadas",
		"discoveries":   "Modernc sqlite requiere WAL mode para alta concurrencia",
		"next_steps":    "Escribir tests de integración E2E",
		"relevant_files": "internal/auth/jwt.go, internal/storage/sqlite.go",
		"project":       "test-project",
		"topic_key":     "session/latest",
	}
	sessionBytes, _ := json.Marshal(sessionArgs)
	sessionRes, isErr := server.executeTool("cogni_session_summary", sessionBytes)
	if isErr {
		t.Fatalf("cogni_session_summary failed: %s", sessionRes)
	}
	if !strings.Contains(sessionRes, "Resumen de sesión persistido") {
		t.Errorf("Unexpected session summary output: %s", sessionRes)
	}

	// 6. Test cogni_context
	contextArgs := map[string]any{
		"project": "test-project",
		"limit":   5,
	}
	contextBytes, _ := json.Marshal(contextArgs)
	contextRes, isErr := server.executeTool("cogni_context", contextBytes)
	if isErr {
		t.Fatalf("cogni_context failed: %s", contextRes)
	}
	if !strings.Contains(contextRes, "Contexto Activo Reciente") || !strings.Contains(contextRes, "Resumen de Sesión") {
		t.Errorf("Expected context output to include active session and memories, got: %s", contextRes)
	}
}
