package auth

import (
	"path/filepath"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestUpdateKubeconfigExecInteractiveMode(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    api.ExecInteractiveMode
		wantErr bool
	}{
		{
			name: "default is IfAvailable",
			args: []string{"--project", "p", "--hostname", "https://api.example.test"},
			want: api.IfAvailableExecInteractiveMode,
		},
		{
			name: "Never",
			args: []string{"--project", "p", "--hostname", "https://api.example.test", "--exec-interactive-mode", "Never"},
			want: api.NeverExecInteractiveMode,
		},
		{
			name: "Always",
			args: []string{"--project", "p", "--hostname", "https://api.example.test", "--exec-interactive-mode", "Always"},
			want: api.AlwaysExecInteractiveMode,
		},
		{
			name:    "invalid value rejected",
			args:    []string{"--project", "p", "--hostname", "https://api.example.test", "--exec-interactive-mode", "Bogus"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kubeconfigPath := filepath.Join(t.TempDir(), "config")
			cmd := updateKubeconfigCmd()
			cmd.SetArgs(append(tc.args, "--kubeconfig", kubeconfigPath))

			err := cmd.Execute()
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
			if err != nil {
				t.Fatalf("failed to load written kubeconfig: %v", err)
			}
			user, ok := cfg.AuthInfos["datum-user"]
			if !ok || user.Exec == nil {
				t.Fatal("datum-user exec auth info not written")
			}
			if user.Exec.InteractiveMode != tc.want {
				t.Errorf("InteractiveMode = %q, want %q", user.Exec.InteractiveMode, tc.want)
			}
		})
	}
}
