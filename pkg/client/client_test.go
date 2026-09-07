package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestExtractList(t *testing.T) {
	// Single array value
	data := map[string]interface{}{
		"credentials": []interface{}{"a", "b"},
	}
	items, err := ExtractList(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}

	// Multiple keys, one is an array
	data = map[string]interface{}{
		"count": 3.0,
		"items": []interface{}{"x", "y", "z"},
	}
	items, err = ExtractList(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}

	// No array value
	data = map[string]interface{}{
		"count": 0.0,
		"name":  "empty",
	}
	_, err = ExtractList(data)
	if err == nil {
		t.Error("expected error for no-list response")
	}
}

func TestExtractListEmpty(t *testing.T) {
	data := map[string]interface{}{
		"items": []interface{}{},
	}
	items, err := ExtractList(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestTokensRoundtrip(t *testing.T) {
	// Override token path to temp dir
	origFn := TokenPathFn
	tmpDir := t.TempDir()
	TokenPathFn = func() (string, error) {
		return filepath.Join(tmpDir, TokenFile), nil
	}
	defer func() { TokenPathFn = origFn }()

	// Start clean
	ClearTokens()

	// Not logged in
	if IsLoggedIn() {
		t.Error("expected not logged in")
	}

	// Save tokens
	if err := SaveTokens("access-123", "refresh-456"); err != nil {
		t.Fatal(err)
	}

	// Now logged in
	if !IsLoggedIn() {
		t.Error("expected logged in after save")
	}

	// Load and verify
	tokens, err := LoadTokens()
	if err != nil {
		t.Fatal(err)
	}
	if tokens == nil {
		t.Fatal("expected tokens")
	}
	if tokens.AccessToken != "access-123" {
		t.Errorf("expected access-123, got %s", tokens.AccessToken)
	}
	if tokens.RefreshToken != "refresh-456" {
		t.Errorf("expected refresh-456, got %s", tokens.RefreshToken)
	}

	// Clear
	if err := ClearTokens(); err != nil {
		t.Fatal(err)
	}
	if IsLoggedIn() {
		t.Error("expected not logged in after clear")
	}
}

func TestLoadTokensNoFile(t *testing.T) {
	origFn := TokenPathFn
	TokenPathFn = func() (string, error) {
		return filepath.Join(t.TempDir(), "nonexistent.json"), nil
	}
	defer func() { TokenPathFn = origFn }()

	tokens, err := LoadTokens()
	if err != nil {
		t.Fatal(err)
	}
	if tokens != nil {
		t.Error("expected nil tokens when file doesn't exist")
	}
}

func TestClearTokensNoFile(t *testing.T) {
	origFn := TokenPathFn
	TokenPathFn = func() (string, error) {
		return filepath.Join(t.TempDir(), "nonexistent.json"), nil
	}
	defer func() { TokenPathFn = origFn }()

	// Should not error
	if err := ClearTokens(); err != nil {
		t.Fatal(err)
	}
}

func TestTokensFileContent(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := TokenPathFn
	TokenPathFn = func() (string, error) {
		return filepath.Join(tmpDir, TokenFile), nil
	}
	defer func() { TokenPathFn = origFn }()

	if err := SaveTokens("abc", "def"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, TokenFile))
	if err != nil {
		t.Fatal(err)
	}

	var tokens Tokens
	if err := json.Unmarshal(data, &tokens); err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken != "abc" || tokens.RefreshToken != "def" {
		t.Error("token content mismatch")
	}
}

func TestIsLoggedIn_RefreshTokenOnly(t *testing.T) {
	origFn := TokenPathFn
	tmpDir := t.TempDir()
	TokenPathFn = func() (string, error) {
		return filepath.Join(tmpDir, TokenFile), nil
	}
	defer func() { TokenPathFn = origFn }()

	// Initially not logged in
	if IsLoggedIn() {
		t.Error("expected not logged in")
	}

	// Save only refresh token
	if err := saveAuth(func(tok *Tokens) {
		tok.RefreshToken = "refresh-only"
	}); err != nil {
		t.Fatal(err)
	}

	if !IsLoggedIn() {
		t.Error("expected logged in when refresh token is present")
	}
}

func TestAuthRoundTripper_SuccessWithout401(t *testing.T) {
	origFn := TokenPathFn
	tmpDir := t.TempDir()
	TokenPathFn = func() (string, error) {
		return filepath.Join(tmpDir, TokenFile), nil
	}
	defer func() { TokenPathFn = origFn }()

	_ = SaveTokens("valid-access", "valid-refresh")

	receivedAuth := ""
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok": true}`))
	}))
	defer ts.Close()

	rt := NewAuthRoundTripper(ts.URL, ts.Client().Transport)
	client := &http.Client{Transport: rt}

	req, err := http.NewRequest("GET", ts.URL+"/api/items", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if receivedAuth != "Bearer valid-access" {
		t.Errorf("expected 'Bearer valid-access', got %q", receivedAuth)
	}
}

