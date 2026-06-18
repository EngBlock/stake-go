# stake-go MCP server

An [MCP (Model Context Protocol)](https://modelcontextprotocol.io) server that exposes a live, authenticated [Stake](https://hellostake.com) brokerage account and market data as tools for MCP-compatible AI clients.

The server is **stdio-only**. It is **read-only by default** — trading and other state-changing tools are registered only when explicitly enabled via flags.

> The Stake API is unofficial and undocumented. There is no promise of backwards compatibility and API calls may break at any time. Do not use this in production software.

## Quick start

```sh
go run ./cmd/mcp-server
```

An MCP client launches the server as a subprocess and talks to it over stdin/stdout. A working client configuration is in [`opencode.example.json`](../../opencode.example.json) at the repo root — copy it, adjust the secret commands, and point your MCP client at it.

## Authentication

The server needs a Stake session to call the API. There are three ways to authenticate, checked in this order:

1. **Existing session token** — set `STAKE_TOKEN` to a `Stake-Session-Token` you retrieved from the Stake web app (Network tab in browser dev tools; valid ~30 days).
2. **Cached token file** — set `STAKE_TOKEN_FILE` (or pass `-token-file`) to a path holding a cached token. If none is given and caching is not disabled, the default is `<UserConfigDir>/stake-go/session-token` (on macOS: `~/Library/Application Support/stake-go/session-token`). The file is written with `0600` permissions inside a `0700` directory.
3. **Username + password (+ optional OTP)** — set `STAKE_USERNAME`, `STAKE_PASSWORD`, and (if 2FA is enabled) `STAKE_OTP`. The server logs in with credentials and caches the resulting session token for later runs.

### Secret resolution: value or command

For each credential, the server reads a literal value from the env var first. If that is empty, it runs the `_COMMAND` (or `_CMD`) variant and uses the command's stdout as the secret:

| Env var | Meaning |
|---|---|
| `STAKE_TOKEN` | An existing Stake session token to use directly. |
| `STAKE_TOKEN_FILE` | Path to the cached session-token file (overridden by `-token-file`). |
| `STAKE_DISABLE_TOKEN_CACHE` | Truthy (`1`/`true`/`yes`/`on`) disables the token cache. |
| `STAKE_USERNAME` | Stake username (email). |
| `STAKE_USERNAME_COMMAND` or `STAKE_USERNAME_CMD` | A command whose stdout provides the username. |
| `STAKE_PASSWORD` / `STAKE_PASSWORD_COMMAND` / `STAKE_PASSWORD_CMD` | Password, or a command that prints it. |
| `STAKE_OTP` / `STAKE_OTP_COMMAND` / `STAKE_OTP_CMD` | One-time code, or a command that prints it. |

The `_COMMAND` form is recommended with a secret manager so credentials never sit in plaintext config. The executable must be an absolute path; output is capped at 4 KiB and the command times out after 30 seconds. The secret value itself is never logged.

### Token refresh

Tools auto-refresh credentials and retry once on a `401 Unauthorized` response, so an expired session token is handled transparently as long as username/password (or a fresh `_COMMAND`) are available.

## Flags

| Flag | Default | Meaning |
|---|---|---|
| `-base-url <url>` | `""` | Override the Stake API base URL (primarily for tests). |
| `-token-file <path>` | `""` | Path to a cached Stake session-token file. |
| `-no-token-cache` | `false` | Disable reading/writing the session-token cache. |
| `-enable-watchlist-mutations` | `false` | Register watchlist create/update/delete tools (these change account state). |
| `-enable-order-cancel` | `false` | Register pending-order cancellation tools (these change account state). |
| `-enable-trading` | `false` | Register buy/sell trading tools (these submit real orders). |
| `-auto` | `false` | Skip MCP confirmation for buy/sell trading tools. **Dangerous:** trades execute without a confirmation prompt. |

## Tools

### Read-only (always registered)

Profile and market status: `me`, `nyse.market.status`, `asx.market.status`, `nyse.market.is_open`, `asx.market.is_open`, `fx.convert`.

Positions & cash: `nyse.positions.list`, `asx.positions.list`, `nyse.cash_available`, `asx.cash_available`.

Funding & transactions: `nyse.funds.in_flight`, `asx.funds.in_flight`, `nyse.fundings.list`, `asx.fundings.list`, `nyse.transactions.list`, `asx.transactions.list`.

Orders & brokerage: `nyse.orders.list`, `asx.orders.list`, `nyse.orders.brokerage`, `asx.orders.brokerage`.

Products & market data: `nyse.product.get`, `asx.product.get`, `nyse.products.search`, `asx.products.search`, `asx.product.depth`, `asx.product.course_of_sales`.

Ratings & statements: `nyse.ratings.list`, `nyse.statements.list`.

Watchlist reads: `nyse.watchlists.list`, `asx.watchlists.list`, `nyse.watchlists.get`, `asx.watchlists.get`.

### Watchlist mutations (`-enable-watchlist-mutations`)

`nyse.watchlists.create`, `asx.watchlists.create`, `nyse.watchlists.add`, `asx.watchlists.add`, `nyse.watchlists.remove`, `asx.watchlists.remove`, `nyse.watchlists.delete`, `asx.watchlists.delete`. These change Stake watchlist state.

### Order cancellation (`-enable-order-cancel`)

`nyse.orders.cancel`, `asx.orders.cancel`. These cancel pending Stake orders.

### Trading (`-enable-trading`)

US market: `nyse.trades.market_buy`, `nyse.trades.limit_buy`, `nyse.trades.stop_buy`, `nyse.trades.market_sell`, `nyse.trades.limit_sell`, `nyse.trades.stop_sell`.

ASX: `asx.trades.market_buy`, `asx.trades.limit_buy`, `asx.trades.market_sell`, `asx.trades.limit_sell`.

These submit **real, money-moving orders** against your Stake account. By default each trade requires MCP confirmation (an elicitation prompt the client must accept before the order is submitted). With `-auto`, confirmation is skipped entirely — use with care.

## Building and testing

```sh
go build ./...
go vet ./...
go test ./...
go test -race ./...
```

All four should pass. See [CONTRIBUTING.md](../../CONTRIBUTING.md) for PR expectations.
