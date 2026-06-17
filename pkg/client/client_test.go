package client

import (
	"encoding/json"
	"os"
	"path/filepath"
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