func TestAuthRoundTripper_SkipAuthEndpoints(t *testing.T) {
	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "bad creds"}`))
	}))
	defer ts.Close()

	rt := NewAuthRoundTripper(ts.URL, ts.Client().Transport)
	client := &http.Client{Transport: rt}

	req, err := http.NewRequest("POST", ts.URL+"/auth/login", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 directly, got %d", resp.StatusCode)
	}
	if !called {
		t.Error("expected server to be called")
	}
}

func TestAuthRoundTripper_AutoRefreshOn401(t *testing.T) {
	origFn := TokenPathFn
	tmpDir := t.TempDir()
	TokenPathFn = func() (string, error) {
		return filepath.Join(tmpDir, TokenFile), nil
	}
	defer func() { TokenPathFn = origFn }()

	_ = SaveTokens("expired-access", "valid-refresh")

	var requestsReceived []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestsReceived = append(requestsReceived, r.URL.Path)
		if r.URL.Path == "/auth/refresh" {
			cookie, err := r.Cookie("refresh_token")
			if err != nil || cookie.Value != "valid-refresh" {
				http.Error(w, "invalid cookie", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token":  "new-access-token",
				"refresh_token": "new-refresh-token",
			})
			return
		}

		if r.URL.Path == "/api/data" {
			auth := r.Header.Get("Authorization")
			if auth == "Bearer new-access-token" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status": "ok"}`))
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": "token expired"}`))
			return
		}

		http.NotFound(w, r)
	}))
	defer ts.Close()

	// Override httpClient transport so DoRefresh routes to test server
	origTransport := httpClient.Transport
	httpClient.Transport = ts.Client().Transport
	defer func() { httpClient.Transport = origTransport }()

	rt := NewAuthRoundTripper(ts.URL, ts.Client().Transport)
	client := &http.Client{Transport: rt}

	req, err := http.NewRequest("GET", ts.URL+"/api/data", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 after refresh, got %d", resp.StatusCode)
	}

	// Verify new tokens saved to disk
	tokens, err := LoadTokens()
	if err != nil || tokens == nil {
		t.Fatal("expected tokens saved")
	}
	if tokens.AccessToken != "new-access-token" {
		t.Errorf("expected new-access-token, got %s", tokens.AccessToken)
	}
	if tokens.RefreshToken != "new-refresh-token" {
		t.Errorf("expected new-refresh-token, got %s", tokens.RefreshToken)
	}
}

