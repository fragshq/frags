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

package chatgpt

import (
	"sort"

	"github.com/fragshq/frags/schema"
)

func SchemaToChatGPTMap(s *schema.Schema) map[string]any {
	if s == nil {
		return nil
	}

	m := map[string]any{}

	// Scalar / metadata fields
	if s.Type != "" {
		m["type"] = s.Type
	}
	if s.Description != "" {
		m["description"] = s.Description
	}
	if s.Title != "" {
		m["title"] = s.Title
	}
	if s.Format != "" {
		m["format"] = s.Format
	}
	if s.Pattern != "" {
		m["pattern"] = s.Pattern
	}
	if s.Default != nil {
		m["default"] = s.Default
	}
	if s.Example != nil {
		m["example"] = s.Example
	}
	// $ref: dropped — not supported by OpenAI structured outputs

	// Numeric constraints
	// minimum, maximum, minLength, maxLength, minItems, maxItems: dropped — not supported
	if s.MinProperties != nil {
		m["minProperties"] = *s.MinProperties
	}
	if s.MaxProperties != nil {
		m["maxProperties"] = *s.MaxProperties
	}

	// Nullable
	if s.Nullable != nil {
		m["nullable"] = *s.Nullable
	}

	// Enum
	if len(s.Enum) > 0 {
		m["enum"] = s.Enum
	}

	// PropertyOrdering (Gemini-style extension, kept for compatibility)
	if len(s.PropertyOrdering) > 0 {
		m["propertyOrdering"] = s.PropertyOrdering
	}

	// Items (arrays)
	if s.Items != nil {
		m["items"] = SchemaToChatGPTMap(s.Items)
	}

	// Properties (objects) — recurse into each value
	if len(s.Properties) > 0 {
		props := make(map[string]any, len(s.Properties))
		required := make([]string, 0, len(s.Properties))
		for k, v := range s.Properties {
			props[k] = SchemaToChatGPTMap(v)
			required = append(required, k)
		}
		m["properties"] = props

		// OpenAI structured outputs: every property must be listed as required,
		// regardless of what the source schema's Required list says.
		sort.Strings(required) // deterministic ordering
		m["required"] = required
	}

	// Inject additionalProperties: false on object layers, as required by OpenAI
	if s.Type == "object" {
		m["additionalProperties"] = false
	}

	// Combiners
	if len(s.OneOf) > 0 {
		oneOf := make([]any, len(s.OneOf))
		for i, sub := range s.OneOf {
			oneOf[i] = SchemaToChatGPTMap(sub)
		}
		m["oneOf"] = oneOf
	}
	if len(s.AnyOf) > 0 {
		anyOf := make([]any, len(s.AnyOf))
		for i, sub := range s.AnyOf {
			anyOf[i] = SchemaToChatGPTMap(sub)
		}
		m["anyOf"] = anyOf
	}

	return m
}
