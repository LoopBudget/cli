package estimate

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Tokens struct {
	In  int
	Out int
}

// FromTranscript sums message.usage when present, else chars/4 on assistant text.
// Never ships transcript content — only derived token counts.
func FromTranscript(path string) Tokens {
	if path == "" {
		return Tokens{}
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return Tokens{}
	}
	f, err := os.Open(resolved)
	if err != nil {
		return Tokens{}
	}
	defer f.Close()

	var tokensIn, tokensOut, textOutChars int
	sc := bufio.NewScanner(f)
	// large transcript lines
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 8*1024*1024)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}
		if usage := usageObj(row); usage != nil {
			tokensIn += num(usage, "input_tokens", "inputTokens")
			tokensOut += num(usage, "output_tokens", "outputTokens")
			continue
		}
		typ, _ := row["type"].(string)
		role, _ := row["role"].(string)
		if typ == "assistant" || role == "assistant" {
			textOutChars += contentLen(row)
		}
	}
	if tokensIn == 0 && tokensOut == 0 && textOutChars > 0 {
		tokensOut = textOutChars / 4
		if tokensOut < 1 {
			tokensOut = 1
		}
	}
	return Tokens{In: tokensIn, Out: tokensOut}
}

func usageObj(row map[string]any) map[string]any {
	if msg, ok := row["message"].(map[string]any); ok {
		if u, ok := msg["usage"].(map[string]any); ok {
			return u
		}
	}
	if u, ok := row["usage"].(map[string]any); ok {
		return u
	}
	return nil
}

func num(m map[string]any, keys ...string) int {
	for _, k := range keys {
		switch v := m[k].(type) {
		case float64:
			return int(v)
		case json.Number:
			n, _ := v.Int64()
			return int(n)
		}
	}
	return 0
}

func contentLen(row map[string]any) int {
	var content any
	if msg, ok := row["message"].(map[string]any); ok {
		content = msg["content"]
	} else {
		content = row["content"]
	}
	switch c := content.(type) {
	case string:
		return len(c)
	case []any:
		n := 0
		for _, part := range c {
			switch p := part.(type) {
			case string:
				n += len(p)
			case map[string]any:
				if t, ok := p["text"].(string); ok {
					n += len(t)
				}
			}
		}
		return n
	default:
		return 0
	}
}
