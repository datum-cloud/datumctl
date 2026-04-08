package ai

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"go.datum.net/datumctl/internal/ai/llm"
)

// PrintPreview writes a human-readable description of a proposed mutating
// action to w. It is called by the agentic loop before prompting for
// confirmation.
func PrintPreview(w io.Writer, call llm.ToolCall) {
	fmt.Fprintf(w, "\n--- Proposed action ---\n")
	fmt.Fprintf(w, "Tool:    %s\n", call.ToolName)
	b, _ := json.MarshalIndent(call.Arguments, "", "  ")
	fmt.Fprintf(w, "Details:\n%s\n", string(b))
	fmt.Fprintf(w, "-----------------------\n")
}

// Ask prompts the user on out and reads a response from in. It returns true
// only if the user explicitly types "y" (case-insensitive). Any other input
// or EOF is treated as "no".
func Ask(in io.Reader, out io.Writer) bool {
	fmt.Fprint(out, "Apply changes? [y/N]: ")
	sc := bufio.NewScanner(in)
	if !sc.Scan() {
		return false
	}
	return strings.ToLower(strings.TrimSpace(sc.Text())) == "y"
}
