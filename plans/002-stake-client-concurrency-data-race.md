# Plan 002 — Make `stake.Client` session state safe for concurrent use

- **Written against commit:** `22636da`
- **Category:** Correctness (concurrency / data race)
- **Effort:** M (≈2–4 hours)
- **Risk of the change:** Low–Medium (touches a core client method; covered by the new race test)
- **Status:** TODO
- **Depends on:** nothing. **Recommended before** any work that increases HTTP concurrency.

> Verify the cited lines still match before editing. If the excerpts below have
> drifted, STOP and report instead of guessing.

## Context (you have no prior knowledge of this repo)

`github.com/nathanbeddoewebdev/stake-go` is a Go client for the Stake brokerage
API (`pkg/stake/`) plus an MCP server (`cmd/mcp-server/`).

`pkg/stake.Client` (`pkg/stake/client.go`) holds mutable session state:

```go
// pkg/stake/client.go:32-46 (abridged)
type Client struct {
	httpClient   *http.Client
	baseURL      string
	sessionToken string          // <-- mutated by Login, read by every request
	credentials  *CredentialsLoginRequest
	exchange     Exchange
	User         *User           // <-- mutated by Login
	NYSE *NYSEServices
	ASX  *ASXServices
}
```

`Login` writes `sessionToken` and `User`:

```go
// pkg/stake/client.go:188-215 (abridged)
func (c *Client) Login(ctx context.Context) (*User, error) {
	if c.credentials != nil {
		...
		c.sessionToken = response.SessionKey   // write
	} else if c.sessionToken == "" {
		return nil, ErrMissingSessionToken
	}
	...
	c.User = &user                              // write
	return &user, nil
}
```

Every request reads `sessionToken` without synchronization:

```go
// pkg/stake/client.go:269-271
if c.sessionToken != "" {
	request.Header.Set("Stake-Session-Token", c.sessionToken)
}
```

`SessionToken()` (`client.go:183-185`) and `SetExchange()` (`client.go:173-180`)
also touch this state.

### Why this races

The MCP server shares **one** `*stake.Client` across all requests. In HTTP mode
(`cmd/mcp-server/main.go:33-35`) the same `*mcp.Server` — and therefore the same
`stakeAuth` and its single `*stake.Client` — serves every concurrent HTTP request.
`mcp.NewStreamableHTTPHandler` processes requests concurrently.

Concretely (`cmd/mcp-server/auth.go`):
- The `me` tool calls `auth.Login(ctx)` → `a.client.Login(ctx)`, which **writes**
  `c.sessionToken` and `c.User` (`auth.go:112-131`).
- Any other tool, via `withStakeClient` → `CurrentClient` (`auth.go:102-110`,
  `214-232`), obtains the **same** `*stake.Client` and then **reads** `c.sessionToken`
  inside `do()` while making its HTTP call.

`stakeAuth` has a mutex, but it only guards swapping `a.client`; it does **not**
guard the `*stake.Client`'s own fields while a handler is mid-call. Two concurrent
tool calls — one logging in, one reading — are an unsynchronized read/write of
`c.sessionToken`: a Go data race (undefined behavior, not just a stale read).

### The fix

Protect `Client`'s mutable session fields (`sessionToken`, `User`) with a
`sync.RWMutex` on the `Client`. Writers (`Login`) take the write lock around the
field assignments; readers (`do`, `SessionToken`) take the read lock (or snapshot
the token under the read lock before issuing the HTTP call). Keep the lock scope
**narrow** — never hold it across the blocking `c.httpClient.Do(...)` network call.

## Repo conventions to follow

- Standard library only; `sync` is already idiomatic here (`cmd/mcp-server/auth.go`
  uses `sync.Mutex`).
- Keep exported behavior identical — this is an internal-safety change, not an API
  change. Do not rename or re-signature any exported method.
- Tests use `net/http/httptest` + `testing` (see `pkg/stake/client_test.go`).
- Doc comments on exported identifiers are full sentences (see existing comments).

## Files in scope

- `pkg/stake/client.go` — add the mutex and guard the session fields.
- `pkg/stake/client_test.go` — add a race-detector test.

## Files explicitly OUT of scope — do not touch

- `cmd/mcp-server/**` (the auth layer's `sync.Mutex` already serializes client
  *swaps* correctly; the bug is inside `pkg/stake`).
- Any other `pkg/stake/*.go` service files. Do **not** add locking to every method
  — only the session-state fields (`sessionToken`, `User`) need it. The service
  structs (`NYSE`, `ASX`) are set once in `NewClient` and never reassigned.

## Implementation steps

### Step 1 — Add a mutex to `Client` and guard the session fields

In `pkg/stake/client.go`:

1. Add `"sync"` to the import block.
2. Add an unexported field to `Client` (place it near `sessionToken`):

```go
type Client struct {
	httpClient   *http.Client
	baseURL      string
	mu           sync.RWMutex // guards sessionToken and User
	sessionToken string
	credentials  *CredentialsLoginRequest
	exchange     Exchange
	User         *User
	NYSE *NYSEServices
	ASX  *ASXServices
}
```

> Note: `User` is an **exported** field that callers may read directly. Document
> that concurrent reads of `User` during a `Login` are not safe and callers should
> prefer the value returned by `Login`. The lock makes the *internal* request path
> safe; we are not changing `User`'s exported-ness.

3. Make `SessionToken()` read under the lock:

```go
func (c *Client) SessionToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sessionToken
}
```

4. In `Login`, take the write lock only around the field writes — **not** around
   the network calls. Restructure so the HTTP calls happen lock-free and only the
   assignments are guarded:

