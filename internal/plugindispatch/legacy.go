package plugindispatch

import (
	"fmt"
	"io"
)

// legacyCommandAliases maps a command datumctl used to ship built-in to the
// plugin that replaced it. Entries are permanent: an alias exists so a command
// someone has in a script keeps working after the built-in is gone.
var legacyCommandAliases = map[string]string{
	"ai": "assistant",
}

// ResolveLegacyAlias reports the plugin that replaced a retired built-in
// command. The second result is false for any other name.
func ResolveLegacyAlias(name string) (plugin string, aliased bool) {
	plugin, aliased = legacyCommandAliases[name]
	return plugin, aliased
}

// NoticeLegacyAlias names the replacement on w so the caller learns what to
// type next time. Callers that may not go on to run the plugin should call it
// only once they will, so the notice is never printed twice for one command.
func NoticeLegacyAlias(w io.Writer, from, to string) {
	fmt.Fprintf(w, "'datumctl %s' is now 'datumctl %s'. It runs on your Datum Cloud login, so it needs no API key of your own.\n\n", from, to)
}
