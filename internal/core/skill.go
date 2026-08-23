package core

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
)

// SkillContent embeds the canonical Cogni skill definition
const SkillContent = `---
name: cogni
description: Autonomous local memory system to query and store synthetic semantic signatures in SQLite, reducing token consumption by up to 95% across AI Agent environments (Antigravity, Cursor, Claude, Copilot, OpenCode, Hermes).
---

# 🧠 Cogni Skill (Autonomous AI Agent Memory System)

> *"Just as a Byte is the fundamental unit of raw data, a Cogni is the unit of synthetic knowledge for your AI agent."*

**Cogni** (*Cognitive Omniscient Grid for Networked Intelligence*) enables AI agents to **query, register, update, and manage synthetic semantic signatures** in a fast local or global SQLite database (` + "`.cogni/memory.db`" + ` or ` + "`~/.cogni/memory.db`" + `).

Its primary objective is to maintain architectural consistency across chat sessions while drastically reducing input token consumption by preventing repetitive reading of source code and documentation.

---

## ⚡ Autonomous Agent Operating Directives

### 1. Two-Step Retrieval Protocol (Token Optimization)
To prevent context inflation, retrieval ALWAYS follows two distinct phases:

- **Step 1: Lightweight Search (Discovery)**
  Run a compact search to inspect matching titles, categories, tags, and 1-line previews:
  ` + "```bash\n  cogni search --query \"<keywords>\" \n  # Or via MCP Tool: cogni_search(query: \"...\")\n  ```" + `
- **Step 2: Full Content Hydration (Only for relevant IDs/Keys)**
  Retrieve the complete synthetic signature only for the chosen ID or TopicKey:
  ` + "```bash\n  cogni get <id_or_topic_key>\n  # Or via MCP Tool: cogni_get(id: 6) / cogni_get(topic_key: \"arch/auth/jwt\")\n  ```" + `

### 1.1 Proactive Preflight Search (Mandatory Triggers)
- **Architecture / New Feature**: Before proposing, designing, or scaffolding a new technical pattern, database table, API, state store, or auth flow, execute ` + "`cogni search`" + ` on the domain keyword.
- **Pre-fix Search**: Before implementing non-trivial bugfixes, search for previous resolutions in that module/error area.
- Adhere strictly to retrieved architectural patterns and previous decisions.

### 2. High-Signal Threshold & When to Save (Postflight Gate)
**GOLDEN RULE**: Call ` + "`cogni save`" + ` (or ` + "`cogni_save`" + `) ONLY if: *If this memory signature does not exist in the future, will an agent waste time investigating, break an architecture, or make a mistake?*

**MANDATORY TIMING**: Execute save/update **before** emitting the final text envelope to the user.

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

### 3. Deterministic Topic Keys & Automatic Upserts
To prevent signature duplication and database fragmentation, use a structured ` + "`--topic-key`" + `:
- Format: ` + "`<domain>/<subdomain>/<topic>`" + ` (ej. ` + "`arch/auth/jwt`" + `, ` + "`sdd/cart/spec`" + `, ` + "`pattern/react/forms`" + `).
- When a ` + "`--topic-key`" + ` already exists in the project, ` + "`cogni save`" + ` **automatically updates (upserts)** the record instead of creating duplicates.

### 4. Synthetic Summary Format (` + "`--summary`" + `)
Every summary MUST follow this high-density 4-part structured format:
` + "`What: <One sentence description of what was done> | Why: <Motivation or root cause> | Where: <Key files/paths affected> | Learned: <Gotchas or key learnings (omit if none)>`" + `

### 5. 3-Layer Tag Taxonomy
Include 3 to 5 lowercase, kebab-case tags:
1. **Layer 1 - Main Concept**: Generic technical domain (` + "`pagination`" + `, ` + "`auth`" + `, ` + "`state-management`" + `, ` + "`database`" + `).
2. **Layer 2 - Technology / Stack**: Exact tech stack (` + "`go`" + `, ` + "`sqlite`" + `, ` + "`zustand`" + `, ` + "`react`" + `, ` + "`css-modules`" + `).
3. **Layer 3 - Specific Module**: Project domain entity (` + "`products-list`" + `, ` + "`jwt-middleware`" + `).

---

## 🛠️ Tooling & CLI Reference

### Native MCP Tools (When running in MCP-compatible environments):
- ` + "`cogni_search(query, project, category, limit)`" + `: Lightweight discovery search (previews).
- ` + "`cogni_get(id, topic_key, project)`" + `: Full content hydration.
- ` + "`cogni_save(title, summary, category, tags, topic_key, project, global)`" + `: High-signal save/upsert.
- ` + "`cogni_update(id, summary, title, category, tags, topic_key)`" + `: Direct update by ID.
- ` + "`cogni_stats()`" + `: Memory usage and token metrics.

### CLI Commands:
` + "```bash\n# 1. Save / Upsert structured memory\ncogni save \\\n  --topic-key \"arch/auth/jwt\" \\\n  --title \"JWT Refresh Token Rotation\" \\\n  --category \"architecture\" \\\n  --tags \"auth,jwt,security\" \\\n  --summary \"What: Added refresh token rotation with blacklist | Why: Mitigates token replay | Where: src/auth/jwt.go | Learned: Requires redis TTL sync\"\n\n# 2. Search memories (Compact 1-line preview)\ncogni search --query \"jwt\"\n\n# 3. Retrieve full memory content (Phase 2)\ncogni get arch/auth/jwt\n# or: cogni get --id 6\n\n# 4. Update memory by ID\ncogni update --id 6 --summary \"What: ... | Why: ... | Where: ... | Learned: ...\"\n\n# 5. Start native MCP stdio server\ncogni mcp\n\n# 6. Open visual web dashboard\ncogni ui\n```" + `
`

