// Package client wraps the generated Flotio API client with auth and config.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	apiclient "github.com/flotio-dev/cli/pkg/api/client"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

// TokenFile is the path to the auth token storage, relative to ~/.flotio/.
const TokenFile = "auth.json"

// Tokens holds the persisted auth tokens and credentials.
type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Email        string `json:"email,omitempty"`
}

// LoadTokens reads tokens from ~/.flotio/auth.json.
// Returns nil if the file doesn't exist.
func LoadTokens() (*Tokens, error) {
	path, err := tokenPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading auth file: %w", err)
	}
	var t Tokens
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parsing auth file: %w", err)
	}
	return &t, nil
}

// SaveTokens persists tokens to ~/.flotio/auth.json.
// If existing credentials (email/password) are stored, they are preserved.
func SaveTokens(access, refresh string) error {
	return saveAuth(func(t *Tokens) {
		t.AccessToken = access
		t.RefreshToken = refresh
	})
}

// SaveCredentials persists email alongside tokens.
// The password is intentionally NOT stored — a refresh token is used for
// silent re-authentication instead (see DoRefresh).
func SaveCredentials(email, access, refresh string) error {
	return saveAuth(func(t *Tokens) {
		t.Email = email
		t.AccessToken = access
		t.RefreshToken = refresh
	})
}

