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

	apiclient "github.com/flotio-dev/cli/pkg/api/client"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

// TokenFile is the path to the auth token storage, relative to ~/.flotio/.
const TokenFile = "auth.json"

// Tokens holds the persisted auth tokens.
type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
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
func SaveTokens(access, refresh string) error {
	path, err := tokenPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	t := Tokens{AccessToken: access, RefreshToken: refresh}
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

func tokenPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, ".flotio", TokenFile), nil
}

// New creates a new Flotio API client connected to the given host.
// If a valid access token is stored, it is injected as Bearer auth.
func New(host string) *apiclient.FlotioAPI {
	schemes := []string{"https"}
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
// and decodes the JSON response into v. The host should be the
// API host (e.g. "api.flotio.ovh"), and path the API path (e.g. "/auth/me").
func apiDo(method, host, path string, body io.Reader, v interface{}) error {
	tokens, err := LoadTokens()
	if err != nil {
		return fmt.Errorf("loading tokens: %w", err)
	}
	if tokens == nil || tokens.AccessToken == "" {
		return fmt.Errorf("not logged in")
	}

	url := "https://" + host + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

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
