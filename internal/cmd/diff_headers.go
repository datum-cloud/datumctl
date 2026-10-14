package cmd

import (
	"bytes"
	"io"
	"regexp"
	"strings"
	"sync"
)

// The diff command writes the live and would-be-applied objects into two
// temporary directories named LIVE-<random> and MERGED-<random>, one file per
// resource named <group>.<version>.<Kind>.<namespace>.<name>, and runs the
// diff tool over the two directories. The tool then names those temp paths in
// its output, which is machine-specific noise that changes on every run.
//
// diffHeaderRewriter sits in front of the diff command's stdout and rewrites
// only the lines that name the compared files, so that
//
//	/var/folders/xx/T/LIVE-123456/compute.datumapis.com.v1alpha.Workload.default.my-app
//
// is shown as
//
//	live/compute.datumapis.com.v1alpha.Workload.default.my-app
//
// and MERGED-<random> as merged/. Timestamps of the temp files are dropped
// from ---/+++ headers for the same reason. Diff body lines are passed
// through untouched.

const (
	diffLiveLabel   = "live"
	diffMergedLabel = "merged"
)

var (
	// ansi matches an SGR colour sequence, as emitted by tools like colordiff
	// around header lines.
	ansi = `(?:\x1b\[[0-9;]*m)*`

	// diffFileHeaderRe matches the "--- <path>\t<timestamp>" and
	// "+++ <path>\t<timestamp>" lines of a unified diff. The path is taken
	// greedily up to the last LIVE-/MERGED- temp directory, so it may contain
	// spaces. Group 1: leading colour codes; 2: marker; 3: temp dir kind;
	// 4: file name; 5: trailing colour codes.
	diffFileHeaderRe = regexp.MustCompile(
		`^(` + ansi + `)(---|\+\+\+) .*[/\\](LIVE|MERGED)-[0-9]+[/\\]([^\s\x1b]+)(?:\t[^\x1b]*)?(` + ansi + `)$`)

	// diffCommandRe matches the "diff <flags> <from> <to>" line that diff
	// prints before each file pair when comparing directories. Group 1:
	// flags (including trailing space); 2: live file name; 3: merged file
	// name.
	diffCommandRe = regexp.MustCompile(
		`^diff ((?:-\S+ )*).*[/\\]LIVE-[0-9]+[/\\](\S+) .*[/\\]MERGED-[0-9]+[/\\](\S+)$`)

	// ansiPrefixRe matches the colour codes at the start of a line.
	ansiPrefixRe = regexp.MustCompile(`^` + ansi)

	// diffOnlyInRe matches "Only in <dir>: <file>", printed by diff tools
	// that are not run with -N when a resource exists on one side only.
	diffOnlyInRe = regexp.MustCompile(`^Only in .*[/\\](LIVE|MERGED)-[0-9]+: `)
)

// diffHeaderPrefixes are the starts of lines the rewriter may change. A
// partial line that cannot grow into one of these is forwarded immediately
// rather than held until its newline arrives.
var diffHeaderPrefixes = []string{"--- ", "+++ ", "diff ", "Only in "}

// rewriteDiffLine returns line with temp directory paths replaced by the
// live/ and merged/ labels. Lines that are not file headers are returned
// unchanged.
func rewriteDiffLine(line string) string {
	if m := diffFileHeaderRe.FindStringSubmatch(line); m != nil {
		return m[1] + m[2] + " " + diffLabel(m[3]) + "/" + m[4] + m[5]
	}
	if m := diffCommandRe.FindStringSubmatch(line); m != nil {
		return "diff " + m[1] + diffLiveLabel + "/" + m[2] + " " + diffMergedLabel + "/" + m[3]
	}
	if m := diffOnlyInRe.FindStringSubmatch(line); m != nil {
		return "Only in " + diffLabel(m[1]) + ": " + line[len(m[0]):]
	}
	return line
}

func diffLabel(tmpDirKind string) string {
	if tmpDirKind == "LIVE" {
		return diffLiveLabel
	}
	return diffMergedLabel
}

// mayBeDiffHeader reports whether a partial line could still turn into a
// header line once the rest of it arrives.
func mayBeDiffHeader(partial string) bool {
	// Colour codes may precede the marker; while the partial line is only
	// escape sequences (or an unfinished one) we cannot tell yet.
	trimmed := ansiPrefixRe.ReplaceAllString(partial, "")
	if trimmed == "" || strings.HasPrefix(trimmed, "\x1b") {
		return true
	}
	for _, p := range diffHeaderPrefixes {
		if strings.HasPrefix(trimmed, p) || strings.HasPrefix(p, trimmed) {
			return true
		}
	}
	return false
}

// diffHeaderRewriter is a line-buffered io.Writer that applies
// rewriteDiffLine to every complete line before forwarding it. It is safe
// for use as the stdout of a child process, whose output arrives in
// arbitrary chunks.
//
// The diff command exits the process directly when differences are found, so
// output cannot rely on a final Flush. Partial lines that cannot be headers
// are therefore forwarded as soon as they arrive; only a partial line that
// still looks like a header is held until its newline. Diff tools always
// terminate their output with a newline, so nothing is lost in practice.
type diffHeaderRewriter struct {
	mu          sync.Mutex
	out         io.Writer
	pending     []byte // start of a line that may still become a header
	passthrough bool   // the current line is known not to be a header
}

func newDiffHeaderRewriter(out io.Writer) *diffHeaderRewriter {
	return &diffHeaderRewriter{out: out}
}

// Write implements io.Writer. It reports len(p) on success as the bytes are
// either forwarded or buffered.
func (w *diffHeaderRewriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	n := len(p)
	for len(p) > 0 {
		nl := bytes.IndexByte(p, '\n')
		if nl < 0 {
			if w.passthrough {
				if _, err := w.out.Write(p); err != nil {
					return 0, err
				}
				return n, nil
			}
			w.pending = append(w.pending, p...)
			if !mayBeDiffHeader(string(w.pending)) {
				w.passthrough = true
				if err := w.flushPending(); err != nil {
					return 0, err
				}
			}
			return n, nil
		}

		chunk := p[:nl+1]
		p = p[nl+1:]
		var line []byte
		if w.passthrough {
			line = chunk
		} else {
			line = append(w.pending, chunk...)
			rewritten := rewriteDiffLine(strings.TrimSuffix(string(line), "\n"))
			line = append([]byte(rewritten), '\n')
		}
		w.pending = w.pending[:0]
		w.passthrough = false
		if _, err := w.out.Write(line); err != nil {
			return 0, err
		}
	}
	return n, nil
}

// Flush forwards any held partial line unchanged. Call it once the diff
// command has returned so a final line without a trailing newline is not
// lost.
func (w *diffHeaderRewriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.flushPending()
}

func (w *diffHeaderRewriter) flushPending() error {
	if len(w.pending) == 0 {
		return nil
	}
	_, err := w.out.Write(w.pending)
	w.pending = w.pending[:0]
	return err
}
