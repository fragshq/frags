/*
 * Copyright (C) 2026 Simone Pezzano
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package frags

import (
	"testing"

	"github.com/fragshq/frags/log"
	"github.com/fragshq/frags/mcpauth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingOauthProvider is a GenericOauthProvider prototype that records whether New was called,
// and with which configuration, so tests can verify when and how the OAuth path is selected.
type recordingOauthProvider struct {
	mcpauth.NopOauthAuthProvider
	newCalled bool
	config    mcpauth.OAuthProviderConfig
}

func (r *recordingOauthProvider) New(config mcpauth.OAuthProviderConfig, _ *log.StreamerLogger) mcpauth.GenericOauthProvider {
	r.newCalled = true
	r.config = config
	return r
}

func (r *recordingOauthProvider) Config() *mcpauth.OAuthProviderConfig {
	return &r.config
}

func TestMcpTool_selectAuthProvider(t *testing.T) {
	ptr := func(s string) *string { return &s }

	t.Run("static token wins over everything else", func(t *testing.T) {
		oauth := &recordingOauthProvider{}
		tool := NewMcpTool("srv", McpServerConfig{
			Url:      "https://example.com/mcp",
			Token:    ptr("my-token"),
			ClientID: ptr("client"),
			Headers:  map[string]string{"Authorization": "Bearer manual", "X-Custom": "1"},
		}).WithOAuthProvider(oauth)

		provider := tool.selectAuthProvider(nil)

		static, ok := provider.(*mcpauth.StaticTokenProvider)
		require.True(t, ok, "expected *StaticTokenProvider, got %T", provider)
		tok, err := static.Token()
		require.NoError(t, err)
		assert.Equal(t, "my-token", tok.AccessToken)
		assert.False(t, oauth.newCalled)
	})

	t.Run("manual Authorization header disables automatic auth", func(t *testing.T) {
		oauth := &recordingOauthProvider{}
		tool := NewMcpTool("srv", McpServerConfig{
			Url:      "https://example.com/mcp",
			ClientID: ptr("client"),
			Headers:  map[string]string{"Authorization": "Bearer manual"},
		}).WithOAuthProvider(oauth)

		assert.Nil(t, tool.selectAuthProvider(nil))
		assert.False(t, oauth.newCalled)
	})

	t.Run("nil client ID means no authentication", func(t *testing.T) {
		oauth := &recordingOauthProvider{}
		tool := NewMcpTool("srv", McpServerConfig{
			Url:     "https://example.com/mcp",
			Headers: map[string]string{"X-Custom": "1"},
		}).WithOAuthProvider(oauth)

		assert.Nil(t, tool.selectAuthProvider(nil))
		assert.False(t, oauth.newCalled)
	})

	t.Run("empty client ID means no authentication", func(t *testing.T) {
		oauth := &recordingOauthProvider{}
		tool := NewMcpTool("srv", McpServerConfig{
			Url:      "https://example.com/mcp",
			ClientID: ptr(""),
			// OAuth-related fields without a client ID must not opt into OAuth.
			ClientSecret: ptr("secret"),
			TokenURL:     ptr("https://example.com/token"),
		}).WithOAuthProvider(oauth)

		assert.Nil(t, tool.selectAuthProvider(nil))
		assert.False(t, oauth.newCalled)
	})

	t.Run("client ID selects OAuth with the server configuration", func(t *testing.T) {
		oauth := &recordingOauthProvider{}
		tool := NewMcpTool("srv", McpServerConfig{
			Url:              "https://example.com/mcp",
			ClientID:         ptr("client"),
			ClientSecret:     ptr("secret"),
			AuthorizationURL: ptr("https://example.com/authorize"),
			TokenURL:         ptr("https://example.com/token"),
			Headers:          map[string]string{"X-Custom": "1"},
		}).WithOAuthProvider(oauth)

		provider := tool.selectAuthProvider(nil)

		require.True(t, oauth.newCalled)
		assert.Same(t, oauth, provider)
		assert.Equal(t, "srv", oauth.config.Name)
		assert.Equal(t, "https://example.com/mcp", oauth.config.BaseURL)
		assert.Equal(t, "client", oauth.config.ClientID)
		assert.Equal(t, "secret", oauth.config.ClientSecret)
		assert.Equal(t, "https://example.com/authorize", oauth.config.AuthorizationURL)
		assert.Equal(t, "https://example.com/token", oauth.config.TokenURL)
		assert.NotNil(t, oauth.config.HTTPClient)
	})
}
