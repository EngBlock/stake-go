# Plan 005 — Document the MCP server: config, secrets, flags, and HTTP risks

- **Written against commit:** `22636da`
- **Category:** Docs
- **Effort:** S (≈1 hour)
- **Risk of the change:** None (documentation only)
- **Status:** TODO
- **Coordinates with:** Plan 001 (introduces `STAKE_MCP_HTTP_TOKEN` and the
  loopback warning). If Plan 001 has landed, document `STAKE_MCP_HTTP_TOKEN` as a
  hard requirement for `-http`. If it has **not** landed, document the current
  behavior and add a prominent security warning (see Step 2).

> Documentation must match the code as it actually exists when you write it.
> Re-read `cmd/mcp-server/main.go` and `cmd/mcp-server/auth.go` first and document
> only what is true. If a flag or env var below is gone or renamed, STOP and report.

## Context (you have no prior knowledge of this repo)

`github.com/nathanbeddoewebdev/stake-go` is a Go client for the Stake brokerage
API (`pkg/stake/`) plus an MCP (Model Context Protocol) server at
`cmd/mcp-server/`. The repo has **no README anywhere** — not at the root, not in
`cmd/mcp-server/`. For a tool that authenticates to a real brokerage account and
can (behind flags) cancel orders and edit watchlists, the absence of any
configuration/security documentation is a real gap: operators have no guidance on
how to supply credentials, what the gating flags do, or the risks of HTTP mode.

This plan creates `cmd/mcp-server/README.md`. **No code changes.**

### Facts to document (verified at `22636da` — re-verify before writing)

**Command-line flags** (`cmd/mcp-server/main.go:13-19`):

| Flag | Default | Meaning |
|---|---|---|
| `-http <addr>` | `""` (stdio) | Serve MCP over streamable HTTP at `<addr>` instead of stdio. |
| `-base-url <url>` | `""` | Override the Stake API base URL (primarily for tests). |
| `-token-file <path>` | `""` | Path to a cached Stake session-token file. |
| `-no-token-cache` | `false` | Disable reading/writing the session-token cache. |
| `-enable-watchlist-mutations` | `false` | Register watchlist create/update/delete tools (these change account state). |
| `-enable-order-cancel` | `false` | Register pending-order cancellation tools (these change account state). |

**Environment variables** (`cmd/mcp-server/auth.go:47-63`, `258-262`):

| Env var | Meaning |
|---|---|
| `STAKE_TOKEN` | An existing Stake session token to use directly. |
| `STAKE_TOKEN_FILE` | Path to the cached session-token file (overridden by `-token-file`). |
| `STAKE_DISABLE_TOKEN_CACHE` | Truthy (`1`/`true`/`yes`/`on`) disables the token cache. |
| `STAKE_USERNAME` | Stake username (email). |
| `STAKE_USERNAME_COMMAND` or `STAKE_USERNAME_CMD` | A shell command whose stdout provides the username. |
| `STAKE_PASSWORD` / `STAKE_PASSWORD_COMMAND` / `STAKE_PASSWORD_CMD` | Password, or a command that prints it. |
| `STAKE_OTP` / `STAKE_OTP_COMMAND` / `STAKE_OTP_CMD` | One-time code, or a command that prints it. |

- **Credential resolution order** (`auth.go:234-256`): a `secretSource` first uses
  the literal value (`STAKE_*`); if empty, it runs the `_COMMAND`/`_CMD` shell
  command (via `/bin/sh -c`, 30s timeout — `auth.go:19,245`) and trims the output.
  Commands are operator-configured; the secret value is never logged or stored.
- **Token caching** (`auth.go:51-53,279-295`): when no token file is specified and
  caching is not disabled, the default is `<UserConfigDir>/stake-go/session-token`
  (on macOS: `~/Library/Application Support/stake-go/session-token`). The file is
  written `0600` inside a `0700` directory.
- **Auth/refresh behavior** (`auth.go:214-232`): tools auto-refresh credentials and
  retry once on a `401 Unauthorized`.
- **Default tool set** (`cmd/mcp-server/tools.go:36-45,57-293`): read-only by
  default (profile, market status, positions, cash, transactions, orders list,
  product lookup/search, ratings, statements, watchlist reads). Trading tools are
  intentionally absent. Mutations are only registered behind the two flags above.

There is an example MCP client configuration already in the repo at
`opencode.json` (stdio mode, using Bitwarden `bw get ...` commands for secrets) —
reference it as the canonical example.

## Repo conventions to follow

- Markdown, GitHub-flavored. Match the plain, direct tone of this codebase's code
  comments (full sentences, no marketing).
- Use fenced code blocks for commands and JSON.
- **Never include real secret values.** Use placeholders (`<your-username>`) and
  the command-based pattern. Do not paste any token, password, or `.env` content.

