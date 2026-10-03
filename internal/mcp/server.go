package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AdelysAlberto/cogni/internal/core"
	"github.com/AdelysAlberto/cogni/internal/storage"
)

// JSON-RPC 2.0 Structures
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type FlexTags string

func (ft *FlexTags) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*ft = FlexTags(str)
		return nil
	}
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*ft = FlexTags(strings.Join(arr, ","))
		return nil
	}
	return nil
}

// MCP Tools Structures
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema *JSONSchema `json:"inputSchema"`
}

type JSONSchema struct {
	Type                 string              `json:"type"`
	Properties           map[string]Property `json:"properties"`
	Required             []string            `json:"required,omitempty"`
	AdditionalProperties *bool               `json:"additionalProperties,omitempty"`
}

type Property struct {
	Type        string     `json:"type,omitempty"`
	Description string     `json:"description,omitempty"`
	Enum        []string   `json:"enum,omitempty"`
	Items       *Property  `json:"items,omitempty"`
	OneOf       []Property `json:"oneOf,omitempty"`
	Default     any        `json:"default,omitempty"`
}

type CallToolResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// MCP Resources Structures
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mimeType,omitempty"`
}

type ResourceTemplate struct {
	URITemplate string `json:"uriTemplate"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mimeType,omitempty"`
}

type ResourceContent struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}

type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents"`
}

// MCP Prompts Structures
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

type PromptMessage struct {
	Role    string      `json:"role"`
	Content ToolContent `json:"content"`
}

type GetPromptResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// Server implements the full MCP stdio server for Cogni (Tools, Resources, Prompts)
type Server struct {
	version string
}

func NewServer(version string) *Server {
	return &Server{version: version}
}

func (s *Server) Run() error {
	reader := bufio.NewReader(os.Stdin)
	writer := os.Stdout

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		line = []byte(strings.TrimSpace(string(line)))
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(writer, nil, -32700, "Parse error: "+err.Error())
			continue
		}

		s.handleRequest(writer, &req)
	}
}

func (s *Server) handleRequest(w io.Writer, req *Request) {
	switch req.Method {
	case "initialize":
		result := map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{
					"listChanged": false,
				},
				"resources": map[string]any{
					"subscribe":   false,
					"listChanged": false,
				},
				"prompts": map[string]any{
					"listChanged": false,
				},
			},
			"serverInfo": map[string]any{
				"name":    "cogni-mcp",
				"version": s.version,
			},
		}
		s.sendResult(w, req.ID, result)

	case "notifications/initialized", "initialized":
		return

	case "ping":
		s.sendResult(w, req.ID, map[string]any{})

	// 1. MCP Tools
	case "tools/list":
		tools := s.getToolsList()
		s.sendResult(w, req.ID, map[string]any{
			"tools": tools,
		})

	case "tools/call":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendError(w, req.ID, -32602, "Invalid params: "+err.Error())
			return
		}

		result, isErr := s.executeTool(params.Name, params.Arguments)
		s.sendResult(w, req.ID, CallToolResult{
			Content: []ToolContent{{Type: "text", Text: result}},
			IsError: isErr,
		})

	// 2. MCP Resources
	case "resources/list":
		resources := s.getResourcesList()
		s.sendResult(w, req.ID, map[string]any{
			"resources": resources,
		})

	case "resources/templates/list":
		templates := s.getResourceTemplatesList()
		s.sendResult(w, req.ID, map[string]any{
			"resourceTemplates": templates,
		})

	case "resources/read":
		var params struct {
			URI string `json:"uri"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendError(w, req.ID, -32602, "Invalid params: "+err.Error())
			return
		}

		res, err := s.readResource(params.URI)
		if err != nil {
			s.sendError(w, req.ID, -32002, "Resource read error: "+err.Error())
			return
		}
		s.sendResult(w, req.ID, res)

	// 3. MCP Prompts
	case "prompts/list":
		prompts := s.getPromptsList()
		s.sendResult(w, req.ID, map[string]any{
			"prompts": prompts,
		})

	case "prompts/get":
		var params struct {
			Name      string            `json:"name"`
			Arguments map[string]string `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendError(w, req.ID, -32602, "Invalid params: "+err.Error())
			return
		}

		res, err := s.getPrompt(params.Name, params.Arguments)
		if err != nil {
			s.sendError(w, req.ID, -32602, "Prompt error: "+err.Error())
			return
		}
		s.sendResult(w, req.ID, res)

	default:
		s.sendError(w, req.ID, -32601, fmt.Sprintf("Method '%s' not found", req.Method))
	}
}

