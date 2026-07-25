package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/LoopBudget/cli/internal/claudehook"
	"github.com/LoopBudget/cli/internal/config"
)

var version = "0.2.0"

var keyOnArgv = regexp.MustCompile(`lb_[A-Za-z0-9]+|LOOPBUDGET_API_KEY=`)

func main() {
	for _, arg := range os.Args[1:] {
		if keyOnArgv.MatchString(arg) {
			fmt.Fprintln(os.Stderr, "[loopbudget] Refusing to run: API key must not appear on the command line. Use `loopbudget-claude-code init`.")
			os.Exit(2)
		}
	}

	cmd := "stop-hook"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	var err error
	switch cmd {
	case "stop-hook", "hook":
		err = cmdStopHook()
	case "init":
		err = cmdInit()
	case "doctor":
		err = cmdDoctor()
	case "version", "--version", "-v":
		fmt.Println(version)
	case "help", "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "[loopbudget] %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Printf(`loopbudget-claude-code v%s

Zero-Node Claude Code Stop hook for LoopBudget.

Usage:
  loopbudget-claude-code stop-hook
  loopbudget-claude-code init
  loopbudget-claude-code doctor
  loopbudget-claude-code version
`, version)
}

func cmdStopHook() error {
	cfg, err := config.LoadClaude()
	if err != nil {
		return err
	}
	return claudehook.RunStopHook(cfg, os.Stdin, os.Stdout, os.Stderr)
}

func cmdInit() error {
	in := bufio.NewReader(os.Stdin)
	defaultURL := "https://loopbudget.com"
	fmt.Printf("LoopBudget URL [%s]: ", defaultURL)
	urlLine, _ := in.ReadString('\n')
	urlLine = strings.TrimSpace(urlLine)
	if urlLine == "" {
		urlLine = defaultURL
	}
	if _, err := config.AssertSafeBaseURL(urlLine); err != nil {
		return err
	}
	fmt.Print("API key (lb_… from /app/setup): ")
	keyLine, _ := in.ReadString('\n')
	keyLine = strings.TrimSpace(keyLine)
	if !strings.HasPrefix(keyLine, "lb_") {
		return fmt.Errorf("API key should look like lb_…")
	}
	path, err := config.WriteCredentials(urlLine, keyLine)
	if err != nil {
		return err
	}
	fmt.Printf("Wrote %s (mode 600)\n", path)
	return nil
}

func cmdDoctor() error {
	cfg, err := config.LoadClaude()
	if err != nil {
		return err
	}
	keyShow := cfg.APIKey
	if len(keyShow) > 6 {
		keyShow = keyShow[:6] + "…"
	}
	cred := cfg.CredentialsPath
	if cred == "" {
		cred = "(env only)"
	}
	fmt.Println("OK")
	fmt.Printf("  url:          %s\n", cfg.BaseURL)
	fmt.Printf("  apiKey:       %s (%d chars)\n", keyShow, len(cfg.APIKey))
	fmt.Printf("  credentials:  %s\n", cred)
	fmt.Printf("  state:        %s\n", cfg.StatePath)
	return nil
}
