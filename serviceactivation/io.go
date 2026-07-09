package serviceactivation

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// IOStreams carries the three streams the flow talks to. Stdout is reserved as
// a clean data channel (-o json|yaml); every prompt, spinner, and status line
// goes to stderr.
type IOStreams struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer

	// interactive overrides TTY auto-detection when non-nil (see WithInteractive).
	interactive *bool
}

// StdIOStreams returns IOStreams bound to the process standard streams.
func StdIOStreams() IOStreams {
	return IOStreams{In: os.Stdin, Out: os.Stdout, Err: os.Stderr}
}

// WithInteractive returns a copy of s with TTY auto-detection overridden: true
// forces the interactive (prompting) path, false forces the non-interactive one.
// Use it to honor a --yes/--no-input flag, or in tests.
func (s IOStreams) WithInteractive(v bool) IOStreams {
	s.interactive = &v
	return s
}

// IsInputTTY reports whether stdin is an interactive terminal. TTY detection
// keys on stdin (not stdout) so a piped `... -o json | jq` with a terminal
// stdin is still treated as interactive, matching how a human would run it.
func (s IOStreams) IsInputTTY() bool {
	if s.interactive != nil {
		return *s.interactive
	}
	f, ok := s.In.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// promptYesNo asks a yes/no question on stderr and reads the answer from stdin.
// The default is No; only "y"/"yes" (case-insensitive) confirms.
func (s IOStreams) promptYesNo(question string) (bool, error) {
	fmt.Fprintf(s.Err, "%s [y/N]: ", question)
	scanner := bufio.NewScanner(s.In)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, err
		}
		return false, nil
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "y" || answer == "yes", nil
}

// spinnerFrames drives the progress indicator.
var spinnerFrames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

// progress renders a bounded-wait progress line while fn runs. On a TTY it
// animates a spinner with an elapsed-seconds counter, rewriting a single line;
// otherwise it emits a periodic keepalive so non-interactive callers see the
// process is alive without control characters polluting the stream.
//
// label is the message shown alongside the elapsed time. progress returns
// whatever fn returns; fn should honor ctx for cancellation.
func (s IOStreams) progress(ctx context.Context, tty bool, label string, fn func(context.Context) error) error {
	done := make(chan error, 1)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() { done <- fn(runCtx) }()

	start := time.Now()
	if tty {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		frame := 0
		for {
			select {
			case err := <-done:
				fmt.Fprint(s.Err, "\r\033[K") // clear the spinner line
				return err
			case <-ticker.C:
				elapsed := int(time.Since(start).Seconds())
				fmt.Fprintf(s.Err, "\r\033[K%c %s (%ds)", spinnerFrames[frame%len(spinnerFrames)], label, elapsed)
				frame++
			}
		}
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			return err
		case <-ticker.C:
			elapsed := int(time.Since(start).Seconds())
			fmt.Fprintf(s.Err, "%s (%ds)\n", label, elapsed)
		}
	}
}