func saveAuth(update func(*Tokens)) error {
	path, err := tokenPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	// Preserve existing fields if they exist
	t, _ := LoadTokens()
	if t == nil {
		t = &Tokens{}
	}
	update(t)

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// ClearTokens removes the auth file.
func ClearTokens() error {
	path, err := tokenPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// httpClient is the shared HTTP client with a sane timeout so a hung API
// never blocks the CLI indefinitely.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// Relogin attempts to silently refresh the access token using the stored
// refresh token (rotation). Returns nil on success (new tokens saved).
// The plaintext password is never persisted — refresh-token rotation replaces it.
func Relogin(baseURL string) error {
	tokens, err := LoadTokens()
	if err != nil || tokens == nil || tokens.RefreshToken == "" {
		return fmt.Errorf("no stored refresh token")
	}
	return DoRefresh(baseURL, tokens.RefreshToken)
}

// DoRefresh exchanges a refresh token for a new access+refresh pair via
// POST /auth/refresh (sending the token in JSON body, cookie, and Bearer header for maximum compatibility).
func DoRefresh(baseURL, refreshToken string) error {
	body := map[string]string{
		"refresh_token": refreshToken,
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", baseURL+"/auth/refresh", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
	req.Header.Set("Authorization", "Bearer "+refreshToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("refresh returned %d: %s", resp.StatusCode, string(errBody))
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	email := ""
	if t, _ := LoadTokens(); t != nil {
		email = t.Email
	}
	return SaveCredentials(email, result.AccessToken, result.RefreshToken)
}

// DoLogin authenticates with email/password and saves tokens + credentials.
func DoLogin(baseURL, email, password string) error {
	body := map[string]string{
		"email":    email,
		"password": password,
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", baseURL+"/auth/login", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("login returned %d: %s", resp.StatusCode, string(errBody))
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	return SaveCredentials(email, result.AccessToken, result.RefreshToken)
}

// TokenPathFn returns the path to the auth token file.
// Overridable in tests.
var TokenPathFn = defaultTokenPath

func defaultTokenPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, ".flotio", TokenFile), nil
}

func tokenPath() (string, error) {
	return TokenPathFn()
}

// AuthRoundTripper wraps an http.RoundTripper and automatically intercepts
// 401 Unauthorized responses to perform refresh-token rotation and retry the
// original request with the renewed access token.
type AuthRoundTripper struct {
	BaseURL      string
	Wrapped      http.RoundTripper
	refreshMutex sync.Mutex
}

// NewAuthRoundTripper creates an AuthRoundTripper with the given baseURL and underlying transport.
func NewAuthRoundTripper(baseURL string, wrapped http.RoundTripper) *AuthRoundTripper {
	if wrapped == nil {
		wrapped = http.DefaultTransport
	}
	return &AuthRoundTripper{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Wrapped: wrapped,
	}
}

func isAuthEndpoint(path string) bool {
	return strings.HasSuffix(path, "/auth/login") ||
		strings.HasSuffix(path, "/auth/refresh") ||
		strings.HasSuffix(path, "/auth/register")
}

// RoundTrip executes a single HTTP transaction. If 401 is received on a non-auth endpoint,
// it refreshes tokens and transparently retries the request with the new access token.
func (rt *AuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Do not intercept auth endpoints to avoid recursion
	if isAuthEndpoint(req.URL.Path) {
		return rt.Wrapped.RoundTrip(req)
	}

	// Buffer body if needed so we can rewind and replay on retry
	if req.Body != nil && req.GetBody == nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("buffering request body: %w", err)
		}
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
	}

	// Track initial token used
	initialToken := ""
	authHeader := req.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		initialToken = strings.TrimPrefix(authHeader, "Bearer ")
	} else {
		if tokens, _ := LoadTokens(); tokens != nil && tokens.AccessToken != "" {
			initialToken = tokens.AccessToken
			req.Header.Set("Authorization", "Bearer "+initialToken)
		}
	}

	resp, err := rt.Wrapped.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}

	// 401 Unauthorized received — attempt token refresh
	rt.refreshMutex.Lock()
	defer rt.refreshMutex.Unlock()

	// Check if another concurrent request already refreshed the token
	currentTokens, err := LoadTokens()
	if err == nil && currentTokens != nil && currentTokens.AccessToken != "" && currentTokens.AccessToken != initialToken {
		_ = resp.Body.Close()
		retryReq := req.Clone(req.Context())
		retryReq.Header.Set("Authorization", "Bearer "+currentTokens.AccessToken)
		if req.GetBody != nil {
			body, bodyErr := req.GetBody()
			if bodyErr != nil {
				return nil, bodyErr
			}
			retryReq.Body = body
		}
		return rt.Wrapped.RoundTrip(retryReq)
	}

	// If no refresh token is stored, we cannot refresh
	if currentTokens == nil || currentTokens.RefreshToken == "" {
		_ = resp.Body.Close()
		_ = ClearTokens()
		return nil, fmt.Errorf("session expired: please run 'flotio login' to authenticate again")
	}

	// Perform refresh
	if refreshErr := DoRefresh(rt.BaseURL, currentTokens.RefreshToken); refreshErr != nil {
		_ = resp.Body.Close()
		_ = ClearTokens()
		return nil, fmt.Errorf("session expired: please run 'flotio login' to authenticate again (refresh failed: %w)", refreshErr)
	}

	newTokens, err := LoadTokens()
	if err != nil || newTokens == nil || newTokens.AccessToken == "" {
		_ = resp.Body.Close()
		_ = ClearTokens()
		return nil, fmt.Errorf("session expired: please run 'flotio login' to authenticate again")
	}

	_ = resp.Body.Close()
	retryReq := req.Clone(req.Context())
	retryReq.Header.Set("Authorization", "Bearer "+newTokens.AccessToken)
	if req.GetBody != nil {
		body, bodyErr := req.GetBody()
		if bodyErr != nil {
			return nil, bodyErr
		}
		retryReq.Body = body
	}

	return rt.Wrapped.RoundTrip(retryReq)
}

// New creates a new Flotio API client connected to the given host.
// If a valid access token is stored, it is injected as Bearer auth.
// An AuthRoundTripper is attached to automatically handle refresh token rotation on 401.
// The host can be a plain hostname or scheme://host (scheme is extracted).
func New(rawHost string) *apiclient.FlotioAPI {
	scheme, host := parseHost(rawHost)
	schemes := []string{scheme}
	transport := httptransport.New(host, "/", schemes)

	baseURL := fmt.Sprintf("%s://%s", scheme, host)
	transport.Transport = NewAuthRoundTripper(baseURL, transport.Transport)

	// Inject dynamic bearer token so updated tokens are always used on new requests.
	transport.DefaultAuthentication = runtime.ClientAuthInfoWriterFunc(func(req runtime.ClientRequest, reg strfmt.Registry) error {
		tokens, _ := LoadTokens()
		if tokens != nil && tokens.AccessToken != "" {
			return req.SetHeaderParam("Authorization", "Bearer "+tokens.AccessToken)
		}
		return nil
	})

	return apiclient.New(transport, strfmt.Default)
}

