package cursor

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/LoopBudget/cli/internal/config"
)

type fileState struct {
	ByteOffset int64 `json:"byteOffset"`
	LineCount  int   `json:"lineCount"`
}

type stateFile struct {
	Files map[string]fileState `json:"files"`
}

type ingestBody struct {
	ExternalSessionID string `json:"externalSessionId"`
	ProfileName       string `json:"profileName"`
	ConnectorKind     string `json:"connectorKind"`
	EventType         string `json:"eventType"`
	TokensIn          int    `json:"tokensIn"`
	TokensOut         int    `json:"tokensOut"`
	CostCentsEstimate int    `json:"costCentsEstimate"`
}

func loadState(path string) stateFile {
	raw, err := os.ReadFile(path)
	if err != nil {
		return stateFile{Files: map[string]fileState{}}
	}
	var s stateFile
	if json.Unmarshal(raw, &s) != nil || s.Files == nil {
		return stateFile{Files: map[string]fileState{}}
	}
	return s
}

func saveState(path string, s stateFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

func walkJSONL(dir, filter string, out *[]string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, ent := range entries {
		full := filepath.Join(dir, ent.Name())
		if ent.IsDir() {
			if ent.Name() == "node_modules" || ent.Name() == ".git" {
				continue
			}
			walkJSONL(full, filter, out)
			continue
		}
		if strings.HasSuffix(ent.Name(), ".jsonl") && strings.Contains(full, "agent-transcripts") {
			if filter != "" && !strings.Contains(full, filter) {
				continue
			}
			*out = append(*out, full)
		}
	}
}

func contentToText(content any) string {
	switch c := content.(type) {
	case string:
		return c
	case []any:
		var parts []string
		for _, block := range c {
			m, ok := block.(map[string]any)
			if !ok {
				continue
			}
			if t, ok := m["text"].(string); ok {
				parts = append(parts, t)
			}
			if typ, _ := m["type"].(string); typ == "tool_use" {
				if name, ok := m["name"].(string); ok {
					parts = append(parts, name)
				}
				if input, ok := m["input"]; ok {
					b, _ := json.Marshal(input)
					parts = append(parts, string(b))
				}
			}
			if typ, _ := m["type"].(string); typ == "tool_result" {
				parts = append(parts, contentToText(m["content"]))
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

func estimateTokens(text string) int {
	if text == "" {
		return 0
	}
	n := (len(text) + 3) / 4
	if n < 1 {
		return 1
	}
	return n
}

func estimateCostCents(tokensIn, tokensOut int, priceIn, priceOut float64) int {
	usd := (float64(tokensIn)/1e6)*priceIn + (float64(tokensOut)/1e6)*priceOut
	cents := int(math.Round(usd * 100))
	if cents < 1 {
		return 1
	}
	return cents
}

func readNewLines(path string, byteOffset int64) (lines []string, nextOffset int64, err error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, byteOffset, err
	}
	size := st.Size()
	if size < byteOffset {
		byteOffset = 0
	}
	if size == byteOffset {
		return nil, byteOffset, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, byteOffset, err
	}
	defer f.Close()
	if _, err := f.Seek(byteOffset, io.SeekStart); err != nil {
		return nil, byteOffset, err
	}
	sc := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 8*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, size, sc.Err()
}

func postIngest(cfg config.Config, body ingestBody) (map[string]any, error) {
	payload, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, cfg.BaseURL+"/api/ingest", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+cfg.APIKey)
	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("ingest %d: %s", res.StatusCode, string(raw))
	}
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

func processFile(cfg config.Config, path string, state *stateFile, log io.Writer) (int, error) {
	prev, seen := state.Files[path]
	st, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	size := st.Size()

	if !seen && !cfg.CatchUp {
		state.Files[path] = fileState{ByteOffset: size, LineCount: 0}
		fmt.Fprintf(log, "[skip-history] %s (%d bytes)\n", strings.TrimSuffix(filepath.Base(path), ".jsonl"), size)
		return 0, nil
	}

	lines, nextOffset, err := readNewLines(path, prev.ByteOffset)
	if err != nil {
		return 0, err
	}
	if len(lines) == 0 {
		state.Files[path] = fileState{ByteOffset: nextOffset, LineCount: prev.LineCount}
		return 0, nil
	}

	sessionID := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	posted := 0
	for _, line := range lines {
		var row map[string]any
		if json.Unmarshal([]byte(line), &row) != nil {
			continue
		}
		role, _ := row["role"].(string)
		if role != "user" && role != "assistant" {
			continue
		}
		var content any
		if msg, ok := row["message"].(map[string]any); ok {
			content = msg["content"]
			if content == nil {
				content = msg
			}
		} else {
			content = row["message"]
		}
		text := contentToText(content)
		tokens := estimateTokens(text)
		tokensIn, tokensOut := 0, 0
		if role == "user" {
			tokensIn = tokens
		} else {
			tokensOut = tokens
		}
		if tokensIn+tokensOut == 0 {
			continue
		}
		cost := estimateCostCents(tokensIn, tokensOut, cfg.PriceIn, cfg.PriceOut)
		result, err := postIngest(cfg, ingestBody{
			ExternalSessionID: sessionID,
			ProfileName:       "cursor",
			ConnectorKind:     "cursor",
			EventType:         "cursor_" + role,
			TokensIn:          tokensIn,
			TokensOut:         tokensOut,
			CostCentsEstimate: cost,
		})
		if err != nil {
			fmt.Fprintf(log, "[error] %v\n", err)
			return posted, err
		}
		posted++
		spend := "?"
		if v, ok := result["spendCents"]; ok {
			spend = fmt.Sprint(v)
		}
		decisions := "-"
		if arr, ok := result["decisions"].([]any); ok && len(arr) > 0 {
			var parts []string
			for _, d := range arr {
				if m, ok := d.(map[string]any); ok {
					if dec, ok := m["decision"].(string); ok {
						parts = append(parts, dec)
					}
				}
			}
			if len(parts) > 0 {
				decisions = strings.Join(parts, ",")
			}
		}
		sid := sessionID
		if len(sid) > 8 {
			sid = sid[:8]
		}
		fmt.Fprintf(log, "[ingest] %s… %s +%d/%d tok ~%d¢ spend=%s¢ decision=%s\n",
			sid, role, tokensIn, tokensOut, cost, spend, decisions)
	}

	state.Files[path] = fileState{ByteOffset: nextOffset, LineCount: prev.LineCount + len(lines)}
	return posted, nil
}

// Run watches Cursor agent transcripts and POSTs usage until the process is killed.
func Run(cfg config.Config, stdout, stderr io.Writer) error {
	projects := filepath.Join(cfg.CursorHome, "projects")
	fmt.Fprintf(stdout, "LoopBudget Cursor sidecar\n")
	fmt.Fprintf(stdout, "  ingest → %s/api/ingest\n", cfg.BaseURL)
	fmt.Fprintf(stdout, "  watch  → %s/**/agent-transcripts/**/*.jsonl\n", projects)
	fmt.Fprintf(stdout, "  state  → %s\n", cfg.StatePath)
	fmt.Fprintf(stdout, "  poll   → %dms  catch-up=%v\n", cfg.PollMS, cfg.CatchUp)
	fmt.Fprintf(stdout, "  price  → $%.0f/M in · $%.0f/M out\n", cfg.PriceIn, cfg.PriceOut)
	if cfg.ProjectFilter != "" {
		fmt.Fprintf(stdout, "  filter → %s\n", cfg.ProjectFilter)
	}

	if res, err := http.Get(cfg.BaseURL + "/"); err != nil {
		fmt.Fprintf(stderr, "[warn] Cannot reach %s — is LoopBudget up?\n", cfg.BaseURL)
	} else {
		res.Body.Close()
		if res.StatusCode >= 500 {
			fmt.Fprintf(stderr, "[warn] LoopBudget responded %d\n", res.StatusCode)
		}
	}

	state := loadState(cfg.StatePath)
	var lastHb time.Time
	ticking := false

	tick := func() {
		if ticking {
			return
		}
		ticking = true
		defer func() { ticking = false }()

		var files []string
		walkJSONL(projects, cfg.ProjectFilter, &files)
		before, _ := json.Marshal(state.Files)
		posted := 0
		for _, f := range files {
			n, err := processFile(cfg, f, &state, stdout)
			posted += n
			if err != nil {
				// keep offsets; retry next poll
				break
			}
		}
		after, _ := json.Marshal(state.Files)
		if !bytes.Equal(before, after) {
			_ = saveState(cfg.StatePath, state)
		}
		if posted == 0 && (lastHb.IsZero() || time.Since(lastHb) > 30*time.Second) {
			fmt.Fprintf(stdout, "[idle] watching %d transcript file(s)\n", len(files))
			lastHb = time.Now()
		}
	}

	tick()
	for range time.Tick(time.Duration(cfg.PollMS) * time.Millisecond) {
		tick()
	}
	return nil
}