func (s *Server) sendResult(w io.Writer, id any, result any) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	data, _ := json.Marshal(resp)
	_, _ = fmt.Fprintf(w, "%s\n", string(data))
}

func (s *Server) sendError(w io.Writer, id any, code int, message string) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
	data, _ := json.Marshal(resp)
	_, _ = fmt.Fprintf(w, "%s\n", string(data))
}

func (s *Server) getToolsList() []Tool {
	additionalPropsFalse := false

	return []Tool{
		{
			Name: "cogni_search",
			Description: "Search synthetic memories lightly (returns compact previews, IDs, and tags). " +
				"Use during Preflight Checks before designing or fixing components to recall prior patterns without inflating context.",
			InputSchema: &JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {
						Type:        "string",
						Description: "Semantic search query or keywords to match against memory signatures.",
					},
					"project": {
						Type:        "string",
						Description: "Project name (optional, automatically detected if omitted).",
					},
					"category": {
						Type:        "string",
						Description: "Filter by memory category (bugfix, architecture, decision, discovery, config, pattern, preference, general).",
						Enum:        []string{"bugfix", "architecture", "decision", "discovery", "config", "pattern", "preference", "general"},
					},
					"all_projects": {
						Type:        "boolean",
						Description: "Search across all projects ignoring project boundaries (default: false).",
						Default:     false,
					},
					"limit": {
						Type:        "integer",
						Description: "Maximum number of results to return (default: 5).",
						Default:     5,
					},
				},
				Required:             []string{"query"},
				AdditionalProperties: &additionalPropsFalse,
			},
		},
		{
			Name: "cogni_get",
			Description: "Retrieve the full synthetic memory signature by numeric ID or deterministic TopicKey (2-Step Retrieval Protocol). " +
				"Use after cogni_search to hydrate only the specific record needed.",
			InputSchema: &JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {
						Type:        "integer",
						Description: "Numeric ID of the memory to fetch.",
					},
					"topic_key": {
						Type:        "string",
						Description: "Deterministic topic key (e.g. 'arch/auth/jwt', 'pattern/react/modals').",
					},
					"project": {
						Type:        "string",
						Description: "Project name (optional when fetching by topic_key).",
					},
				},
				AdditionalProperties: &additionalPropsFalse,
			},
		},
		{
			Name: "cogni_save",
			Description: "Save or upsert a high-density synthetic memory signature into local or global storage. " +
				"Accepts structured discrete fields (what, why, where, learned) or a pre-built signature string: 'What: ... | Why: ... | Where: ... | Learned: ...'.",
			InputSchema: &JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"title": {
						Type:        "string",
						Description: "Concise title describing the milestone, decision, or discovery.",
					},
					"summary": {
						Type:        "string",
						Description: "Synthetic signature in format: 'What: ... | Why: ... | Where: ... | Learned: ...' (optional if discrete fields what/why/where/learned are provided).",
					},
					"what": {
						Type:        "string",
						Description: "Single descriptive sentence of what was implemented or resolved.",
					},
					"why": {
						Type:        "string",
						Description: "Motivation or root cause behind the change.",
					},
					"where": {
						Type:        "string",
						Description: "Key affected file paths or modules.",
					},
					"learned": {
						Type:        "string",
						Description: "Edge cases, gotchas, or lessons learned.",
					},
					"category": {
						Type:        "string",
						Description: "Memory classification category.",
						Enum:        []string{"bugfix", "architecture", "decision", "discovery", "config", "pattern", "preference", "general"},
					},
					"tags": {
						Description: "Tags in 3 tiers. Can be a comma-separated string (e.g. 'auth,jwt,tokens') or an array of tag strings.",
						OneOf: []Property{
							{Type: "string"},
							{Type: "array", Items: &Property{Type: "string"}},
						},
					},
					"topic_key": {
						Type:        "string",
						Description: "Deterministic topic key to enable automatic upsert without duplication (e.g. 'arch/auth/jwt', 'spec/storage/wal').",
					},
					"project": {
						Type:        "string",
						Description: "Project name (optional, automatically detected).",
					},
					"global": {
						Type:        "boolean",
						Description: "If true, saves into ~/.cogni/memory.db (global cross-project). Default is false (project-local .cogni/).",
						Default:     false,
					},
				},
				Required:             []string{"title", "category", "tags"},
				AdditionalProperties: &additionalPropsFalse,
			},
		},
		{
			Name: "cogni_session_summary",
			Description: "Save or upsert a structured session milestone or post-compaction summary (Goal, Accomplished, Discoveries, Next Steps, Relevant Files). " +
				"Vital for maintaining continuity between agent sessions and recovering state after context compaction.",
			InputSchema: &JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"goal": {
						Type:        "string",
						Description: "Primary objective or task worked on during the session.",
					},
					"accomplished": {
						Type:        "string",
						Description: "Completed milestones and code changes with technical details.",
					},
					"discoveries": {
						Type:        "string",
						Description: "Technical findings, architectural decisions, or gotchas discovered.",
					},
					"next_steps": {
						Type:        "string",
						Description: "Pending tasks or recommendations for the subsequent session.",
					},
					"relevant_files": {
						Type:        "string",
						Description: "Key modified or created files.",
					},
					"instructions": {
						Type:        "string",
						Description: "User preferences or constraints learned during the session.",
					},
					"topic_key": {
						Type:        "string",
						Description: "Deterministic key for the session (default: 'session/latest').",
						Default:     "session/latest",
					},
					"project": {
						Type:        "string",
						Description: "Project name (optional).",
					},
					"tags": {
						Type:        "string",
						Description: "Additional comma-separated tags.",
					},
					"global": {
						Type:        "boolean",
						Description: "Save into global storage (~/.cogni/). Default is false (local).",
						Default:     false,
					},
				},
				Required:             []string{"goal", "accomplished"},
				AdditionalProperties: &additionalPropsFalse,
			},
		},
		{
			Name: "cogni_context",
			Description: "Quickly retrieve recent active context for the project (previous sessions, architecture decisions, and bugfixes) " +
				"to bootstrap sessions with high signal and minimal tokens (< 100 tokens).",
			InputSchema: &JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"project": {
						Type:        "string",
						Description: "Project name (optional, automatically detected).",
					},
					"limit": {
						Type:        "integer",
						Description: "Maximum number of recent memories to return (default: 5).",
						Default:     5,
					},
				},
				AdditionalProperties: &additionalPropsFalse,
			},
		},
		{
			Name:        "cogni_update",
			Description: "Update fields of an existing synthetic memory by its numeric ID.",
			InputSchema: &JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {
						Type:        "integer",
						Description: "Numeric ID of the memory to update.",
					},
					"summary": {
						Type:        "string",
						Description: "Updated synthetic signature string.",
					},
					"title": {
						Type:        "string",
						Description: "Updated title.",
					},
					"category": {
						Type:        "string",
						Description: "Updated category.",
					},
					"tags": {
						Type:        "string",
						Description: "Updated tags.",
					},
					"topic_key": {
						Type:        "string",
						Description: "Updated deterministic topic key.",
					},
				},
				Required:             []string{"id"},
				AdditionalProperties: &additionalPropsFalse,
			},
		},
		{
			Name:        "cogni_stats",
			Description: "Retrieve memory usage statistics and estimated tokens saved across sessions.",
			InputSchema: &JSONSchema{
				Type:                 "object",
				Properties:           map[string]Property{},
				AdditionalProperties: &additionalPropsFalse,
			},
		},
	}
}

