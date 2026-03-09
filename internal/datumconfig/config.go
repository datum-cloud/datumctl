package datumconfig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

const (
	DefaultAPIVersion = "datumctl.config.datum.net/v1alpha1"
	DefaultKind       = "DatumctlConfig"
	DefaultNamespace  = "default"
)

type Config struct {
	APIVersion     string         `json:"apiVersion" yaml:"apiVersion"`
	Kind           string         `json:"kind" yaml:"kind"`
	Clusters       []NamedCluster `json:"clusters,omitempty" yaml:"clusters,omitempty"`
	Users          []NamedUser    `json:"users,omitempty" yaml:"users,omitempty"`
	Contexts       []NamedContext `json:"contexts,omitempty" yaml:"contexts,omitempty"`
	CurrentContext string         `json:"current-context,omitempty" yaml:"current-context,omitempty"`
}

type NamedCluster struct {
	Name    string  `json:"name" yaml:"name"`
	Cluster Cluster `json:"cluster" yaml:"cluster"`
}

type Cluster struct {
	Server                   string `json:"server" yaml:"server"`
	TLSServerName            string `json:"tls-server-name,omitempty" yaml:"tls-server-name,omitempty"`
	InsecureSkipTLSVerify    bool   `json:"insecure-skip-tls-verify,omitempty" yaml:"insecure-skip-tls-verify,omitempty"`
	CertificateAuthorityData string `json:"certificate-authority-data,omitempty" yaml:"certificate-authority-data,omitempty"`
}

type NamedContext struct {
	Name    string  `json:"name" yaml:"name"`
	Context Context `json:"context" yaml:"context"`
}

type Context struct {
	Cluster        string `json:"cluster" yaml:"cluster"`
	User           string `json:"user,omitempty" yaml:"user,omitempty"`
	Namespace      string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	ProjectID      string `json:"project_id,omitempty" yaml:"project_id,omitempty"`
	OrganizationID string `json:"organization_id,omitempty" yaml:"organization_id,omitempty"`
}

type NamedUser struct {
	Name string `json:"name" yaml:"name"`
	User User   `json:"user" yaml:"user"`
}

type User struct {
	Key string `json:"key" yaml:"key"`
}

func New() *Config {
	return &Config{
		APIVersion: DefaultAPIVersion,
		Kind:       DefaultKind,
	}
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".datumctl", "config"), nil
}

func Load() (*Config, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return LoadFromPath(path)
}

func LoadFromPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return New(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return New(), nil
	}

	cfg := New()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	cfg.ensureDefaults()
	return cfg, nil
}

func Save(cfg *Config) error {
	path, err := DefaultPath()
	if err != nil {
		return err
	}
	return SaveToPath(cfg, path)
}

func SaveToPath(cfg *Config, path string) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	cfg.ensureDefaults()

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}

	return nil
}

func (c *Config) ensureDefaults() {
	if c.APIVersion == "" {
		c.APIVersion = DefaultAPIVersion
	}
	if c.Kind == "" {
		c.Kind = DefaultKind
	}
}

func (c *Config) ContextByName(name string) (*NamedContext, bool) {
	for i := range c.Contexts {
		if c.Contexts[i].Name == name {
			return &c.Contexts[i], true
		}
	}
	return nil, false
}

func (c *Config) ClusterByName(name string) (*NamedCluster, bool) {
	for i := range c.Clusters {
		if c.Clusters[i].Name == name {
			return &c.Clusters[i], true
		}
	}
	return nil, false
}

func (c *Config) UserByName(name string) (*NamedUser, bool) {
	for i := range c.Users {
		if c.Users[i].Name == name {
			return &c.Users[i], true
		}
	}
	return nil, false
}

func (c *Config) UpsertCluster(entry NamedCluster) {
	for i := range c.Clusters {
		if c.Clusters[i].Name == entry.Name {
			c.Clusters[i] = entry
			return
		}
	}
	c.Clusters = append(c.Clusters, entry)
}

func (c *Config) UpsertUser(entry NamedUser) {
	for i := range c.Users {
		if c.Users[i].Name == entry.Name {
			c.Users[i] = entry
			return
		}
	}
	c.Users = append(c.Users, entry)
}

func (c *Config) UpsertContext(entry NamedContext) {
	for i := range c.Contexts {
		if c.Contexts[i].Name == entry.Name {
			c.Contexts[i] = entry
			return
		}
	}
	c.Contexts = append(c.Contexts, entry)
}

func (c *Config) CurrentContextEntry() (*NamedContext, bool) {
	if c.CurrentContext == "" {
		return nil, false
	}
	return c.ContextByName(c.CurrentContext)
}

func (c *Config) EnsureContextDefaults(ctx *Context) {
	if ctx.Namespace == "" {
		ctx.Namespace = DefaultNamespace
	}
}

func (c *Config) ValidateContext(ctx Context) error {
	if ctx.Cluster == "" {
		return errors.New("context cluster is required")
	}
	if ctx.ProjectID != "" && ctx.OrganizationID != "" {
		return errors.New("context cannot set both project_id and organization_id")
	}
	return nil
}

func (c *Config) ValidateCluster(cluster Cluster) error {
	if cluster.Server == "" {
		return errors.New("cluster server is required")
	}
	return nil
}

func LoadCurrentContext() (*Config, *NamedContext, *NamedCluster, error) {
	cfg, err := Load()
	if err != nil {
		return nil, nil, nil, err
	}
	ctx, ok := cfg.CurrentContextEntry()
	if !ok {
		return cfg, nil, nil, nil
	}
	if ctx.Context.Cluster == "" {
		return cfg, ctx, nil, fmt.Errorf("context %q is missing cluster", ctx.Name)
	}
	cluster, ok := cfg.ClusterByName(ctx.Context.Cluster)
	if !ok {
		return cfg, ctx, nil, fmt.Errorf("cluster %q referenced by context %q not found", ctx.Context.Cluster, ctx.Name)
	}
	cfg.EnsureContextDefaults(&ctx.Context)
	return cfg, ctx, cluster, nil
}

func EnsureScheme(server string) string {
	if server == "" {
		return server
	}
	if strings.HasPrefix(server, "http://") || strings.HasPrefix(server, "https://") {
		return server
	}
	return "https://" + server
}

func CleanBaseServer(server string) string {
	if server == "" {
		return server
	}
	return strings.TrimRight(server, "/")
}
