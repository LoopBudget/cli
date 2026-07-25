package main

import (
	"fmt"
	"os"
	"regexp"

	"github.com/LoopBudget/cli/internal/config"
	"github.com/LoopBudget/cli/internal/cursor"
)

var version = "0.2.0"

var keyOnArgv = regexp.MustCompile(`lb_[A-Za-z0-9]+|LOOPBUDGET_API_KEY=`)

func main() {
	for _, arg := range os.Args[1:] {
		if keyOnArgv.MatchString(arg) {
			fmt.Fprintln(os.Stderr, "[loopbudget] Refusing to run: API key must not appear on the command line. Use `loopbudget-claude-code init`.")
			os.Exit(2)
		}
		if arg == "version" || arg == "--version" || arg == "-v" {
			fmt.Println(version)
			return
		}
		if arg == "help" || arg == "--help" || arg == "-h" {
			usage()
			return
		}
	}

	cfg, err := config.LoadCursor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[loopbudget] %v\n", err)
		os.Exit(1)
	}
	if err := cursor.Run(cfg, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "[loopbudget] %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Printf(`loopbudget-cursor v%s

Zero-Node Cursor transcript sidecar for LoopBudget.

Uses ~/.loopbudget/credentials (from loopbudget-claude-code init).

Env overrides: LOOPBUDGET_URL, LOOPBUDGET_API_KEY, PROJECT_FILTER, CATCH_UP,
CURSOR_HOME, POLL_MS, PRICE_IN_PER_M, PRICE_OUT_PER_M, STATE_PATH

Usage:
  loopbudget-cursor
  PROJECT_FILTER=my-app loopbudget-cursor
`, version)
}