func TestAuthRoundTripper_ReplayRequestBody(t *testing.T) {
	origFn := TokenPathFn
	tmpDir := t.TempDir()
	TokenPathFn = func() (string, error) {
		return filepath.Join(tmpDir, TokenFile), nil
	}
	defer func() { TokenPathFn = origFn }()

	_ = SaveTokens("expired-access", "valid-refresh")

	var bodiesReceived []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/refresh" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token":  "new-access-token",
				"refresh_token": "new-refresh-token",
			})
			return
		}

		if r.URL.Path == "/api/create" {
			b, _ := io.ReadAll(r.Body)
			bodiesReceived = append(bodiesReceived, string(b))
			if r.Header.Get("Authorization") == "Bearer new-access-token" {
				w.WriteHeader(http.StatusCreated)
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}))
	defer ts.Close()

	origTransport := httpClient.Transport
	httpClient.Transport = ts.Client().Transport
	defer func() { httpClient.Transport = origTransport }()

	rt := NewAuthRoundTripper(ts.URL, ts.Client().Transport)
	client := &http.Client{Transport: rt}

	reqBody := `{"name":"test-project"}`
	req, err := http.NewRequest("POST", ts.URL+"/api/create", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201 Created, got %d", resp.StatusCode)
	}

	// Both the initial failed request and the retry should have seen the full body
	if len(bodiesReceived) != 2 {
		t.Fatalf("expected 2 requests received, got %d", len(bodiesReceived))
	}
	if bodiesReceived[0] != reqBody || bodiesReceived[1] != reqBody {
		t.Errorf("body not replayed correctly: %v", bodiesReceived)
	}
}

func TestAuthRoundTripper_RefreshFailureClearsTokens(t *testing.T) {
	origFn := TokenPathFn
	tmpDir := t.TempDir()
	TokenPathFn = func() (string, error) {
		return filepath.Join(tmpDir, TokenFile), nil
	}
	defer func() { TokenPathFn = origFn }()

	_ = SaveTokens("expired-access", "invalid-refresh")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/refresh" {
			http.Error(w, "invalid refresh token", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	origTransport := httpClient.Transport
	httpClient.Transport = ts.Client().Transport
	defer func() { httpClient.Transport = origTransport }()

	rt := NewAuthRoundTripper(ts.URL, ts.Client().Transport)
	client := &http.Client{Transport: rt}

	req, err := http.NewRequest("GET", ts.URL+"/api/secure", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err == nil {
		if resp != nil {
			resp.Body.Close()
		}
		t.Fatalf("expected error on failed refresh, got nil with status %d", resp.StatusCode)
	}

	if !strings.Contains(err.Error(), "flotio login") {
		t.Errorf("expected error to mention 'flotio login', got: %v", err)
	}

	// Ensure tokens were cleared
	if IsLoggedIn() {
		t.Error("expected tokens to be cleared after refresh failure")
	}
}

func TestAuthRoundTripper_ConcurrentRequestsSingleRefresh(t *testing.T) {
	origFn := TokenPathFn
	tmpDir := t.TempDir()
	TokenPathFn = func() (string, error) {
		return filepath.Join(tmpDir, TokenFile), nil
	}
	defer func() { TokenPathFn = origFn }()

	_ = SaveTokens("expired-access", "valid-refresh")

	var refreshCount int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/refresh" {
			atomic.AddInt32(&refreshCount, 1)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token":  "new-access-token",
				"refresh_token": "new-refresh-token",
			})
			return
		}

		if r.Header.Get("Authorization") == "Bearer new-access-token" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok": true}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	origTransport := httpClient.Transport
	httpClient.Transport = ts.Client().Transport
	defer func() { httpClient.Transport = origTransport }()

	rt := NewAuthRoundTripper(ts.URL, ts.Client().Transport)
	client := &http.Client{Transport: rt}

	const numGoroutines = 5
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errs := make(chan error, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			req, err := http.NewRequest("GET", ts.URL+"/api/parallel", nil)
			if err != nil {
				errs <- err
				return
			}
			resp, err := client.Do(req)
			if err != nil {
				errs <- err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				errs <- fmt.Errorf("unexpected status %d", resp.StatusCode)
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent request error: %v", err)
	}

	if count := atomic.LoadInt32(&refreshCount); count != 1 {
		t.Errorf("expected exactly 1 refresh call, got %d", count)
	}
}