// Resources Implementation
func (s *Server) getResourcesList() []Resource {
	return []Resource{
		{
			URI:         "cogni://context/recent",
			Name:        "recent-project-context",
			Description: "Active context, recent architectural decisions, and bugfix signatures for the current workspace",
			MIMEType:    "application/json",
		},
		{
			URI:         "cogni://session/latest",
			Name:        "latest-session-summary",
			Description: "Latest recorded session summary and pending next steps for the current workspace",
			MIMEType:    "application/json",
		},
	}
}

func (s *Server) getResourceTemplatesList() []ResourceTemplate {
	return []ResourceTemplate{
		{
			URITemplate: "cogni://memory/{id}",
			Name:        "memory-by-id",
			Description: "Retrieve full synthetic memory signature by numeric ID",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "cogni://topic/{topic_key}",
			Name:        "memory-by-topic-key",
			Description: "Retrieve full synthetic memory signature by deterministic TopicKey",
			MIMEType:    "application/json",
		},
	}
}

func (s *Server) readResource(rawURI string) (*ReadResourceResult, error) {
	localStorage, globalStorage := s.getStorages()
	if localStorage != nil {
		defer localStorage.Close()
	}
	if globalStorage != nil {
		defer globalStorage.Close()
	}

	parsedURI, _ := url.Parse(rawURI)
	var queryProject string
	if parsedURI != nil {
		queryProject = parsedURI.Query().Get("project")
	}

	project := queryProject
	if project == "" {
		project = core.DetectProjectName()
	}
	if project == "/" || project == "." || project == "default_project" {
		project = ""
	}

	cleanURI := rawURI
	if parsedURI != nil {
		cleanURI = fmt.Sprintf("%s://%s%s", parsedURI.Scheme, parsedURI.Host, parsedURI.Path)
	}

	switch {
	case cleanURI == "cogni://context/recent":
		var memories []core.Memory
		if localStorage != nil {
			mems, _ := localStorage.GetRecentContext(project, 5)
			if len(mems) == 0 && project != "" {
				// Fallback to all memories in project-local storage
				mems, _ = localStorage.GetRecentContext("", 5)
			}
			memories = append(memories, mems...)
		}
		if globalStorage != nil && len(memories) < 5 {
			mems, _ := globalStorage.GetRecentContext(project, 5-len(memories))
			memories = append(memories, mems...)
		}
		data, err := json.MarshalIndent(memories, "", "  ")
		if err != nil {
			return nil, err
		}
		return &ReadResourceResult{
			Contents: []ResourceContent{
				{
					URI:      rawURI,
					MIMEType: "application/json",
					Text:     string(data),
				},
			},
		}, nil

	case cleanURI == "cogni://session/latest":
		var mem *core.Memory
		if localStorage != nil {
			mem, _ = localStorage.GetMemoryByTopicKey(project, "session/latest")
			if mem == nil && project != "" {
				mem, _ = localStorage.GetMemoryByTopicKey("", "session/latest")
			}
		}
		if mem == nil && globalStorage != nil {
			mem, _ = globalStorage.GetMemoryByTopicKey(project, "session/latest")
			if mem == nil && project != "" {
				mem, _ = globalStorage.GetMemoryByTopicKey("", "session/latest")
			}
		}
		if mem == nil {
			return &ReadResourceResult{
				Contents: []ResourceContent{
					{
						URI:      rawURI,
						MIMEType: "application/json",
						Text:     `{"status": "no_recent_session"}`,
					},
				},
			}, nil
		}
		data, err := json.MarshalIndent(mem, "", "  ")
		if err != nil {
			return nil, err
		}
		return &ReadResourceResult{
			Contents: []ResourceContent{
				{
					URI:      rawURI,
					MIMEType: "application/json",
					Text:     string(data),
				},
			},
		}, nil

	case strings.HasPrefix(cleanURI, "cogni://memory/"):
		idStr := strings.TrimPrefix(cleanURI, "cogni://memory/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid memory ID in URI '%s': %w", rawURI, err)
		}
		var mem *core.Memory
		if localStorage != nil {
			mem, _ = localStorage.GetMemoryByID(id)
		}
		if mem == nil && globalStorage != nil {
			mem, _ = globalStorage.GetMemoryByID(id)
		}
		if mem == nil {
			return nil, fmt.Errorf("memory #%d not found", id)
		}
		data, err := json.MarshalIndent(mem, "", "  ")
		if err != nil {
			return nil, err
		}
		return &ReadResourceResult{
			Contents: []ResourceContent{
				{
					URI:      rawURI,
					MIMEType: "application/json",
					Text:     string(data),
				},
			},
		}, nil

	case strings.HasPrefix(cleanURI, "cogni://topic/"):
		topicKey := strings.TrimPrefix(cleanURI, "cogni://topic/")
		var mem *core.Memory
		if localStorage != nil {
			mem, _ = localStorage.GetMemoryByTopicKey(project, topicKey)
			if mem == nil && project != "" {
				mem, _ = localStorage.GetMemoryByTopicKey("", topicKey)
			}
		}
		if mem == nil && globalStorage != nil {
			mem, _ = globalStorage.GetMemoryByTopicKey(project, topicKey)
			if mem == nil && project != "" {
				mem, _ = globalStorage.GetMemoryByTopicKey("", topicKey)
			}
		}
		if mem == nil {
			return nil, fmt.Errorf("memory for topic_key '%s' not found", topicKey)
		}
		data, err := json.MarshalIndent(mem, "", "  ")
		if err != nil {
			return nil, err
		}
		return &ReadResourceResult{
			Contents: []ResourceContent{
				{
					URI:      rawURI,
					MIMEType: "application/json",
					Text:     string(data),
				},
			},
		}, nil

	default:
		return nil, fmt.Errorf("unknown resource URI: %s", rawURI)
	}
}

