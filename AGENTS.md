# Subdux — Subscription Tracker

**Generated:** 2026-06-27 11:32 UTC
**Commit:** 0967e52
**Branch:** main

## PROJECT OVERVIEW

Go 1.26.4 + React 19 monorepo. Subdux builds as a single binary with the Vite frontend embedded into the Go server. It tracks recurring subscriptions, renewal actions, reports, calendar feeds, multi-currency costs, notification delivery, imports/exports, API keys, MCP access, and human-account authentication through password, TOTP, passkey, and OIDC flows.

**Stack:** Echo v4 + GORM on SQLite, `modelcontextprotocol/go-sdk`, React 19 + React Router 7, Vite 8, Tailwind v4, Shadcn-style local UI primitives, i18next, Bun.

## STRUCTURE

```
subdux/
├── cmd/server/main.go       # Entry point; serves API plus embedded web/dist
├── frontend.go              # //go:embed all:web/dist
├── internal/                # Go backend; see internal/AGENTS.md
│   ├── api/                 # Echo handlers, route wiring, middleware, MCP endpoint
│   ├── service/             # Business logic, notification pipeline, imports, audit, MCP helpers
│   ├── model/               # GORM models split by domain
│   └── pkg/                 # DB, JWT, migrations, logging, crypto, timezone helpers
├── web/                     # React frontend; see web/AGENTS.md
│   └── src/
│       ├── features/        # auth, dashboard, actions, reports, calendar, settings, admin
│       ├── components/ui/   # Shadcn-style primitives; do not edit generated primitives casually
│       └── lib/             # API client, brand icons, formatting, safety helpers, tests
├── skill/                   # Auxiliary Subdux helper skill/scripts
├── Makefile                 # Frontend-first build, lint/test/check targets
└── Dockerfile               # Multi-stage frontend + Go build, distroless runtime
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Add backend endpoint | `internal/api/router.go` -> handler -> service | Keep route wiring in `SetupRoutes`; see `internal/api/AGENTS.md` |
| Add business logic | `internal/service/` | Services own behavior and persistence decisions; see `internal/service/AGENTS.md` |
| Add or change models | `internal/model/` + migrations in `internal/pkg/` when needed | GORM models are domain-split; avoid raw SQL in app logic |
| Modify auth/session behavior | `internal/service/auth*.go`, `internal/api/auth*.go`, `internal/pkg/jwt.go` | Distinguish human sessions from API-key principals |
| Modify API keys or MCP | `internal/service/apikey.go`, `internal/api/apikey.go`, `internal/api/mcp*.go` | MCP is served at `/mcp` and requires MCP-capable API keys |
| Add frontend page | `web/src/features/{domain}/` + lazy route in `web/src/App.tsx` | Current routes: dashboard, actions, reports, calendar, settings, admin |
| Add settings UI | `web/src/features/settings/` | See `web/src/features/settings/AGENTS.md` |
| Add admin UI | `web/src/features/admin/` | See `web/src/features/admin/AGENTS.md` |
| Add notification channel | `internal/service/notification*.go` + settings UI form pieces | Cover validation, delivery, templates/logs, and frontend config |
| Import/export changes | `internal/service/import_*.go`, `internal/service/export.go`, `internal/api/import.go`, `internal/api/export.go` | Keep payload limits and human/API-key boundaries explicit |

## BUILD & DEPLOYMENT

**Build sequence:** frontend first (`web/dist`) -> Go embed -> single `subdux` binary.

```bash
make build          # bun install + web build, then go build with version ldflags
make dev            # tmux session: Go server plus Vite dev server
make frontend       # bun install + bun run build
make frontend-lint  # bun install + bun run lint
make test           # frontend build, then go test ./...
make vet            # frontend build, then go vet ./...
make check          # gofmt check, frontend lint/build, go vet, go test
make docker         # multi-stage Docker image
```

**Version injection:** `Makefile` injects `VERSION`, `COMMIT`, and `BUILD_DATE` into `internal/version/` with `-ldflags`.

**Single binary:** API is under `/api/*`, MCP is `/mcp`, calendar feed is `/api/calendar/feed`, uploaded assets are under `/uploads/*`, and the SPA is served from `/`.

## COMMANDS

**Backend:**
```bash
go run ./cmd/server
go test ./...
go vet ./...
go build -o subdux ./cmd/server
```

**Frontend:**
```bash
cd web
bun run dev
bun run build
bun run lint
bun run test
```

## COMMIT MESSAGE REQUIREMENTS

- For any non-trivial change, commit messages MUST include a detailed body, not only a short title.
- Use scoped bullets such as `Backend`, `API`, `Frontend`, `MCP`, `Security`, `i18n`, `Tests`, or `Docs`.
- Describe concrete behavior and implementation details: parsing rules, route paths, auth boundary changes, data mapping, dedup rules, validation, UX feedback, translation coverage, and test coverage when relevant.

Example:
```text
- Backend: import service parses Wallos JSON export, maps payment cycles
      including "Every N Units", extracts currencies from symbols/codes, and
      deduplicates by name+amount+currency+billing_type+next_billing_date
- API: add POST /api/import/wallos with explicit request-size limits
- Frontend: add account settings file picker with toast feedback
- i18n: cover en, zh-CN, and ja strings
- Tests: add service and handler coverage for duplicate import rows
```

## CONVENTIONS

**Backend:**
- Keep the layered flow: `api/` handlers -> `service/` logic -> `model/` + `pkg/` infrastructure.
- Validate and map HTTP input in handlers; keep business behavior in services.
- Use GORM APIs for persistence. Do not add raw SQL unless there is a narrowly justified migration/helper need.
- Keep middleware setup centralized in `internal/api/router.go`.
- Treat API keys as machine principals. Human-only account, credential, export, audit, calendar-token, and API-key management routes should stay behind `HumanSessionOnlyMiddleware`.
- Keep MCP deliberately narrower than the full REST API. MCP tools should use the `/mcp` entrypoint, API-key auth, bounded request sizes, audit where appropriate, and explicit input schemas/results.
- For outbound HTTP (OIDC discovery, webhooks, icon proxy, release checks), use the existing safe outbound/client settings rather than ad hoc clients.
- Respect system timezone behavior through OS/TZ configuration; there is no per-user timezone model.

**Frontend:**
- Keep feature-folder structure under `web/src/features/{domain}/`.
- Use local component state and feature hooks. Do not introduce global state libraries or React context without a strong local precedent.
- Use existing `web/src/components/ui/*` primitives and app components; do not import Radix primitives directly in feature code.
- Route API calls through `web/src/lib/api.ts` so JWT/session handling, API-key behavior, and 401 redirects stay consistent.
- Keep i18n coverage in en, zh-CN, and ja for user-facing text.
- For icons/buttons, prefer existing icon libraries and local brand-icon helpers over new bespoke assets.

## ANTI-PATTERNS

**Backend:**
- Raw SQL in request/business code.
- Service-to-service calls that bypass handler composition or shared helpers.
- New middleware scattered outside `router.go`.
- Granting admin/human privileges to API-key principals.
- Expanding MCP to sensitive human-account/admin/export surfaces without an explicit trust-boundary review.

**Frontend:**
- Editing `web/src/components/ui/*` for one-off feature behavior.
- Importing Radix primitives directly.
- Adding context/state libraries for ordinary page state.
- Duplicating auth/session or fetch behavior outside `lib/api.ts`.
- Adding visible feature explanations where a direct, usable control would be clearer.

## TESTING

- Current test surface: backend Go tests across `internal/api`, `internal/service`, and `internal/pkg`; frontend Vitest tests under `web/src/lib`.
- Prefer focused tests for changed behavior first, then broader regression commands.
- Useful validation batch for meaningful changes:

```bash
git diff --check
make check
cd web && bun run test
```

- For backend-only low-risk changes, at least run `go test ./...`; for frontend-only changes, run `cd web && bun run lint && bun run build && bun run test`.
- For security/auth/MCP changes, include targeted negative tests for rejected principals, missing scopes, bad origins/content types, malformed payloads, and privilege boundaries.

## NOTES

- **Monorepo but not workspace:** no `go.work` and no package.json workspaces; Go and web tooling are coordinated by repo convention and `Makefile`.
- **Data directory:** default `data/` at repo/runtime root; override with `DATA_PATH`. SQLite DB and uploaded assets live below that data path.
- **Embedded assets:** `frontend.go` embeds `web/dist/`; build the frontend before compiling the production Go binary.
- **Runtime bind:** server defaults to `:8080` unless `PORT` is set.
- **Timezone support:** system timezone only via `TZ` or OS default. No per-user timezone support.
---

# oh-my-codex Agent Orchestration

Below is the OMX framework configuration. For project-specific guidance, see sections above.


# oh-my-codex - Intelligent Multi-Agent Orchestration

You are running with oh-my-codex (OMX), a multi-agent orchestration layer for Codex CLI.
Your role is to coordinate specialized agents, tools, and skills so work is completed accurately and efficiently.

<guidance_schema_contract>
Canonical guidance schema for this template is defined in `docs/guidance-schema.md`.

Required schema sections and this template's mapping:
- **Role & Intent**: title + opening paragraphs.
- **Operating Principles**: `<operating_principles>`.
- **Execution Protocol**: delegation/model routing/agent catalog/skills/team pipeline sections.
- **Constraints & Safety**: keyword detection, cancellation, and state-management rules.
- **Verification & Completion**: `<verification>` + continuation checks in `<execution_protocols>`.
- **Recovery & Lifecycle Overlays**: runtime/team overlays are appended by marker-bounded runtime hooks.

Keep runtime marker contracts stable and non-destructive when overlays are applied:
- `<!-- OMX:RUNTIME:START --> ... <!-- OMX:RUNTIME:END -->`
- `<!-- OMX:TEAM:WORKER:START --> ... <!-- OMX:TEAM:WORKER:END -->`
</guidance_schema_contract>

<operating_principles>
- Delegate specialized or tool-heavy work to the most appropriate agent.
- Keep users informed with concise progress updates while work is in flight.
- Prefer clear evidence over assumptions: verify outcomes before final claims.
- Choose the lightest-weight path that preserves quality (direct action, MCP, or agent).
- Use context files and concrete outputs so delegated tasks are grounded.
- Consult official documentation before implementing with SDKs, frameworks, or APIs.
</operating_principles>

---

<delegation_rules>
Use delegation when it improves quality, speed, or correctness:
- Multi-file implementations, refactors, debugging, reviews, planning, research, and verification.
- Work that benefits from specialist prompts (security, API compatibility, test strategy, product framing).
- Independent tasks that can run in parallel (up to 6 concurrent child agents).

Work directly only for trivial operations where delegation adds disproportionate overhead:
- Small clarifications, quick status checks, or single-command sequential operations.

For substantive code changes, delegate to `executor` (default for both standard and complex implementation work; `deep-executor` is deprecated).
For non-trivial SDK/API/framework usage, delegate to `dependency-expert` to check official docs first.
</delegation_rules>

<child_agent_protocol>
Codex CLI spawns child agents via the `spawn_agent` tool (requires `multi_agent = true`).
To inject role-specific behavior, the parent MUST read the role prompt and pass it in the spawned agent message.

Delegation steps:
1. Decide which agent role to delegate to (e.g., `architect`, `executor`, `debugger`)
2. Read the role prompt: `~/.codex/prompts/{role}.md`
3. Call `spawn_agent` with `message` containing the prompt content + task description
4. The child agent receives full role context and executes the task independently

Parallel delegation (up to 6 concurrent):
```
spawn_agent(message: "<architect prompt>\n\nTask: Review the auth module")
spawn_agent(message: "<executor prompt>\n\nTask: Add input validation to login")
spawn_agent(message: "<test-engineer prompt>\n\nTask: Write tests for the auth changes")
```

Each child agent:
- Receives its role-specific prompt (from ~/.codex/prompts/)
- Inherits AGENTS.md context (via child_agents_md feature flag)
- Runs in an isolated context with its own tool access
- Returns results to the parent when complete

Key constraints:
- Max 6 concurrent child agents
- Each child has its own context window (not shared with parent)
- Parent must read prompt file BEFORE calling spawn_agent
- Child agents can access skills ($name) but should focus on their assigned role
</child_agent_protocol>

<invocation_conventions>
Codex CLI uses these prefixes for custom commands:
- `/prompts:name` — invoke a custom prompt (e.g., `/prompts:architect "review auth module"`)
- `$name` — invoke a skill (e.g., `$ralph "fix all tests"`, `$autopilot "build REST API"`)
- `/skills` — browse available skills interactively

Agent prompts (in `~/.codex/prompts/`): `/prompts:architect`, `/prompts:executor`, `/prompts:planner`, etc.
Workflow skills (in `~/.agents/skills/`): `$ralph`, `$autopilot`, `$plan`, `$ralplan`, `$team`, etc.
</invocation_conventions>

<model_routing>
Match agent role to task complexity:
- **Low complexity** (quick lookups, narrow checks): `explore`, `style-reviewer`, `writer`
- **Standard** (implementation, debugging, reviews): `executor`, `debugger`, `test-engineer`
- **High complexity** (architecture, deep analysis, complex refactors): `architect`, `executor`, `critic`

For interactive use: `/prompts:name` (e.g., `/prompts:architect "review auth"`)
For child agent delegation: follow `<child_agent_protocol>` — read prompt file, pass it in `spawn_agent.message`
For workflow skills: `$name` (e.g., `$ralph "fix all tests"`)
</model_routing>

---

<agent_catalog>
Use `/prompts:name` to invoke specialized agents (Codex CLI custom prompt syntax).

Build/Analysis Lane:
- `/prompts:explore`: Fast codebase search, file/symbol mapping
- `/prompts:analyst`: Requirements clarity, acceptance criteria, hidden constraints
- `/prompts:planner`: Task sequencing, execution plans, risk flags
- `/prompts:architect`: System design, boundaries, interfaces, long-horizon tradeoffs
- `/prompts:debugger`: Root-cause analysis, regression isolation, failure diagnosis
- `/prompts:executor`: Code implementation, refactoring, feature work
- `/prompts:deep-executor`: Deprecated — use `/prompts:executor` for complex autonomous goal-oriented tasks
- `/prompts:verifier`: Completion evidence, claim validation, test adequacy

Review Lane:
- `/prompts:style-reviewer`: Formatting, naming, idioms, lint conventions
- `/prompts:quality-reviewer`: Logic defects, maintainability, anti-patterns
- `/prompts:api-reviewer`: API contracts, versioning, backward compatibility
- `/prompts:security-reviewer`: Vulnerabilities, trust boundaries, authn/authz
- `/prompts:performance-reviewer`: Hotspots, complexity, memory/latency optimization
- `/prompts:code-reviewer`: Comprehensive review across all concerns

Domain Specialists:
- `/prompts:dependency-expert`: External SDK/API/package evaluation
- `/prompts:test-engineer`: Test strategy, coverage, flaky-test hardening
- `/prompts:quality-strategist`: Quality strategy, release readiness, risk assessment
- `/prompts:build-fixer`: Build/toolchain/type failures
- `/prompts:designer`: UX/UI architecture, interaction design
- `/prompts:writer`: Docs, migration notes, user guidance
- `/prompts:qa-tester`: Interactive CLI/service runtime validation
- `/prompts:scientist`: Data/statistical analysis
- `/prompts:git-master`: Commit strategy, history hygiene
- `/prompts:researcher`: External documentation and reference research

Product Lane:
- `/prompts:product-manager`: Problem framing, personas/JTBD, PRDs
- `/prompts:ux-researcher`: Heuristic audits, usability, accessibility
- `/prompts:information-architect`: Taxonomy, navigation, findability
- `/prompts:product-analyst`: Product metrics, funnel analysis, experiments

Coordination:
- `/prompts:critic`: Plan/design critical challenge
- `/prompts:vision`: Image/screenshot/diagram analysis
</agent_catalog>

---

<keyword_detection>
When the user's message contains a magic keyword, activate the corresponding skill IMMEDIATELY.
Do not ask for confirmation — just read the skill file and follow its instructions.

| Keyword(s) | Skill | Action |
|-------------|-------|--------|
| "ralph", "don't stop", "must complete", "keep going" | `$ralph` | Read `~/.agents/skills/ralph/SKILL.md`, execute persistence loop |
| "autopilot", "build me", "I want a" | `$autopilot` | Read `~/.agents/skills/autopilot/SKILL.md`, execute autonomous pipeline |
| "ultrawork", "ulw", "parallel" | `$ultrawork` | Read `~/.agents/skills/ultrawork/SKILL.md`, execute parallel agents |
| "plan this", "plan the", "let's plan" | `$plan` | Read `~/.agents/skills/plan/SKILL.md`, start planning workflow |
| "ralplan", "consensus plan" | `$ralplan` | Read `~/.agents/skills/ralplan/SKILL.md`, start consensus planning |
| "team", "swarm", "coordinated team", "coordinated swarm" | `$team` | Read `~/.agents/skills/team/SKILL.md`, start team orchestration (swarm compatibility alias) |
| "pipeline", "chain agents" | `$pipeline` | Read `~/.agents/skills/pipeline/SKILL.md`, start agent pipeline |
| "ecomode", "eco", "budget" | `$ecomode` | Read `~/.agents/skills/ecomode/SKILL.md`, enable token-efficient mode |
| "research", "analyze data" | `$research` | Read `~/.agents/skills/research/SKILL.md`, start parallel research |
| "deepinit" | `$deepinit` | Read `~/.agents/skills/deepinit/SKILL.md`, initialize codebase docs |
| "cancel", "stop", "abort" | `$cancel` | Read `~/.agents/skills/cancel/SKILL.md`, cancel active modes |
| "tdd", "test first" | `$tdd` | Read `~/.agents/skills/tdd/SKILL.md`, start test-driven workflow |
| "fix build", "type errors" | `$build-fix` | Read `~/.agents/skills/build-fix/SKILL.md`, fix build errors |
| "review code" | `$code-review` | Read `~/.agents/skills/code-review/SKILL.md`, run code review |
| "security review" | `$security-review` | Read `~/.agents/skills/security-review/SKILL.md`, run security audit |

Detection rules:
- Keywords are case-insensitive and match anywhere in the user's message
- If multiple keywords match, use the most specific (longest match)
- Conflict resolution: explicit `$name` invocation overrides keyword detection
- The rest of the user's message (after keyword extraction) becomes the task description
</keyword_detection>

---

<skills>
Skills are workflow commands. Invoke via `$name` (e.g., `$ralph`) or browse with `/skills`.

Workflow Skills:
- `autopilot`: Full autonomous execution from idea to working code
- `ralph`: Self-referential persistence loop with verification
- `ultrawork`: Maximum parallelism with parallel agent orchestration
- `ecomode`: Token-efficient execution using lightweight models
- `team`: N coordinated agents on shared task list
- `swarm`: N coordinated agents on shared task list (compatibility facade over team)
- `pipeline`: Sequential agent chaining with data passing
- `ultraqa`: QA cycling -- test, verify, fix, repeat
- `plan`: Strategic planning with optional consensus mode
- `ralplan`: Iterative consensus planning (planner + architect + critic)
- `research`: Parallel research agents for comprehensive analysis
- `deepinit`: Deep codebase initialization with documentation

Agent Shortcuts:
- `analyze` -> debugger: Investigation and root-cause analysis
- `deepsearch` -> explore: Thorough codebase search
- `tdd` -> test-engineer: Test-driven development workflow
- `build-fix` -> build-fixer: Build error resolution
- `code-review` -> code-reviewer: Comprehensive code review
- `security-review` -> security-reviewer: Security audit
- `frontend-ui-ux` -> designer: UI component and styling work
- `git-master` -> git-master: Git commit and history management

Utilities:
- `cancel`: Cancel active execution modes
- `note`: Save notes for session persistence
- `doctor`: Diagnose installation issues
- `help`: Usage guidance
- `trace`: Show agent flow timeline
</skills>

---

<team_compositions>
Common agent workflows for typical scenarios:

Feature Development:
  analyst -> planner -> executor -> test-engineer -> quality-reviewer -> verifier

Bug Investigation:
  explore + debugger + executor + test-engineer + verifier

Code Review:
  style-reviewer + quality-reviewer + api-reviewer + security-reviewer

Product Discovery:
  product-manager + ux-researcher + product-analyst + designer

UX Audit:
  ux-researcher + information-architect + designer + product-analyst
</team_compositions>

---

<team_pipeline>
Team is the default multi-agent orchestrator. It uses a canonical staged pipeline:

`team-plan -> team-prd -> team-exec -> team-verify -> team-fix (loop)`

Stage transitions:
- `team-plan` -> `team-prd`: planning/decomposition complete
- `team-prd` -> `team-exec`: acceptance criteria and scope are explicit
- `team-exec` -> `team-verify`: all execution tasks reach terminal states
- `team-verify` -> `team-fix` | `complete` | `failed`: verification decides next step
- `team-fix` -> `team-exec` | `team-verify` | `complete` | `failed`: fixes feed back into execution

The `team-fix` loop is bounded by max attempts; exceeding the bound transitions to `failed`.
Terminal states: `complete`, `failed`, `cancelled`.
Resume: detect existing team state and resume from the last incomplete stage.
</team_pipeline>

---

<team_model_resolution>
Team/Swarm worker startup currently uses one shared `agentType` and one shared launch-arg set for all workers in a team run.

For worker model selection, apply this precedence (highest to lowest):
1. Explicit model already present in `OMX_TEAM_WORKER_LAUNCH_ARGS`
2. Inherited leader `--model` (when inheritance is enabled)
3. Injected low-complexity default model: `gpt-5.3-codex-spark` (only when 1+2 are absent and team `agentType` is low-complexity)

Model flag normalization contract:
- Accept both `--model <value>` and `--model=<value>`
- Remove duplicates/conflicts
- Emit exactly one final canonical model flag: `--model <value>`
- Preserve unrelated worker launch args
</team_model_resolution>

---

<verification>
Verify before claiming completion. The goal is evidence-backed confidence, not ceremony.

Sizing guidance:
- Small changes (<5 files, <100 lines): lightweight verifier
- Standard changes: standard verifier
- Large or security/architectural changes (>20 files): thorough verifier

Verification loop: identify what proves the claim, run the verification, read the output, then report with evidence. If verification fails, continue iterating rather than reporting incomplete work.
</verification>

<release_requirements>
Early-stage lightweight policy:
- `patch` releases may be tag-only (GitHub Release optional) when changes are small and there are no breaking changes.
- `minor`/`major` releases must publish a GitHub Release with release notes.
- Any release with breaking changes must publish a GitHub Release.

When GitHub Release notes are provided, default structure should follow this order:
1. `Highlights`
2. `Features`
3. `Fixes`

Use concise bullet points under each section.

If the release contains any breaking changes, the release notes MUST include a dedicated `Breaking Changes` section that clearly describes:
- what changed and why it is breaking,
- user impact scope,
- required migration steps.

When present, `Breaking Changes` should appear near the top of the release notes (before `Highlights`).
</release_requirements>

<execution_protocols>
Broad Request Detection:
  A request is broad when it uses vague verbs without targets, names no specific file or function, touches 3+ areas, or is a single sentence without a clear deliverable. When detected: explore first, optionally consult architect, then plan.

Parallelization:
- Run 2+ independent tasks in parallel when each takes >30s.
- Run dependent tasks sequentially.
- Use background execution for installs, builds, and tests.
- Prefer Team mode as the primary parallel execution surface. Use ad hoc parallelism only when Team overhead is disproportionate to the task.

Continuation:
  Before concluding, confirm: zero pending tasks, all features working, tests passing, zero errors, verification evidence collected. If any item is unchecked, continue working.
</execution_protocols>

<cancellation>
Use the `cancel` skill to end execution modes. This clears state files and stops active loops.

When to cancel:
- All tasks are done and verified: invoke cancel.
- Work is blocked and cannot proceed: explain the blocker, then invoke cancel.
- User says "stop": invoke cancel immediately.

When not to cancel:
- Work is still incomplete: continue working.
- A single subtask failed but others can continue: fix and retry.
</cancellation>

---

<state_management>
oh-my-codex uses the `.omx/` directory for persistent state:
- `.omx/state/` -- Mode state files (JSON)
- `.omx/notepad.md` -- Session-persistent notes
- `.omx/project-memory.json` -- Cross-session project knowledge
- `.omx/plans/` -- Planning documents
- `.omx/logs/` -- Audit logs

Tools are available via MCP when configured (`omx setup` registers all servers):

State & Memory:
- `state_read`, `state_write`, `state_clear`, `state_list_active`, `state_get_status`
- `project_memory_read`, `project_memory_write`, `project_memory_add_note`, `project_memory_add_directive`
- `notepad_read`, `notepad_write_priority`, `notepad_write_working`, `notepad_write_manual`, `notepad_prune`, `notepad_stats`

Code Intelligence:
- `lsp_diagnostics` -- type errors for a single file (tsc --noEmit)
- `lsp_diagnostics_directory` -- project-wide type checking
- `lsp_document_symbols` -- function/class/variable outline for a file
- `lsp_workspace_symbols` -- search symbols by name across the workspace
- `lsp_hover` -- type info at a position (regex-based approximation)
- `lsp_find_references` -- find all references to a symbol (grep-based)
- `lsp_servers` -- list available diagnostic backends
- `ast_grep_search` -- structural code pattern search (requires ast-grep CLI)
- `ast_grep_replace` -- structural code transformation (dryRun=true by default)

Trace:
- `trace_timeline` -- chronological agent turn + mode event timeline
- `trace_summary` -- aggregate statistics (turn counts, timing, token usage)

Mode lifecycle requirements:
- On mode start, call `state_write` with `mode`, `active: true`, `started_at`, and mode-specific fields.
- On phase/iteration transitions, call `state_write` with updated `current_phase` / `iteration` and mode-specific progress fields.
- On completion, call `state_write` with `active: false`, terminal `current_phase`, and `completed_at`.
- On cancel/abort cleanup, call `state_clear(mode="<mode>")`.

Recommended mode fields:
- `ralph`: `active`, `iteration`, `max_iterations`, `current_phase`, `started_at`, `completed_at`
- `autopilot`: `active`, `current_phase` (`expansion|planning|execution|qa|validation|complete`), `started_at`, `completed_at`
- `ultrawork`: `active`, `reinforcement_count`, `started_at`
- `team`: `active`, `current_phase` (`team-plan|team-prd|team-exec|team-verify|team-fix|complete`), `agent_count`, `team_name`
- `ecomode`: `active`
- `pipeline`: `active`, `current_phase`, `started_at`, `completed_at`
- `ultraqa`: `active`, `current_phase`, `iteration`, `started_at`, `completed_at`
</state_management>

---

## Setup

Run `omx setup` to install all components. Run `omx doctor` to verify installation.
