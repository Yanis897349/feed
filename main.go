package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	appName        = "investinglive-news"
	defaultFeedURL = "https://investinglive.com/feed/news/"
	maxFeedSize    = 5 << 20
)

// version is replaced at build time for releases.
var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := &http.Client{Timeout: 15 * time.Second}
	var err error
	if len(os.Args) == 1 && interactiveTerminal() {
		err = runTUI(ctx, os.Stdin, os.Stdout, client, time.Now, tuiOptionsFromEnv())
	} else {
		err = run(ctx, os.Args[1:], os.Stdout, os.Stderr, client, time.Now)
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stderr, "cancelled")
			os.Exit(130)
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func interactiveTerminal() bool {
	stdin, stdinErr := os.Stdin.Stat()
	stdout, stdoutErr := os.Stdout.Stat()
	return stdinErr == nil && stdoutErr == nil &&
		stdin.Mode()&os.ModeCharDevice != 0 && stdout.Mode()&os.ModeCharDevice != 0
}
