package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePaths(t *testing.T) {
	// Save current state
	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(origCwd)
	}()

	origXdg := os.Getenv("XDG_CONFIG_HOME")
	origAppData := os.Getenv("APPDATA")
	origHome := os.Getenv("HOME")
	defer func() {
		_ = os.Setenv("XDG_CONFIG_HOME", origXdg)
		_ = os.Setenv("APPDATA", origAppData)
		_ = os.Setenv("HOME", origHome)
	}()

	t.Run("tools.json in current directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Change working directory to tmpDir
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}

		// Create tools.json in current directory
		if err := os.WriteFile("tools.json", []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		// Set mock user config dir environment variables
		mockUserConfig := filepath.Join(tmpDir, "user_config")
		_ = os.Setenv("XDG_CONFIG_HOME", mockUserConfig)
		_ = os.Setenv("APPDATA", mockUserConfig)
		_ = os.Setenv("HOME", mockUserConfig)

		// Get the active user config directory
		uConfigDir, err := os.UserConfigDir()
		if err != nil {
			t.Fatal(err)
		}

		// Create tools.json in user config directory to ensure current dir takes precedence
		userFragsDir := filepath.Join(uConfigDir, "frags")
		if err := os.MkdirAll(userFragsDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(userFragsDir, "tools.json"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		// Reset state
		pathsResolved = false

		tools := getToolsPath()
		tokens := getTokensPath()

		if tools != "tools.json" {
			t.Errorf("expected tools.json, got %s", tools)
		}
		if tokens != "tokens.json" {
			t.Errorf("expected tokens.json, got %s", tokens)
		}
	})

	t.Run("tools.json in user config directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Change working directory to tmpDir (which has NO tools.json)
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}

		// Set mock user config dir environment variables
		mockUserConfig := filepath.Join(tmpDir, "user_config")
		_ = os.Setenv("XDG_CONFIG_HOME", mockUserConfig)
		_ = os.Setenv("APPDATA", mockUserConfig)
		_ = os.Setenv("HOME", mockUserConfig)

		// Get the active user config directory
		uConfigDir, err := os.UserConfigDir()
		if err != nil {
			t.Fatal(err)
		}

		userFragsDir := filepath.Join(uConfigDir, "frags")
		if err := os.MkdirAll(userFragsDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Create tools.json in user config directory
		userToolsPath := filepath.Join(userFragsDir, "tools.json")
		if err := os.WriteFile(userToolsPath, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		// Reset state
		pathsResolved = false

		tools := getToolsPath()
		tokens := getTokensPath()

		expectedTools := userToolsPath
		expectedTokens := filepath.Join(userFragsDir, "tokens.json")

		if tools != expectedTools {
			t.Errorf("expected %s, got %s", expectedTools, tools)
		}
		if tokens != expectedTokens {
			t.Errorf("expected %s, got %s", expectedTokens, tokens)
		}
	})

	t.Run("tools.json doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Change working directory to empty tmpDir
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}

		// Set mock user config dir environment variables
		mockUserConfig := filepath.Join(tmpDir, "user_config")
		_ = os.Setenv("XDG_CONFIG_HOME", mockUserConfig)
		_ = os.Setenv("APPDATA", mockUserConfig)
		_ = os.Setenv("HOME", mockUserConfig)

		// Reset state
		pathsResolved = false

		tools := getToolsPath()
		tokens := getTokensPath()

		if tools != "tools.json" {
			t.Errorf("expected tools.json, got %s", tools)
		}
		if tokens != "tokens.json" {
			t.Errorf("expected tokens.json, got %s", tokens)
		}
	})
}
