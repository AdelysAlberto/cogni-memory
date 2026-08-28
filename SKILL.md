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

### 1. Two-Step Retrieval & Smart Task Tag-Matching
To prevent context inflation and avoid re-analyzing codebases:

- **Step 1: Lightweight Search (Discovery & Task Matching)**
  When requested to do any non-trivial task or bugfix, extract the main technical concept/tags from the prompt and execute a compact search:
  ```bash
  cogni search --query "<keywords_or_tags>"
  # Or via MCP Tool: cogni_search(query: "auth jwt")
  ```
- **Step 2: Full Content Hydration (Only for matching IDs/Keys)**
  Retrieve the complete synthetic signature only for the relevant ID or TopicKey to know exactly how to address the task without reading large files:
  ```bash
  cogni get <id_or_topic_key>
  # Or via MCP Tool: cogni_get(id: 6) / cogni_get(topic_key: "arch/auth/jwt")
  ```
- **Step 0: Quick Context Bootstrapping (`cogni_context`)**
  At session start or after compaction, call `cogni_context` to load recent sessions, decisions, and active conventions in under 100 tokens.

### 1.1 Proactive Preflight Search (Mandatory Triggers)
- **Architecture / New Feature**: Before proposing, designing, or scaffolding a new technical pattern, database table, API, state store, or auth flow, execute `cogni search` on the domain keyword.
- **Pre-fix Search**: Before implementing non-trivial bugfixes, search for previous resolutions in that module/error area.
- Adhere strictly to retrieved architectural patterns and previous decisions.

### 2. High-Signal Threshold & When to Save (Postflight Gate)
**GOLDEN RULE**: Call `cogni save` (or `cogni_save`) ONLY if: *If this memory signature does not exist in the future, will an agent waste time investigating, break an architecture, or make a mistake?*

**DELIVERY GUARANTEE (Saving is not replying)**:
- Saving to memory is internal bookkeeping. It NEVER counts as answering the user.
- Always save/update memory **BEFORE** generating your final text reply.
- End every turn with your complete user-facing answer as the final message (no tool calls after it).
- A failed or slow memory operation NEVER blocks or replaces your user reply.

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
- **`session`**: End-of-session or post-compaction milestone summaries.

### 3. High-Density Synthetic Signature Format (What / Why / Where / Learned)
Cogni is designed to eliminate context saturation by replacing 500-line file reads with High-Density Synthetic Signatures occupying under 5% of tokens:

- **Topic**: Hierarchical key (`<domain>/<subdomain>/<topic>`, e.g., `standards/i18n/ui`, `arch/auth/jwt`).
- **What**: One concise sentence — what was done or decided.
- **Why**: Motivation or root cause.
- **Where**: Affected relative files or paths.
- **Learned**: Non-obvious gotchas or learnings (omit if none).

*Format in signature*: `What: ... | Why: ... | Where: ... | Learned: ...`

```yaml
# Ideal Cogni Signature Example:
Topic: standards/i18n/ui
What: Todo texto visible en JSX/TSX debe usar t('namespace:key'). Prohibido texto literal.
Why: Estándar global del proyecto para soporte multi-idioma (es, en, pt, fr, ar).
Where: src/providers/i18n/, src/modules/*, src/layouts/
Learned: Cadenas en toast o modales también deben internacionalizarse.
```

### 4. Diagnostic & Maintenance Tooling
- **`cogni stats` / `cogni_stats()`**: Displays memory health, entry count, and estimated token savings metrics.
- **`cogni session_summary` / `cogni_session_summary()`**: Summarizes progress, discoveries, and next steps to resume context without reloading long chat histories.

### 5. Compaction & Session Lifecycle Protocol

#### End of Session (`cogni_session_summary`)
Before ending a session or stating "done", call `cogni_session_summary` (or `cogni session-summary`) with:
- **goal**: Main objective worked on.
- **accomplished**: Completed items with key technical details.
- **discoveries**: Findings, gotchas, or architectural decisions.
- **next_steps**: Pending items for the next session.
- **relevant_files**: Key files modified.

#### After Compaction / Context Reset (`FIRST ACTION REQUIRED`)
If a compaction message or reset occurs:
1. IMMEDIATELY call `cogni_session_summary` with the compacted summary content to persist pre-compaction progress into SQLite.
2. Call `cogni_context` to retrieve active project context.
3. Only THEN proceed with your task.

### 6. Deterministic Topic Keys & Automatic Upserts
To prevent duplicate records:
- Format: `<domain>/<subdomain>/<topic>` (ej. `arch/auth/jwt`, `standards/i18n/ui`, `session/latest`).
- When a `--topic-key` already exists, `cogni save` automatically updates (**upserts**) the record.

---

## 🛠️ Tooling & CLI Reference

### Native MCP Tools:
- `cogni_context(project, limit)`: Active context & recent sessions in < 100 tokens.
- `cogni_session_summary(goal, accomplished, discoveries, next_steps, relevant_files)`: Persist session summary.
- `cogni_search(query, project, category, limit)`: Lightweight discovery search.
- `cogni_get(id, topic_key, project)`: Full content hydration (Phase 2).
- `cogni_save(title, summary, what, why, where, learned, category, tags, topic_key, project, global)`: Structured save/upsert.
- `cogni_update(id, summary, title, category, tags, topic_key)`: Direct update by ID.
- `cogni_stats()`: Memory usage, health, and token metrics.

### CLI Commands:
```bash
# 1. Quick active context bootstrapping
cogni context

# 2. Save structured memory with discrete fields
cogni save \
  --topic-key "arch/auth/jwt" \
  --title "JWT Refresh Token Rotation" \
  --what "Implemented refresh token rotation with Redis blacklist" \
  --why "Mitigates replay attacks after security audit" \
  --where "src/auth/jwt.go, src/middleware/auth.go" \
  --learned "Redis TTL automatically manages expired blacklist keys" \
  --category "architecture" \
  --tags "auth,jwt,security"

# 3. Save end-of-session or post-compaction summary
cogni session-summary \
  --goal "Implement JWT Auth" \
  --accomplished "Created tokens endpoints and migrations" \
  --where "src/auth/jwt.go"

# 4. Search memories (Compact 1-line preview)
cogni search --query "jwt"

# 5. Retrieve full memory content (Phase 2)
cogni get arch/auth/jwt
```
