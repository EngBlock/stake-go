# stake-go

An unofficial Go client and MCP server for the [Stake](https://hellostake.com) brokerage API. The API is not public, so there is no promise of backwards compatibility and API calls may break at any time. Do not use this in production software.

## What's here

- **`pkg/stake`** — an idiomatic Go HTTP client for Stake's US (NYSE) and Australian (ASX) market endpoints: authentication, positions, cash, transactions, orders, products, market data, ratings, statements, watchlists, FX, and trades.
- **`pkg/secretsauce`** — a small, reusable secret-resolution helper (literal value or command stdout) used by the MCP server.
- **`cmd/mcp-server`** — an [MCP](https://modelcontextprotocol.io) server that exposes a live, authenticated Stake account as tools for MCP-compatible AI clients. Read-only by default; trading and mutations behind flags. See [`cmd/mcp-server/README.md`](cmd/mcp-server/README.md).

## Install

```sh
go install github.com/EngBlock/stake-go/cmd/mcp-server@latest
```

## Quick start (SDK)

```go
client, err := stake.NewClient(stake.WithSessionToken(token))
if err != nil {
    log.Fatal(err)
}
positions, err := client.NYSE.Equities.List(ctx)
orders, err := client.ASX.Orders.List(ctx)
```

See [`pkg/stake/doc.go`](pkg/stake/doc.go) for the package overview. Authentication is via a session token or username/password (with optional OTP).

## Quick start (MCP server)

```sh
go run ./cmd/mcp-server
```

Point your MCP client at it. A working configuration template (with secret-manager command patterns) is in [`opencode.example.json`](opencode.example.json). Full configuration, flags, env vars, and security notes: [`cmd/mcp-server/README.md`](cmd/mcp-server/README.md).

## Building and testing

```sh
go build ./...
go vet ./...
go test ./...
go test -race ./...
```

## Repository notes

- The `stake-python/` directory is a git submodule pointing at a third-party Python Stake client ([stabacco/stake-python](https://github.com/stabacco/stake-python)) included as a reference. You do **not** need to clone it (or run `git submodule update --init`) to build or use the Go code.
- Requires Go 1.26+.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Security reports: [SECURITY.md](SECURITY.md).

## License

MIT — see [LICENSE](LICENSE).
