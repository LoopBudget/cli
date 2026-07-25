package config_test

import (
	"testing"

	"github.com/LoopBudget/cli/internal/config"
)

func TestAssertSafeBaseURL(t *testing.T) {
	ok, err := config.AssertSafeBaseURL("https://loopbudget.com/foo")
	if err != nil || ok != "https://loopbudget.com" {
		t.Fatalf("got %q %v", ok, err)
	}
	if _, err := config.AssertSafeBaseURL("https://evil.example"); err == nil {
		t.Fatal("expected allowlist error")
	}
	if _, err := config.AssertSafeBaseURL("http://loopbudget.com"); err == nil {
		t.Fatal("expected https required")
	}
	ok, err = config.AssertSafeBaseURL("http://127.0.0.1:3100")
	if err != nil || ok != "http://127.0.0.1:3100" {
		t.Fatalf("got %q %v", ok, err)
	}
}
