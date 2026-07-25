package config

import "testing"

func TestAssertSafeBaseURL(t *testing.T) {
	ok, err := AssertSafeBaseURL("https://loopbudget.com/foo")
	if err != nil || ok != "https://loopbudget.com" {
		t.Fatalf("got %q %v", ok, err)
	}
	if _, err := AssertSafeBaseURL("https://evil.example"); err == nil {
		t.Fatal("expected allowlist error")
	}
	ok, err = AssertSafeBaseURL("http://127.0.0.1:3100")
	if err != nil || ok != "http://127.0.0.1:3100" {
		t.Fatalf("got %q %v", ok, err)
	}
}
