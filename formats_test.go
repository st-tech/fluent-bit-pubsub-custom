package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeToJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
		hasErr   bool
	}{
		{
			name: "simple map",
			input: map[interface{}]interface{}{
				"k1": "v1",
				"k2": "v2",
			},
			expected: `{"k1":"v1","k2":"v2"}`,
			hasErr:   false,
		},
		{
			name: "nested map",
			input: map[interface{}]interface{}{
				"k1": map[interface{}]interface{}{
					"sk1": "sv1",
				},
				"k2": "v2",
			},
			expected: `{"k1":{"sk1":"sv1"},"k2":"v2"}`,
			hasErr:   false,
		},
		{
			name: "invalid key type",
			input: map[interface{}]interface{}{
				12345: "v1",
			},
			expected: "",
			hasErr:   true,
		},
		{
			name: "nested slice",
			input: map[interface{}]interface{}{
				"k1": []interface{}{
					"v1",
					map[interface{}]interface{}{
						"sk1": "sv1",
					},
				},
			},
			expected: `{"k1":["v1",{"sk1":"sv1"}]}`,
			hasErr:   false,
		},
		{
			name: "byte value",
			input: map[interface{}]interface{}{
				"k1": []byte("v1"),
			},
			expected: `{"k1":"v1"}`,
			hasErr:   false,
		},
		{
			name: "nested byte slice",
			input: map[interface{}]interface{}{
				"k1": []interface{}{
					[]byte("v1"),
					map[interface{}]interface{}{
						"sk1": []byte("sv1"),
					},
				},
			},
			expected: `{"k1":["v1",{"sk1":"sv1"}]}`,
			hasErr:   false,
		},
		{
			name: "nested interface",
			input: map[interface{}]interface{}{
				"k1": []interface{}{
					interface{}("v1"),
					map[interface{}]interface{}{
						"sk1": interface{}("sv1"),
					},
				},
			},
			expected: `{"k1":["v1",{"sk1":"sv1"}]}`,
			hasErr:   false,
		},
		{
			name: "deeply nested slice",
			input: map[interface{}]interface{}{
				"k1": []interface{}{
					"v1",
					[]interface{}{
						"v2",
						[]interface{}{
							"v3",
							map[interface{}]interface{}{
								"sk1": "sv1",
							},
						},
					},
				},
			},
			expected: `{"k1":["v1",["v2",["v3",{"sk1":"sv1"}]]]}`,
			hasErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeToJSON(tt.input)
			if tt.hasErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.JSONEq(t, tt.expected, string(got))
			}
		})
	}
}
