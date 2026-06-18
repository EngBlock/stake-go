# Contributing

Thanks for considering a contribution. This is a small project; keep PRs focused.

## Before you start

- The Stake API is unofficial and undocumented. There is no backwards-compatibility promise, so expect breakage and design for it.
- Read [`cmd/mcp-server/README.md`](cmd/mcp-server/README.md) and [`SECURITY.md`](SECURITY.md) before touching auth, secrets, or anything that changes account state.

## Development workflow

```sh
go build ./...
go vet ./...
go test ./...
go test -race ./...
```

All four must pass before a PR is mergeable. The `-race` gate is required because `pkg/stake.Client` is shared across goroutines; a change that introduces a data race will be rejected.

## Conventions

- **Standard library first.** New third-party deps need a clear justification. The only current external dep is the MCP `go-sdk`.
- **No comments unless asked** is not the rule here — doc comments on exported identifiers are expected (full sentences, matching the existing style in `pkg/stake`).
- **Env vars use the `STAKE_` prefix.** Flags use lowercase kebab-case.
- **No new state-changing MCP tools without a gating flag.** Read-only tools register by default; anything that mutates the Stake account (orders, watchlists, cancellations) must sit behind an `-enable-*` flag and carry a `destructiveHint` annotation (see `tradingToolAnnotations` in `cmd/mcp-server/tools.go`).
- **Hand-written `UnmarshalJSON`** that mirrors a public struct must ship with a drift-guard test (see `pkg/stake/equity_test.go` for the pattern).

## Tests

- New endpoints or decode paths need tests using `net/http/httptest` (see `pkg/stake/client_test.go` for the house style and the `writeJSON` helper).
- Concurrency-sensitive changes need a `-race` test.

## Commits

Conventional-commits style (`feat:`, `fix:`, `test:`, `docs:`, `chore:`). Keep the subject line short; put context and constraints in the body.

## Security

Never commit real Stake credentials, session tokens, `.env` files, or `.har` captures. If you add a new secret-handling path, document it in `SECURITY.md` and `cmd/mcp-server/README.md`.
