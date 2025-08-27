package kube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Kubectl struct {
	Path      string   // default: "kubectl"
	Context   string   // optional --context
	Namespace string   // optional -n
	Env       []string // defaults to os.Environ()
}

func New() *Kubectl { return &Kubectl{Path: "kubectl", Env: os.Environ()} }

func (k *Kubectl) args(base ...string) []string {
	args := make([]string, 0, len(base)+4)
	if k.Context != "" {
		args = append(args, "--context", k.Context)
	}
	if k.Namespace != "" {
		args = append(args, "-n", k.Namespace)
	}
	args = append(args, base...)
	return args
}

// like args() but DOES NOT inject -n (namespace)
func (k *Kubectl) argsNoNS(base ...string) []string {
	args := make([]string, 0, len(base)+2)
	if k.Context != "" {
		args = append(args, "--context", k.Context)
	}
	args = append(args, base...)
	return args
}

func (k *Kubectl) run(ctx context.Context, stdin []byte, base ...string) ([]byte, []byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, k.Path, k.args(base...)...)
	cmd.Env = k.Env
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var out, err bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &err
	runErr := cmd.Run()
	return out.Bytes(), err.Bytes(), runErr
}

// runNoNS injects --context (if set) but NOT -n (namespace).
func (k *Kubectl) runNoNS(ctx context.Context, stdin []byte, base ...string) ([]byte, []byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, k.Path, k.argsNoNS(base...)...)
	cmd.Env = k.Env
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var out, err bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &err
	runErr := cmd.Run()
	return out.Bytes(), err.Bytes(), runErr
}

// runBare injects neither --context nor -n; used for `kubectl config ...`
func (k *Kubectl) runBare(ctx context.Context, stdin []byte, base ...string) ([]byte, []byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, k.Path, base...)
	cmd.Env = k.Env
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var out, err bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &err
	runErr := cmd.Run()
	return out.Bytes(), err.Bytes(), runErr
}

// ---------------- high-level operations ----------------

type CRDItem struct {
	Name     string   `json:"name"`     // metadata.name, e.g. httpproxies.networking.example.io
	Group    string   `json:"group"`    // spec.group
	Kind     string   `json:"kind"`     // spec.names.kind
	Versions []string `json:"versions"` // served versions
}

func (k *Kubectl) ListCRDs(ctx context.Context) ([]CRDItem, error) {
	stdout, stderr, err := k.run(ctx, nil, "get", "crd", "-o", "json")
	if err != nil {
		return nil, fmt.Errorf("kubectl get crd: %v: %s", err, strings.TrimSpace(string(stderr)))
	}
	var payload struct {
		Items []struct {
			Metadata struct{ Name string `json:"name"` } `json:"metadata"`
			Spec     struct {
				Group    string `json:"group"`
				Names    struct{ Kind string `json:"kind"` } `json:"names"`
				Versions []struct {
					Name   string `json:"name"`
					Served bool   `json:"served"`
				} `json:"versions"`
			} `json:"spec"`
		} `json:"items"`
	}
	if err := json.Unmarshal(stdout, &payload); err != nil {
		return nil, fmt.Errorf("parse CRDs json: %w", err)
	}
	out := make([]CRDItem, 0, len(payload.Items))
	for _, it := range payload.Items {
		var vs []string
		for _, v := range it.Spec.Versions {
			if v.Served {
				vs = append(vs, v.Name)
			}
		}
		out = append(out, CRDItem{
			Name:     it.Metadata.Name,
			Group:    it.Spec.Group,
			Kind:     it.Spec.Names.Kind,
			Versions: vs,
		})
	}
	return out, nil
}

func (k *Kubectl) GetCRD(ctx context.Context, name, mode string) (string, error) {
	switch mode {
	case "", "yaml", "json":
		if mode == "" {
			mode = "yaml"
		}
		stdout, stderr, err := k.run(ctx, nil, "get", "crd", name, "-o", mode)
		if err != nil {
			return "", fmt.Errorf("kubectl get crd %s: %v: %s", name, err, strings.TrimSpace(string(stderr)))
		}
		return string(stdout), nil
	case "describe":
		stdout, stderr, err := k.run(ctx, nil, "describe", "crd", name)
		if err != nil {
			return "", fmt.Errorf("kubectl describe crd %s: %v: %s", name, err, strings.TrimSpace(string(stderr)))
		}
		return string(stdout), nil
	default:
		return "", fmt.Errorf("unsupported mode %q (use yaml|json|describe)", mode)
	}
}

func (k *Kubectl) ValidateYAML(ctx context.Context, manifest string) (ok bool, output string, err error) {
	args := []string{"apply", "--dry-run=server", "--validate=true", "-f", "-"}
	stdout, stderr, runErr := k.run(ctx, []byte(manifest), args...)
	out := strings.TrimSpace(string(stdout))
	errs := strings.TrimSpace(string(stderr))
	if runErr != nil {
		// API server returns non-zero for validation failures; surface combined output
		if out != "" && errs != "" {
			return false, out + "\n" + errs, nil
		}
		if out != "" {
			return false, out, nil
		}
		return false, errs, nil
	}
	return true, out, nil
}

// ---------------- preflight (no auto-login; friendly errors) ----------------

// Preflight verifies context exists, cluster is reachable, and caller can list CRDs.
// It does NOT log anyone in; it only reports what's missing.
func (k *Kubectl) Preflight(ctx context.Context) error {
	if err := k.ensureContextExists(ctx); err != nil {
		return err
	}
	if err := k.ensureClusterReachable(ctx); err != nil {
		return err
	}
	ok, _, _ := k.canI(ctx, "get", "crd")
	if !ok {
		return fmt.Errorf("not logged in or insufficient RBAC for CRDs. " +
			"Please run `datumctl auth login`, ensure the correct kube context, and retry.")
	}
	return nil
}

func (k *Kubectl) ensureContextExists(ctx context.Context) error {
	if k.Context == "" {
		out, _, err := k.runBare(ctx, nil, "config", "current-context")
		if err != nil || strings.TrimSpace(string(out)) == "" {
			return fmt.Errorf("no current kubectl context. Use --kube-context or run `kubectl config use-context ...`")
		}
		return nil
	}
	// Exit non-zero if the named context does not exist.
	_, errOut, err := k.runBare(ctx, nil, "config", "get-contexts", k.Context)
	if err != nil {
		return fmt.Errorf("kube context %q not found. Run `kubectl config get-contexts` and choose a valid one", k.Context)
	}
	_ = errOut
	return nil
}

func (k *Kubectl) ensureClusterReachable(ctx context.Context) error {
	// Prefer a raw /version probe; it doesn't assume core/v1 Services exist.
	_, errOut, err := k.runNoNS(ctx, nil, "get", "--raw", "/version")
	if err == nil {
		return nil
	}
	// Fallback: kubectl version --short (older kubectl may not support --raw)
	_, errOut2, err2 := k.runNoNS(ctx, nil, "version", "--short")
	if err2 == nil {
		return nil
	}
	return fmt.Errorf("cannot reach cluster for the selected context. %s %s",
		strings.TrimSpace(string(errOut)), strings.TrimSpace(string(errOut2)))
}

// returns (allowed, output, errorRunningCmd)
func (k *Kubectl) canI(ctx context.Context, verb, resource string) (bool, string, error) {
	// --quiet exits 0 if allowed, 1 if denied
	_, errOut, err := k.runNoNS(ctx, nil, "auth", "can-i", "--quiet", verb, resource)
	if err != nil {
		return false, strings.TrimSpace(string(errOut)), nil
	}
	return true, "", nil
}
