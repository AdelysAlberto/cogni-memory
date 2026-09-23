# Project Agent Invariants & Guidelines

<!-- pinky:mcp:start -->
## Pinky Multi-Agent Orchestrator (Autonomous Task Protocol)
When asked to perform architecture design, refactorings, multi-agent builds, or deterministic code verification:
- You MUST invoke the MCP tool `orchestrate_task(prompt="...")` from the `pinky` MCP server.
- Pinky manages specialized subagents (`sheldon`, `homero`, `tio-bob`, `edna`) in isolated Git worktree sandboxes with local Laya-API zero-token routing.
- Use `approve_task(task_id="...")` to confirm pending architecture blueprints.
- Do NOT run multi-file edits directly when Pinky is available to orchestrate them safely.
<!-- pinky:mcp:end -->
