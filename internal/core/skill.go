package core

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// SkillContent embeds the canonical Cogni skill definition
const SkillContent = `---
name: cogni
description: Autonomous local memory system to query and store synthetic semantic signatures in SQLite, reducing token consumption by up to 95% across AI Agent environments (Antigravity, Cursor, Claude, Copilot, OpenCode, Hermes, Codex).
---

# 🧠 Cogni Skill (Autonomous AI Agent Memory System)

> *"Just as a Byte is the fundamental unit of raw data, a Cogni is the unit of synthetic knowledge for your AI agent."*

**Cogni** (*Cognitive Omniscient Grid for Networked Intelligence*) enables AI agents to **query, register, update, and manage synthetic semantic signatures** in a fast local or global SQLite database (` + "`.cogni/memory.db`" + ` or ` + "`~/.cogni/memory.db`" + `).

Its primary objective is to maintain architectural consistency across chat sessions while drastically reducing input token consumption by preventing repetitive reading of source code and documentation.

---

## ⚡ Autonomous Agent Operating Directives

### 1. Two-Step Retrieval & Smart Task Tag-Matching
To prevent context inflation and avoid re-analyzing codebases:

- **Step 1: Lightweight Search (Discovery & Task Matching)**
  When requested to do any non-trivial task or bugfix, extract the main technical concept/tags from the prompt and execute a compact search:
  ` + "```bash\n  cogni search --query \"<keywords_or_tags>\"\n  # Or via MCP Tool: cogni_search(query: \"auth jwt\")\n  ```" + `
- **Step 2: Full Content Hydration (Only for matching IDs/Keys)**
  Retrieve the complete synthetic signature only for the relevant ID or TopicKey to know exactly how to address the task without reading large files:
  ` + "```bash\n  cogni get <id_or_topic_key>\n  # Or via MCP Tool: cogni_get(id: 6) / cogni_get(topic_key: \"arch/auth/jwt\")\n  ```" + `
- **Step 0: Quick Context Bootstrapping (` + "`cogni_context`" + `)**
  At session start or after compaction, call ` + "`cogni_context`" + ` to load recent sessions, decisions, and active conventions in under 100 tokens.

### 1.1 Proactive Preflight Search (Mandatory Triggers)
- **Architecture / New Feature**: Before proposing, designing, or scaffolding a new technical pattern, database table, API, state store, or auth flow, execute ` + "`cogni search`" + ` on the domain keyword.
- **Pre-fix Search**: Before implementing non-trivial bugfixes, search for previous resolutions in that module/error area.
- Adhere strictly to retrieved architectural patterns and previous decisions.

### 2. High-Signal Threshold & When to Save (Postflight Gate)
**GOLDEN RULE**: Call ` + "`cogni save`" + ` (or ` + "`cogni_save`" + `) ONLY if: *If this memory signature does not exist in the future, will an agent waste time investigating, break an architecture, or make a mistake?*

**DELIVERY GUARANTEE (Saving is not replying)**:
- Saving to memory is internal bookkeeping. It NEVER counts as answering the user.
- Always save/update memory **BEFORE** generating your final text reply.
- End every turn with your complete user-facing answer as the final message (no tool calls after it).
- A failed or slow memory operation NEVER blocks or replaces your user reply.

**DO NOT SAVE (Noise / Skip)**:
- ❌ Trivial metadata tasks (creating/modifying ` + "`LICENSE`" + `, ` + "`.gitignore`" + `, ` + "`.prettierrc`" + `, cosmetic assets).
- ❌ Typo fixes, code formatting (` + "`fmt`" + `, ` + "`lint`" + `), or minor documentation polishing.
- ❌ Self-evident information easily discovered by reading the first few lines of a file.

**HIGH-SIGNAL CATEGORIES (Must Save)**:
- **` + "`bugfix`" + `**: Resolution of a non-trivial error with a non-obvious root cause.
- **` + "`architecture`" + ` / ` + "`decision`" + `**: Choice of libraries, data schemas, API contracts, or system structures.
- **` + "`discovery`" + `**: Non-obvious technical finding or gotcha about runtime/codebase behavior.
- **` + "`config`" + `**: Non-trivial tooling, environment, script, or build setup.
- **` + "`pattern`" + `**: Established naming convention, folder structure, or coding standard.
- **` + "`preference`" + `**: User preference or technical constraint learned during the session.
- **` + "`session`" + `**: End-of-session or post-compaction milestone summaries.

### 3. High-Density Synthetic Signature Format (What / Why / Where / Learned)
Cogni is designed to eliminate context saturation by replacing 500-line file reads with High-Density Synthetic Signatures occupying under 5% of tokens:

- **Topic**: Hierarchical key (` + "`<domain>/<subdomain>/<topic>`" + `, e.g., ` + "`standards/i18n/ui`" + `, ` + "`arch/auth/jwt`" + `).
- **What**: One concise sentence — what was done or decided.
- **Why**: Motivation or root cause.
- **Where**: Affected relative files or paths.
- **Learned**: Non-obvious gotchas or learnings (omit if none).

*Format in signature*: ` + "`What: ... | Why: ... | Where: ... | Learned: ...`" + `

` + "```yaml\n# Ideal Cogni Signature Example:\nTopic: standards/i18n/ui\nWhat: Todo texto visible en JSX/TSX debe usar t('namespace:key'). Prohibido texto literal.\nWhy: Estándar global del proyecto para soporte multi-idioma (es, en, pt, fr, ar).\nWhere: src/providers/i18n/, src/modules/*, src/layouts/\nLearned: Cadenas en toast o modales también deben internacionalizarse.\n```" + `

### 4. Diagnostic & Maintenance Tooling
- **` + "`cogni stats`" + ` / ` + "`cogni_stats()`" + `**: Displays memory health, entry count, and estimated token savings metrics.
- **` + "`cogni session_summary`" + ` / ` + "`cogni_session_summary()`" + `**: Summarizes progress, discoveries, and next steps to resume context without reloading long chat histories.

### 5. Compaction & Session Lifecycle Protocol

#### End of Session (` + "`cogni_session_summary`" + `)
Before ending a session or stating "done", call ` + "`cogni_session_summary`" + ` (or ` + "`cogni session-summary`" + `) with:
- **goal**: Main objective worked on.
- **accomplished**: Completed items with key technical details.
- **discoveries**: Findings, gotchas, or architectural decisions.
- **next_steps**: Pending items for the next session.
- **relevant_files**: Key files modified.

#### After Compaction / Context Reset (` + "`FIRST ACTION REQUIRED`" + `)
If a compaction message or reset occurs:
1. IMMEDIATELY call ` + "`cogni_session_summary`" + ` with the compacted summary content to persist pre-compaction progress into SQLite.
2. Call ` + "`cogni_context`" + ` to retrieve active project context.
3. Only THEN proceed with your task.

### 6. Deterministic Topic Keys & Automatic Upserts
To prevent duplicate records:
- Format: ` + "`<domain>/<subdomain>/<topic>`" + ` (ej. ` + "`arch/auth/jwt`" + `, ` + "`standards/i18n/ui`" + `, ` + "`session/latest`" + `).
- When a ` + "`--topic-key`" + ` already exists, ` + "`cogni save`" + ` automatically updates (**upserts**) the record.

---

## 🛠️ Tooling & CLI Reference

### Native MCP Tools:
- ` + "`cogni_context(project, limit)`" + `: Active context & recent sessions in < 100 tokens.
- ` + "`cogni_session_summary(goal, accomplished, discoveries, next_steps, relevant_files)`" + `: Persist session summary.
- ` + "`cogni_search(query, project, category, limit)`" + `: Lightweight discovery search.
- ` + "`cogni_get(id, topic_key, project)`" + `: Full content hydration (Phase 2).
- ` + "`cogni_save(title, summary, what, why, where, learned, category, tags, topic_key, project, global)`" + `: Structured save/upsert.
- ` + "`cogni_update(id, summary, title, category, tags, topic_key)`" + `: Direct update by ID.
- ` + "`cogni_stats()`" + `: Memory usage, health, and token metrics.

### CLI Commands:
` + "```bash\n# 1. Quick active context bootstrapping\ncogni context\n\n# 2. Save structured memory with discrete fields\ncogni save \\\n  --topic-key \"arch/auth/jwt\" \\\n  --title \"JWT Refresh Token Rotation\" \\\n  --what \"Implemented refresh token rotation with Redis blacklist\" \\\n  --why \"Mitigates replay attacks after security audit\" \\\n  --where \"src/auth/jwt.go, src/middleware/auth.go\" \\\n  --learned \"Redis TTL automatically manages expired blacklist keys\" \\\n  --category \"architecture\" \\\n  --tags \"auth,jwt,security\"\n\n# 3. Save end-of-session or post-compaction summary\ncogni session-summary \\\n  --goal \"Implement JWT Auth\" \\\n  --accomplished \"Created tokens endpoints and migrations\" \\\n  --where \"src/auth/jwt.go\"\n\n# 4. Search memories (Compact 1-line preview)\ncogni search --query \"jwt\"\n\n# 5. Retrieve full memory content (Phase 2)\ncogni get arch/auth/jwt\n```" + `
`

