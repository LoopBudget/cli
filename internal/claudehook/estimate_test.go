package claudehook_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LoopBudget/cli/internal/claudehook"
)

func TestFromTranscriptUsage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.jsonl")
	content := `{"message":{"usage":{"input_tokens":100,"output_tokens":50}}}
{"type":"assistant","message":{"content":"hello world"}}
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	tok := claudehook.FromTranscript(path)
	if tok.In != 100 || tok.Out != 50 {
		t.Fatalf("got %+v", tok)
	}
}

func TestFromTranscriptCharFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.jsonl")
	// 40 chars → 10 tokens
	content := `{"type":"assistant","message":{"content":"0123456789012345678901234567890123456789"}}
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	tok := claudehook.FromTranscript(path)
	if tok.In != 0 || tok.Out != 10 {
		t.Fatalf("got %+v want out=10", tok)
	}
}
