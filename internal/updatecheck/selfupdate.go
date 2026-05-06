package updatecheck

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// EnvUpdated is set on the re-executed process after a successful
	// auto-update so the new run skips the auto-update path entirely. This
	// guards against unbounded loops if a freshly installed binary still
	// reports as outdated for any reason.
	EnvUpdated = "DATUMCTL_UPDATED"

	// downloadTimeout bounds the binary download. Archives are typically a
	// few MB, so 60s leaves plenty of headroom on slow links.
	downloadTimeout = 60 * time.Second

	releasesDownloadBase = "https://github.com/datum-cloud/datumctl/releases/download"
)

// SelfUpdate downloads the release archive for the given version and replaces
// the currently running executable. The version must be in vX.Y.Z form.
func SelfUpdate(ctx context.Context, version string) error {
	if !strings.HasPrefix(version, "v") {
		return fmt.Errorf("invalid version %q", version)
	}

	url, err := downloadURL(version)
	if err != nil {
		return err
	}

	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate current binary: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(binPath)
	if err == nil {
		binPath = resolved
	}

	client := &http.Client{Timeout: downloadTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "datumctl-self-update")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}

	tmp, err := os.CreateTemp(filepath.Dir(binPath), ".datumctl-update-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	binName := "datumctl"
	if runtime.GOOS == "windows" {
		binName = "datumctl.exe"
	}

	if runtime.GOOS == "windows" {
		if err := extractFromZip(resp.Body, binName, tmp); err != nil {
			tmp.Close()
			return err
		}
	} else {
		if err := extractFromTarGz(resp.Body, binName, tmp); err != nil {
			tmp.Close()
			return err
		}
	}

	if err := tmp.Chmod(0o755); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := replaceBinary(binPath, tmpPath); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func downloadURL(version string) (string, error) {
	name, err := archiveName()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s/%s", releasesDownloadBase, version, name), nil
}

func archiveName() (string, error) {
	var osName string
	switch runtime.GOOS {
	case "linux":
		osName = "Linux"
	case "darwin":
		osName = "Darwin"
	case "windows":
		osName = "Windows"
	default:
		return "", fmt.Errorf("unsupported os %q for auto-update", runtime.GOOS)
	}

	var arch string
	switch runtime.GOARCH {
	case "amd64":
		arch = "x86_64"
	case "386":
		arch = "i386"
	case "arm64":
		arch = "arm64"
	default:
		return "", fmt.Errorf("unsupported arch %q for auto-update", runtime.GOARCH)
	}

	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("datumctl_%s_%s.%s", osName, arch, ext), nil
}

func extractFromTarGz(r io.Reader, binName string, dst io.Writer) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gunzip archive: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("binary %q not found in archive", binName)
		}
		if err != nil {
			return fmt.Errorf("read tar archive: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(hdr.Name) != binName {
			continue
		}
		if _, err := io.Copy(dst, tr); err != nil {
			return fmt.Errorf("extract %s: %w", binName, err)
		}
		return nil
	}
}

func extractFromZip(r io.Reader, binName string, dst io.Writer) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read zip body: %w", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("open zip archive: %w", err)
	}
	for _, f := range zr.File {
		if filepath.Base(f.Name) != binName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open %s in zip: %w", binName, err)
		}
		_, copyErr := io.Copy(dst, rc)
		rc.Close()
		if copyErr != nil {
			return fmt.Errorf("extract %s: %w", binName, copyErr)
		}
		return nil
	}
	return fmt.Errorf("binary %q not found in archive", binName)
}

// ReExec replaces the current process with the freshly installed binary,
// passing through the original argv. On unix-like systems this uses
// syscall.Exec so the original PID is preserved. On Windows (which has no
// exec(2)), it spawns a child process, mirrors the standard streams, and
// exits with the child's exit code. Sets EnvUpdated in the new environment
// so the auto-update path is skipped on the next run.
func ReExec() error {
	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate binary: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(binPath); err == nil {
		binPath = resolved
	}
	env := append(os.Environ(), EnvUpdated+"=1")
	return reExec(binPath, os.Args, env)
}

// AlreadyUpdated reports whether this process was started as the post-update
// re-exec, in which case auto-update logic must be skipped.
func AlreadyUpdated() bool {
	return os.Getenv(EnvUpdated) != ""
}

// replaceBinary atomically replaces dst with src. On Windows the running
// executable cannot be overwritten, so the current binary is moved aside to
// dst+".old"; the OS cleans it up on next boot or the user can remove it.
func replaceBinary(dst, src string) error {
	if runtime.GOOS == "windows" {
		old := dst + ".old"
		_ = os.Remove(old)
		if err := os.Rename(dst, old); err != nil {
			return fmt.Errorf("move current binary aside: %w", err)
		}
		if err := os.Rename(src, dst); err != nil {
			_ = os.Rename(old, dst)
			return fmt.Errorf("install new binary: %w", err)
		}
		return nil
	}
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("install new binary: %w", err)
	}
	return nil
}