// RuleContent defines the mandatory behavior directives for the agent
const RuleContent = `---
trigger: always_on
description: "Cogni Autonomous Memory Protocol & Synthetic Signatures"
applyTo: "**"
---

# Cogni Memory Invariants & Context Optimization

## 1. Preflight Search & Task Tag Matching (Mandatory)
- Before proposing, designing, or implementing a new feature, API route, schema, state store, or bugfix, execute **` + "`cogni_search`" + `** (MCP) or **` + "`cogni search`" + `** (CLI) with the task tags/domain keywords.
- When previous memories exist, adhere strictly to established patterns and hydrate only required records with **` + "`cogni_get`" + `** to avoid reading entire source files.
- At session start, call **` + "`cogni_context`" + `** to load the active project context in minimal tokens.

## 2. Postflight Save Gate & Delivery Guarantee (Mandatory)
- Before completing any high-signal task (bugfix, architectural decision, library selection, build setup, convention), save or update it in memory.
- Structure every summary as: ` + "`What: ... | Why: ... | Where: ... | Learned: ...`" + `
- Use a deterministic **` + "`topic_key`" + `** (` + "`<domain>/<subdomain>/<topic>`" + `) so subsequent runs **upsert** existing records.
- **Delivery Guarantee**: Saving memory is internal bookkeeping. Always save BEFORE composing the final reply and never replace the complete user answer with a one-line "saved" acknowledgement.

## 3. Compaction & Session Summary Protocol
- When a context compaction happens or you see "FIRST ACTION REQUIRED":
  1. Call **` + "`cogni_session_summary`" + `** immediately with the compacted summary to persist state.
  2. Call **` + "`cogni_context`" + `** to recover active project context.
  3. Continue with the task.
`