// Prompts Implementation
func (s *Server) getPromptsList() []Prompt {
	return []Prompt{
		{
			Name:        "cogni_preflight_check",
			Description: "Prompt template to execute preflight memory search before designing or implementing a feature or bugfix",
			Arguments: []PromptArgument{
				{
					Name:        "task",
					Description: "Description or technical domain of the task to be performed",
					Required:    true,
				},
				{
					Name:        "project",
					Description: "Project name (optional, automatically detected)",
					Required:    false,
				},
			},
		},
		{
			Name:        "cogni_session_summary",
			Description: "Prompt template to guide creating and persisting an end-of-session or post-compaction summary into Cogni",
			Arguments: []PromptArgument{
				{
					Name:        "goal",
					Description: "Primary objective worked on",
					Required:    true,
				},
				{
					Name:        "accomplished",
					Description: "Tasks, milestones, and code changes completed",
					Required:    true,
				},
				{
					Name:        "discoveries",
					Description: "Gotchas, lessons learned, or architectural decisions",
					Required:    false,
				},
				{
					Name:        "next_steps",
					Description: "Pending items for the next session",
					Required:    false,
				},
			},
		},
	}
}

func (s *Server) getPrompt(name string, args map[string]string) (*GetPromptResult, error) {
	switch name {
	case "cogni_preflight_check":
		task := args["task"]
		if task == "" {
			return nil, fmt.Errorf("missing required argument 'task'")
		}
		project := args["project"]
		if project == "" {
			project = core.DetectProjectName()
		}

		promptText := fmt.Sprintf(
			"Please perform a Cogni Preflight Memory Check for project '%s'.\n"+
				"1. Call cogni_search(query: \"%s\", project: \"%s\") to check for existing architecture decisions, conventions, or bugfixes.\n"+
				"2. If relevant records are found, call cogni_get with the ID or TopicKey to hydrate the complete signature.\n"+
				"3. Adhere strictly to the retrieved architectural invariants before proceeding with implementation.",
			project, task, project,
		)

		return &GetPromptResult{
			Description: "Preflight Check instructions for Cogni memory retrieval",
			Messages: []PromptMessage{
				{
					Role: "user",
					Content: ToolContent{
						Type: "text",
						Text: promptText,
					},
				},
			},
		}, nil

	case "cogni_session_summary":
		goal := args["goal"]
		accomplished := args["accomplished"]
		if goal == "" || accomplished == "" {
			return nil, fmt.Errorf("arguments 'goal' and 'accomplished' are required")
		}
		discoveries := args["discoveries"]
		nextSteps := args["next_steps"]

		promptText := fmt.Sprintf(
			"Persist session progress to Cogni by calling cogni_session_summary with:\n"+
				"- goal: %s\n"+
				"- accomplished: %s\n"+
				"- discoveries: %s\n"+
				"- next_steps: %s\n"+
				"- topic_key: 'session/latest'",
			goal, accomplished, discoveries, nextSteps,
		)

		return &GetPromptResult{
			Description: "Session Summary persistence prompt",
			Messages: []PromptMessage{
				{
					Role: "user",
					Content: ToolContent{
						Type: "text",
						Text: promptText,
					},
				},
			},
		}, nil

	default:
		return nil, fmt.Errorf("unknown prompt: %s", name)
	}
}

