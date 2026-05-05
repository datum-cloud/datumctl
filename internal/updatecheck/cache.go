package updatecheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type cacheFile struct {
	CheckedAt     time.Time `json:"checkedAt"`
	LatestVersion string    `json:"latestVersion"`
}

func defaultCachePath() string {
	dir, err := os.UserCacheDir()
	if err != nil || dir == "" {
		return ""
	}
	return filepath.Join(dir, "datumctl", "update-check.json")
}

func (c *Checker) cachedLatest() (string, bool) {
	if c.cachePath == "" {
		return "", false
	}
	data, err := os.ReadFile(c.cachePath)
	if err != nil {
		return "", false
	}
	var f cacheFile
	if err := json.Unmarshal(data, &f); err != nil {
		return "", false
	}
	if c.now().Sub(f.CheckedAt) >= CacheTTL {
		return "", false
	}
	if f.LatestVersion == "" {
		return "", false
	}
	return f.LatestVersion, true
}

func (c *Checker) saveCache(latest string) error {
	if c.cachePath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(c.cachePath), 0o755); err != nil {
		return err
	}
	payload, err := json.Marshal(cacheFile{CheckedAt: c.now(), LatestVersion: latest})
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(c.cachePath), "update-check-*.tmp")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(payload); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), c.cachePath)
}
