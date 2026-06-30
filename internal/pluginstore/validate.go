package pluginstore

import (
	"context"
	"fmt"
	"net/url"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// FetchAndParseCatalog resolves a catalog source (HTTPS URL, GitHub owner/repo,
// or local path), fetches the manifest, and parses it WITHOUT touching any
// cache or registry. It is the read-only primitive behind `plugin index
// validate`.
func FetchAndParseCatalog(ctx context.Context, source string) (*PluginList, error) {
	resolved, err := ResolveCatalogSource(source)
	if err != nil {
		return nil, err
	}
	raw, err := fetchCatalogManifest(ctx, resolved)
	if err != nil {
		return nil, err
	}
	// fetchCatalogManifest caps raw to MaxManifestBytes; re-assert it here so the
	// bound on the YAML parser's input is explicit at the unmarshal site.
	if int64(len(raw)) > MaxManifestBytes {
		return nil, fmt.Errorf("catalog manifest exceeds the maximum allowed size of %d bytes", MaxManifestBytes)
	}
	var list PluginList
	if err := yaml.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("parse catalog manifest: %w", err)
	}
	return &list, nil
}

// LintCatalog returns a list of human-readable problems with a catalog manifest:
// missing plugin names/versions, missing or non-HTTPS download URIs, missing
// checksums, and invalid platform selectors. An empty result means the manifest
// is valid.
func LintCatalog(list *PluginList) []string {
	var problems []string
	if list == nil {
		return []string{"manifest is empty"}
	}
	if len(list.Items) == 0 {
		problems = append(problems, "catalog contains no plugins")
	}
	for i := range list.Items {
		p := &list.Items[i]
		label := p.Name
		if label == "" {
			label = fmt.Sprintf("plugin #%d", i)
			problems = append(problems, fmt.Sprintf("%s: missing metadata.name", label))
		}
		if p.Spec.Version == "" {
			problems = append(problems, fmt.Sprintf("%s: missing spec.version", label))
		}
		if len(p.Spec.Platforms) == 0 {
			problems = append(problems, fmt.Sprintf("%s: no platforms declared", label))
		}
		for j := range p.Spec.Platforms {
			plat := &p.Spec.Platforms[j]
			switch {
			case plat.URI == "":
				problems = append(problems, fmt.Sprintf("%s platform %d: missing uri", label, j))
			default:
				if u, err := url.Parse(plat.URI); err != nil {
					problems = append(problems, fmt.Sprintf("%s platform %d: invalid uri %q", label, j, plat.URI))
				} else if u.Scheme != "https" {
					problems = append(problems, fmt.Sprintf("%s platform %d: uri %q must use HTTPS", label, j, plat.URI))
				}
			}
			if plat.SHA256 == "" {
				problems = append(problems, fmt.Sprintf("%s platform %d: missing sha256 checksum", label, j))
			}
			if plat.Selector != nil {
				if _, err := metav1.LabelSelectorAsSelector(plat.Selector); err != nil {
					problems = append(problems, fmt.Sprintf("%s platform %d: invalid selector: %v", label, j, err))
				}
			}
		}
	}
	return problems
}
