package stake

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

const defaultPlatformType = "WEB_f5K2x3"

// Exchange identifies a Stake market supported by the client.
type Exchange string

const (
	// ExchangeNYSE selects Stake's US market endpoints.
	ExchangeNYSE Exchange = "nyse"
	// ExchangeASX selects Stake's Australian market endpoints.
	ExchangeASX Exchange = "asx"
)

// Option configures a Client.
type Option func(*Client) error

// Client is an HTTP client for Stake's API.
type Client struct {
	httpClient   *http.Client
	baseURL      string
	mu           sync.RWMutex // guards sessionToken and User
	sessionToken string
	credentials  *CredentialsLoginRequest
	exchange     Exchange

	// User is populated after Login succeeds. It is an exported field for
	// convenience, but callers performing concurrent Login calls should prefer
	// the value returned by Login; reading User directly while Login is in
	// progress is not safe.
	User *User

	// NYSE exposes typed services for Stake's US market endpoints.
	NYSE *NYSEServices
	// ASX exposes typed services for Stake's Australian market endpoints.
	ASX *ASXServices
}

// NewClient constructs a Stake API client.
func NewClient(options ...Option) (*Client, error) {
	c := &Client{
		httpClient: http.DefaultClient,
		exchange:   ExchangeNYSE,
	}

	for _, option := range options {
		if option == nil {
			continue
		}

		if err := option(c); err != nil {
			return nil, err
		}
	}

	if c.httpClient == nil {
		c.httpClient = http.DefaultClient
	}

	if c.exchange != ExchangeNYSE && c.exchange != ExchangeASX {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedExchange, c.exchange)
	}

	c.initServices()
	return c, nil
}

func (c *Client) initServices() {
	c.NYSE = newNYSEServices(c)
	c.ASX = newASXServices(c)
}

// WithHTTPClient sets the HTTP client used for requests.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) error {
		if httpClient == nil {
			return errors.New("stake: nil http client")
		}

		c.httpClient = httpClient
		return nil
	}
}

// WithBaseURL rewrites absolute Stake endpoints to the provided base URL.
// It is primarily useful with httptest.Server.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) error {
		if strings.TrimSpace(baseURL) == "" {
			return errors.New("stake: empty base URL")
		}

		parsed, err := url.Parse(baseURL)
		if err != nil {
			return fmt.Errorf("stake: parse base URL: %w", err)
		}
		if parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("stake: base URL must be absolute: %q", baseURL)
		}

		c.baseURL = strings.TrimRight(baseURL, "/")
		return nil
	}
}

// WithSessionToken configures token-based authentication.
func WithSessionToken(token string) Option {
	return func(c *Client) error {
		c.sessionToken = token
		c.credentials = nil
		return nil
	}
}

// WithSessionTokenFromEnv configures token-based authentication from an environment variable.
// If name is empty, STAKE_TOKEN is used.
func WithSessionTokenFromEnv(name string) Option {
	return func(c *Client) error {
		if name == "" {
			name = "STAKE_TOKEN"
		}

		c.sessionToken = os.Getenv(name)
		c.credentials = nil
		return nil
	}
}

// WithCredentials configures username/password authentication.
func WithCredentials(username, password string) Option {
	return func(c *Client) error {
		c.credentials = (&CredentialsLoginRequest{
			Username: username,
			Password: password,
		}).withDefaults()
		c.sessionToken = ""
		return nil
	}
}

// WithCredentialsRequest configures username/password authentication with all request fields.
func WithCredentialsRequest(request CredentialsLoginRequest) Option {
	return func(c *Client) error {
		c.credentials = request.withDefaults()
		c.sessionToken = ""
		return nil
	}
}

// WithExchange selects the exchange used by Login when retrieving the user record.
func WithExchange(exchange Exchange) Option {
	return func(c *Client) error {
		c.exchange = exchange
		return nil
	}
}

// Exchange returns the client's selected exchange.
func (c *Client) Exchange() Exchange {
	return c.exchange
}

