package sources

import (
	"encoding/json"
	"strings"
)

// NotebookToMarkdown flattens a Jupyter notebook: markdown cells as they
// are, code cells as fenced blocks, text outputs kept short, everything
// else (images, HTML widgets) left out.
func NotebookToMarkdown(data []byte) (string, error) {
	var nb struct {
		Metadata struct {
			LanguageInfo struct {
				Name string `json:"name"`
			} `json:"language_info"`
		} `json:"metadata"`
		Cells []struct {
			Type    string          `json:"cell_type"`
			Source  json.RawMessage `json:"source"`
			Outputs []struct {
				Type string          `json:"output_type"`
				Text json.RawMessage `json:"text"`
				Data map[string]json.RawMessage
			} `json:"outputs"`
		} `json:"cells"`
	}
	if err := json.Unmarshal(data, &nb); err != nil {
		return "", err
	}
	lang := nb.Metadata.LanguageInfo.Name
	if lang == "" {
		lang = "python"
	}
	var b strings.Builder
	for _, cell := range nb.Cells {
		src := joined(cell.Source)
		switch cell.Type {
		case "markdown":
			b.WriteString(strings.TrimRight(src, "\n") + "\n\n")
		case "code":
			if strings.TrimSpace(src) == "" {
				continue
			}
			b.WriteString("```" + lang + "\n" + strings.TrimRight(src, "\n") + "\n```\n")
			for _, out := range cell.Outputs {
				var text string
				if out.Text != nil {
					text = joined(out.Text)
				} else if raw, ok := out.Data["text/plain"]; ok {
					text = joined(raw)
				}
				text = strings.TrimSpace(text)
				if text == "" {
					continue
				}
				if lines := strings.Split(text, "\n"); len(lines) > 12 {
					text = strings.Join(lines[:12], "\n") + "\n..."
				}
				b.WriteString("```\n" + text + "\n```\n")
			}
			b.WriteString("\n")
		}
	}
	return tidy(b.String()), nil
}

// joined reads a notebook string field, which is a string or a list of lines.
func joined(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var lines []string
	if json.Unmarshal(raw, &lines) == nil {
		return strings.Join(lines, "")
	}
	return ""
}