// RuleContent defines the mandatory behavior directives for the agent
const RuleContent = `# 🧠 Cogni Memory Invariants

## 🔍 1. Preflight Search (Mandatory)
- Before proposing, designing, or implementing a new feature, database schema, API route, state store, or non-trivial logic, execute **` + "`cogni_search`" + `** (MCP tool) or **` + "`cogni search`" + `** (CLI) for the domain keyword.
- If previous memories are retrieved:
  - Adhere strictly to the established patterns, configurations, and decisions.
  - Hydrate only the relevant memories using **` + "`cogni_get`" + `** or **` + "`cogni get`" + `**.

## 💾 2. Postflight Save Gate (Mandatory)
- Before completing any high-signal task (e.g. bugfix with non-obvious cause, architectural decision, library selection, build setup, coding standard), save or update it in memory.
- Use a deterministic **` + "`topic_key`" + `** (format: ` + "`<domain>/<subdomain>/<topic>`" + `, ej. ` + "`arch/auth/jwt`" + `) so that subsequent runs **upsert** existing records instead of generating duplicates.
- Structure every summary strictly as: ` + "`What: ... | Why: ... | Where: ... | Learned: ...`" + `
- Tags must follow the 3-layer taxonomy: main concept, tech stack, specific module.
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

// InstallRules writes the global/local rules to their respective directories
func InstallRules(homeDir string) error {
	rulesDirs := []string{
		filepath.Join(homeDir, ".gemini", "config", "rules"),
		filepath.Join(homeDir, ".cursor", "rules"),
		filepath.Join(homeDir, ".claude", "rules"),
		filepath.Join(homeDir, ".config", "opencode", "rules"),
		filepath.Join(homeDir, ".hermes", "rules"),
	}

	for _, dir := range rulesDirs {
		parentDir := filepath.Dir(dir)
		if _, err := os.Stat(parentDir); err == nil {
			_ = os.MkdirAll(dir, 0755)
			dest := filepath.Join(dir, "cogni.rules.md")
			_ = os.WriteFile(dest, []byte(RuleContent), 0644)
		}
	}

	// Instalar en local si .agents existe
	if _, err := os.Stat(".agents"); err == nil {
		localRules := filepath.Join(".agents", "rules")
		_ = os.MkdirAll(localRules, 0755)
		_ = os.WriteFile(filepath.Join(localRules, "cogni.rules.md"), []byte(RuleContent), 0644)
	}

	return nil
}

// GetHarnessSkillPaths returns all supported AI harness skill directory paths
func GetHarnessSkillPaths(homeDir string) map[string][]string {
	return map[string][]string{
		"local": {
			filepath.Join(".agents", "skills"),
		},
		"antigravity": {
			filepath.Join(homeDir, ".gemini", "config", "skills"),
		},
		"cursor": {
			filepath.Join(homeDir, ".cursor", "skills"),
		},
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
	}
}

// GetHarnessMCPPaths returns supported AI harness MCP configuration file paths
func GetHarnessMCPPaths(homeDir string) map[string][]string {
	return map[string][]string{
		"antigravity": {
			filepath.Join(homeDir, ".gemini", "antigravity-ide", "mcp_config.json"),
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
			filepath.Join(homeDir, ".config", "opencode", "mcp.json"),
		},
		"hermes": {
			filepath.Join(homeDir, ".hermes", "mcp.json"),
		},
	}
}

// ConfigureHarnessMCP inyecta automáticamente el servidor MCP 'cogni' en las configuraciones de MCP
func ConfigureHarnessMCP(homeDir string) map[string]string {
	mcpConfigs := GetHarnessMCPPaths(homeDir)
	configured := make(map[string]string)

	cogniBin := "cogni"
	localBin := filepath.Join(homeDir, ".local", "bin", "cogni")
	if _, err := os.Stat(localBin); err == nil {
		cogniBin = localBin
	}

	cogniEntry := map[string]any{
		"command": cogniBin,
		"args":    []string{"mcp"},
	}

	for harness, paths := range mcpConfigs {
		for _, cfgPath := range paths {
			parentDir := filepath.Dir(cfgPath)
			// Solo configurar si el directorio padre del arnés existe (evita crear carpetas fantasmas)
			if _, err := os.Stat(parentDir); err != nil {
				continue
			}

			if err := injectMCPServer(cfgPath, "cogni", cogniEntry); err == nil {
				configured[harness] = cfgPath
				break
			}
		}
	}

	// También configurar en workspace local si .agents/ existe
	if _, err := os.Stat(".agents"); err == nil {
		localMCP := filepath.Join(".agents", "mcp_config.json")
		if err := injectMCPServer(localMCP, "cogni", map[string]any{
			"command": "cogni",
			"args":    []string{"mcp"},
		}); err == nil {
			configured["workspace"] = localMCP
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
