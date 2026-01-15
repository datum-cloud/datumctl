// Command update-nix-hash automatically updates the vendorHash in flake.nix
// after Go dependencies have changed.
//
// Usage:
//
//	task update-nix-hash
//
// Or directly:
//
//	go run bin/update-nix-hash.go
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	flakeFile = "flake.nix"
	fakeHash  = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
)

func main() {
	log.SetFlags(0)

	// Check if flake.nix exists
	if _, err := os.Stat(flakeFile); err != nil {
		log.Fatalf("Error: %s not found. Run this from the project root.", flakeFile)
	}

	// Read the current flake.nix
	content, err := os.ReadFile(flakeFile)
	if err != nil {
		log.Fatalf("Error reading %s: %v", flakeFile, err)
	}

	// Find current vendorHash
	hashRe := regexp.MustCompile(`vendorHash = "(sha256-[A-Za-z0-9+/=]+)";`)
	matches := hashRe.FindSubmatch(content)
	if matches == nil {
		log.Fatalf("Error: Could not find vendorHash in %s", flakeFile)
	}
	currentHash := string(matches[1])

	log.Printf("Current vendorHash: %s", currentHash)
	log.Println("Updating vendorHash in flake.nix...")

	// Replace with fake hash to trigger Nix error
	updatedContent := hashRe.ReplaceAll(content, []byte(fmt.Sprintf(`vendorHash = "%s";`, fakeHash)))

	// Write temporary flake.nix
	if err := os.WriteFile(flakeFile, updatedContent, 0644); err != nil {
		log.Fatalf("Error writing %s: %v", flakeFile, err)
	}

	// Run nix build to get the correct hash
	log.Println("Running nix build to determine correct hash...")
	cmd := exec.Command("nix", "build", ".#default")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = os.Stdout

	// We expect this to fail
	_ = cmd.Run()

	// Extract the correct hash from stderr
	output := stderr.String()
	correctHash := extractHash(output)

	if correctHash == "" {
		// Restore original content
		if err := os.WriteFile(flakeFile, content, 0644); err != nil {
			log.Printf("Warning: Failed to restore original flake.nix: %v", err)
		}
		log.Fatalf("Error: Could not extract hash from nix build output.\n%s", output)
	}

	log.Printf("Extracted hash: %s", correctHash)

	// If hash hasn't changed, we're done
	if correctHash == currentHash {
		// Restore original
		if err := os.WriteFile(flakeFile, content, 0644); err != nil {
			log.Fatalf("Error restoring %s: %v", flakeFile, err)
		}
		log.Println("✓ vendorHash is already up to date")
		return
	}

	// Update with correct hash
	finalContent := bytes.ReplaceAll(content, []byte(currentHash), []byte(correctHash))
	if err := os.WriteFile(flakeFile, finalContent, 0644); err != nil {
		log.Fatalf("Error writing %s: %v", flakeFile, err)
	}

	log.Printf("Updated vendorHash: %s -> %s", currentHash, correctHash)

	// Verify build works
	log.Println("Verifying build with new hash...")
	cmd = exec.Command("nix", "build", ".#default")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatalf("Error: Build failed with new hash: %v", err)
	}

	log.Println("✓ Build successful with new hash")
}

// extractHash parses the nix build error output to find the correct hash
func extractHash(output string) string {
	// Look for "got:    sha256-..."
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "got:") {
			// Extract the hash after "got:"
			parts := strings.Fields(line)
			if len(parts) >= 2 && strings.HasPrefix(parts[1], "sha256-") {
				return parts[1]
			}
		}
	}
	return ""
}
