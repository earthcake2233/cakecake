// Package toolkit provides JSON Schema helpers for tool definitions.
package toolkit

// Schema helpers for building JSON Schema objects without raw JSON.

// S creates a string property schema.
func S(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"description": description,
	}
}

// I creates an integer property schema.
func I(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "integer",
		"description": description,
	}
}

// Object builds a JSON Schema object with the given properties.
// props is a map of property name -> schema.
// required lists the property names that must be provided.
func Object(props map[string]interface{}, required ...string) map[string]interface{} {
	obj := map[string]interface{}{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		obj["required"] = required
	}
	return obj
}