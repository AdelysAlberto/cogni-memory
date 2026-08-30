package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema *JSONSchema `json:"inputSchema"`
}

type JSONSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

type CallToolResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Server implements the MCP stdio server for Cogni
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
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "cogni-mcp",
				"version": s.version,
			},
		}
		s.sendResult(w, req.ID, result)

	case "notifications/initialized", "initialized":
		// No response required for notifications
		return

	case "ping":
		s.sendResult(w, req.ID, map[string]any{})

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
	return []Tool{
		{
			Name: "cogni_search",
			Description: "Busca memorias sintéticas de forma ligera (retorna IDs, títulos, tags y resúmenes compactos). " +
				"Usa este tool en el Preflight Check antes de diseñar o fixear un componente para recuperar patrones previos sin inflar el contexto.",
			InputSchema: &JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {
						Type:        "string",
						Description: "Término de búsqueda semántica o palabras clave.",
					},
					"project": {
						Type:        "string",
						Description: "Nombre del proyecto (opcional, auto-detectado si no se especifica).",
					},
					"category": {
						Type:        "string",
						Description: "Filtrar por categoría (bugfix, architecture, decision, discovery, config, pattern, preference, general).",
						Enum:        []string{"bugfix", "architecture", "decision", "discovery", "config", "pattern", "preference", "general"},
					},
					"limit": {
						Type:        "integer",
						Description: "Límite máximo de resultados (por defecto: 5).",
					},
				},
				Required: []string{"query"},
			},
		},
		{
			Name: "cogni_get",
			Description: "Recupera la memoria sintética completa por ID o por TopicKey determinístico (Protocolo de 2 Fases). " +
				"Úsalo tras `cogni_search` para hidratar únicamente el registro que necesitas.",
			InputSchema: &JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {
						Type:        "integer",
						Description: "ID numérico de la memoria a consultar.",
					},
					"topic_key": {
						Type:        "string",
						Description: "TopicKey determinístico (ej: 'arch/auth/jwt', 'pattern/react/modals').",
					},
					"project": {
						Type:        "string",
						Description: "Nombre del proyecto (opcional para filtrar por topic_key).",
					},
				},
			},
		},
		{
			Name: "cogni_save",
			Description: "Guarda o actualiza (upsert si topic_key existe) una firma de memoria sintética estructurada de alta densidad. " +
				"Acepta campos estructurados (what, why, where, learned) o una firma en formato: 'What: <desc> | Why: <motivo> | Where: <archivos> | Learned: <gotchas>'",
			InputSchema: &JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"title": {
						Type:        "string",
						Description: "Título conciso del hito o aprendizaje.",
					},
					"summary": {
						Type:        "string",
						Description: "Firma sintética en formato: 'What: ... | Why: ... | Where: ... | Learned: ...' (opcional si se pasan what/why/where/learned).",
					},
					"what": {
						Type:        "string",
						Description: "Una oración descriptiva de qué se hizo.",
					},
					"why": {
						Type:        "string",
						Description: "Motivo o causa raíz de la acción.",
					},
					"where": {
						Type:        "string",
						Description: "Archivos o rutas afectadas clave.",
					},
					"learned": {
						Type:        "string",
						Description: "Aprendizajes, gotchas o casos borde descubiertos.",
					},
					"category": {
						Type:        "string",
						Description: "Categoría de la memoria.",
						Enum:        []string{"bugfix", "architecture", "decision", "discovery", "config", "pattern", "preference", "general"},
					},
					"tags": {
						Type:        "string",
						Description: "Tags separados por coma en 3 capas (ej: 'auth,jwt,tokens-middleware').",
					},
					"topic_key": {
						Type:        "string",
						Description: "Clave temática determinística para posibilitar upserts sin duplicar (ej: 'sdd/auth/spec', 'arch/db/indexes').",
					},
					"project": {
						Type:        "string",
						Description: "Nombre del proyecto (opcional, auto-detectado).",
					},
					"global": {
						Type:        "boolean",
						Description: "Si es true, se guarda en ~/.cogni/memory.db (global). Por defecto false (local en el proyecto).",
					},
				},
				Required: []string{"title", "category", "tags"},
			},
		},
		{
			Name: "cogni_session_summary",
			Description: "Guarda o actualiza un resumen estructurado al finalizar la sesión o tras una compactación (Goal, Accomplished, Discoveries, Next Steps, Relevant Files). " +
				"Esencial para mantener continuidad entre sesiones y recuperar contexto tras la compactación.",
			InputSchema: &JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"goal": {
						Type:        "string",
						Description: "Objetivo principal de la sesión de trabajo.",
					},
					"accomplished": {
						Type:        "string",
						Description: "Logros y tareas completadas con detalles clave.",
					},
					"discoveries": {
						Type:        "string",
						Description: "Hallazgos técnicos, decisiones o gotchas descubiertos.",
					},
					"next_steps": {
						Type:        "string",
						Description: "Próximos pasos pendientes para la siguiente sesión.",
					},
					"relevant_files": {
						Type:        "string",
						Description: "Archivos modificados o creados principales.",
					},
					"instructions": {
						Type:        "string",
						Description: "Preferencias o restricciones aprendidas del usuario.",
					},
					"topic_key": {
						Type:        "string",
						Description: "Clave determinística (por defecto: 'session/latest').",
					},
					"project": {
						Type:        "string",
						Description: "Nombre del proyecto (opcional).",
					},
					"tags": {
						Type:        "string",
						Description: "Tags adicionales separados por coma.",
					},
					"global": {
						Type:        "boolean",
						Description: "Guardar en BD global (~/.cogni/). Por defecto false (local).",
					},
				},
				Required: []string{"goal", "accomplished"},
			},
		},
		{
			Name: "cogni_context",
			Description: "Recupera de forma rápida y compacta el contexto activo reciente del proyecto (sesiones previas, decisiones de arquitectura y fixes recientes) para iniciar la sesión con alta señal y sin inflar el contexto.",
			InputSchema: &JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"project": {
						Type:        "string",
						Description: "Nombre del proyecto (opcional, auto-detectado).",
					},
					"limit": {
						Type:        "integer",
						Description: "Límite máximo de memorias recientes (por defecto: 5).",
					},
				},
			},
		},
		{
			Name: "cogni_update",
			Description: "Actualiza campos de una memoria sintética existente por su ID.",
			InputSchema: &JSONSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {
						Type:        "integer",
						Description: "ID de la memoria a actualizar.",
					},
					"summary": {
						Type:        "string",
						Description: "Nueva firma sintética.",
					},
					"title": {
						Type:        "string",
						Description: "Nuevo título.",
					},
					"category": {
						Type:        "string",
						Description: "Nueva categoría.",
					},
					"tags": {
						Type:        "string",
						Description: "Nuevos tags.",
					},
					"topic_key": {
						Type:        "string",
						Description: "Nuevo topic_key.",
					},
				},
				Required: []string{"id"},
			},
		},
		{
			Name: "cogni_stats",
			Description: "Obtiene estadísticas de uso de memoria y cantidad de tokens ahorrados.",
			InputSchema: &JSONSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
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
			Query    string `json:"query"`
			Project  string `json:"project"`
			Category string `json:"category"`
			Limit    int    `json:"limit"`
		}
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return "Error parseando argumentos: " + err.Error(), true
		}
		if args.Limit <= 0 {
			args.Limit = 5
		}
		project := args.Project
		if project == "" {
			project = core.DetectProjectName()
		}

		var results []core.Memory
		if localStorage != nil {
			res, _ := localStorage.SearchMemories(project, args.Query, args.Category, args.Limit)
			results = append(results, res...)
		}
		if globalStorage != nil && len(results) < args.Limit {
			res, _ := globalStorage.SearchMemories(project, args.Query, args.Category, args.Limit-len(results))
			results = append(results, res...)
		}

		if len(results) == 0 {
			return "No se encontraron memorias para la búsqueda: " + args.Query, false
		}

		// Retornar formato ligero (ID, TopicKey, Title, Preview de 1 línea, Tags) para ahorrar tokens
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("🔍 Encontradas %d memoria(s) [Usa cogni_get con ID o TopicKey para ver el contenido completo]:\n\n", len(results)))
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
			return "Error parseando argumentos: " + err.Error(), true
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
			return "Error: Debes especificar 'id' o 'topic_key'.", true
		}

		if err != nil {
			return "Error recuperando memoria: " + err.Error(), true
		}
		if mem == nil {
			return "Memoria no encontrada.", false
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("━━━ [#%d] [%s] %s ━━━\n", mem.ID, mem.ProjectName, mem.Title))
		if mem.TopicKey != "" {
			sb.WriteString(fmt.Sprintf("🔑 Topic Key: %s\n", mem.TopicKey))
		}
		sb.WriteString(fmt.Sprintf("📂 Categoría: %s | 🏷️ Tags: %s\n", mem.Category, mem.Tags))
		sb.WriteString(fmt.Sprintf("📅 Fecha: %s\n\n", mem.UpdatedAt.Format("2006-01-02 15:04:05")))
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
			return "Error parseando argumentos: " + err.Error(), true
		}

		if args.Title == "" {
			return "Error: 'title' es requerido.", true
		}

		finalSummary := args.Summary
		if finalSummary == "" && (args.What != "" || args.Why != "" || args.Where != "" || args.Learned != "") {
			finalSummary = core.BuildSummarySignature(args.What, args.Why, args.Where, args.Learned)
		}

		if finalSummary == "" {
			return "Error: Debes proporcionar 'summary' o los campos estructurados 'what', 'why', 'where', 'learned'.", true
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
				// Crear local si no existe
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
			// Fallback de emergencia a la base de datos global de usuario
			homeDir, _ := os.UserHomeDir()
			if homeDir != "" {
				globalDir := filepath.Join(homeDir, ".cogni")
				_ = os.MkdirAll(globalDir, 0755)
				globalPath := filepath.Join(globalDir, "memory.db")
				targetStorage, _ = storage.NewWithSource(globalPath, "global")
			}
		}

		if targetStorage == nil {
			return "Error: No se pudo inicializar el almacenamiento.", true
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
			return "Error guardando memoria: " + err.Error(), true
		}

		dest := "LOCAL"
		if args.Global {
			dest = "GLOBAL"
		}

		return fmt.Sprintf("💾 Memoria guardada [%s #%d]: [%s] \"%s\" (Category: #%s, Tags: %s)",
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
			return "Error parseando argumentos: " + err.Error(), true
		}

		if args.Goal == "" || args.Accomplished == "" {
			return "Error: 'goal' y 'accomplished' son requeridos para el resumen de sesión.", true
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
			return "Error: No se pudo inicializar el almacenamiento.", true
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
			return "Error guardando resumen de sesión: " + err.Error(), true
		}

		dest := "LOCAL"
		if args.Global {
			dest = "GLOBAL"
		}

		return fmt.Sprintf("📋 Resumen de sesión persistido [%s #%d]: [%s] \"%s\" (TopicKey: %s)\n%s",
			dest, saved.ID, saved.ProjectName, saved.Title, saved.TopicKey, saved.SummarySignature), false

	case "cogni_context":
		var args struct {
			Project string `json:"project"`
			Limit   int    `json:"limit"`
		}
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return "Error parseando argumentos: " + err.Error(), true
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
			return fmt.Sprintf("No hay contexto reciente guardado para el proyecto '%s'.", project), false
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("⚡ Contexto Activo Reciente para '%s' (%d memorias clave):\n\n", project, len(memories)))
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
			return "Error parseando argumentos: " + err.Error(), true
		}
		if args.ID <= 0 {
			return "Error: 'id' es requerido.", true
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
			return fmt.Sprintf("Memoria #%d no encontrada.", args.ID), true
		}

		updated, err := targetStorage.UpdateMemory(args.ID, args.Title, args.Summary, args.Category, args.Tags, args.TopicKey)
		if err != nil {
			return "Error actualizando memoria: " + err.Error(), true
		}

		return fmt.Sprintf("🔄 Memoria #%d actualizada con éxito: \"%s\"", updated.ID, updated.Title), false

	case "cogni_stats":
		var stats *core.Stats
		if localStorage != nil {
			stats, _ = localStorage.GetStats()
		}
		if stats == nil && globalStorage != nil {
			stats, _ = globalStorage.GetStats()
		}
		if stats == nil {
			return "No se pudieron obtener estadísticas.", true
		}

		return fmt.Sprintf("📊 Estadísticas de Cogni:\n- Total Memorias: %d\n- Proyectos: %d\n- Tokens Estimados Ahorrados: ~%d tokens",
			stats.TotalMemories, stats.TotalProjects, stats.EstimatedTokensSaved), false

	default:
		return fmt.Sprintf("Herramienta desconocida: %s", name), true
	}
}
