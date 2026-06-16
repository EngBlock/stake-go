package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nathanbeddoewebdev/stake-go/pkg/secretsauce"
	"github.com/nathanbeddoewebdev/stake-go/pkg/stake"
)

type stakeAuthConfig struct {
	BaseURL           string
	TokenFile         string
	DisableTokenCache bool
	Token             string
	Username          secretsauce.Source
	Password          secretsauce.Source
	OTP               secretsauce.Source
}

type stakeAuth struct {
	mu                sync.Mutex
	client            *stake.Client
	baseURL           string
	tokenFile         string
	disableTokenCache bool
	username          secretsauce.Source
	password          secretsauce.Source
	otp               secretsauce.Source
}

func newStakeAuthFromEnv(baseURL, tokenFile string, disableTokenCache bool) (*stakeAuth, error) {
	if tokenFile == "" {
		tokenFile = os.Getenv("STAKE_TOKEN_FILE")
	}
	if !disableTokenCache && tokenFile == "" && !truthy(os.Getenv("STAKE_DISABLE_TOKEN_CACHE")) {
		tokenFile = defaultTokenFile()
	}

	return newStakeAuth(stakeAuthConfig{
		BaseURL:           baseURL,
		TokenFile:         tokenFile,
		DisableTokenCache: disableTokenCache || truthy(os.Getenv("STAKE_DISABLE_TOKEN_CACHE")),
		Token:             os.Getenv("STAKE_TOKEN"),
		Username:          secretsauce.FromEnv("STAKE_USERNAME"),
		Password:          secretsauce.FromEnv("STAKE_PASSWORD"),
		OTP:               secretsauce.FromEnv("STAKE_OTP"),
	})
}

func newStakeAuth(config stakeAuthConfig) (*stakeAuth, error) {
	tokenFile := ""
	if !config.DisableTokenCache {
		path, err := expandPath(config.TokenFile)
		if err != nil {
			return nil, err
		}
		tokenFile = path
	}

	token := strings.TrimSpace(config.Token)
	if token == "" && tokenFile != "" {
		cached, err := readTokenFile(tokenFile)
		if err != nil {
			return nil, err
		}
		token = cached
	}

	auth := &stakeAuth{
		baseURL:           config.BaseURL,
		tokenFile:         tokenFile,
		disableTokenCache: config.DisableTokenCache,
		username:          config.Username,
		password:          config.Password,
		otp:               config.OTP,
	}

	client, err := auth.newClientWithToken(token)
	if err != nil {
		return nil, err
	}
	auth.client = client
	return auth, nil
}

func (a *stakeAuth) CurrentClient(ctx context.Context) (*stake.Client, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.client.SessionToken() != "" {
		return a.client, nil
	}
	return a.loginLocked(ctx)
}

func (a *stakeAuth) Login(ctx context.Context) (*stake.User, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.client.SessionToken() != "" {
		user, err := a.client.Login(ctx)
		if err == nil {
			return user, a.saveTokenLocked()
		}
		if !isUnauthorized(err) {
			return nil, err
		}
	}

	client, err := a.loginLocked(ctx)
	if err != nil {
		return nil, err
	}
	return client.User, nil
}

func (a *stakeAuth) Refresh(ctx context.Context) (*stake.Client, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.loginLocked(ctx)
}

func (a *stakeAuth) loginLocked(ctx context.Context) (*stake.Client, error) {
	request, err := a.credentials(ctx)
	if err != nil {
		return nil, err
	}

	client, err := a.newClient(stake.WithCredentialsRequest(request))
	if err != nil {
		return nil, err
	}
	if _, err := client.Login(ctx); err != nil {
		return nil, err
	}

	a.client = client
	if err := a.saveTokenLocked(); err != nil {
		return nil, err
	}
	return client, nil
}

