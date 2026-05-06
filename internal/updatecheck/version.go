package updatecheck

import "golang.org/x/mod/semver"

// isNewer returns true when latest is a strictly newer semantic version than
// current. Both inputs must be vX.Y.Z[-suffix]. Invalid inputs result in
// false (no warning shown).
func isNewer(current, latest string) bool {
	if !semver.IsValid(current) || !semver.IsValid(latest) {
		return false
	}
	return semver.Compare(latest, current) > 0
}
