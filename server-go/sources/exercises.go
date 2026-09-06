package sources

import (
	"context"
	"regexp"
	"strconv"
	"strings"
)

// An Exercise is one question a source poses, with its worked answer when
// the source gives one. Prompts and solutions are Markdown as fetched.
type Exercise struct {
	Number   int    `json:"number" yaml:"number"`
	Label    string `json:"label,omitempty" yaml:"label,omitempty"`
	Prompt   string `json:"prompt" yaml:"prompt"`
	Solution string `json:"solution,omitempty" yaml:"solution,omitempty"`
}

var (
	mystDirective = regexp.MustCompile("^```\\{([a-z-]+)\\}\\s*(\\S*)\\s*$")
	boldMarker    = regexp.MustCompile(`^\*\*(?:Exercise|Problem)(?:\s+(\d+))?[:.]?\*\*:?\s*`)
	headingMarker = regexp.MustCompile(`^(#+)\s*(?:Exercise|Problem)\s+(\d+)\b.*$`)
	answerHeading = regexp.MustCompile(`^#+\s*(?:Solution|Answer)s?\b`)
	placeholder   = regexp.MustCompile(`(?i)^#\s*solution goes here\.?$`)
)

// Exercises finds the exercises in a chunk of fetched Markdown. Three
// shapes are read: MyST exercise and solution directives (QuantEcon),
// bold "**Exercise:**" markers with the answer in the cells that follow
// (Downey's notebooks), and "Exercise N" or "Problem N" headings with a
// Solution or Answer heading under them. The first shape the text has wins.
func Exercises(md string) []Exercise {
	lines := strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n")
	if out := mystExercises(lines); len(out) > 0 {
		return out
	}
	if out := boldExercises(lines); len(out) > 0 {
		return out
	}
	return headingExercises(lines)
}

func mystExercises(lines []string) []Exercise {
	var out []Exercise
	solutions := map[string]string{}
	var loose []string
	for i := 0; i < len(lines); i++ {
		m := mystDirective.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		kind, arg := m[1], m[2]
		// The directive's own fence: options, then either the body (plain
		// form) or a closing fence with the body following (start/end form).
		j := i + 1
		label := ""
		for j < len(lines) && strings.HasPrefix(lines[j], ":") {
			if strings.HasPrefix(lines[j], ":label:") {
				label = strings.TrimSpace(strings.TrimPrefix(lines[j], ":label:"))
			}
			j++
		}
		var body []string
		end := j
		switch kind {
		case "exercise", "solution":
			for end < len(lines) && lines[end] != "```" {
				end++
			}
			body = lines[j:end]
		case "exercise-start", "solution-start":
			for end < len(lines) && lines[end] != "```" {
				end++
			}
			closer := "```{" + strings.TrimSuffix(kind, "-start") + "-end}"
			k := end + 1
			for k < len(lines) && lines[k] != closer {
				k++
			}
			body = lines[end+1 : min(k, len(lines))]
			end = k + 1 // the end directive's own closing fence
			for end < len(lines) && lines[end] != "```" {
				end++
			}
		default:
			continue
		}
		text := strings.TrimSpace(strings.Join(body, "\n"))
		switch kind {
		case "exercise", "exercise-start":
			out = append(out, Exercise{Number: len(out) + 1, Label: label, Prompt: text})
		default:
			if arg != "" {
				solutions[arg] = text
			} else {
				loose = append(loose, text)
			}
		}
		i = end
	}
	for i := range out {
		if s, ok := solutions[out[i].Label]; ok && out[i].Label != "" {
			out[i].Solution = s
		} else if i < len(loose) {
			out[i].Solution = loose[i]
		}
	}
	return out
}

func boldExercises(lines []string) []Exercise {
	var starts []int
	for i, l := range lines {
		if boldMarker.MatchString(l) {
			starts = append(starts, i)
		}
	}
	var out []Exercise
	for n, at := range starts {
		end := len(lines)
		if n+1 < len(starts) {
			end = starts[n+1]
		}
		end = cutAtHeading(lines, at+1, end, func(string) bool { return true })
		m := boldMarker.FindStringSubmatch(lines[at])
		first := strings.TrimSpace(lines[at][len(m[0]):])
		span := append([]string{first}, lines[at+1:end]...)
		prompt, solution := splitAtFirstFence(span)
		num := n + 1
		if m[1] != "" {
			num, _ = strconv.Atoi(m[1])
		}
		out = append(out, Exercise{Number: num, Prompt: prompt, Solution: solution})
	}
	return out
}

