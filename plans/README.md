# Improvement plans — stake-go (MCP & auth focus)

Self-contained implementation plans produced by a read-only advisory audit of the
**uncommitted changes** to this repo, scoped to the MCP server (`cmd/mcp-server/`)
and the authentication path. Each plan is written for a fresh executor with **no
context from the audit session** — read the plan top to bottom before starting.

- **Audit baseline commit:** `22636da` (every plan stamps this; use it for drift detection).
- **Baseline health at audit time:** `go build ./...`, `go vet ./...`, and
  `go test ./...` all pass (`ok cmd/mcp-server`, `ok pkg/stake`).
- **Verification commands (gates for every plan):**
  ```sh
  go build ./...
  go vet ./...
  go test ./...
  go test -race ./...   # required by plan 002
  ```

## Plans

| # | Title | Category | Effort | Risk | Status |
|---|---|---|---|---|---|
| 001 | [HTTP transport auth + bind guard](001-http-transport-auth-bind-guard.md) | Security | S | Low | TODO |
| 002 | [`stake.Client` concurrency data race](002-stake-client-concurrency-data-race.md) | Correctness | M | Low–Med | TODO |
| 003 | [Equity `UnmarshalJSON` drift-guard test](003-equity-unmarshal-drift-guard-test.md) | Tech debt | S | None | TODO |
| 005 | [MCP server documentation](005-mcp-server-documentation.md) | Docs | S | None | TODO |

> Numbering is non-monotonic by design: findings #4 (ASX/NYSE flexible-decode
> parity) and #6 (empty `internal/mcpserver/` dir) were surfaced in the audit but
> **not selected** for planning by the maintainer. If revisited later, give them
> the next free numbers (004, 006) rather than renumbering these.

## Recommended execution order

1. **003** — pure test addition, zero production risk; lands a safety net first.
2. **001** — closes the highest-impact security gap (unauthenticated HTTP transport).
3. **002** — the concurrency fix; touches a core client method, so do it with the
   `-race` suite green and after 001 (they touch different files; no conflict, but
   001 is higher leverage).
4. **005** — documentation; **write it last** so it can describe 001's
   `STAKE_MCP_HTTP_TOKEN` accurately. Plan 005 Step 2 explains how to handle the
   case where 001 has not yet landed.

## Dependencies

- **005 → 001 (soft):** Plan 005 documents `STAKE_MCP_HTTP_TOKEN` introduced by
  001. 005 can ship before 001 using its "warning" variant, but the cleanest order
  is 001 then 005.
- 001, 002, 003 are mutually independent and touch disjoint files
  (`main.go` / `client.go` / `equity_test.go` respectively).

## Considered and rejected (do not re-audit)

- **Secret resolution via `/bin/sh -c`** (`cmd/mcp-server/auth.go:245`) — the
  commands come from operator-controlled env vars (`STAKE_*_COMMAND`), not
  untrusted input. By design; matches the Bitwarden pattern in `opencode.json`.
  Not a vulnerability.
- **`FlexibleString` on `InceptionDate`** (`pkg/stake/product.go:49`) — decoding an
  epoch number into the string `"315532800000"` loses date semantics, but is
  consistent with the codebase's flexible-decode philosophy and is not a bug.
- **`go.mod` `go-sdk` indirect→direct promotion** — correct now that
  `cmd/mcp-server` imports it directly. No action needed.

## Not audited

Only the changed files plus the MCP/auth path were audited, per the focus request.
Not covered: the broader `pkg/stake` API surface (trade/order execution, ASX
endpoints beyond product parity) and the `stake-python` submodule (reference mirror).

## How to execute a plan

Pick a plan, follow its steps exactly, and run its verification gates. Update this
table's **Status** column (`TODO` → `IN PROGRESS` → `DONE`, or `BLOCKED` with a
note) as you go. If a plan's escape-hatch condition triggers, STOP and report
rather than improvising.
