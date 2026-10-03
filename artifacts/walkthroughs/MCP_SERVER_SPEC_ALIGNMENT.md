---
title: MCP Server Spec Alignment & Agent Skills Standardization
module: mcp / core / skills
author: homero
date: 2026-10-03
status: Done
priority: P1
scope: "[Phase 1 & 2]"
source: user_request
---

# Walkthrough: MCP Server Spec Alignment & Agent Skills Standardization

## User Request
Evaluate Cogni's MCP server and Agent Skill against official standards ([server-concepts](https://modelcontextprotocol.io/docs/2026-07-28/learn/server-concepts), [build-with-agent-skills](https://modelcontextprotocol.io/docs/2026-07-28/develop/build-with-agent-skills), [build-server](https://modelcontextprotocol.io/docs/2026-07-28/develop/build-server)), eliminate false positives and assumptions, and implement complete improvements across Phase 1 (MCP server) and Phase 2 (Skills & Harness integration).

## Solution Applied

### 1. Phase 1: Complete Model Context Protocol (MCP) Triad Implementation
Prior to this task, Cogni only exposed Tools, omitting the other two foundational pillars of MCP.
- **Capabilities Handshake**: Updated `initialize` to negotiate `protocolVersion: 2024-11-05` and declare full capabilities for `tools`, `resources`, and `prompts`.
- **MCP Resources**:
  - `resources/list`: Exposes direct resources `cogni://context/recent` and `cogni://session/latest`.
  - `resources/templates/list`: Exposes dynamic resource templates `cogni://memory/{id}` and `cogni://topic/{topic_key}`.
  - `resources/read`: Implemented with support for URL query parameters (e.g. `?project=...`) and intelligent fallbacks to local storage if the folder was renamed.
- **MCP Prompts**:
  - `prompts/list`: Exposes guided workflows `cogni_preflight_check` and `cogni_session_summary`.
  - `prompts/get`: Renders parameter-populated prompt messages for clients that support UI slash commands.
- **JSON Schema Hardening**:
  - Extended JSON Schema structures to support `items`, `oneOf`, and `additionalProperties: false`.
  - Fixed `tags` parameter in `cogni_save` to explicitly allow both comma-separated strings and string arrays (`oneOf`).
  - Standardized tool and property descriptions into clear, universal technical English to maximize LLM tool-calling accuracy across models.

### 2. Phase 2: Agent Skill & Harness Distribution Harmonization
- **Skill Discovery (`SKILL.md`)**:
  - Rewrote the YAML frontmatter `description` to state explicit activation triggers (`preflight search`, `postflight save`, `context reset`).
  - Added a strict **Tool Invocation Hierarchy** section: Native MCP tools (Tier 1) > Resources/Prompts (Tier 2) > Terminal CLI fallback (Tier 3).
- **Embedded Skill (`internal/core/skill.go`)**:
  - Synchronized `SkillContent` with the updated `SKILL.md`.
  - Removed erroneous deletion logic in `InstallRules` that previously wiped `skills/cogni` in Antigravity and Cursor environments.
  - Enabled skill paths for Antigravity (`~/.gemini/config/skills`) and Cursor (`~/.cursor/skills`) in `GetHarnessSkillPaths` so rules and skills coexist harmoniously.
- **Documentation**:
  - Updated `README.md` to document the full triad of Tools, Resources, and Prompts.

## Relevant Technical Decisions
- **Passive Context vs. Active Function Calls**: Exposing `cogni://context/recent` allows clients to inject active context without burning a tool call roundtrip.
- **Universal English in Schema**: Model tool classification benchmarks show higher accuracy when tool metadata is in English, while allowing multilingual content in the payload.

## Validation Performed
- `go test -v ./...`: All unit and integration tests passed across all packages (`internal/mcp`, `internal/core`, `internal/storage`, `internal/network`, `internal/platform`).
- `go build -v -o bin/cogni ./cmd/cogni`: Binary builds cleanly without warnings.
- Stdio JSON-RPC verification: Confirmed via piped JSON-RPC commands that `./bin/cogni mcp` handles `initialize`, `resources/list`, and `prompts/list` flawlessly.