// SetExchange changes the client's selected exchange.
func (c *Client) SetExchange(exchange Exchange) error {
	if exchange != ExchangeNYSE && exchange != ExchangeASX {
		return fmt.Errorf("%w: %s", ErrUnsupportedExchange, exchange)
	}

	c.exchange = exchange
	return nil
}

// SessionToken returns the current Stake session token.
func (c *Client) SessionToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sessionToken
}

// Login authenticates with Stake and retrieves the current user.
func (c *Client) Login(ctx context.Context) (*User, error) {
	c.mu.RLock()
	hasCreds := c.credentials != nil
	tokenEmpty := c.sessionToken == ""
	c.mu.RUnlock()

	if hasCreds {
		var response createSessionResponse
		if err := c.do(ctx, http.MethodPost, NYSE.CreateSession, c.credentials, &response); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidLogin, err)
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
		return nil, fmt.Errorf("%w: %w", ErrInvalidLogin, err)
	}

	c.mu.Lock()
	c.User = &user
	c.mu.Unlock()
	return &user, nil
}

func (c *Client) userEndpoint() (string, error) {
	switch c.exchange {
	case ExchangeNYSE:
		return NYSE.Users, nil
	case ExchangeASX:
		return ASX.Users, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedExchange, c.exchange)
	}
}

// Get performs a GET request against a Stake endpoint and decodes JSON into out.
func (c *Client) Get(ctx context.Context, endpoint string, out any) error {
	return c.do(ctx, http.MethodGet, endpoint, nil, out)
}

// Post performs a POST request against a Stake endpoint and decodes JSON into out.
func (c *Client) Post(ctx context.Context, endpoint string, in any, out any) error {
	return c.do(ctx, http.MethodPost, endpoint, in, out)
}

// Delete performs a DELETE request against a Stake endpoint and decodes JSON into out.
func (c *Client) Delete(ctx context.Context, endpoint string, in any, out any) error {
	return c.do(ctx, http.MethodDelete, endpoint, in, out)
}

func (c *Client) do(ctx context.Context, method, endpoint string, in any, out any) error {
	if ctx == nil {
		ctx = context.Background()
	}

	resolved, err := c.resolveEndpoint(endpoint)
	if err != nil {
		return err
	}

	var body io.Reader
	if in != nil {
		payload, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("stake: encode %s %s: %w", method, resolved, err)
		}
		body = bytes.NewReader(payload)
	}

	request, err := http.NewRequestWithContext(ctx, method, resolved, body)
	if err != nil {
		return fmt.Errorf("stake: build %s %s: %w", method, resolved, err)
	}

	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")

	c.mu.RLock()
	token := c.sessionToken
	c.mu.RUnlock()
	if token != "" {
		request.Header.Set("Stake-Session-Token", token)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("stake: %s %s: %w", method, resolved, err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("stake: read %s %s response: %w", method, resolved, err)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return &APIError{
			StatusCode: response.StatusCode,
			Method:     method,
			URL:        resolved,
			Body:       responseBody,
		}
	}

	if out == nil || len(bytes.TrimSpace(responseBody)) == 0 {
		return nil
	}

	if err := json.Unmarshal(responseBody, out); err != nil {
		return fmt.Errorf("stake: decode %s %s response: %w", method, resolved, err)
	}

	return nil
}

func (c *Client) resolveEndpoint(endpoint string) (string, error) {
	if c.baseURL == "" {
		return endpoint, nil
	}

	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("stake: parse base URL: %w", err)
	}

	target, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("stake: parse endpoint: %w", err)
	}

	if target.IsAbs() {
		target.Scheme = base.Scheme
		target.Host = base.Host
		if base.Path != "" && base.Path != "/" {
			target.Path = strings.TrimRight(base.Path, "/") + target.Path
		}
		return target.String(), nil
	}

	return base.ResolveReference(target).String(), nil
}

type createSessionResponse struct {
	SessionKey string `json:"sessionKey"`
}
