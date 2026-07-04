# MCP Transport

**Updated:** 2026-07-04
**Commit:** b895c6b
**Branch:** main

## OVERVIEW

`internal/api/mcp` implements the `/mcp` endpoint using the MCP Go SDK. It owns MCP transport checks, tool definitions, argument schemas, structured results, search/reference helpers, and exactly-once semantics for write tools through idempotency keys.

## STRUCTURE

```text
mcp/
├── mcp_handler.go        # MCPHandler, SDK server wiring, auth, method handling
├── mcp_schema.go         # Tool definitions, JSON-schema helpers, read/write annotations
├── mcp_tools.go          # Tool implementations for subscriptions and reference data
├── mcp_args.go           # Argument parsing helpers
├── mcp_results.go        # Structured MCP result helpers
├── mcp_search.go         # Search/filter helpers
├── mcp_idempotency.go    # Idempotent write wrapper and replay semantics
├── response.go           # MCP response helpers
└── mcp_test.go           # End-to-end and tool-level MCP tests
```

## WHERE TO LOOK

| Task | Start here | Notes |
|------|------------|-------|
| Add or change a tool | `mcp_schema.go`, `mcp_tools.go` | Update schema and implementation together |
| Change MCP transport behavior | `mcp_handler.go` | Preserve SDK transport and method-handling assumptions |
| Change write idempotency | `mcp_idempotency.go` | Keep replay semantics, audit ordering, and stored-result behavior stable |
| Change tool result shape | `mcp_results.go` | Prefer structured objects, not ad hoc string blobs |
| Change search/reference helpers | `mcp_search.go`, `mcp_tools.go` | Keep result fields stable for clients |

## CONVENTIONS

- MCP remains API-key based and narrower than the REST API surface.
- Keep `X-API-Key`, `Origin`, `Content-Type`, `Accept`, and protocol-version checks in front of SDK tool execution.
- Define schemas and implementations together. Unknown or stray write arguments should remain invalid so idempotency fingerprints stay stable.
- Write tools must require `idempotency_key` and run through the shared idempotent write path.
- Preserve audit behavior and transaction boundaries for write tools: mutation, audit event, and idempotency record should succeed or fail together.
- Do not casually expose admin, export, account, calendar-token, or notification-management capabilities through MCP.

## TESTING

```bash
go test -count=1 ./internal/api/mcp ./internal/api
```

Cover transport errors, schema validation, idempotency replay, and hostile request cases when changing MCP behavior.