func (s *Server) getStorages() (*storage.Storage, *storage.Storage) {
	globalPath := core.ResolveDatabasePath("", true)
	globalStorage, _ := storage.NewWithSource(globalPath, "global")

	var localStorage *storage.Storage
	localDir := core.FindLocalCogniDir()
	if localDir != "" {
		localPath := filepath.Join(localDir, "memory.db")
		if filepath.Clean(localPath) != filepath.Clean(globalPath) {
			localStorage, _ = storage.NewWithSource(localPath, "local")
		}
	}

	return localStorage, globalStorage
}

func (s *Server) executeTool(name string, argsRaw json.RawMessage) (string, bool) {
	localStorage, globalStorage := s.getStorages()
	if localStorage != nil {
		defer localStorage.Close()
	}
	if globalStorage != nil {
		defer globalStorage.Close()
	}

	switch name {
	case "cogni_search":
		var args struct {
			Query       string `json:"query"`
			Project     string `json:"project"`
			Category    string `json:"category"`
			Limit       int    `json:"limit"`
			AllProjects bool   `json:"all_projects"`
		}
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return "Error parsing arguments: " + err.Error(), true
		}
		if args.Limit <= 0 {
			args.Limit = 5
		}
		project := args.Project
		if project == "" && !args.AllProjects {
			project = core.DetectProjectName()
		}
		if project == "/" || project == "." || project == "default_project" {
			project = ""
		}

		var results []core.Memory
		seen := make(map[int64]bool)

		if localStorage != nil {
			res, _ := localStorage.SearchMemoriesAdvanced(project, args.Query, args.Category, args.Limit, args.AllProjects)
			for _, m := range res {
				if !seen[m.ID] {
					seen[m.ID] = true
					results = append(results, m)
				}
			}
		}
		if globalStorage != nil && len(results) < args.Limit {
			res, _ := globalStorage.SearchMemoriesAdvanced(project, args.Query, args.Category, args.Limit-len(results), args.AllProjects)
			for _, m := range res {
				if !seen[m.ID] {
					seen[m.ID] = true
					results = append(results, m)
				}
			}
		}

		if len(results) == 0 {
			return "No memories found for query: " + args.Query, false
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("🔍 Found %d memory signature(s) [Use cogni_get with ID or TopicKey to view full signature]:\n\n", len(results)))
		for _, m := range results {
			srcBadge := "LOCAL"
			if m.Source == "global" {
				srcBadge = "GLOBAL"
			}
			topicStr := ""
			if m.TopicKey != "" {
				topicStr = fmt.Sprintf(" | Key: %s", m.TopicKey)
			}
			preview := m.SummarySignature
			if idx := strings.Index(preview, "|"); idx != -1 {
				preview = strings.TrimSpace(preview[:idx])
			} else if len(preview) > 90 {
				preview = preview[:87] + "..."
			}

			sb.WriteString(fmt.Sprintf("- [#%d %s] [%s] %s%s\n  Tags: %s | %s\n",
				m.ID, srcBadge, m.Category, m.Title, topicStr, m.Tags, preview))
		}
		return sb.String(), false

	case "cogni_get":
		var args struct {
			ID       int64  `json:"id"`
			TopicKey string `json:"topic_key"`
			Project  string `json:"project"`
		}
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return "Error parsing arguments: " + err.Error(), true
		}

		project := args.Project
		if project == "" {
			project = core.DetectProjectName()
		}

		var mem *core.Memory
		var err error

		if args.ID > 0 {
			if localStorage != nil {
				mem, err = localStorage.GetMemoryByID(args.ID)
			}
			if mem == nil && globalStorage != nil {
				mem, err = globalStorage.GetMemoryByID(args.ID)
			}
		} else if args.TopicKey != "" {
			if localStorage != nil {
				mem, err = localStorage.GetMemoryByTopicKey(project, args.TopicKey)
			}
			if mem == nil && globalStorage != nil {
				mem, err = globalStorage.GetMemoryByTopicKey(project, args.TopicKey)
			}
		} else {
			return "Error: You must specify either 'id' or 'topic_key'.", true
		}

		if err != nil {
			return "Error retrieving memory: " + err.Error(), true
		}
		if mem == nil {
			return "Memory signature not found.", false
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("━━━ [#%d] [%s] %s ━━━\n", mem.ID, mem.ProjectName, mem.Title))
		if mem.TopicKey != "" {
			sb.WriteString(fmt.Sprintf("🔑 Topic Key: %s\n", mem.TopicKey))
		}
		sb.WriteString(fmt.Sprintf("📂 Category: %s | 🏷️ Tags: %s\n", mem.Category, mem.Tags))
		sb.WriteString(fmt.Sprintf("📅 Date: %s\n\n", mem.UpdatedAt.Format("2006-01-02 15:04:05")))
		sb.WriteString(fmt.Sprintf("📝 %s\n", mem.SummarySignature))

		return sb.String(), false

	case "cogni_save":
		var args struct {
			Title    string   `json:"title"`
			Summary  string   `json:"summary"`
			What     string   `json:"what"`
			Why      string   `json:"why"`
			Where    string   `json:"where"`
			Learned  string   `json:"learned"`
			Category string   `json:"category"`
			Tags     FlexTags `json:"tags"`
			TopicKey string   `json:"topic_key"`
			Project  string   `json:"project"`
			Global   bool     `json:"global"`
		}
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return "Error parsing arguments: " + err.Error(), true
		}

		if args.Title == "" {
			return "Error: 'title' is required.", true
		}

		finalSummary := args.Summary
		if finalSummary == "" && (args.What != "" || args.Why != "" || args.Where != "" || args.Learned != "") {
			finalSummary = core.BuildSummarySignature(args.What, args.Why, args.Where, args.Learned)
		}

		if finalSummary == "" {
			return "Error: You must provide either 'summary' or discrete fields 'what', 'why', 'where', 'learned'.", true
		}

		if args.Category == "" {
			args.Category = "general"
		}

		project := args.Project
		if project == "" {
			project = core.DetectProjectName()
		}

		var targetStorage *storage.Storage
		if args.Global {
			targetStorage = globalStorage
		} else {
			if localStorage == nil {
				localDir := filepath.Join(".", ".cogni")
				_ = os.MkdirAll(localDir, 0755)
				localPath := filepath.Join(localDir, "memory.db")
				localStorage, _ = storage.NewWithSource(localPath, "local")
			}
			targetStorage = localStorage
		}

		if targetStorage == nil {
			targetStorage = globalStorage
		}

		if targetStorage == nil {
			homeDir, _ := os.UserHomeDir()
			if homeDir != "" {
				globalDir := filepath.Join(homeDir, ".cogni")
				_ = os.MkdirAll(globalDir, 0755)
				globalPath := filepath.Join(globalDir, "memory.db")
				targetStorage, _ = storage.NewWithSource(globalPath, "global")
			}
		}

		if targetStorage == nil {
			return "Error: Could not initialize database storage.", true
		}

		formattedTags := core.FormatTags(string(args.Tags), project)
		mem := &core.Memory{
			ProjectName:      project,
			Category:         args.Category,
			Title:            args.Title,
			TopicKey:         args.TopicKey,
			SummarySignature: finalSummary,
			Tags:             formattedTags,
		}

		saved, err := targetStorage.SaveMemory(mem)
		if err != nil {
			return "Error saving memory: " + err.Error(), true
		}

		dest := "LOCAL"
		if args.Global {
			dest = "GLOBAL"
		}

		return fmt.Sprintf("💾 Memory saved [%s #%d]: [%s] \"%s\" (Category: #%s, Tags: %s)",
			dest, saved.ID, saved.ProjectName, saved.Title, saved.Category, saved.Tags), false

	case "cogni_session_summary":
		var args struct {
			Goal          string `json:"goal"`
			Accomplished  string `json:"accomplished"`
			Discoveries   string `json:"discoveries"`
			NextSteps     string `json:"next_steps"`
			RelevantFiles string `json:"relevant_files"`
			Instructions  string `json:"instructions"`
			TopicKey      string `json:"topic_key"`
			Project       string `json:"project"`
			Tags          string `json:"tags"`
			Global        bool   `json:"global"`
		}
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return "Error parsing arguments: " + err.Error(), true
		}

		if args.Goal == "" || args.Accomplished == "" {
			return "Error: 'goal' and 'accomplished' are required for session summary.", true
		}

		project := args.Project
		if project == "" {
			project = core.DetectProjectName()
		}

		var targetStorage *storage.Storage
		if args.Global {
			targetStorage = globalStorage
		} else {
			if localStorage == nil {
				localDir := filepath.Join(".", ".cogni")
				_ = os.MkdirAll(localDir, 0755)
				localPath := filepath.Join(localDir, "memory.db")
				localStorage, _ = storage.NewWithSource(localPath, "local")
			}
			targetStorage = localStorage
		}

		if targetStorage == nil {
			return "Error: Could not initialize storage.", true
		}

		sessionSummary := core.SessionSummary{
			Goal:          args.Goal,
			Accomplished:  args.Accomplished,
			Discoveries:   args.Discoveries,
			NextSteps:     args.NextSteps,
			RelevantFiles: args.RelevantFiles,
			Instructions:  args.Instructions,
		}

		saved, err := targetStorage.SaveSessionSummary(project, args.TopicKey, sessionSummary, args.Tags)
		if err != nil {
			return "Error saving session summary: " + err.Error(), true
		}

		dest := "LOCAL"
		if args.Global {
			dest = "GLOBAL"
		}

		return fmt.Sprintf("📋 Session summary persisted [%s #%d]: [%s] \"%s\" (TopicKey: %s)\n%s",
			dest, saved.ID, saved.ProjectName, saved.Title, saved.TopicKey, saved.SummarySignature), false

	case "cogni_context":
		var args struct {
			Project string `json:"project"`
			Limit   int    `json:"limit"`
		}
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return "Error parsing arguments: " + err.Error(), true
		}
		if args.Limit <= 0 {
			args.Limit = 5
		}
		project := args.Project
		if project == "" {
			project = core.DetectProjectName()
		}

		var memories []core.Memory
		if localStorage != nil {
			mems, _ := localStorage.GetRecentContext(project, args.Limit)
			memories = append(memories, mems...)
		}
		if globalStorage != nil && len(memories) < args.Limit {
			mems, _ := globalStorage.GetRecentContext(project, args.Limit-len(memories))
			memories = append(memories, mems...)
		}

		if len(memories) == 0 {
			return fmt.Sprintf("No active context found for project '%s'.", project), false
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("⚡ Active Context for '%s' (%d key signatures):\n\n", project, len(memories)))
		for _, m := range memories {
			srcBadge := "LOCAL"
			if m.Source == "global" {
				srcBadge = "GLOBAL"
			}
			topicStr := ""
			if m.TopicKey != "" {
				topicStr = fmt.Sprintf(" | Key: %s", m.TopicKey)
			}
			sb.WriteString(fmt.Sprintf("▶ [#%d %s] [%s] %s%s\n  %s\n",
				m.ID, srcBadge, m.Category, m.Title, topicStr, m.SummarySignature))
		}
		return sb.String(), false

	case "cogni_update":
		var args struct {
			ID       int64  `json:"id"`
			Summary  string `json:"summary"`
			Title    string `json:"title"`
			Category string `json:"category"`
			Tags     string `json:"tags"`
			TopicKey string `json:"topic_key"`
		}
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return "Error parsing arguments: " + err.Error(), true
		}
		if args.ID <= 0 {
			return "Error: 'id' is required.", true
		}

		var targetStorage *storage.Storage
		if localStorage != nil {
			if m, _ := localStorage.GetMemoryByID(args.ID); m != nil {
				targetStorage = localStorage
			}
		}
		if targetStorage == nil && globalStorage != nil {
			if m, _ := globalStorage.GetMemoryByID(args.ID); m != nil {
				targetStorage = globalStorage
			}
		}

		if targetStorage == nil {
			return fmt.Sprintf("Memory #%d not found.", args.ID), true
		}

		updated, err := targetStorage.UpdateMemory(args.ID, args.Title, args.Summary, args.Category, args.Tags, args.TopicKey)
		if err != nil {
			return "Error updating memory: " + err.Error(), true
		}

		return fmt.Sprintf("🔄 Memory #%d updated successfully: \"%s\"", updated.ID, updated.Title), false

	case "cogni_stats":
		var stats *core.Stats
		if localStorage != nil {
			stats, _ = localStorage.GetStats()
		}
		if stats == nil && globalStorage != nil {
			stats, _ = globalStorage.GetStats()
		}
		if stats == nil {
			return "Could not retrieve statistics.", true
		}

		return fmt.Sprintf("📊 Cogni Memory Statistics:\n- Total Memories: %d\n- Projects: %d\n- Estimated Tokens Saved: ~%d tokens",
			stats.TotalMemories, stats.TotalProjects, stats.EstimatedTokensSaved), false

	default:
		return fmt.Sprintf("Unknown tool: %s", name), true
	}
}