// IsLoggedIn returns true if a valid-looking token or refresh token is stored.
func IsLoggedIn() bool {
	tokens, _ := LoadTokens()
	return tokens != nil && (tokens.AccessToken != "" || tokens.RefreshToken != "")
}

// apiDo is a helper that performs an authenticated HTTP request
// and decodes the JSON response into v. The baseURL should be a
//
//	full URL (e.g. "https://api.flotio.ovh"), and path the API path (e.g. "/auth/@me").
//
// On 401, automatically attempts re-login with stored credentials.
func apiDo(method, baseURL, path string, body io.Reader, v interface{}) error {
	tokens, err := LoadTokens()
	if err != nil {
		return fmt.Errorf("loading tokens: %w", err)
	}
	if tokens == nil || tokens.AccessToken == "" {
		// Try re-login with stored credentials
		if err := Relogin(baseURL); err != nil {
			return fmt.Errorf("not logged in — run 'flotio login' first")
		}
		tokens, _ = LoadTokens()
	}

	return apiDoWithToken(method, baseURL, path, body, v, tokens.AccessToken)
}

func apiDoWithToken(method, baseURL, path string, body io.Reader, v interface{}, token string) error {
	url := baseURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		// Access token expired — try refresh-token rotation once, then retry.
		if err := Relogin(baseURL); err != nil {
			_ = ClearTokens()
			errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			return fmt.Errorf("session expired (refresh failed: %v): %s\n\nRun 'flotio login' again.", err, string(errBody))
		}
		tokens, _ := LoadTokens()
		if tokens != nil && tokens.AccessToken != "" && tokens.AccessToken != token {
			return apiDoWithToken(method, baseURL, path, body, v, tokens.AccessToken)
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("API returned %d: %s", resp.StatusCode, string(errBody))
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}
	}
	return nil
}

// APIError is returned for non-2xx responses.
type APIError struct {
	Code int
	Body string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API returned %d: %s", e.Code, e.Body)
}

// WhoAmI fetches the current user info from GET /auth/@me.
func WhoAmI(host string) (map[string]interface{}, error) {
	var user map[string]interface{}
	if err := apiDo("GET", host, "/auth/@me", nil, &user); err != nil {
		return nil, err
	}
	return user, nil
}

// GetJSON performs an authenticated GET and decodes the JSON response into v.
func GetJSON(host, path string, v interface{}) error {
	return apiDo("GET", host, path, nil, v)
}

// PostJSON performs an authenticated POST with a JSON body, decoding the response into v.
func PostJSON(host, path string, body, v interface{}) error {
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request: %w", err)
		}
		r = bytes.NewReader(data)
	}
	return apiDo("POST", host, path, r, v)
}

// DeleteJSON performs an authenticated DELETE.
func DeleteJSON(host, path string) error {
	return apiDo("DELETE", host, path, nil, nil)
}

// PutJSON performs an authenticated PUT with a JSON body, decoding the response into v.
func PutJSON(host, path string, body, v interface{}) error {
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request: %w", err)
		}
		r = bytes.NewReader(data)
	}
	return apiDo("PUT", host, path, r, v)
}

// ExtractList tries to find a JSON array in the response object.
// Many Flotio API endpoints wrap lists in objects with keys like
// "credentials", "keystores", "environments", etc.
func ExtractList(data map[string]interface{}) ([]interface{}, error) {
	for _, v := range data {
		if arr, ok := v.([]interface{}); ok {
			return arr, nil
		}
	}
	return nil, fmt.Errorf("no list found in response")
}

// parseHost splits "scheme://host:port" into (scheme, host).
func parseHost(raw string) (scheme, host string) {
	idx := strings.Index(raw, "://")
	if idx >= 0 {
		return raw[:idx], raw[idx+3:]
	}
	return "https", raw
}