// InstallSkill writes the embedded SKILL.md to the specified directory
func InstallSkill(targetDir string) error {
	skillDir := filepath.Join(targetDir, "cogni")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return err
	}

	dest := filepath.Join(skillDir, "SKILL.md")
	return os.WriteFile(dest, []byte(SkillContent), 0644)
}

// RemoveSkill removes the cogni skill directory from the target path if it exists
func RemoveSkill(targetDir string) error {
	skillDir := filepath.Join(targetDir, "cogni")
	if _, err := os.Stat(skillDir); err == nil {
		return os.RemoveAll(skillDir)
	}
	return nil
}

// InstallRules writes the global/local rules to their respective directories
func InstallRules(homeDir string, allowedHarnesses []string) error {
	rulesDirs := map[string]string{
		"antigravity": filepath.Join(homeDir, ".gemini", "config", "rules"),
		"cursor":      filepath.Join(homeDir, ".cursor", "rules"),
		"claude":      filepath.Join(homeDir, ".claude", "rules"),
		"opencode":    filepath.Join(homeDir, ".config", "opencode", "rules"),
		"hermes":      filepath.Join(homeDir, ".hermes", "rules"),
	}

	harnessAllowed := func(h string) bool {
		if len(allowedHarnesses) == 0 {
			return true
		}
		for _, ah := range allowedHarnesses {
			if ah == h || ah == "all" {
				return true
			}
		}
		return false
	}

	for harness, dir := range rulesDirs {
		if !harnessAllowed(harness) {
			continue
		}
		parentDir := filepath.Dir(dir)
		if _, err := os.Stat(parentDir); err == nil {
			_ = os.MkdirAll(dir, 0755)
			dest := filepath.Join(dir, "cogni.rules.md")
			_ = os.WriteFile(dest, []byte(RuleContent), 0644)
		}
	}

	// Instalar en local si .agents existe y está permitido
	if harnessAllowed("local") || harnessAllowed("workspace") {
		if _, err := os.Stat(".agents"); err == nil {
			localRules := filepath.Join(".agents", "rules")
			_ = os.MkdirAll(localRules, 0755)
			_ = os.WriteFile(filepath.Join(localRules, "cogni.rules.md"), []byte(RuleContent), 0644)
		}
	}

	// Limpiar cualquier skill legado previo en arneses que usan Always-On Rules + MCP
	if harnessAllowed("antigravity") {
		_ = RemoveSkill(filepath.Join(homeDir, ".gemini", "config", "skills"))
		_ = RemoveSkill(filepath.Join(".agents", "skills"))
	}
	if harnessAllowed("cursor") {
		_ = RemoveSkill(filepath.Join(homeDir, ".cursor", "skills"))
	}

	return nil
}

