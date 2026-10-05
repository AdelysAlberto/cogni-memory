package mcp

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestMCPServerToolsAndResources(t *testing.T) {
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

	// 1. Test cogni_save with structured fields and tags as array
	saveStructuredArgs := map[string]any{
		"title":     "JWT Refresh Flow",
		"topic_key": "arch/auth/jwt",
		"category":  "refactor",
		"tags":      []string{"auth", "jwt", "security"},
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
	if !strings.Contains(res, "Memory saved") {
		t.Errorf("Unexpected cogni_save output: %s", res)
	}

	// 2. Test cogni_search (2-step protocol, step 1)
	searchArgs := map[string]any{
		"query":    "refresh",
		"project":  "test-project",
		"category": "refactor",
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
	if !strings.Contains(getRes, "Category: refactor") {
		t.Errorf("Expected persisted refactor category, got: %s", getRes)
	}

	// 4. Test cogni_save upsert behavior with same topic_key
	updateSaveArgs := map[string]any{
		"title":     "JWT Refresh Flow V2",
		"topic_key": "arch/auth/jwt",
		"category":  "refactor",
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
	getRes2, isErr := server.executeTool("cogni_get", getBytes)
	if isErr || !strings.Contains(getRes2, "Category: refactor") {
		t.Fatalf("Expected persisted refactor category after upsert, got: %s", getRes2)
	}
	if !strings.Contains(getRes2, "JWT Refresh Flow V2") {
		t.Errorf("Expected upserted title in get result, got: %s", getRes2)
	}

	// 5. Test cogni_session_summary
	sessionArgs := map[string]any{
		"goal":           "Integrar autenticación JWT y optimización de contexto",
		"accomplished":   "Endpoints de auth creados, migraciones aplicadas",
		"discoveries":    "Modernc sqlite requiere WAL mode para alta concurrencia",
		"next_steps":     "Escribir tests de integración E2E",
		"relevant_files": "internal/auth/jwt.go, internal/storage/sqlite.go",
		"project":        "test-project",
		"topic_key":      "session/latest",
	}
	sessionBytes, _ := json.Marshal(sessionArgs)
	sessionRes, isErr := server.executeTool("cogni_session_summary", sessionBytes)
	if isErr {
		t.Fatalf("cogni_session_summary failed: %s", sessionRes)
	}
	if !strings.Contains(sessionRes, "Session summary persisted") {
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
	if !strings.Contains(contextRes, "Active Context for") {
		t.Errorf("Expected context output to include active session and memories, got: %s", contextRes)
	}

	// 7. Test MCP Resources list and read
	resources := server.getResourcesList()
	if len(resources) != 2 {
		t.Errorf("Expected 2 resources, got %d", len(resources))
	}

	templates := server.getResourceTemplatesList()
	if len(templates) != 2 {
		t.Errorf("Expected 2 resource templates, got %d", len(templates))
	}

	// Read cogni://context/recent
	resContext, err := server.readResource("cogni://context/recent")
	if err != nil {
		t.Fatalf("readResource(cogni://context/recent) failed: %v", err)
	}
	if len(resContext.Contents) == 0 || !strings.Contains(resContext.Contents[0].Text, "JWT Refresh Flow V2") {
		t.Errorf("Expected recent context resource to include memory, got: %+v", resContext)
	}

	// Read cogni://session/latest
	resSession, err := server.readResource("cogni://session/latest")
	if err != nil {
		t.Fatalf("readResource(cogni://session/latest) failed: %v", err)
	}
	if len(resSession.Contents) == 0 || !strings.Contains(resSession.Contents[0].Text, "session/latest") {
		t.Errorf("Expected session resource to include session/latest, got: %+v", resSession)
	}

	// Read cogni://topic/arch/auth/jwt
	resTopic, err := server.readResource("cogni://topic/arch/auth/jwt")
	if err != nil {
		t.Fatalf("readResource(cogni://topic/arch/auth/jwt) failed: %v", err)
	}
	if len(resTopic.Contents) == 0 || !strings.Contains(resTopic.Contents[0].Text, "JWT Refresh Flow V2") {
		t.Errorf("Expected topic resource to include memory, got: %+v", resTopic)
	}

	// 8. Test MCP Prompts list and get
	prompts := server.getPromptsList()
	if len(prompts) != 2 {
		t.Errorf("Expected 2 prompts, got %d", len(prompts))
	}

	pPreflight, err := server.getPrompt("cogni_preflight_check", map[string]string{
		"task":    "Auth JWT Middleware",
		"project": "test-project",
	})
	if err != nil {
		t.Fatalf("getPrompt(cogni_preflight_check) failed: %v", err)
	}
	if len(pPreflight.Messages) == 0 || !strings.Contains(pPreflight.Messages[0].Content.Text, "Auth JWT Middleware") {
		t.Errorf("Expected prompt messages to include task, got: %+v", pPreflight)
	}

	pSession, err := server.getPrompt("cogni_session_summary", map[string]string{
		"goal":         "Complete Phase 1",
		"accomplished": "Built MCP Resources",
		"discoveries":  "None",
		"next_steps":   "Test Phase 2",
	})
	if err != nil {
		t.Fatalf("getPrompt(cogni_session_summary) failed: %v", err)
	}
	if len(pSession.Messages) == 0 || !strings.Contains(pSession.Messages[0].Content.Text, "Complete Phase 1") {
		t.Errorf("Expected session prompt to include goal, got: %+v", pSession)
	}
}

func TestMCPServerJSONRPCProtocol(t *testing.T) {
	server := NewServer("test-v2")
	var buf strings.Builder

	// 1. Test initialize
	initReq := &Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}
	server.handleRequest(&buf, initReq)

	var initResp Response
	if err := json.Unmarshal([]byte(buf.String()), &initResp); err != nil {
		t.Fatalf("Failed to parse initialize response: %v", err)
	}
	resultMap, ok := initResp.Result.(map[string]any)
	if !ok {
		t.Fatalf("Expected resultMap in initialize, got: %+v", initResp.Result)
	}
	if resultMap["protocolVersion"] != "2024-11-05" {
		t.Errorf("Expected protocolVersion 2024-11-05, got %v", resultMap["protocolVersion"])
	}
	caps, ok := resultMap["capabilities"].(map[string]any)
	if !ok || caps["tools"] == nil || caps["resources"] == nil || caps["prompts"] == nil {
		t.Errorf("Expected tools, resources, and prompts in capabilities, got: %+v", caps)
	}

	// 2. Test ping
	buf.Reset()
	server.handleRequest(&buf, &Request{JSONRPC: "2.0", ID: 2, Method: "ping"})
	if !strings.Contains(buf.String(), `"id":2`) {
		t.Errorf("Unexpected ping response: %s", buf.String())
	}

	// 3. Test unknown method
	buf.Reset()
	server.handleRequest(&buf, &Request{JSONRPC: "2.0", ID: 3, Method: "unknown_method"})
	if !strings.Contains(buf.String(), `-32601`) {
		t.Errorf("Expected -32601 Method Not Found error, got: %s", buf.String())
	}
}
