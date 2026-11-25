package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePaths(t *testing.T) {
	t.Run("tools.json in current directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Change working directory to tmpDir
		t.Chdir(tmpDir)

		// Create tools.json in current directory
		require.NoError(t, os.WriteFile("tools.json", []byte("{}"), 0644))

		// Set mock user config dir environment variables
		mockUserConfig := filepath.Join(tmpDir, "user_config")
		t.Setenv("XDG_CONFIG_HOME", mockUserConfig)
		t.Setenv("APPDATA", mockUserConfig)
		t.Setenv("HOME", mockUserConfig)

		// Get the active user config directory
		uConfigDir, err := os.UserConfigDir()
		require.NoError(t, err)

		// Create tools.json in user config directory to ensure current dir takes precedence
		userFragsDir := filepath.Join(uConfigDir, "frags")
		require.NoError(t, os.MkdirAll(userFragsDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(userFragsDir, "tools.json"), []byte("{}"), 0644))

		// Reset state
		pathsResolved = false

		tools := getToolsPath()
		tokens := getTokensPath()

		assert.Equal(t, "tools.json", tools)
		assert.Equal(t, "tokens.json", tokens)
	})

	t.Run("tools.json in user config directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Change working directory to tmpDir (which has NO tools.json)
		t.Chdir(tmpDir)

		// Set mock user config dir environment variables
		mockUserConfig := filepath.Join(tmpDir, "user_config")
		t.Setenv("XDG_CONFIG_HOME", mockUserConfig)
		t.Setenv("APPDATA", mockUserConfig)
		t.Setenv("HOME", mockUserConfig)

		// Get the active user config directory
		uConfigDir, err := os.UserConfigDir()
		require.NoError(t, err)

		userFragsDir := filepath.Join(uConfigDir, "frags")
		require.NoError(t, os.MkdirAll(userFragsDir, 0755))

		// Create tools.json in user config directory
		userToolsPath := filepath.Join(userFragsDir, "tools.json")
		require.NoError(t, os.WriteFile(userToolsPath, []byte("{}"), 0644))

		// Reset state
		pathsResolved = false

		tools := getToolsPath()
		tokens := getTokensPath()

		expectedTools := userToolsPath
		expectedTokens := filepath.Join(userFragsDir, "tokens.json")

		assert.Equal(t, expectedTools, tools)
		assert.Equal(t, expectedTokens, tokens)
	})

	t.Run("tools.json doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Change working directory to empty tmpDir
		t.Chdir(tmpDir)

		// Set mock user config dir environment variables
		mockUserConfig := filepath.Join(tmpDir, "user_config")
		t.Setenv("XDG_CONFIG_HOME", mockUserConfig)
		t.Setenv("APPDATA", mockUserConfig)
		t.Setenv("HOME", mockUserConfig)

		// Reset state
		pathsResolved = false

		tools := getToolsPath()
		tokens := getTokensPath()

		assert.Equal(t, "tools.json", tools)
		assert.Equal(t, "tokens.json", tokens)
	})
}