// GetHarnessSkillPaths returns supported AI harness skill directory paths.
// Nota: Arneses modernos como Antigravity y Cursor usan Always-On Rules + MCP
// y NO requieren inyectar cogni como skill (evita lecturas forzadas de SKILL.md).
func GetHarnessSkillPaths(homeDir string) map[string][]string {
	return map[string][]string{
		"local": {
			filepath.Join(".agents", "skills"),
		},
		"antigravity": {},
		"cursor":      {},
		"claude": {
			filepath.Join(homeDir, ".claude", "skills"),
		},
		"opencode": {
			filepath.Join(homeDir, ".config", "opencode", "skills"),
			filepath.Join(homeDir, ".agents", "skills"),
		},
		"copilot": {
			filepath.Join(homeDir, ".agents", "skills"),
			filepath.Join(homeDir, ".copilot", "skills"),
		},
		"hermes": {
			filepath.Join(homeDir, ".hermes", "skills"),
		},
		"codex": {
			filepath.Join(homeDir, ".agents", "skills"),
		},
	}
}

// GetHarnessMCPPaths returns supported AI harness MCP configuration file paths
func GetHarnessMCPPaths(homeDir string) map[string][]string {
	return map[string][]string{
		"antigravity": {
			filepath.Join(homeDir, ".gemini", "config", "mcp_config.json"),
		},
		"cursor": {
			filepath.Join(homeDir, ".cursor", "mcp.json"),
		},
		"claude": {
			filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json"),
			filepath.Join(homeDir, ".config", "Claude", "claude_desktop_config.json"),
			filepath.Join(homeDir, ".claude.json"),
		},
		"opencode": {
			filepath.Join(homeDir, ".config", "opencode", "opencode.json"),
			filepath.Join(homeDir, ".config", "opencode", "opencode.jsonc"),
		},
		"hermes": {
			filepath.Join(homeDir, ".hermes", "mcp.json"),
		},
		"codex": {
			filepath.Join(homeDir, ".codex", "config.toml"),
		},
	}
}

