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

package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type s2 struct {
	P1 float64 `json:"p1"`
	P2 float64 `json:"p2"`
}

type s3 struct {
	M1 float64 `json:"m1"`
	M2 float64 `json:"m2"`
}

type s1 struct {
	S2        s2            `json:"s2"`
	ArrayOfS2 []s2          `json:"arrayOfS2"`
	MapOfS3   map[string]s3 `json:"mapOfS3"`
}

func TestSchema_Validate(t *testing.T) {
	t.Run("base type", func(t *testing.T) {
		s := Schema{Type: String}
		err := s.Validate("foo", nil)
		assert.NoError(t, err)

		s = Schema{Type: Integer}
		err = s.Validate("foo", nil)
		assert.Error(t, err)
	})
	t.Run("map", func(t *testing.T) {
		s := Schema{
			Type: Object,
			Properties: map[string]*Schema{
				"foo": {Type: String},
				"bar": {Type: Integer},
			},
		}
		err := s.Validate(map[string]any{"foo": "123", "bar": 123}, nil)
		assert.NoError(t, err)

		err = s.Validate(map[string]any{"foo": "123"}, nil)
		assert.NoError(t, err)

		err = s.Validate(map[string]any{"foo": 123}, nil)
		assert.Error(t, err)

		s.Required = []string{"foo", "bar"}
		err = s.Validate(map[string]any{"foo": "123"}, nil)
		assert.Error(t, err)
	})
	t.Run("array", func(t *testing.T) {
		s := Schema{
			Type:  Array,
			Items: &Schema{Type: String},
		}
		err := s.Validate([]string{"foo", "bar"}, nil)
		assert.NoError(t, err)

		s = Schema{
			Type: Array,
			Items: &Schema{
				Type:     Object,
				Required: []string{"foo", "bar"},
				Properties: map[string]*Schema{
					"foo": {Type: String},
					"bar": {Type: Integer},
				},
			},
		}
		err = s.Validate([]map[string]any{{"foo": "123", "bar": 123}}, nil)
		assert.NoError(t, err)

		err = s.Validate([]map[string]any{{"foo": "123", "bar": 123}, {"foo": "123", "bar": "123"}}, nil)
		assert.Error(t, err)
	})
	t.Run("composite struct", func(t *testing.T) {
		s := Schema{
			Type:     Object,
			Required: []string{"s2", "arrayOfS2", "mapOfS3"},
			Properties: map[string]*Schema{
				"s2": {
					Type:     Object,
					Required: []string{"p1", "p2"},
					Properties: map[string]*Schema{
						"p1": {Type: Number},
						"p2": {Type: Integer},
					},
				},
				"arrayOfS2": {
					Type: Array,
					Items: &Schema{
						Type: Object,
						Properties: map[string]*Schema{
							"p1": {Type: Number},
							"p2": {Type: Integer},
						},
					},
				},
				"mapOfS3": {
					Type:     Object,
					Required: []string{"v1"},
					Properties: map[string]*Schema{
						"v1": {
							Type:     Object,
							Required: []string{"m1", "m2"},
							Properties: map[string]*Schema{
								"m1": {Type: Integer},
								"m2": {Type: Integer},
							},
						},
					},
				},
			},
		}
		struct1 := s1{
			S2: s2{
				P1: 32.5,
				P2: 123,
			},
			ArrayOfS2: []s2{{
				P1: 32.5,
				P2: 123,
			}},
			MapOfS3: map[string]s3{
				"v1": {
					M1: 123,
					M2: 456,
				},
			},
		}
		err := s.Validate(struct1, nil)
		assert.NoError(t, err)
	})
}

