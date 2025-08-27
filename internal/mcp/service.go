package mcp

import (
	"context"
	"fmt"
	"strings"
	"encoding/json"

	"go.datum.net/datumctl/internal/client"
	syaml "sigs.k8s.io/yaml"
)

type Service struct {
	K *client.K8sClient
}

func NewService(k *client.K8sClient) *Service { return &Service{K: k} }

// ---------- API types ----------

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

// ---------- Methods ----------

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