// ConfigureHarnessMCP inyecta automáticamente el servidor MCP 'cogni' en las configuraciones de MCP
func ConfigureHarnessMCP(homeDir string, allowedHarnesses []string) map[string]string {
	mcpConfigs := GetHarnessMCPPaths(homeDir)
	configured := make(map[string]string)

	harnessAllowed := func(h string) bool {
		if len(allowedHarnesses) == 0 {
			return true
		}
		for _, ah := range allowedHarnesses {
			if ah == h || ah == "all" {
				return true
			}
		}
		return false
	}

	cogniBin := "cogni"
	localBin := filepath.Join(homeDir, ".local", "bin", "cogni")
	if _, err := os.Stat(localBin); err == nil {
		cogniBin = localBin
	}

	cogniEntry := map[string]any{
		"command": cogniBin,
		"args":    []string{"mcp"},
		"type":    "stdio",
	}

	for harness, paths := range mcpConfigs {
		if !harnessAllowed(harness) {
			continue
		}
		for _, cfgPath := range paths {
			parentDir := filepath.Dir(cfgPath)
			// Solo configurar si el directorio padre del arnés existe (evita crear carpetas fantasmas)
			if _, err := os.Stat(parentDir); err != nil {
				continue
			}

			if err := injectMCPServer(cfgPath, "cogni", cogniEntry); err == nil {
				configured[harness+" ("+filepath.Base(parentDir)+")"] = cfgPath
			}
		}
	}

	// También configurar en workspace local si .agents/ existe y está permitido
	if harnessAllowed("local") || harnessAllowed("workspace") {
		if _, err := os.Stat(".agents"); err == nil {
			localMCP := filepath.Join(".agents", "mcp_config.json")
			if err := injectMCPServer(localMCP, "cogni", map[string]any{
				"command": "cogni",
				"args":    []string{"mcp"},
			}); err == nil {
				configured["workspace"] = localMCP
			}
		}
	}

	return configured
}

func injectMCPServer(filePath string, serverName string, serverConfig map[string]any) error {
	var root map[string]any

	if data, err := os.ReadFile(filePath); err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, &root)
	}

	if root == nil {
		root = make(map[string]any)
	}

	// OpenCode usa formato especial: { "mcp": { "servers": { "name": {...} } } }
	baseName := filepath.Base(filePath)
	isOpenCode := baseName == "opencode.json" || baseName == "opencode.jsonc"
	if isOpenCode {
		return injectOpenCodeMCP(filePath, root, serverName, serverConfig)
	}

	// Codex CLI usa TOML: [mcp_servers.<name>] con command + args
	isCodex := baseName == "config.toml" && filepath.Base(filepath.Dir(filePath)) == ".codex"
	if isCodex {
		cogniCmd, _ := serverConfig["command"].(string)
		cogniArgs, _ := serverConfig["args"].([]string)
		return injectCodexMCP(filePath, serverName, cogniCmd, cogniArgs)
	}

	// Formato estandar (Claude, Cursor, Gemini): { "mcpServers": { "name": {...} } }
	var servers map[string]any
	if existing, ok := root["mcpServers"].(map[string]any); ok && existing != nil {
		servers = existing
	} else {
		servers = make(map[string]any)
	}

	servers[serverName] = serverConfig
	root["mcpServers"] = servers

	_ = os.MkdirAll(filepath.Dir(filePath), 0755)

	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, append(data, '\n'), 0644)
}

