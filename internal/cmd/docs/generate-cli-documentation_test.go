package docs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceH1WithH3(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "h1 to h3", in: "# Title\n", want: "### Title\n"},
		{name: "h2 unchanged", in: "## Title\n", want: "## Title\n"},
		{name: "h6 unchanged", in: "###### Title\n", want: "###### Title\n"},
		{name: "no space unchanged", in: "#Title\n", want: "#Title\n"},
		{name: "indent preserved", in: "  # Title\n", want: "  ### Title\n"},
		{name: "not header", in: "Title\n", want: "Title\n"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := replaceH1WithH3(test.in); got != test.want {
				t.Fatalf("replaceH1WithH3(%q) = %q, want %q", test.in, got, test.want)
			}
		})
	}
}

func TestNormalizeSeeAlsoLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "heading", in: "### SEE ALSO\n", want: "### See also\n"},
		{name: "heading with indent", in: "  ### SEE ALSO\n", want: "  ### See also\n"},
		{name: "link label", in: "* [SEE ALSO](link)\n", want: "* [See also](link)\n"},
		{name: "other", in: "### Other\n", want: "### Other\n"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizeSeeAlsoLine(test.in); got != test.want {
				t.Fatalf("normalizeSeeAlsoLine(%q) = %q, want %q", test.in, got, test.want)
			}
		})
	}
}

func TestDropFirstH2HeaderLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		want    string
		dropped bool
	}{
		{name: "h2 header dropped", in: "## Title\n", want: "", dropped: true},
		{name: "h2 with indent dropped", in: "  ## Title\n", want: "", dropped: true},
		{name: "h3 not dropped", in: "### Title\n", want: "### Title\n", dropped: false},
		{name: "non-header", in: "Title\n", want: "Title\n", dropped: false},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, dropped := dropFirstH2HeaderLine(test.in)
			if got != test.want || dropped != test.dropped {
				t.Fatalf("dropFirstH2HeaderLine(%q) = (%q, %v), want (%q, %v)", test.in, got, dropped, test.want, test.dropped)
			}
		})
	}
}

func TestDownscaleMarkdownHeadersInFile_RespectsFences(t *testing.T) {
	t.Parallel()

	content := "" +
		"# A\n" +
		"### SEE ALSO\n" +
		"```bash\n" +
		"# Not\n" +
		"```\n" +
		"## B\n" +
		"~~~\n" +
		"### Not2\n" +
		"~~~\n"

	dir := t.TempDir()
	file := filepath.Join(dir, "test.md")
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	if err := downscaleMarkdownHeadersInFile(file); err != nil {
		t.Fatalf("downscaleMarkdownHeadersInFile: %v", err)
	}

	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read transformed file: %v", err)
	}

	want := "" +
		"### A\n" +
		"### See also\n" +
		"```bash\n" +
		"# Not\n" +
		"```\n" +
		"~~~\n" +
		"### Not2\n" +
		"~~~\n"

	if string(got) != want {
		t.Fatalf("unexpected transformed content:\n%s", string(got))
	}
}
