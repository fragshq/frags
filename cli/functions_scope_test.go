package main

import (
	"maps"
	"slices"
	"testing"

	"github.com/fragshq/frags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scopeTestConfig() ExtendedToolsConfig {
	return ExtendedToolsConfig{
		ToolsConfig: frags.ToolsConfig{
			McpServers: frags.McpServerConfigs{
				"s1": {Url: "http://localhost/s1"},
				"s2": {Url: "http://localhost/s2"},
			},
			Collections: frags.ToolsCollectionConfigs{
				"http":         {ToolType: "http"},
				"call_me_once": {ToolType: "http"},
			},
		},
		ApiCPs: map[string]ApiCPConfig{
			"crm": {},
			"hr":  {},
		},
	}
}

func TestExtendedToolsConfig_ScopedTo(t *testing.T) {
	scoped := scopeTestConfig().ScopedTo(frags.RequiredTools{
		{Name: "s1", Type: frags.ToolTypeMCP},
		{Name: "http", Type: frags.ToolTypeCollection},
		{Name: "crm", Type: frags.ToolTypeApicp},
	})
	assert.Equal(t, []string{"s1"}, sortedKeys(scoped.McpServers))
	assert.Equal(t, []string{"http"}, sortedKeys(scoped.Collections))
	assert.Equal(t, []string{"crm"}, sortedKeys(scoped.ApiCPs))
}

func TestExtendedToolsConfig_ScopedTo_NoRequirements(t *testing.T) {
	cfg := scopeTestConfig()
	assert.Equal(t, cfg, cfg.ScopedTo(nil), "nil requirements should keep the full configuration")
	assert.Equal(t, cfg, cfg.ScopedTo(frags.RequiredTools{}), "empty requirements should keep the full configuration")
}

func TestExtendedToolsConfig_ScopedTo_MissingRequirement(t *testing.T) {
	// a required tool that isn't configured is simply absent; RequiredTools.Check reports it at run time
	scoped := scopeTestConfig().ScopedTo(frags.RequiredTools{{Name: "nope", Type: frags.ToolTypeMCP}})
	assert.Empty(t, scoped.McpServers)
	assert.Empty(t, scoped.Collections)
	assert.Empty(t, scoped.ApiCPs)
}

// TestExtendedToolsConfig_ScopedTo_DefaultHttp is the case that motivated scoping: a plan that requires only the
// default "http" collection can call bare http_fetch even though tools.json also configures other http collections.
func TestExtendedToolsConfig_ScopedTo_DefaultHttp(t *testing.T) {
	sm, err := parsePlan("plan.fml", []byte("require collection http\n\nsession(\"test\") {\n    - hello\n}\n"))
	require.NoError(t, err)
	scoped := scopeTestConfig().ScopedTo(sm.RequiredTools)
	assert.Equal(t, []string{"http"}, sortedKeys(scoped.Collections))
	assert.Empty(t, scoped.McpServers)
}

func sortedKeys[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}
