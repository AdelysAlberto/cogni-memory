# 🧠 Cogni Memory Invariants & Context Optimization

## 🔍 1. Preflight Search & Task Tag Matching (Mandatory)
- Before proposing, designing, or implementing a new feature, API route, schema, state store, or bugfix, execute **`cogni_search`** (MCP) or **`cogni search`** (CLI) with the task tags/domain keywords.
- When previous memories exist, adhere strictly to established patterns and hydrate only required records with **`cogni_get`** to avoid reading entire source files.
- At session start, call **`cogni_context`** to load the active project context in minimal tokens.

## 💾 2. Postflight Save Gate & Delivery Guarantee (Mandatory)
- Before completing any high-signal task (bugfix, architectural decision, library selection, build setup, convention), save or update it in memory.
- Structure every summary as: `What: ... | Why: ... | Where: ... | Learned: ...`
- Use a deterministic **`topic_key`** (`<domain>/<subdomain>/<topic>`) so subsequent runs **upsert** existing records.
- **Delivery Guarantee**: Saving memory is internal bookkeeping. Always save BEFORE composing the final reply and never replace the complete user answer with a one-line "saved" acknowledgement.

## ⚡ 3. Compaction & Session Summary Protocol
- When a context compaction happens or you see "FIRST ACTION REQUIRED":
  1. Call **`cogni_session_summary`** immediately with the compacted summary to persist state.
  2. Call **`cogni_context`** to recover active project context.
  3. Continue with the task.