```go
func (c *Client) Login(ctx context.Context) (*User, error) {
	c.mu.RLock()
	hasCreds := c.credentials != nil
	tokenEmpty := c.sessionToken == ""
	c.mu.RUnlock()

	if hasCreds {
		var response createSessionResponse
		if err := c.do(ctx, http.MethodPost, NYSE.CreateSession, c.credentials, &response); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidLogin, err)
		}
		if response.SessionKey == "" {
			return nil, fmt.Errorf("%w: missing session key", ErrInvalidLogin)
		}
		c.mu.Lock()
		c.sessionToken = response.SessionKey
		c.mu.Unlock()
	} else if tokenEmpty {
		return nil, ErrMissingSessionToken
	}

	endpoint, err := c.userEndpoint()
	if err != nil {
		return nil, err
	}

	var user User
	if err := c.do(ctx, http.MethodGet, endpoint, nil, &user); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidLogin, err)
	}

	c.mu.Lock()
	c.User = &user
	c.mu.Unlock()
	return &user, nil
}
```

   `c.credentials` is set once during `NewClient`'s option application and never
   mutated afterward, so reading it lock-free after the snapshot is acceptable;
   the snapshot above is for clarity. If unsure, also guard `credentials` reads —
   but do not over-lock the network path.

5. In `do()`, snapshot the token under the read lock before building the request,
   so the read no longer races with `Login`'s write:

```go
func (c *Client) do(ctx context.Context, method, endpoint string, in any, out any) error {
	...
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")

	c.mu.RLock()
	token := c.sessionToken
	c.mu.RUnlock()
	if token != "" {
		request.Header.Set("Stake-Session-Token", token)
	}

	response, err := c.httpClient.Do(request)   // network call happens with no lock held
	...
}
```

> **Escape hatch:** if guarding `do()` causes a deadlock (e.g. `Login` indirectly
> calls `do()` while holding `c.mu`), that means a write lock is being held across
> a `do()` call — it must not be. Re-check that all `c.mu.Lock()/Unlock()` pairs in
> `Login` wrap only the assignment lines, never a `c.do(...)`. If you cannot satisfy
> this, STOP and report.

### Step 2 — Add a race-detector test

Add to `pkg/stake/client_test.go`. The test fires concurrent `Login` and
`SessionToken`/request calls against an `httptest` server and is meaningful only
under `-race`. Follow the existing test style in that file (it already imports
`context`, `net/http`, `net/http/httptest`, `sync` may need adding, `testing`).

```go
func TestClientLoginIsRaceFreeUnderConcurrency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/sessions/v2/createSession":
			writeJSON(t, w, map[string]string{"sessionKey": "tok"})
		case "/api/user":
			writeJSON(t, w, map[string]any{
				"userId": "u1", "firstName": "Ada", "lastName": "L",
				"emailAddress": "a@example.com", "macStatus": "OK",
				"accountType": "INDIVIDUAL", "regionIdentifier": "AU",
			})
		case "/api/utils/marketStatus":
			writeJSON(t, w, map[string]any{"response": map[string]any{"status": map[string]any{"current": "OPEN"}}})
		default:
			writeJSON(t, w, map[string]any{})
		}
	}))
	defer server.Close()

	client, err := NewClient(
		WithBaseURL(server.URL),
		WithCredentials("user@example.com", "secret"),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); _, _ = client.Login(context.Background()) }()
		go func() { defer wg.Done(); _ = client.SessionToken() }()
	}
	wg.Wait()
}
```

> Confirm `writeJSON` is already a test helper in `pkg/stake` (it is used by
> existing tests in `client_test.go`). If the `/api/utils/marketStatus` path or
> the user fixture shape differs, mirror whatever the existing passing tests in
> this file use — do not invent new fixture shapes.

## Verification gates (run from repo root)

```sh
go build ./...
go vet ./...
go test -race ./pkg/stake/ -run TestClientLoginIsRaceFreeUnderConcurrency -v
go test -race ./...
go test ./...
```

Expected:
- Build/vet: clean, exit 0.
- The targeted `-race` test: `PASS`, **no** `WARNING: DATA RACE` in output.
- `go test -race ./...`: all packages `ok`, no data-race warnings.
- `go test ./...`: all `ok`.

**Baseline note for the executor:** the full suite already passes at `22636da`
(`ok cmd/mcp-server`, `ok pkg/stake`). Your change must keep it green.

## Done criteria (machine-checkable)

- [ ] `go build ./...` and `go vet ./...` exit 0.
- [ ] `pkg/stake/client.go` has a `sync.RWMutex` guarding `sessionToken` and `User`.
- [ ] No `c.mu.Lock()`/`RLock()` is held across a `c.httpClient.Do(...)` or `c.do(...)` call.
- [ ] `go test -race ./...` passes with no `DATA RACE` warnings.
- [ ] `go test ./...` passes.
- [ ] No exported method signatures changed.

## Test plan

New test `TestClientLoginIsRaceFreeUnderConcurrency` in `pkg/stake/client_test.go`,
modeled on existing `httptest`-based tests there. It only proves its value under
`-race`; the verification gates run it that way. No existing tests should need
changes.

## Maintenance note

Any future field added to `Client` that is mutated after construction must be
guarded by `c.mu` too. Reviewers: reject changes that read or write `sessionToken`
or `User` outside the lock, and reject any lock held across a network call. If the
MCP layer ever creates a fresh `*stake.Client` per request instead of sharing one,
this locking becomes belt-and-suspenders but should stay (the client is a public
package and may be shared by other consumers).
