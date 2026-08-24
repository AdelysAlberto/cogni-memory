---
name: cogni
description: Autonomous local memory system to query and store synthetic semantic signatures in SQLite, reducing token consumption by up to 95% across AI Agent environments (Antigravity, Cursor, Claude, Copilot, OpenCode, Hermes).
---

# 🧠 Cogni Skill (Autonomous AI Agent Memory System)

> *"Just as a Byte is the fundamental unit of raw data, a Cogni is the unit of synthetic knowledge for your AI agent."*

**Cogni** (*Cognitive Omniscient Grid for Networked Intelligence*) enables AI agents to **query, register, update, and manage synthetic semantic signatures** in a fast local or global SQLite database (`.cogni/memory.db` or `~/.cogni/memory.db`).

Its primary objective is to maintain architectural consistency across chat sessions while drastically reducing input token consumption by preventing repetitive reading of source code and documentation.

---

## ⚡ Autonomous Agent Operating Directives

### 1. Two-Step Retrieval Protocol (Token Optimization)
To prevent context inflation, retrieval ALWAYS follows two distinct phases:

- **Step 1: Lightweight Search (Discovery)**
  Run a compact search to inspect matching titles, categories, tags, and 1-line previews:
  ```bash
  cogni search --query "<keywords>"
  # Or via MCP Tool: cogni_search(query: "...")
  ```
- **Step 2: Full Content Hydration (Only for relevant IDs/Keys)**
  Retrieve the complete synthetic signature only for the chosen ID or TopicKey:
  ```bash
  cogni get <id_or_topic_key>
  # Or via MCP Tool: cogni_get(id: 6) / cogni_get(topic_key: "arch/auth/jwt")
  ```

### 1.1 Proactive Preflight Search (Mandatory Triggers)
- **Architecture / New Feature**: Before proposing, designing, or scaffolding a new technical pattern, database table, API, state store, or auth flow, execute `cogni search` on the domain keyword.
- **Pre-fix Search**: Before implementing non-trivial bugfixes, search for previous resolutions in that module/error area.
- Adhere strictly to retrieved architectural patterns and previous decisions.

### 2. High-Signal Threshold & When to Save (Postflight Gate)
**GOLDEN RULE**: Call `cogni save` (or `cogni_save`) ONLY if: *If this memory signature does not exist in the future, will an agent waste time investigating, break an architecture, or make a mistake?*

**MANDATORY TIMING**: Execute save/update **before** emitting the final text envelope to the user.

**DO NOT SAVE (Noise / Skip)**:
- ❌ Trivial metadata tasks (creating/modifying `LICENSE`, `.gitignore`, `.prettierrc`, cosmetic assets).
- ❌ Typo fixes, code formatting (`fmt`, `lint`), or minor documentation polishing.
- ❌ Self-evident information easily discovered by reading the first few lines of a file.

**HIGH-SIGNAL CATEGORIES (Must Save)**:
- **`bugfix`**: Resolution of a non-trivial error with a non-obvious root cause.
- **`architecture` / `decision`**: Choice of libraries, data schemas, API contracts, or system structures.
- **`discovery`**: Non-obvious technical finding or gotcha about runtime/codebase behavior.
- **`config`**: Non-trivial tooling, environment, script, or build setup.
- **`pattern`**: Established naming convention, folder structure, or coding standard.
- **`preference`**: User preference or technical constraint learned during the session.

### 3. Deterministic Topic Keys & Automatic Upserts
To prevent signature duplication and database fragmentation, use a structured `--topic-key`:
- Format: `<domain>/<subdomain>/<topic>` (ej. `arch/auth/jwt`, `sdd/cart/spec`, `pattern/react/forms`).
- When a `--topic-key` already exists in the project, `cogni save` **automatically updates (upserts)** the record instead of creating duplicates.

### 4. Synthetic Summary Format (`--summary`)
Every summary MUST follow this high-density 4-part structured format:
`What: <One sentence description of what was done> | Why: <Motivation or root cause> | Where: <Key files/paths affected> | Learned: <Gotchas or key learnings (omit if none)>`

### 5. 3-Layer Tag Taxonomy & Cross-Language Keywords
Include 3 to 5 lowercase, kebab-case tags:
1. **Layer 1 - Main Concept**: Generic technical domain (`pagination`, `auth`, `state-management`, `database`, `utils`).
2. **Layer 2 - Technology / Stack**: Exact tech stack (`go`, `sqlite`, `zustand`, `react`, `css-modules`, `redis`).
3. **Layer 3 - Specific Module**: Project domain entity (`products-list`, `jwt-middleware`, `config-util`).

**Bilingual & Technical Keyword Rule**:
- `topic_key` MUST ALWAYS be technical English (`config.util`, `arch/auth/jwt`).
- `title`: If Spanish is used in the title (e.g. `"Utilidad de Configuración Dinámica"`), include technical English terms/code identifiers in parentheses or topic suffix, e.g., `"Utilidad de Configuración Dinámica (Dynamic Config Utils) (config.util)"`.
- `tags`: Always include primary English technical keywords (e.g. `utils,config,redis`) so searches in either English or Spanish match effortlessly.

---

## 🛠️ Tooling & CLI Reference

### Native MCP Tools (When running in MCP-compatible environments):
- `cogni_search(query, project, category, limit)`: Lightweight discovery search (previews).
- `cogni_get(id, topic_key, project)`: Full content hydration.
- `cogni_save(title, summary, category, tags, topic_key, project, global)`: High-signal save/upsert.
- `cogni_update(id, summary, title, category, tags, topic_key)`: Direct update by ID.
- `cogni_stats()`: Memory usage and token metrics.

### CLI Commands:
```bash
# 1. Save / Upsert structured memory
cogni save \
  --topic-key "arch/auth/jwt" \
  --title "JWT Refresh Token Rotation" \
  --category "architecture" \
  --tags "auth,jwt,security" \
  --summary "What: Added refresh token rotation with blacklist | Why: Mitigates token replay | Where: src/auth/jwt.go | Learned: Requires redis TTL sync"

# 2. Search memories (Compact 1-line preview)
cogni search --query "jwt"

# 3. Retrieve full memory content (Phase 2)
cogni get arch/auth/jwt
# or: cogni get --id 6

# 4. Update memory by ID
cogni update --id 6 --summary "What: ... | Why: ... | Where: ... | Learned: ..."

# 5. Start native MCP stdio server
cogni mcp

# 6. Open visual web dashboard
cogni ui
```
