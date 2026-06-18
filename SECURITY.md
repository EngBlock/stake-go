# Security policy

This project authenticates to a real brokerage account (Stake) and, behind opt-in flags, can place real money-moving orders and cancel pending orders. Treat credentials and the session token with care.

## Reporting a vulnerability

Do not open a public GitHub issue for security vulnerabilities. Instead, email the maintainer at `nbedd2@protonmail.com` with a description and, if possible, a reproduction.

Please do not include real Stake credentials, session tokens, or passwords in any report. Replace them with placeholders.

## Credential handling

- The server never logs secret values. Credentials are read from environment variables or the stdout of a `_COMMAND` (e.g. a secret manager such as Bitwarden or 1Password). See [`cmd/mcp-server/README.md`](cmd/mcp-server/README.md) for the full env-var table.
- The cached session token file is written with `0600` permissions inside a `0700` directory. On read, the server rejects files that are symlinks, non-regular, or world/group-readable.
- Credential commands must reference an absolute-path executable; output is capped at 4 KiB and times out after 30 seconds. Commands are executed directly (argv, no shell).

## Trading and mutations

- Trading tools are **not registered by default**. They require the `-enable-trading` flag.
- By default each trade requires MCP confirmation (an elicitation prompt the client must accept). The `-auto` flag skips this confirmation — use it only in automated, trusted contexts and understand it submits real orders without a prompt.
- Watchlist mutations and order cancellation are likewise opt-in (`-enable-watchlist-mutations`, `-enable-order-cancel`).
- Read-only tools (positions, cash, market data, etc.) are always safe to call.

## Transport

The server is **stdio-only**. There is no HTTP transport. An earlier `-http` flag was removed because it served an authenticated brokerage session with no authentication; if an HTTP transport is reintroduced, it must require bearer-token authentication before the handler reaches the network.
