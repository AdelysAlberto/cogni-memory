# 🧠 Cogni Memory Invariants

## 🔍 1. Preflight Search (Mandatory)
- Before proposing, designing, or implementing a new feature, database schema, API route, state store, or non-trivial logic, execute **`cogni_search`** (MCP tool) or **`cogni search`** (CLI) for the domain keyword.
- If previous memories are retrieved:
  - Adhere strictly to the established patterns, configurations, and decisions.
  - Hydrate only the relevant memories using **`cogni_get`** or **`cogni get`**.

## 💾 2. Postflight Save Gate (Mandatory)
- Before completing any high-signal task (e.g. bugfix with non-obvious cause, architectural decision, library selection, build setup, coding standard), save or update it in memory.
- Use a deterministic **`topic_key`** (format: `<domain>/<subdomain>/<topic>`, ej. `arch/auth/jwt`) so that subsequent runs **upsert** existing records instead of generating duplicates.
- Structure every summary strictly as: `What: ... | Why: ... | Where: ... | Learned: ...`
- Tags must follow the 3-layer taxonomy (main concept, tech stack, specific module) and include primary technical English keywords (e.g. `utils`, `config`, `auth`).
- `topic_key` must be technical English (`config.util`, `arch/auth/jwt`). If `title` is in Spanish, include the English code alias in parentheses (e.g. `"Utilidad de Configuración (config.util)"`).
