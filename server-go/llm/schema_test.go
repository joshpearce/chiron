package llm

import (
	"encoding/json"
	"testing"
)

func TestStrictSchemaClosesEveryObject(t *testing.T) {
	in := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{"type": "string"},
			"sections": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":       "object",
					"properties": map[string]any{"heading": map[string]any{"type": "string"}},
				},
			},
			"choice": map[string]any{"anyOf": []any{
				map[string]any{"type": "object", "properties": map[string]any{}},
				map[string]any{"type": "string"},
			}},
		},
		"required": []string{"title"},
	}
	before, _ := json.Marshal(in)
	out := strictSchema(in)
	after, _ := json.Marshal(in)
	if string(before) != string(after) {
		t.Fatal("the input schema was modified")
	}
	if out["additionalProperties"] != false {
		t.Fatalf("root not closed: %v", out)
	}
	props := out["properties"].(map[string]any)
	items := props["sections"].(map[string]any)["items"].(map[string]any)
	if items["additionalProperties"] != false {
		t.Fatalf("array item object not closed: %v", items)
	}
	alt := props["choice"].(map[string]any)["anyOf"].([]any)
	if alt[0].(map[string]any)["additionalProperties"] != false {
		t.Fatalf("anyOf object not closed: %v", alt[0])
	}
	if _, has := alt[1].(map[string]any)["additionalProperties"]; has {
		t.Fatal("a string schema was given additionalProperties")
	}
	if _, has := props["title"].(map[string]any)["additionalProperties"]; has {
		t.Fatal("a string property was given additionalProperties")
	}
}

func TestStrictSchemaKeepsAnExplicitChoice(t *testing.T) {
	out := strictSchema(map[string]any{"type": "object", "additionalProperties": true})
	if out["additionalProperties"] != true {
		t.Fatal("an explicit additionalProperties was overridden")
	}
}
