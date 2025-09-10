package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.datum.net/datumctl/internal/client"
	syaml "sigs.k8s.io/yaml"
)

type Service struct {
	K *client.K8sClient
}

func NewService(k *client.K8sClient) *Service { return &Service{K: k} }

type CRDItem struct {
	Name     string   `json:"name"`
	Group    string   `json:"group"`
	Kind     string   `json:"kind"`
	Versions []string `json:"versions"`
}

type ListCRDsResp struct {
	Items []CRDItem `json:"items"`
}

type GetCRDReq struct {
	Name string `json:"name"` // CRD name, e.g. httpproxies.networking.example.io
	Mode string `json:"mode"` // yaml|json|describe (default yaml)
}
type GetCRDResp struct {
	Text string `json:"text"`
}

type ValidateReq struct {
	YAML string `json:"yaml"`
}
type ValidateResp struct {
	Valid  bool   `json:"valid"`
	Output string `json:"output"`
}

func (s *Service) ListCRDs(ctx context.Context) (ListCRDsResp, error) {
	crds, err := s.K.ListCRDs(ctx) // native CRDs
	if err != nil {
		return ListCRDsResp{}, err
	}
	items := make([]CRDItem, 0, len(crds))
	for _, crd := range crds {
		versions := make([]string, 0, len(crd.Spec.Versions))
		for _, v := range crd.Spec.Versions {
			versions = append(versions, v.Name)
		}
		items = append(items, CRDItem{
			Name:     crd.Name,
			Group:    crd.Spec.Group,
			Kind:     crd.Spec.Names.Kind,
			Versions: versions,
		})
	}
	return ListCRDsResp{Items: items}, nil
}

func (s *Service) GetCRD(ctx context.Context, r GetCRDReq) (GetCRDResp, error) {
	if r.Name == "" {
		return GetCRDResp{}, fmt.Errorf("missing CRD name")
	}
	crd, err := s.K.GetCRD(ctx, r.Name) // native CRD from client
	if err != nil {
		return GetCRDResp{}, err
	}

	switch strings.ToLower(r.Mode) {
	case "json":
		b, err := json.MarshalIndent(crd, "", "  ")
		if err != nil {
			return GetCRDResp{}, err
		}
		return GetCRDResp{Text: string(b)}, nil

	case "describe":
		var sb strings.Builder
		fmt.Fprintf(&sb, "Name: %s\n", crd.Name)
		fmt.Fprintf(&sb, "Group: %s\n", crd.Spec.Group)
		fmt.Fprintf(&sb, "Kind: %s\n", crd.Spec.Names.Kind)
		fmt.Fprintf(&sb, "Scope: %s\n", crd.Spec.Scope)
		versions := make([]string, 0, len(crd.Spec.Versions))
		for _, v := range crd.Spec.Versions {
			versions = append(versions, v.Name)
		}
		fmt.Fprintf(&sb, "Versions: %s\n", strings.Join(versions, ", "))
		return GetCRDResp{Text: sb.String()}, nil

	default: // yaml
		b, err := json.Marshal(crd)
		if err != nil {
			return GetCRDResp{}, err
		}
		y, err := syaml.JSONToYAML(b)
		if err != nil {
			return GetCRDResp{}, err
		}
		return GetCRDResp{Text: string(y)}, nil
	}
}

func (s *Service) ValidateYAML(ctx context.Context, r ValidateReq) ValidateResp {
	ok, out, _ := s.K.ValidateYAML(ctx, r.YAML)
	return ValidateResp{Valid: ok, Output: out}
}

