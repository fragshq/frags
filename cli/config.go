/*
 * Copyright (C) 2025 Simone Pezzano
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

package main

import "strings"

// supported AI engines
const (
	engineGemini    = "gemini"
	engineOllama    = "ollama"
	engineChatgpt   = "chatgpt"
	engineAnthropic = "anthropic"
	engineDummy     = "dummy"
)

// supported output formats
const (
	formatTemplate = "template"
	formatYAML     = "yaml"
	formatJSON     = "json"
	formatEnv      = "bash"
)

type Config struct {
	GeminiServiceAccountPath string  `mapstructure:"GEMINI_SERVICE_ACCOUNT_PATH" yaml:"GEMINI_SERVICE_ACCOUNT_PATH" tui:"label=Gemini Service Account Path"`
	GeminiProjectID          string  `mapstructure:"GEMINI_PROJECT_ID" yaml:"GEMINI_PROJECT_ID" tui:"label=Gemini Project ID"`
	GeminiLocation           string  `mapstructure:"GEMINI_LOCATION" yaml:"GEMINI_LOCATION" tui:"label=Gemini Location"`
	ParallelWorkers          int     `mapstructure:"PARALLEL_WORKERS" yaml:"PARALLEL_WORKERS" tui:"label=Parallel Workers"`
	OllamaBaseURL            string  `mapstructure:"OLLAMA_BASE_URL" yaml:"OLLAMA_BASE_URL" tui:"label=Ollama Base URL"`
	Model                    string  `mapstructure:"MODEL" yaml:"MODEL" tui:"label=Model Name,list"`
	AiEngine                 string  `mapstructure:"AI_ENGINE" yaml:"AI_ENGINE" tui:"label=AI Engine,list,enum=gemini|ollama|chatgpt|anthropic|dummy"`
	Temperature              float32 `mapstructure:"TEMPERATURE" yaml:"TEMPERATURE" tui:"label=Temperature"`
	TopK                     float32 `mapstructure:"TOP_K" yaml:"TOP_K" tui:"label=Top K"`
	TopP                     float32 `mapstructure:"TOP_P" yaml:"TOP_P" tui:"label=Top P"`
	NumPredict               int     `mapstructure:"NUM_PREDICT" yaml:"NUM_PREDICT" tui:"label=Num Predict"`
	ChatGptApiKey            string  `mapstructure:"CHATGPT_API_KEY" yaml:"CHATGPT_API_KEY" tui:"label=ChatGPT API Key"`
	ChatGptBaseURL           string  `mapstructure:"CHATGPT_BASE_URL" yaml:"CHATGPT_BASE_URL" tui:"label=ChatGPT Base URL"`
	AnthropicApiKey          string  `mapstructure:"ANTHROPIC_API_KEY" yaml:"ANTHROPIC_API_KEY" tui:"label=Anthropic API Key"`
	ThinkingLevel            string  `mapstructure:"THINKING_LEVEL" yaml:"THINKING_LEVEL" tui:"label=Thinking Level,enum=LOW|MEDIUM|HIGH"`
	OauthDisabled            bool    `mapstructure:"OAUTH_DISABLED" yaml:"OAUTH_DISABLED" tui:"label=OAuth Disabled"`
}

// guessAi tries to guess the AI engine based on the configuration.
func (c Config) guessAi() string {
	switch strings.ToLower(c.AiEngine) {
	case engineOllama:
		return engineOllama
	case engineGemini:
		return engineGemini
	case engineChatgpt:
		return engineChatgpt
	case engineAnthropic:
		return engineAnthropic
	case engineDummy:
		return engineDummy
	}
	if c.OllamaBaseURL != "" && c.Model != "" {
		return engineOllama
	}
	if c.GeminiServiceAccountPath != "" && c.GeminiProjectID != "" && c.GeminiLocation != "" {
		return engineGemini
	}
	if c.ChatGptApiKey != "" && c.ChatGptBaseURL != "" {
		return engineChatgpt
	}
	if c.AnthropicApiKey != "" {
		return engineAnthropic
	}
	return ""
}

var cfg = Config{}
