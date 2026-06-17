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

	apiclient "github.com/flotio-dev/cli/pkg/api/client"
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
	Password     string `json:"password,omitempty"`
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

// SaveCredentials persists email/password alongside tokens.
func SaveCredentials(email, password, access, refresh string) error {
	return saveAuth(func(t *Tokens) {
		t.Email = email
		t.Password = password
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

// Relogin attempts to re-authenticate using stored credentials.
// Returns nil on success (new tokens saved to auth.json).
func Relogin(baseURL string) error {
	tokens, err := LoadTokens()
	if err != nil || tokens == nil || tokens.Email == "" {
		return fmt.Errorf("no stored credentials")
	}
	return DoLogin(baseURL, tokens.Email, tokens.Password)
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

	resp, err := http.DefaultClient.Do(req)
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

	return SaveCredentials(email, password, result.AccessToken, result.RefreshToken)
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

// New creates a new Flotio API client connected to the given host.
// If a valid access token is stored, it is injected as Bearer auth.
// The host can be a plain hostname or scheme://host (scheme is extracted).
func New(rawHost string) *apiclient.FlotioAPI {
	scheme, host := parseHost(rawHost)
	schemes := []string{scheme}
	transport := httptransport.New(host, "/", schemes)

	// Inject stored bearer token if available.
	tokens, _ := LoadTokens()
	if tokens != nil && tokens.AccessToken != "" {
		transport.DefaultAuthentication = httptransport.BearerToken(tokens.AccessToken)
	}

	return apiclient.New(transport, strfmt.Default)
}

// IsLoggedIn returns true if a valid-looking token is stored.
func IsLoggedIn() bool {
	tokens, _ := LoadTokens()
	return tokens != nil && tokens.AccessToken != ""
}

// apiDo is a helper that performs an authenticated HTTP request
// and decodes the JSON response into v. The baseURL should be a
// full URL (e.g. "https://api.flotio.ovh"), and path the API path (e.g. "/auth/me").
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		// Token expired — try re-login once, then retry
		if err := Relogin(baseURL); err != nil {
			errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			return fmt.Errorf("API returned 401 (and re-login failed: %v): %s", err, string(errBody))
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

// WhoAmI fetches the current user info from GET /auth/me.
func WhoAmI(host string) (map[string]interface{}, error) {
	var user map[string]interface{}
	if err := apiDo("GET", host, "/auth/me", nil, &user); err != nil {
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
