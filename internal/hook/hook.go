package hook

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/LoopBudget/cli/internal/config"
	"github.com/LoopBudget/cli/internal/estimate"
)

type stateFile struct {
	Sessions map[string]tokenPair `json:"sessions"`
}

type tokenPair struct {
	TokensIn  int `json:"tokensIn"`
	TokensOut int `json:"tokensOut"`
}

type hookPayload struct {
	SessionID      string `json:"session_id"`
	SessionIDCamel string `json:"sessionId"`
	TranscriptPath string `json:"transcript_path"`
	TranscriptCamel string `json:"transcriptPath"`
}

type ingestBody struct {
	ExternalSessionID  string `json:"externalSessionId"`
	ProfileName        string `json:"profileName"`
	ConnectorKind      string `json:"connectorKind"`
	TokensIn           int    `json:"tokensIn"`
	TokensOut          int    `json:"tokensOut"`
	CostCentsEstimate  int    `json:"costCentsEstimate"`
	EventType          string `json:"eventType"`
}

func costCents(tokensIn, tokensOut int, priceIn, priceOut float64) int {
	usd := (float64(tokensIn)/1e6)*priceIn + (float64(tokensOut)/1e6)*priceOut
	cents := int(math.Round(usd * 100))
	if cents < 1 {
		return 1
	}
	return cents
}

func loadState(path string) stateFile {
	raw, err := os.ReadFile(path)
	if err != nil {
		return stateFile{Sessions: map[string]tokenPair{}}
	}
	var s stateFile
	if json.Unmarshal(raw, &s) != nil || s.Sessions == nil {
		return stateFile{Sessions: map[string]tokenPair{}}
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

// RunStopHook reads Claude Code Stop JSON from stdin and POSTs usage deltas.
func RunStopHook(cfg config.Config, stdin io.Reader, stdout, stderr io.Writer) error {
	raw, _ := io.ReadAll(stdin)
	var hook hookPayload
	_ = json.Unmarshal(raw, &hook)

	sessionID := hook.SessionID
	if sessionID == "" {
		sessionID = hook.SessionIDCamel
	}
	if sessionID == "" {
		sum := sha1.Sum(raw)
		if len(raw) == 0 {
			sum = sha1.Sum([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
		}
		sessionID = hex.EncodeToString(sum[:])[:12]
	}
	transcript := hook.TranscriptPath
	if transcript == "" {
		transcript = hook.TranscriptCamel
	}

	tok := estimate.FromTranscript(transcript)
	st := loadState(cfg.StatePath)
	prev := st.Sessions[sessionID]
	deltaIn := tok.In - prev.TokensIn
	if deltaIn < 0 {
		deltaIn = 0
	}
	deltaOut := tok.Out - prev.TokensOut
	if deltaOut < 0 {
		deltaOut = 0
	}
	st.Sessions[sessionID] = tokenPair{TokensIn: tok.In, TokensOut: tok.Out}
	_ = saveState(cfg.StatePath, st)

	if deltaIn == 0 && deltaOut == 0 {
		_, _ = io.WriteString(stdout, "{}\n")
		return nil
	}

	body := ingestBody{
		ExternalSessionID: sessionID,
		ProfileName:       "claude-code",
		ConnectorKind:     "claude_code",
		TokensIn:          deltaIn,
		TokensOut:         deltaOut,
		CostCentsEstimate: costCents(deltaIn, deltaOut, cfg.PriceIn, cfg.PriceOut),
		EventType:         "sidecar",
	}
	payload, _ := json.Marshal(body)

	req, err := http.NewRequest(http.MethodPost, cfg.BaseURL+"/api/ingest", bytes.NewReader(payload))
	if err != nil {
		fmt.Fprintf(stderr, "[loopbudget] ingest failed: %v\n", err)
		_, _ = io.WriteString(stdout, "{}\n")
		return nil
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+cfg.APIKey)

	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(stderr, "[loopbudget] ingest failed: %v\n", err)
		_, _ = io.WriteString(stdout, "{}\n")
		return nil
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		t, _ := io.ReadAll(io.LimitReader(res.Body, 200))
		fmt.Fprintf(stderr, "[loopbudget] ingest %d: %s\n", res.StatusCode, string(t))
	}
	_, _ = io.WriteString(stdout, "{}\n")
	return nil
}
