# Plan 001 — Add authentication and a bind guard to the MCP HTTP transport

- **Written against commit:** `22636da`
- **Category:** Security
- **Effort:** S (≈1–2 hours)
- **Risk of the change:** Low
- **Status:** TODO

> Verify the cited lines still match before editing. If `cmd/mcp-server/main.go`
> no longer looks like the excerpt below, STOP and report drift instead of guessing.

## Context (read this first — you have no prior knowledge of this repo)

This repository (`github.com/nathanbeddoewebdev/stake-go`) is a Go client for the
Stake brokerage API plus an MCP (Model Context Protocol) server at
`cmd/mcp-server/`. The MCP server exposes tools that operate on a **live,
authenticated brokerage account** (viewing positions, cash, transactions, and —
when optional flags are set — cancelling orders and editing watchlists).

The server can run two ways (see `cmd/mcp-server/main.go`):

1. **stdio** (default) — the MCP client launches the server as a subprocess and
   talks over stdin/stdout. This is the normal, safe mode used by
   `opencode.json` in the repo root.
2. **streamable HTTP** — enabled with the `-http <addr>` flag. The server listens
   on a TCP address and serves MCP over HTTP.

### The problem

In HTTP mode the server has **no authentication and no bind restriction**. Any
client that can reach the listen address gets a fully-authenticated session
against the operator's Stake account. The current code:

```go
// cmd/mcp-server/main.go:32-38
if *httpAddr != "" {
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, nil)
	log.Printf("MCP streamable HTTP listening at %s", *httpAddr)
	log.Fatal(http.ListenAndServe(*httpAddr, handler))
}
```

`http.ListenAndServe(*httpAddr, handler)` binds to whatever the operator passes.
If they pass `:8080` or `0.0.0.0:8080` (a common mistake), the brokerage account
is exposed to the entire local network with zero auth. There is no bearer-token
check, and nothing nudges the operator toward loopback-only.

### The fix (two layers, defense in depth)

1. **Require a bearer token** for HTTP mode, read from a new env var
   `STAKE_MCP_HTTP_TOKEN`. Wrap the MCP handler so every request must present
   `Authorization: Bearer <token>`; reject with `401` otherwise. If the env var
   is empty when `-http` is used, refuse to start (fail closed).
2. **Warn loudly when binding to a non-loopback address.** If `*httpAddr`
   resolves to anything other than localhost/loopback, log a prominent warning
   (do not block — the operator may intentionally front it with a reverse proxy).

stdio mode must be completely unaffected.

## Repo conventions to follow

- Plain standard-library Go. No new third-party dependencies — `net`, `net/http`,
  `crypto/subtle`, `os`, `strings` are all already available or standard.
- Errors that should stop startup use `log.Fatal(...)` (see existing `main.go`).
- Env vars use the `STAKE_` prefix (see `auth.go`: `STAKE_TOKEN`,
  `STAKE_USERNAME`, etc.). Helpers `truthy(...)` and `firstNonEmpty(...)` already
  exist in `cmd/mcp-server/auth.go` if useful.
- Tests use `net/http/httptest` and the standard `testing` package
  (see `cmd/mcp-server/auth_test.go` and `tools_test.go` for the house style).
- This is `package main` under `cmd/mcp-server/`. All `.go` files there are in
  the same package, so unexported helpers are shared freely.

## Files in scope

- `cmd/mcp-server/main.go` — modify the HTTP branch.
- `cmd/mcp-server/main_test.go` — **new file** for the auth-wrapper tests
  (there is currently no `main_test.go`; create it).

## Files explicitly OUT of scope — do not touch

- `cmd/mcp-server/auth.go`, `tools.go`, `types.go` (and their `_test.go` files).
- Anything under `pkg/stake/`.
- `opencode.json` (it uses stdio mode, so it is unaffected; do not add the new
  env var there).

## Implementation steps

### Step 1 — Add the bearer-auth wrapper and bind warning to `main.go`

Replace the HTTP branch (`main.go:32-38`) so that:

- A new helper reads `STAKE_MCP_HTTP_TOKEN` from the environment.
- If `-http` is set but the token is empty, `log.Fatal` with a clear message
  (fail closed — never serve an unauthenticated brokerage session).
- The MCP handler is wrapped in an `http.Handler` that checks the
  `Authorization: Bearer <token>` header using `crypto/subtle.ConstantTimeCompare`
  (constant-time to avoid timing leaks) and returns `http.StatusUnauthorized`
  with a short plain-text body on mismatch or missing header.
- Before `ListenAndServe`, call a helper that checks whether the host portion of
  `*httpAddr` is loopback; if not, log a warning.

Suggested shape (adapt names to match house style; keep it standard-library only):

```go
if *httpAddr != "" {
	token := strings.TrimSpace(os.Getenv("STAKE_MCP_HTTP_TOKEN"))
	if token == "" {
		log.Fatal("stake: -http requires STAKE_MCP_HTTP_TOKEN to be set; refusing to serve an unauthenticated brokerage session")
	}

	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, nil)
	handler := requireBearerToken(token, mcpHandler)

	warnIfNotLoopback(*httpAddr)
	log.Printf("MCP streamable HTTP listening at %s", *httpAddr)
	log.Fatal(http.ListenAndServe(*httpAddr, handler))
}
```

