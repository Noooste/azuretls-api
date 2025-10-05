package utils

import (
	"encoding/json"
	"fmt"
	"strings"
)

// OrderedMap preserves the order of JSON keys during unmarshaling
type OrderedMap struct {
	Keys   []string
	Values map[string]any
}

// UnmarshalJSON implements custom unmarshaling to preserve key order
func (om *OrderedMap) UnmarshalJSON(data []byte) error {
	// Initialize the map
	om.Values = make(map[string]any)
	om.Keys = []string{}

	// Use json.RawMessage to parse without losing order
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// We need a different approach - parse manually to preserve order
	decoder := json.NewDecoder(strings.NewReader(string(data)))

	// Expect opening brace
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token != json.Delim('{') {
		return fmt.Errorf("expected {, got %v", token)
	}

	// Parse key-value pairs in order
	for decoder.More() {
		// Get key
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := token.(string)
		if !ok {
			return fmt.Errorf("expected string key, got %T", token)
		}

		// Store key in order
		om.Keys = append(om.Keys, key)

		// Get value
		var value any
		if err := decoder.Decode(&value); err != nil {
			return err
		}
		om.Values[key] = value
	}

	// Expect closing brace
	token, err = decoder.Token()
	if err != nil {
		return err
	}
	if token != json.Delim('}') {
		return fmt.Errorf("expected }, got %v", token)
	}

	return nil
}