## Files in scope

- `cmd/mcp-server/README.md` — **new file**.

## Files explicitly OUT of scope — do not touch

- Any `.go` file.
- `opencode.json` (reference it, do not modify it).
- The root directory (do not create a root README in this plan).

## Implementation steps

### Step 1 — Write `cmd/mcp-server/README.md`

Include these sections:

1. **Overview** — what the server is (an MCP server exposing Stake account and
   market data as tools), that it is read-only by default, and that trading tools
   are intentionally not provided.
2. **Quick start (stdio)** — `go run ./cmd/mcp-server`, and how an MCP client
   launches it. Point to `opencode.json` as a working example and show its shape:
   ```jsonc
   {
     "mcp": {
       "stake": {
         "type": "local",
         "command": ["go", "run", "./cmd/mcp-server"],
         "environment": {
           "STAKE_TOKEN_FILE": "<path-to-session-token>",
           "STAKE_USERNAME_COMMAND": "<command that prints your username>",
           "STAKE_PASSWORD_COMMAND": "<command that prints your password>",
           "STAKE_OTP_COMMAND": "<command that prints your TOTP>"
         }
       }
     }
   }
   ```
3. **Authentication** — the three ways to authenticate (existing token via
   `STAKE_TOKEN`; cached token file; username/password/OTP), the
   value-then-command resolution order, and the recommendation to use the
   `_COMMAND` form with a secret manager (e.g. Bitwarden `bw get ...`, 1Password
   `op read ...`) so secrets never sit in plaintext config.
4. **Token cache** — default location, `0600` permissions, and how
   `-no-token-cache` / `STAKE_DISABLE_TOKEN_CACHE` disable it.
5. **Flags** — reproduce the flags table above.
6. **Environment variables** — reproduce the env-var table above.
7. **Mutation tools** — explain that `-enable-watchlist-mutations` and
   `-enable-order-cancel` register state-changing tools that are off by default,
   and to enable them only when intended.
8. **HTTP transport & security** — the most important section:
   - **If Plan 001 has landed:** document that `-http` requires
     `STAKE_MCP_HTTP_TOKEN`, that clients must send
     `Authorization: Bearer <token>`, and that binding to a non-loopback address
     exposes an authenticated brokerage session to the network (prefer
     `127.0.0.1`). Show an example:
     ```sh
     STAKE_MCP_HTTP_TOKEN=<random-token> go run ./cmd/mcp-server -http 127.0.0.1:8765
     ```
   - **If Plan 001 has NOT landed:** add a bold warning that the `-http` transport
     currently performs **no authentication** and serves a live authenticated
     Stake session to anyone who can reach the listen address; advise using stdio,
     or at minimum binding strictly to `127.0.0.1` behind an authenticating proxy,
     until Plan 001 ships.
9. **Building / testing** — `go build ./...`, `go test ./...` (the suite passes at
   `22636da`).

### Step 2 — Decide the HTTP section based on actual code

Before writing section 8, grep to determine which world you are in:

```sh
grep -n "STAKE_MCP_HTTP_TOKEN" cmd/mcp-server/main.go
```

- Non-empty result → Plan 001 landed → document the authenticated behavior.
- Empty result → Plan 001 not landed → write the warning variant.

## Verification gates (run from repo root)

```sh
go build ./...
go test ./...
test -f cmd/mcp-server/README.md && echo "README present"
```

Expected:
- Build and tests unaffected (`ok` for both packages — this plan changes no code).
- `README present` prints.

There is no markdown linter configured in this repo (no `.markdownlint*`, no CI
markdown step — verify with `ls -a` and by checking for a CI config). If one
exists, run it; otherwise visual review is sufficient.

## Done criteria (machine-checkable)

- [ ] `cmd/mcp-server/README.md` exists.
- [ ] It documents every flag in `main.go` and every env var in `auth.go`
      (cross-check the tables against the source — no flag/env var omitted, none invented).
- [ ] The HTTP section matches the actual code (authenticated vs. warning variant per Step 2).
- [ ] No real secret values appear anywhere in the file.
- [ ] `go build ./...` and `go test ./...` still pass.

## Test plan

No automated tests (documentation only). Verification is the build/test gates
above plus a manual read-through confirming the flag and env-var tables match
`cmd/mcp-server/main.go` and `cmd/mcp-server/auth.go` exactly.

## Maintenance note

Keep this README in sync whenever flags or env vars change in `main.go`/`auth.go`,
and whenever Plan 001's `STAKE_MCP_HTTP_TOKEN` lands or changes. Reviewers should
treat a new flag or env var without a corresponding README update as incomplete. A
future improvement (out of scope here) is a root-level README pointing at this one.