type ChangeContextReq struct {
	Project   string `json:"project,omitempty"`
	Org       string `json:"org,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}
type ChangeContextResp struct {
	OK        bool   `json:"ok"`
	Project   string `json:"project,omitempty"`
	Org       string `json:"org,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type CreateResourceReq struct {
	Kind         string            `json:"kind"`
	APIVersion   string            `json:"apiVersion,omitempty"`
	Name         string            `json:"name,omitempty"`
	GenerateName string            `json:"generateName,omitempty"`
	Namespace    string            `json:"namespace,omitempty"`
	Spec         map[string]any    `json:"spec,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
	DryRun       *bool             `json:"dryRun,omitempty"` // default true
}
type GetResourceReq struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion,omitempty"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"`
	Format     string `json:"format,omitempty"` // yaml (default) | json
}
type UpdateResourceReq struct {
	Kind        string            `json:"kind"`
	APIVersion  string            `json:"apiVersion,omitempty"`
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Spec        map[string]any    `json:"spec,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	DryRun      *bool             `json:"dryRun,omitempty"` // default true
	Force       bool              `json:"force,omitempty"`  // SSA force
}
type DeleteResourceReq struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion,omitempty"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"`
	DryRun     *bool  `json:"dryRun,omitempty"` // default true
}

