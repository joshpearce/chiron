package render

import "strings"

// Math spans are found by scanning rather than by regex. The Python original
// leans on lookbehind/lookahead (`(?<!\\)\$(?!\s)...(?<!\s)\$`) and Go's RE2
// has neither, so the rules are spelled out here instead:
//
//   - `$$...$$` is display math and is matched first, so it is never mistaken
//     for two empty inline spans.
//   - an inline `$` opens math only if it is not backslash-escaped and is not
//     followed by whitespace (so "costs $5 or $10" is prose, not math).
//   - it closes on the next unescaped `$` that is not preceded by whitespace.
//   - a span never spans a blank line; an unclosed `$` is prose, not a runaway
//     equation swallowing the rest of the chapter.
type span struct{ start, end int } // byte offsets, end exclusive

func findMathSpans(s string) []span {
	var out []span
	i := 0
	for i < len(s) {
		if s[i] != '$' {
			i++
			continue
		}
		// Escaped dollar: not math at all.
		if i > 0 && s[i-1] == '\\' {
			i++
			continue
		}
		if strings.HasPrefix(s[i:], "$$") {
			if end := strings.Index(s[i+2:], "$$"); end >= 0 {
				out = append(out, span{i, i + 2 + end + 2})
				i = i + 2 + end + 2
				continue
			}
			i += 2
			continue
		}
		if end := inlineEnd(s, i); end > 0 {
			out = append(out, span{i, end})
			i = end
			continue
		}
		i++
	}
	return out
}

// inlineEnd returns the exclusive end offset of the inline span opening at
// `open`, or 0 if this `$` does not open one.
func inlineEnd(s string, open int) int {
	if open+1 >= len(s) || isSpace(s[open+1]) || s[open+1] == '$' {
		return 0
	}
	for j := open + 1; j < len(s); j++ {
		switch s[j] {
		case '\\':
			j++ // skip the escaped character, including \$
		case '\n':
			// A blank line ends the paragraph; math does not cross it.
			if j+1 < len(s) && (s[j+1] == '\n' || s[j+1] == '\r') {
				return 0
			}
		case '$':
			if isSpace(s[j-1]) {
				return 0 // "$x $" is not a span; treat the opener as prose
			}
			return j + 1
		}
	}
	return 0
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' || b == '\v'
}