Add these helpers to `main.go`:

```go
func requireBearerToken(token string, next http.Handler) http.Handler {
	want := []byte("Bearer " + token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := []byte(r.Header.Get("Authorization"))
		if len(got) != len(want) || subtle.ConstantTimeCompare(got, want) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func warnIfNotLoopback(addr string) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	host = strings.TrimSpace(host)
	// Empty host (e.g. ":8080") binds all interfaces — treat as non-loopback.
	if host == "" {
		log.Printf("WARNING: -http %q binds all interfaces; bind to 127.0.0.1 to restrict to this machine", addr)
		return
	}
	if ip := net.ParseIP(host); ip != nil {
		if !ip.IsLoopback() {
			log.Printf("WARNING: -http address %q is not loopback; the authenticated Stake session is reachable from the network", addr)
		}
		return
	}
	if host != "localhost" {
		log.Printf("WARNING: -http host %q is not localhost; verify it resolves only to loopback", addr)
	}
}
```

Add `crypto/subtle`, `net`, `os`, and `strings` to the import block as needed
(`os` and `strings` may need adding — check; `net/http` and `log` are already there).

> **Escape hatch:** if the installed `go-sdk` version exposes a built-in auth
> middleware for `StreamableHTTPHandler` that supersedes this, STOP and report it
> — do not duplicate framework functionality. As of `go-sdk v1.6.1` (see
> `go.mod`) none is wired in here, so the wrapper above is expected to be needed.

### Step 2 — Add tests in a new `cmd/mcp-server/main_test.go`

Test the two helpers directly (they are pure/standard-library, so no MCP plumbing
needed). Follow the `package main` + `testing` style already used in the directory.

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireBearerTokenRejectsMissingAndWrongTokens(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := requireBearerToken("s3cret", next)

	cases := []struct {
		name       string
		header     string
		wantStatus int
		wantCalled bool
	}{
		{"missing", "", http.StatusUnauthorized, false},
		{"wrong", "Bearer nope", http.StatusUnauthorized, false},
		{"no-prefix", "s3cret", http.StatusUnauthorized, false},
		{"correct", "Bearer s3cret", http.StatusOK, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called = false
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if called != tc.wantCalled {
				t.Fatalf("next called = %v, want %v", called, tc.wantCalled)
			}
		})
	}
}

func TestWarnIfNotLoopbackDoesNotPanic(t *testing.T) {
	// Smoke test: these must not panic for representative inputs.
	for _, addr := range []string{"127.0.0.1:8080", "localhost:8080", ":8080", "0.0.0.0:8080", "garbage"} {
		warnIfNotLoopback(addr)
	}
}
```

## Verification gates (run from repo root)

Run each; every one must pass before marking this plan DONE:

```sh
go build ./...
go vet ./...
go test ./cmd/mcp-server/ -run 'TestRequireBearerToken|TestWarnIfNotLoopback' -v
go test ./...
```

Expected:
- `go build` / `go vet`: no output, exit 0.
- The targeted test run: `PASS`, with all four `requireBearerToken` subcases and
  the warn smoke test passing.
- `go test ./...`: `ok` for both `cmd/mcp-server` and `pkg/stake` (no regressions).

Manual sanity check (optional, no real credentials needed):

```sh
# Should refuse to start (fail closed):
go run ./cmd/mcp-server -http 127.0.0.1:8765
# Expect a fatal log: "...requires STAKE_MCP_HTTP_TOKEN..."
```

## Done criteria (machine-checkable)

- [ ] `go build ./...` exits 0.
- [ ] `go vet ./...` exits 0.
- [ ] `go test ./...` exits 0.
- [ ] `requireBearerToken` and `warnIfNotLoopback` exist in `cmd/mcp-server/main.go`.
- [ ] `cmd/mcp-server/main_test.go` exists and its tests pass.
- [ ] Starting with `-http` and no `STAKE_MCP_HTTP_TOKEN` calls `log.Fatal`.
- [ ] stdio mode (no `-http`) builds and runs unchanged — no new env var required.

## Test plan

New tests live in `cmd/mcp-server/main_test.go`, modeled on the assertion style in
`cmd/mcp-server/tools_test.go`. They cover: bearer rejection (missing/wrong/no-prefix),
bearer acceptance, and that the bind-warning helper handles loopback, all-interfaces,
and malformed inputs without panicking.

## Maintenance note

If a future change adds more HTTP-served transports or a web UI, route them through
`requireBearerToken` too. If the `go-sdk` later ships first-class auth middleware,
migrate to it and delete the hand-rolled wrapper. Reviewers should reject any change
that serves the MCP handler over HTTP without an auth check. Plan 005 (docs) should
document `STAKE_MCP_HTTP_TOKEN`; keep them in sync.
