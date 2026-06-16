package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nathanbeddoewebdev/stake-go/pkg/secretsauce"
)

func TestStakeAuthLogsInWithCredentialsAndCachesToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/sessions/v2/createSession":
			var payload struct {
				Username string  `json:"username"`
				Password string  `json:"password"`
				OTP      *string `json:"otp"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode login payload: %v", err)
			}
			if payload.Username != "user@example.com" || payload.Password != "secret" || payload.OTP == nil || *payload.OTP != "123456" {
				t.Fatalf("unexpected login payload: %+v", payload)
			}
			writeJSON(t, w, map[string]string{"sessionKey": "new-token"})
		case "/api/user":
			if got := r.Header.Get("Stake-Session-Token"); got != "new-token" {
				t.Fatalf("Stake-Session-Token = %q, want new-token", got)
			}
			writeJSON(t, w, map[string]any{
				"userId":           "user-1",
				"firstName":        "Ada",
				"lastName":         "Lovelace",
				"emailAddress":     "ada@example.com",
				"macStatus":        "OK",
				"accountType":      "INDIVIDUAL",
				"regionIdentifier": "AU",
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	tokenFile := filepath.Join(t.TempDir(), "session-token")
	auth, err := newStakeAuth(stakeAuthConfig{
		BaseURL:   server.URL,
		TokenFile: tokenFile,
		Username:  secretsauce.Source{Value: "user@example.com"},
		Password:  secretsauce.Source{Value: "secret"},
		OTP:       secretsauce.Source{Value: "123456"},
	})
	if err != nil {
		t.Fatalf("newStakeAuth: %v", err)
	}

	client, err := auth.CurrentClient(context.Background())
	if err != nil {
		t.Fatalf("CurrentClient: %v", err)
	}
	if client.SessionToken() != "new-token" {
		t.Fatalf("session token = %q, want new-token", client.SessionToken())
	}

	data, err := os.ReadFile(tokenFile)
	if err != nil {
		t.Fatalf("read token file: %v", err)
	}
	if string(data) != "new-token\n" {
		t.Fatalf("token file = %q, want new-token", string(data))
	}
}

func TestStakeAuthRejectsInsecureTokenFilePermissions(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "session-token")
	if err := os.WriteFile(tokenFile, []byte("cached-token\n"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}
	if err := os.Chmod(tokenFile, 0o644); err != nil {
		t.Fatalf("chmod token file: %v", err)
	}

	_, err := newStakeAuth(stakeAuthConfig{TokenFile: tokenFile})
	if err == nil || !strings.Contains(err.Error(), "permissions are too broad") {
		t.Fatalf("error = %v, want broad permissions error", err)
	}
}

func TestStakeAuthRejectsSymlinkTokenFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target-token")
	if err := os.WriteFile(target, []byte("cached-token\n"), 0o600); err != nil {
		t.Fatalf("write target token file: %v", err)
	}
	link := filepath.Join(dir, "session-token")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	_, err := newStakeAuth(stakeAuthConfig{TokenFile: link})
	if err == nil || !strings.Contains(err.Error(), "must not be a symlink") {
		t.Fatalf("error = %v, want symlink error", err)
	}
}

func TestWriteTokenFileSecuresExistingFilePermissions(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "session-token")
	if err := os.WriteFile(tokenFile, []byte("old-token\n"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}
	if err := os.Chmod(tokenFile, 0o644); err != nil {
		t.Fatalf("chmod token file: %v", err)
	}

	if err := writeTokenFile(tokenFile, "new-token"); err != nil {
		t.Fatalf("writeTokenFile: %v", err)
	}
	info, err := os.Stat(tokenFile)
	if err != nil {
		t.Fatalf("stat token file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("token file permissions = %o, want 600", got)
	}
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		t.Fatalf("read token file: %v", err)
	}
	if string(data) != "new-token\n" {
		t.Fatalf("token file = %q, want new-token", string(data))
	}
}

func TestStakeToolRefreshesCachedTokenAfterUnauthorized(t *testing.T) {
	var sawBadToken bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/users/accounts/v2/equityPositions":
			switch got := r.Header.Get("Stake-Session-Token"); got {
			case "bad-token":
				sawBadToken = true
				w.WriteHeader(http.StatusUnauthorized)
				writeJSON(t, w, map[string]string{"message": "expired"})
			case "new-token":
				writeJSON(t, w, map[string]any{
					"equityPositions": []map[string]any{{"symbol": "AAPL"}},
					"equityValue":     100,
					"pricesOnly":      false,
				})
			default:
				t.Fatalf("Stake-Session-Token = %q, want bad-token or new-token", got)
			}
		case "/api/sessions/v2/createSession":
			writeJSON(t, w, map[string]string{"sessionKey": "new-token"})
		case "/api/user":
			if got := r.Header.Get("Stake-Session-Token"); got != "new-token" {
				t.Fatalf("Stake-Session-Token = %q, want new-token", got)
			}
			writeJSON(t, w, map[string]any{
				"userId":           "user-1",
				"firstName":        "Ada",
				"lastName":         "Lovelace",
				"emailAddress":     "ada@example.com",
				"macStatus":        "OK",
				"accountType":      "INDIVIDUAL",
				"regionIdentifier": "AU",
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	tokenFile := filepath.Join(t.TempDir(), "session-token")
	if err := os.WriteFile(tokenFile, []byte("bad-token\n"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}
	auth, err := newStakeAuth(stakeAuthConfig{
		BaseURL:   server.URL,
		TokenFile: tokenFile,
		Username:  secretsauce.Source{Value: "user@example.com"},
		Password:  secretsauce.Source{Value: "secret"},
	})
	if err != nil {
		t.Fatalf("newStakeAuth: %v", err)
	}

	session := connectMCP(t, newMCPServer(auth, serverConfig{}))
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "nyse.positions.list"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error content: %+v", result.Content)
	}
	if !sawBadToken {
		t.Fatal("expected first request to use stale cached token")
	}

	data, err := os.ReadFile(tokenFile)
	if err != nil {
		t.Fatalf("read token file: %v", err)
	}
	if string(data) != "new-token\n" {
		t.Fatalf("token file = %q, want new-token", string(data))
	}
}