func (a *stakeAuth) credentials(ctx context.Context) (stake.CredentialsLoginRequest, error) {
	username, err := a.username.Resolve(ctx, "STAKE_USERNAME")
	if err != nil {
		return stake.CredentialsLoginRequest{}, err
	}
	password, err := a.password.Resolve(ctx, "STAKE_PASSWORD")
	if err != nil {
		return stake.CredentialsLoginRequest{}, err
	}
	if username == "" || password == "" {
		return stake.CredentialsLoginRequest{}, fmt.Errorf("stake: missing session token and credentials; set STAKE_TOKEN, STAKE_TOKEN_FILE, or STAKE_USERNAME and STAKE_PASSWORD")
	}

	otp, err := a.otp.Resolve(ctx, "STAKE_OTP")
	if err != nil {
		return stake.CredentialsLoginRequest{}, err
	}
	request := stake.CredentialsLoginRequest{
		Username: username,
		Password: password,
	}
	if otp != "" {
		request.OTP = &otp
	}
	return request, nil
}

func (a *stakeAuth) saveTokenLocked() error {
	if a.disableTokenCache || a.tokenFile == "" {
		return nil
	}
	token := strings.TrimSpace(a.client.SessionToken())
	if token == "" {
		return nil
	}
	return writeTokenFile(a.tokenFile, token)
}

func (a *stakeAuth) newClientWithToken(token string) (*stake.Client, error) {
	if strings.TrimSpace(token) == "" {
		return a.newClient()
	}
	return a.newClient(stake.WithSessionToken(strings.TrimSpace(token)))
}

func (a *stakeAuth) newClient(options ...stake.Option) (*stake.Client, error) {
	allOptions := make([]stake.Option, 0, len(options)+1)
	if strings.TrimSpace(a.baseURL) != "" {
		allOptions = append(allOptions, stake.WithBaseURL(a.baseURL))
	}
	allOptions = append(allOptions, options...)
	return stake.NewClient(allOptions...)
}

func withStakeClient[In any](auth *stakeAuth, handler func(context.Context, *mcp.CallToolRequest, *stake.Client, In) (*mcp.CallToolResult, any, error)) mcp.ToolHandlerFor[In, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input In) (*mcp.CallToolResult, any, error) {
		client, err := auth.CurrentClient(ctx)
		if err != nil {
			return nil, nil, err
		}

		result, output, err := handler(ctx, req, client, input)
		if err == nil || !isUnauthorized(err) {
			return result, output, err
		}

		client, refreshErr := auth.Refresh(ctx)
		if refreshErr != nil {
			return nil, nil, fmt.Errorf("%w; credential refresh failed: %v", err, refreshErr)
		}
		return handler(ctx, req, client, input)
	}
}

func readTokenFile(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if err := validateTokenFileForRead(path); err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("read Stake token file: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

func writeTokenFile(path, token string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create Stake token directory: %w", err)
	}
	if err := validateTokenFileForWrite(path); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open Stake token file: %w", err)
	}
	defer file.Close()
	if err := file.Chmod(0o600); err != nil {
		return fmt.Errorf("secure Stake token file permissions: %w", err)
	}
	if _, err := file.WriteString(token + "\n"); err != nil {
		return fmt.Errorf("write Stake token file: %w", err)
	}
	return nil
}

func validateTokenFileForRead(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("inspect Stake token file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("Stake token file must not be a symlink: %s", path)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("Stake token file must be a regular file: %s", path)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("Stake token file permissions are too broad: %s must be readable only by the owner", path)
	}
	return nil
}

func validateTokenFileForWrite(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("inspect Stake token file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("Stake token file must not be a symlink: %s", path)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("Stake token file must be a regular file: %s", path)
	}
	return nil
}

func defaultTokenFile() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		return ""
	}
	return filepath.Join(dir, "stake-go", "session-token")
}

func expandPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand token file path: %w", err)
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

func isUnauthorized(err error) bool {
	if errors.Is(err, stake.ErrMissingSessionToken) {
		return true
	}
	var apiErr *stake.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == 401
}

func truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