func TestSchema_Validate_Soft(t *testing.T) {
	t.Run("all matches", func(t *testing.T) {
		s := Schema{
			Type: Object,
			Properties: map[string]*Schema{
				"int":     {Type: Integer},
				"str":     {Type: String},
				"bool":    {Type: Boolean},
				"realInt": {Type: Integer},
			},
		}
		err := s.Validate(map[string]any{
			"int":     "123",
			"str":     "123",
			"bool":    "true",
			"realInt": 123,
		}, &ValidatorOptions{SoftValidation: true})
		assert.NoError(t, err)
	})
	t.Run("unmatched promise", func(t *testing.T) {
		s := Schema{
			Type: Object,
			Properties: map[string]*Schema{
				"int":  {Type: Integer},
				"str":  {Type: String},
				"bool": {Type: Boolean},
			},
		}
		err := s.Validate(map[string]any{
			"int":  "abc",
			"str":  "123",
			"bool": true,
		}, &ValidatorOptions{SoftValidation: true})
		assert.Error(t, err)
	})
	t.Run("unmatched promise", func(t *testing.T) {
		s := Schema{
			Type: Object,
			Properties: map[string]*Schema{
				"int":  {Type: Integer},
				"str":  {Type: String},
				"bool": {Type: Boolean},
			},
		}
		err := s.Validate(map[string]any{
			"int":  "123",
			"str":  "123",
			"bool": "foo",
		}, &ValidatorOptions{SoftValidation: true})
		assert.Error(t, err)
	})

}

func TestSchema_Validate_OneOf(t *testing.T) {
	s := Schema{OneOf: []*Schema{
		{Type: String},
		{Type: Integer},
	}}
	t.Run("matches exactly one branch", func(t *testing.T) {
		assert.NoError(t, s.Validate("hello", nil))
		assert.NoError(t, s.Validate(7, nil))
	})
	t.Run("matches no branch", func(t *testing.T) {
		assert.Error(t, s.Validate(true, nil))
		assert.Error(t, s.Validate(2.5, nil))
	})
	t.Run("matches more than one branch", func(t *testing.T) {
		overlapping := Schema{OneOf: []*Schema{
			{Type: Integer},
			{Type: Number},
		}}
		assert.Error(t, overlapping.Validate(1, nil))
		assert.NoError(t, overlapping.Validate(2.5, nil))
	})
	t.Run("nested in object", func(t *testing.T) {
		obj := Schema{Type: Object, Properties: map[string]*Schema{"v": &s}}
		assert.NoError(t, obj.Validate(map[string]any{"v": "x"}, nil))
		assert.Error(t, obj.Validate(map[string]any{"v": false}, nil))
	})
	t.Run("null branch", func(t *testing.T) {
		nullable := true
		withNull := Schema{OneOf: []*Schema{
			{Type: String},
			{Type: Integer, Nullable: &nullable},
		}}
		assert.NoError(t, withNull.Validate(nil, nil))
		assert.Error(t, s.Validate(nil, nil))
	})
	t.Run("soft validation prefers strict match", func(t *testing.T) {
		// "7" coerces to an integer under soft validation, but it is a string strictly:
		// the strict match must win instead of being rejected as ambiguous.
		assert.NoError(t, s.Validate("7", &ValidatorOptions{SoftValidation: true}))
		intOrBool := Schema{OneOf: []*Schema{{Type: Integer}, {Type: Boolean}}}
		assert.NoError(t, intOrBool.Validate("7", &ValidatorOptions{SoftValidation: true}))
		assert.Error(t, intOrBool.Validate("7", nil))
	})
}

func TestGuessSchema_ValidatesItsInput(t *testing.T) {
	inputs := []any{
		[]any{1, 2.5},
		[]any{"hello", 42, true},
		[]any{"a", 1, nil},
		[]any{[]any{1, "a"}, []any{"b"}},
		[]any{[]any{1, 2.5}, []any{3}},
		[]any{map[string]any{"value": "hello"}, map[string]any{"value": 42}},
		[]any{map[string]any{"value": 1}, map[string]any{"value": 2.5}, map[string]any{"value": nil}},
	}
	for _, in := range inputs {
		s := GuessSchema(in)
		assert.NoError(t, s.Validate(in, nil), "input %#v", in)
	}
}