type ListResourcesReq struct {
	Kind          string `json:"kind"`
	APIVersion    string `json:"apiVersion,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
	LabelSelector string `json:"labelSelector,omitempty"`
	FieldSelector string `json:"fieldSelector,omitempty"`
	Limit         *int64 `json:"limit,omitempty"`
	Continue      string `json:"continue,omitempty"`
	Format        string `json:"format,omitempty"` // "yaml" (default) | "names"
}

type OkResp struct {
	OK              bool   `json:"ok"`
	DryRun          bool   `json:"dryRun,omitempty"`
	Validated       bool   `json:"validated,omitempty"`
	ResourceVersion string `json:"resourceVersion,omitempty"`
}
type GetResourceResp struct {
	Text string `json:"text"`
}
type ListResourcesResp struct {
	Text string `json:"text"`
}

func (s *Service) ChangeContext(ctx context.Context, r ChangeContextReq) (ChangeContextResp, error) {
	// namespace-only change
	if r.Project == "" && r.Org == "" && r.Namespace != "" {
		s.K.Namespace = r.Namespace
		return ChangeContextResp{OK: true, Namespace: s.K.Namespace}, nil
	}
	// require exactly one of project or org
	if (r.Project == "") == (r.Org == "") {
		return ChangeContextResp{}, fmt.Errorf("exactly one of project or org is required")
	}

	var (
		next *client.K8sClient
		err  error
	)
	if r.Project != "" {
		next, err = client.NewForProject(ctx, r.Project, firstNonEmpty(r.Namespace, s.K.Namespace))
	} else {
		next, err = client.NewForOrg(ctx, r.Org, firstNonEmpty(r.Namespace, s.K.Namespace))
	}
	if err != nil {
		return ChangeContextResp{}, err
	}
	if err := next.Preflight(ctx); err != nil {
		return ChangeContextResp{}, err
	}
	s.K = next
	return ChangeContextResp{OK: true, Project: r.Project, Org: r.Org, Namespace: s.K.Namespace}, nil
}

func (s *Service) CreateResource(ctx context.Context, r CreateResourceReq) (OkResp, error) {
	if r.Kind == "" {
		return OkResp{}, fmt.Errorf("kind is required")
	}
	dry := true
	if r.DryRun != nil {
		dry = *r.DryRun
	}
	obj, err := s.K.Create(ctx, client.CreateOptions{
		Kind:         r.Kind,
		APIVersion:   r.APIVersion,
		Name:         r.Name,
		GenerateName: r.GenerateName,
		Namespace:    firstNonEmpty(r.Namespace, s.K.Namespace),
		Labels:       r.Labels,
		Annotations:  r.Annotations,
		Spec:         r.Spec,
		DryRun:       dry,
	})
	if err != nil {
		return OkResp{}, err
	}
	if dry {
		return OkResp{OK: true, DryRun: true, Validated: true}, nil
	}
	return OkResp{OK: true, ResourceVersion: obj.GetResourceVersion()}, nil
}

func (s *Service) GetResource(ctx context.Context, r GetResourceReq) (GetResourceResp, error) {
	if r.Kind == "" || r.Name == "" {
		return GetResourceResp{}, fmt.Errorf("kind and name are required")
	}
	obj, err := s.K.Get(ctx, client.GetOptions{
		Kind:       r.Kind,
		APIVersion: r.APIVersion,
		Name:       r.Name,
		Namespace:  firstNonEmpty(r.Namespace, s.K.Namespace),
	})
	if err != nil {
		return GetResourceResp{}, err
	}
	if strings.ToLower(r.Format) == "json" {
		jb, _ := json.MarshalIndent(obj.Object, "", "  ")
		return GetResourceResp{Text: string(jb)}, nil
	}
	yb, err := client.ToYAML(obj)
	if err != nil {
		return GetResourceResp{}, err
	}
	return GetResourceResp{Text: yb}, nil
}

func (s *Service) UpdateResource(ctx context.Context, r UpdateResourceReq) (OkResp, error) {
	if r.Kind == "" || r.Name == "" {
		return OkResp{}, fmt.Errorf("kind and name are required")
	}
	dry := true
	if r.DryRun != nil {
		dry = *r.DryRun
	}
	obj, err := s.K.Apply(ctx, client.ApplyOptions{
		Kind:        r.Kind,
		APIVersion:  r.APIVersion,
		Name:        r.Name,
		Namespace:   firstNonEmpty(r.Namespace, s.K.Namespace),
		Labels:      r.Labels,
		Annotations: r.Annotations,
		Spec:        r.Spec,
		DryRun:      dry,
		Force:       r.Force,
	})
	if err != nil {
		return OkResp{}, err
	}
	if dry {
		return OkResp{OK: true, DryRun: true, Validated: true}, nil
	}
	return OkResp{OK: true, ResourceVersion: obj.GetResourceVersion()}, nil
}

func (s *Service) DeleteResource(ctx context.Context, r DeleteResourceReq) (OkResp, error) {
	if r.Kind == "" || r.Name == "" {
		return OkResp{}, fmt.Errorf("kind and name are required")
	}
	dry := true
	if r.DryRun != nil {
		dry = *r.DryRun
	}
	if err := s.K.Delete(ctx, client.DeleteOptions{
		Kind:       r.Kind,
		APIVersion: r.APIVersion,
		Name:       r.Name,
		Namespace:  firstNonEmpty(r.Namespace, s.K.Namespace),
		DryRun:     dry,
	}); err != nil {
		return OkResp{}, err
	}
	if dry {
		return OkResp{OK: true, DryRun: true, Validated: true}, nil
	}
	return OkResp{OK: true}, nil
}

func (s *Service) ListResources(ctx context.Context, r ListResourcesReq) (ListResourcesResp, error) {
	if r.Kind == "" {
		return ListResourcesResp{}, fmt.Errorf("kind is required")
	}
	var lim int64
	if r.Limit != nil {
		lim = *r.Limit
	}
	lst, err := s.K.List(ctx, client.ListOptions{
		Kind:          r.Kind,
		APIVersion:    r.APIVersion,
		Namespace:     firstNonEmpty(r.Namespace, s.K.Namespace),
		LabelSelector: r.LabelSelector,
		FieldSelector: r.FieldSelector,
		Limit:         lim,
		Continue:      r.Continue,
	})
	if err != nil {
		return ListResourcesResp{}, err
	}

	switch strings.ToLower(r.Format) {
	case "names":
		var b strings.Builder
		for i := range lst.Items {
			n := lst.Items[i].GetName()
			if ns := lst.Items[i].GetNamespace(); ns != "" {
				fmt.Fprintf(&b, "%s/%s\n", ns, n)
			} else {
				fmt.Fprintln(&b, n)
			}
		}
		return ListResourcesResp{Text: b.String()}, nil
	default: // yaml
		y, err := client.ToYAMLList(lst)
		if err != nil {
			return ListResourcesResp{}, err
		}
		return ListResourcesResp{Text: y}, nil
	}
}

// helper for getting the first non-empty string
func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
