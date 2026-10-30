package cmd

import (
	"bytes"
	"strings"
	"testing"
)

const diffFile = "compute.datumapis.com.v1alpha.Workload.default.my-app"

func TestRewriteDiffLine(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "macOS live header",
			in:   "--- /var/folders/q1/5n3pq8h12wd0bxyz8s2w9j1r0000gn/T/LIVE-1234567890/" + diffFile + "\t2026-10-14 13:51:41",
			want: "--- live/" + diffFile,
		},
		{
			name: "macOS merged header",
			in:   "+++ /var/folders/q1/5n3pq8h12wd0bxyz8s2w9j1r0000gn/T/MERGED-987654321/" + diffFile + "\t2026-10-14 13:51:41",
			want: "+++ merged/" + diffFile,
		},
		{
			name: "linux live header",
			in:   "--- /tmp/LIVE-2893740213/" + diffFile + "\t2026-10-14 13:51:41.123456789 +0000",
			want: "--- live/" + diffFile,
		},
		{
			name: "linux merged header",
			in:   "+++ /tmp/MERGED-1093847562/" + diffFile + "\t2026-10-14 13:51:41.123456789 +0000",
			want: "+++ merged/" + diffFile,
		},
		{
			name: "header without timestamp",
			in:   "--- /tmp/LIVE-1/" + diffFile,
			want: "--- live/" + diffFile,
		},
		{
			name: "temp dir with spaces",
			in:   "+++ /Users/jane doe/My Temp/MERGED-42/" + diffFile + "\t2026-10-14 13:51:41",
			want: "+++ merged/" + diffFile,
		},
		{
			name: "windows path",
			in:   `--- C:\Users\jane\AppData\Local\Temp\LIVE-42\` + diffFile + "\t2026-10-14 13:51:41",
			want: "--- live/" + diffFile,
		},
		{
			name: "diff command echo",
			in:   "diff -u -N /tmp/LIVE-2893740213/" + diffFile + " /tmp/MERGED-1093847562/" + diffFile,
			want: "diff -u -N live/" + diffFile + " merged/" + diffFile,
		},
		{
			name: "diff command echo with extra flags and spaces in path",
			in:   "diff --color -u -N /Users/jane doe/My Temp/LIVE-1/" + diffFile + " /Users/jane doe/My Temp/MERGED-2/" + diffFile,
			want: "diff --color -u -N live/" + diffFile + " merged/" + diffFile,
		},
		{
			name: "only in",
			in:   "Only in /tmp/MERGED-7: " + diffFile,
			want: "Only in merged: " + diffFile,
		},
		{
			name: "colordiff header keeps colour codes",
			in:   "\x1b[0;36m--- /tmp/LIVE-2893740213/" + diffFile + "\t2026-10-14 13:51:41\x1b[0;0m",
			want: "\x1b[0;36m--- live/" + diffFile + "\x1b[0;0m",
		},
		{
			name: "removed body line that mentions a temp path",
			in:   "-  path: /tmp/LIVE-1/" + diffFile,
			want: "-  path: /tmp/LIVE-1/" + diffFile,
		},
		{
			name: "added body line",
			in:   "+    replicas: 3",
			want: "+    replicas: 3",
		},
		{
			name: "context body line",
			in:   "   name: my-app",
			want: "   name: my-app",
		},
		{
			name: "hunk header",
			in:   "@@ -12,7 +12,7 @@",
			want: "@@ -12,7 +12,7 @@",
		},
		{
			name: "yaml document separator in body",
			in:   "---",
			want: "---",
		},
		{
			name: "header for a non-temp file is untouched",
			in:   "--- a/project.yaml\t2026-10-14 13:51:41",
			want: "--- a/project.yaml\t2026-10-14 13:51:41",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rewriteDiffLine(tt.in); got != tt.want {
				t.Errorf("rewriteDiffLine(%q)\n got %q\nwant %q", tt.in, got, tt.want)
			}
		})
	}
}

const sampleDiff = "diff -u -N /tmp/LIVE-2893740213/" + diffFile + " /tmp/MERGED-1093847562/" + diffFile + "\n" +
	"--- /tmp/LIVE-2893740213/" + diffFile + "\t2026-10-14 13:51:41.123456789 +0000\n" +
	"+++ /tmp/MERGED-1093847562/" + diffFile + "\t2026-10-14 13:51:41.123456789 +0000\n" +
	"@@ -5,7 +5,7 @@\n" +
	"   name: my-app\n" +
	"   namespace: default\n" +
	" spec:\n" +
	"-  replicas: 2\n" +
	"+  replicas: 3\n" +
	"   template:\n" +
	"     metadata: {}\n"

const wantDiff = "diff -u -N live/" + diffFile + " merged/" + diffFile + "\n" +
	"--- live/" + diffFile + "\n" +
	"+++ merged/" + diffFile + "\n" +
	"@@ -5,7 +5,7 @@\n" +
	"   name: my-app\n" +
	"   namespace: default\n" +
	" spec:\n" +
	"-  replicas: 2\n" +
	"+  replicas: 3\n" +
	"   template:\n" +
	"     metadata: {}\n"

// TestDiffHeaderRewriterChunking feeds the same diff in different chunk sizes
// to prove the rewriter does not depend on where the child process's writes
// happen to split lines.
func TestDiffHeaderRewriterChunking(t *testing.T) {
	for _, size := range []int{1, 2, 7, 64, len(sampleDiff)} {
		var out bytes.Buffer
		w := newDiffHeaderRewriter(&out)
		for i := 0; i < len(sampleDiff); i += size {
			end := i + size
			if end > len(sampleDiff) {
				end = len(sampleDiff)
			}
			chunk := []byte(sampleDiff[i:end])
			n, err := w.Write(chunk)
			if err != nil {
				t.Fatalf("chunk %d: Write: %v", size, err)
			}
			if n != len(chunk) {
				t.Fatalf("chunk %d: Write returned %d, want %d", size, n, len(chunk))
			}
		}
		if err := w.Flush(); err != nil {
			t.Fatalf("chunk %d: Flush: %v", size, err)
		}
		if out.String() != wantDiff {
			t.Errorf("chunk size %d:\n got %q\nwant %q", size, out.String(), wantDiff)
		}
	}
}

// TestDiffHeaderRewriterPartialLines checks that body text is forwarded as
// soon as it is clearly not a header, while a possible header is held until
// its newline arrives.
func TestDiffHeaderRewriterPartialLines(t *testing.T) {
	var out bytes.Buffer
	w := newDiffHeaderRewriter(&out)

	if _, err := w.Write([]byte("+  replicas:")); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "+  replicas:" {
		t.Fatalf("body partial not forwarded: got %q", got)
	}
	if _, err := w.Write([]byte(" 3\n--- /tmp/LIVE-1/")); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "+  replicas: 3\n" {
		t.Fatalf("header partial should be held: got %q", got)
	}
	if _, err := w.Write([]byte(diffFile + "\t2026-10-14 13:51:41\n")); err != nil {
		t.Fatal(err)
	}
	want := "+  replicas: 3\n--- live/" + diffFile + "\n"
	if got := out.String(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	// A held header-looking line with no newline is emitted unchanged on Flush.
	if _, err := w.Write([]byte("--")); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); !strings.HasSuffix(got, "\n--") {
		t.Fatalf("Flush should emit the held partial line: got %q", got)
	}
}
