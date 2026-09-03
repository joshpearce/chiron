package llm

// strictSchema returns a copy of a JSON schema with additionalProperties set
// to false on every object, which Anthropic's structured output requires
// and which no role should have to remember. Nested objects, array items,
// and alternatives are walked too.
func strictSchema(schema map[string]any) map[string]any {
	out := make(map[string]any, len(schema)+1)
	for k, v := range schema {
		switch k {
		case "properties", "definitions", "$defs":
			if props, ok := v.(map[string]any); ok {
				copied := make(map[string]any, len(props))
				for name, sub := range props {
					copied[name] = strictValue(sub)
				}
				out[k] = copied
				continue
			}
		case "items", "not":
			out[k] = strictValue(v)
			continue
		case "anyOf", "oneOf", "allOf":
			if list, ok := v.([]any); ok {
				copied := make([]any, len(list))
				for i, sub := range list {
					copied[i] = strictValue(sub)
				}
				out[k] = copied
				continue
			}
		}
		out[k] = v
	}
	if t, _ := out["type"].(string); t == "object" {
		if _, set := out["additionalProperties"]; !set {
			out["additionalProperties"] = false
		}
	}
	return out
}

func strictValue(v any) any {
	if m, ok := v.(map[string]any); ok {
		return strictSchema(m)
	}
	return v
}
