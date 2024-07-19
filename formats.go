package main

import (
	"encoding/json"
	"fmt"
)

type Formatter struct {
	Encode func(interface{}) ([]byte, error)
}

var supportFormats = map[string]Formatter{
	"json": {Encode: encodeToJSON},
}

func convertKeysToString(m map[interface{}]interface{}) map[string]interface{} {
	nm := make(map[string]interface{})
	for k, v := range m {
		sk, ok := k.(string)
		if !ok {
			return nil
		}

		switch sv := v.(type) {
		case map[interface{}]interface{}:
			nm[sk] = convertKeysToString(sv)
		case []interface{}:
			nm[sk] = convertNestedSlice(sv)
		case []byte:
			nm[sk] = string(sv)
		default:
			nm[sk] = sv
		}
	}
	return nm
}

func convertNestedSlice(s []interface{}) []interface{} {
	ns := make([]interface{}, len(s))
	for i, v := range s {
		switch sv := v.(type) {
		case map[interface{}]interface{}:
			ns[i] = convertKeysToString(sv)
		case []interface{}:
			ns[i] = convertNestedSlice(sv)
		case []byte:
			ns[i] = string(sv)
		default:
			ns[i] = sv
		}
	}
	return ns
}

func encodeToJSON(v interface{}) ([]byte, error) {
	m, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid type, got %sv", v)
	}
	payload := convertKeysToString(m)
	if payload == nil {
		return nil, fmt.Errorf("invalid key type")
	}
	return json.Marshal(payload)
}