func injectOpenCodeMCP(filePath string, root map[string]any, serverName string, serverConfig map[string]any) error {
	// Asegurar estructura { "mcp": { "servers": {...} } }
	var mcpObj map[string]any
	if existing, ok := root["mcp"].(map[string]any); ok && existing != nil {
		mcpObj = existing
	} else {
		mcpObj = make(map[string]any)
	}

	var servers map[string]any
	if existing, ok := mcpObj["servers"].(map[string]any); ok && existing != nil {
		servers = existing
	} else {
		servers = make(map[string]any)
		// Migrar formato V1 (servidores directamente bajo mcp) a V2 (mcp.servers)
		for key, val := range mcpObj {
			if key == "servers" {
				continue
			}
			// Si parece un servidor MCP (tiene "command" o "type" o "url")
			if serverVal, ok := val.(map[string]any); ok {
				if _, hasCommand := serverVal["command"]; hasCommand {
					if _, hasType := serverVal["type"]; hasType {
						servers[key] = val
						delete(mcpObj, key)
					}
				}
			}
		}
	}

	// Convertir formato estandar a formato OpenCode:
	// De: { "command": "cogni", "args": ["mcp"] }
	// A:  { "type": "local", "command": ["cogni", "mcp"], "enabled": true }
	cogniCmd, _ := serverConfig["command"].(string)
	cogniArgs, _ := serverConfig["args"].([]string)

	openCodeServer := map[string]any{
		"type":    "local",
		"command": append([]string{cogniCmd}, cogniArgs...),
		"enabled": true,
	}

	servers[serverName] = openCodeServer
	mcpObj["servers"] = servers
	root["mcp"] = mcpObj

	_ = os.MkdirAll(filepath.Dir(filePath), 0755)

	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, append(data, '\n'), 0644)
}

// injectCodexMCP escribe/actualiza la sección [mcp_servers.cogni] en un archivo
// TOML de configuración de Codex CLI sin destruir las demás claves del archivo.
//
// Si la sección ya existe, se reemplaza su bloque (upsert). Si la sección
// [mcp_servers] no existe, se agrega al final respetando las claves previas.
func injectCodexMCP(filePath string, serverName string, command string, args []string) error {
	if command == "" {
		command = "cogni"
	}
	if args == nil {
		args = []string{"mcp"}
	}

	// 1. Parsear el archivo TOML existente en un mapa genérico.
	var root map[string]any
	if data, err := os.ReadFile(filePath); err == nil && len(data) > 0 {
		if _, err := toml.Decode(string(data), &root); err != nil {
			// Si no se puede parsear, empezar desde cero preservando un backup lógico
			root = make(map[string]any)
		}
	}
	if root == nil {
		root = make(map[string]any)
	}

	// 2. Obtener/crear la tabla [mcp_servers].
	mcpServers, ok := root["mcp_servers"].(map[string]any)
	if !ok || mcpServers == nil {
		mcpServers = make(map[string]any)
	}

	// 3. Upsert del servidor específico.
	mcpServers[serverName] = map[string]any{
		"command": command,
		"args":    args,
		"enabled": true,
	}
	root["mcp_servers"] = mcpServers

	// 4. Serializar preservando el orden natural de Go map (alfabético).
	//    En Codex no se garantiza el orden de las claves TOML, pero
	//    serializar de forma estable evita diffs espurios entre ejecuciones.
	_ = os.MkdirAll(filepath.Dir(filePath), 0755)

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(root); err != nil {
		return err
	}

	return nil
}

// RemoveCodexMCPServer elimina la entrada [mcp_servers.<serverName>] de un
// archivo TOML de Codex CLI. Si la tabla [mcp_servers] queda vacía, también
// se elimina. Si el archivo no existe, no hace nada.
func RemoveCodexMCPServer(filePath string, serverName string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var root map[string]any
	if len(data) > 0 {
		if _, err := toml.Decode(string(data), &root); err != nil {
			return err
		}
	}
	if root == nil {
		return nil
	}

	mcpServers, ok := root["mcp_servers"].(map[string]any)
	if !ok {
		return nil
	}

	delete(mcpServers, serverName)
	if len(mcpServers) == 0 {
		delete(root, "mcp_servers")
	} else {
		root["mcp_servers"] = mcpServers
	}

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	return encoder.Encode(root)
}
