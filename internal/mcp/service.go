package mcp

import (
	"context"
	"fmt"

	"go.datum.net/datumctl/internal/kube"
)

type Service struct {
	K *kube.Kubectl
}

func NewService(k *kube.Kubectl) *Service { return &Service{K: k} }

// ---------- API types ----------

type ListCRDsResp struct {
	Items []kube.CRDItem `json:"items"`
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
	items, err := s.K.ListCRDs(ctx)
	if err != nil {
		return ListCRDsResp{}, err
	}
	return ListCRDsResp{Items: items}, nil
}

func (s *Service) GetCRD(ctx context.Context, r GetCRDReq) (GetCRDResp, error) {
	if r.Name == "" {
		return GetCRDResp{}, fmt.Errorf("missing CRD name")
	}
	text, err := s.K.GetCRD(ctx, r.Name, r.Mode)
	if err != nil {
		return GetCRDResp{}, err
	}
	return GetCRDResp{Text: text}, nil
}

func (s *Service) ValidateYAML(ctx context.Context, r ValidateReq) ValidateResp {
	ok, out, _ := s.K.ValidateYAML(ctx, r.YAML)
	return ValidateResp{Valid: ok, Output: out}
}
