# Project Agent Invariants & Guidelines

<!-- pinky:mcp:start -->
## Pinky Multi-Agent Orchestrator (Autonomous Task Protocol)
When asked to perform architecture design, refactorings, multi-agent builds, or deterministic code verification:
- You MUST invoke the MCP tool `orchestrate_task(prompt="...")` from the `pinky` MCP server.
- Pinky manages specialized subagents (`sheldon`, `homero`, `tio-bob`, `edna`) in isolated Git worktree sandboxes with local Laya-API zero-token routing.
- Use `approve_task(task_id="...")` to confirm pending architecture blueprints.
- Do NOT run multi-file edits directly when Pinky is available to orchestrate them safely.
<!-- pinky:mcp:end -->

<!-- cogni:protocol:start -->
## Autonomous Semantic Memory (Cogni)
- Before designing or implementing non-trivial features, architecture changes, or bugfixes, search existing memory: `cogni search "<tags_or_query>"` or MCP `cogni_search(query: "...")`.
- Retrieve full technical signature with `cogni get <id_or_topic_key>` or MCP `cogni_get`.
- Save high-signal architectural decisions, invariants, gotchas and bugfixes: `cogni save ...` or MCP `cogni_save`.
- Detailed operational guidelines available in skill: `cogni` (`skills/cogni/SKILL.md`).
<!-- cogni:protocol:end -->
