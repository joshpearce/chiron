package generate

import (
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// The author writes prose into YAML as plain scalars, and prose breaks
// them: a value that opens with a quotation mark reads as a quoted
// scalar, a colon followed by a space reads as a nested key. Thirteen of
// eighteen misconception files came back that way from the first book
// built from a PDF. RepairYAMLProse rewrites every multi-line value, and
// every single-line value a plain scalar cannot hold, as a block scalar,
// which takes prose as it is. Anything that already parses is returned
// unchanged.
func RepairYAMLProse(text string) string {
	var probe any
	if yaml.Unmarshal([]byte(text), &probe) == nil {
		return text
	}
	lines := strings.Split(text, "\n")
	var out []string
	for i := 0; i < len(lines); i++ {
		m := keyValue.FindStringSubmatch(lines[i])
		if m == nil {
			out = append(out, lines[i])
			continue
		}
		indent, key, value := m[1], m[2], m[3]
		// The value's continuation: every following line indented deeper
		// than the key, up to a blank line or a key at the key's own
		// level. A key with an inline value is never followed by a list,
		// so a deeper line opening with a dash is prose too.
		j := i + 1
		for ; j < len(lines); j++ {
			l := lines[j]
			if strings.TrimSpace(l) == "" {
				break
			}
			deeper := len(l)-len(strings.TrimLeft(l, " ")) > len(indent)
			if !deeper || keyValue.MatchString(l) && len(keyValue.FindStringSubmatch(l)[1]) <= len(indent) {
				break
			}
		}
		parts := []string{value}
		for _, l := range lines[i+1 : j] {
			parts = append(parts, strings.TrimSpace(l))
		}
		if len(parts) == 1 && plainSafe(value) {
			out = append(out, lines[i])
			continue
		}
		out = append(out, indent+key+": |")
		for _, p := range parts {
			out = append(out, indent+"  "+strings.ReplaceAll(p, `\"`, `"`))
		}
		i = j - 1
	}
	return strings.Join(out, "\n")
}

// A key with an inline value: `  name: The value`. A value that is
// itself a block indicator, a flow collection, or empty is not prose.
var keyValue = regexp.MustCompile(`^( *)([A-Za-z_][A-Za-z0-9_]*): ([^|>\[{\s].*)$`)

// plainSafe: a single-line value a plain scalar can hold as written.
func plainSafe(v string) bool {
	if strings.HasPrefix(v, `"`) || strings.HasPrefix(v, `'`) {
		var probe any
		return yaml.Unmarshal([]byte("k: "+v), &probe) == nil
	}
	return !strings.Contains(v, ": ") && !strings.Contains(v, " #") && !strings.HasSuffix(v, ":")
}

// The author names a unit's own misconceptions u3-m1 in one file and
// U3-M1 in the next; the ids are case-sensitive, and a bank citing the
// other spelling fails lint. NormaliseLocalIDs lowercases every unit-local
// id wherever it appears.
func NormaliseLocalIDs(text string) string {
	return localID.ReplaceAllStringFunc(text, strings.ToLower)
}

var localID = regexp.MustCompile(`\b[Uu][0-9]+-[Mm][0-9]+\b`)