// cutAtHeading returns the index of the first heading in [from, to) that
// ends satisfies, or to. A "#" inside a code block is a comment, not a
// heading, so fenced lines are skipped.
func cutAtHeading(lines []string, from, to int, ends func(string) bool) int {
	fenced := false
	for k := from; k < to; k++ {
		if strings.HasPrefix(lines[k], "```") {
			fenced = !fenced
			continue
		}
		if !fenced && strings.HasPrefix(lines[k], "#") && ends(lines[k]) {
			return k
		}
	}
	return to
}

// splitAtFirstFence: the prompt is the prose before the first code block;
// what follows is the worked answer, unless it is only the placeholder a
// notebook leaves for the reader.
func splitAtFirstFence(span []string) (string, string) {
	for i, l := range span {
		if strings.HasPrefix(l, "```") {
			prompt := strings.TrimSpace(strings.Join(span[:i], "\n"))
			rest := dropPlaceholders(span[i:])
			return prompt, strings.TrimSpace(strings.Join(rest, "\n"))
		}
	}
	return strings.TrimSpace(strings.Join(span, "\n")), ""
}

func dropPlaceholders(lines []string) []string {
	var out []string
	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "```") {
			j := i + 1
			for j < len(lines) && !strings.HasPrefix(lines[j], "```") {
				j++
			}
			inner := strings.TrimSpace(strings.Join(lines[i+1:min(j, len(lines))], "\n"))
			if inner == "" || placeholder.MatchString(inner) {
				i = j
				continue
			}
			out = append(out, lines[i:min(j+1, len(lines))]...)
			i = j
			continue
		}
		out = append(out, lines[i])
	}
	return out
}

func headingExercises(lines []string) []Exercise {
	var starts []int
	for i, l := range lines {
		if headingMarker.MatchString(l) {
			starts = append(starts, i)
		}
	}
	var out []Exercise
	for n, at := range starts {
		m := headingMarker.FindStringSubmatch(lines[at])
		level := len(m[1])
		end := len(lines)
		if n+1 < len(starts) {
			end = starts[n+1]
		}
		end = cutAtHeading(lines, at+1, end, func(l string) bool {
			return !answerHeading.MatchString(l) && len(l)-len(strings.TrimLeft(l, "#")) <= level
		})
		span := lines[at+1 : end]
		prompt, solution := strings.Join(span, "\n"), ""
		for k, l := range span {
			if answerHeading.MatchString(l) {
				prompt = strings.Join(span[:k], "\n")
				solution = strings.Join(span[k+1:], "\n")
				break
			}
		}
		num, _ := strconv.Atoi(m[2])
		out = append(out, Exercise{Number: num, Prompt: strings.TrimSpace(prompt), Solution: strings.TrimSpace(solution)})
	}
	return out
}

// Exercises fetches a locator and returns its exercises, with answers
// from the recipe's solutions companion when the text itself has none.
func (c *Client) Exercises(ctx context.Context, s *Source, locator string) ([]Exercise, error) {
	ch, err := c.Fetch(ctx, s, locator)
	if err != nil {
		return nil, err
	}
	out := Exercises(ch.Markdown)
	if len(out) == 0 || s.Fetch.Solutions == "" {
		return out, nil
	}
	companion := *s
	companion.Fetch.Path = s.Fetch.Solutions
	companion.Contents = nil
	sol, err := c.Fetch(ctx, &companion, locator)
	if err != nil {
		return out, nil
	}
	answers := Exercises(sol.Markdown)
	byNumber := map[int]string{}
	for _, a := range answers {
		byNumber[a.Number] = a.Solution
	}
	for i := range out {
		if out[i].Solution != "" {
			continue
		}
		if len(answers) == len(out) {
			out[i].Solution = answers[i].Solution
		} else if a, ok := byNumber[out[i].Number]; ok {
			out[i].Solution = a
		}
	}
	return out, nil
}
