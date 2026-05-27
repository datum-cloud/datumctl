package plugin

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// WithFlagCompletion wraps a ValidArgsFunction so that the command's flags are
// included as completion candidates on plain <TAB>, not only when the user has
// already typed "--". This is useful for commands whose primary input is flags
// (e.g. deploy) so users discover available options without needing to type a
// prefix first.
//
// If inner is nil, only flags are returned.
func WithFlagCompletion(inner cobra.CompletionFunc) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var results []string
		directive := cobra.ShellCompDirectiveNoFileComp

		if inner != nil {
			results, directive = inner(cmd, args, toComplete)
		}

		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if !f.Hidden {
				entry := "--" + f.Name
				if f.Usage != "" {
					entry += "\t" + f.Usage
				}
				results = append(results, entry)
			}
		})

		return results, directive
	}
}
