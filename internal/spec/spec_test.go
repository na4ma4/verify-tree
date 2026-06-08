package spec_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/na4ma4/verify-tree/internal/spec"
)

func TestSpec_LoadAndDefaults(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	yamlContent := `
defaults:
  owner: "app"
  group: "appgroup"
  file_mode: "0640"
  dir_mode: "0750"

entries:
  - path: "/opt/app"
    type: directory
    mode: "0755"

  - path: "/opt/app/config.yml"
    type: file

  - path: "/opt/app/logs"
    type: symlink
    target: "/var/log/app"
`
	specPath := filepath.Join(dir, "spec.yaml")
	if err := os.WriteFile(specPath, []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := spec.Load(specPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if s.Defaults == nil {
		t.Fatal("Defaults should not be nil")
	}
	if s.Defaults.Owner != "app" {
		t.Errorf("Default owner = %q, want app", s.Defaults.Owner)
	}
	if len(s.Entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(s.Entries))
	}

	s.ApplyDefaults()

	if s.Entries[0].Mode != "0755" {
		t.Errorf("Entry 0 mode = %q, want 0755 (explicit)", s.Entries[0].Mode)
	}
	if s.Entries[0].Owner != "app" {
		t.Errorf("Entry 0 owner = %q, want app", s.Entries[0].Owner)
	}

	if s.Entries[1].Mode != "0640" {
		t.Errorf("Entry 1 mode = %q, want 0640 (file default)", s.Entries[1].Mode)
	}
	if s.Entries[1].Owner != "app" {
		t.Errorf("Entry 1 owner = %q, want app", s.Entries[1].Owner)
	}
	if s.Entries[1].Group != "appgroup" {
		t.Errorf("Entry 1 group = %q, want appgroup", s.Entries[1].Group)
	}
}

func TestSpec_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "valid spec",
			yaml: `
entries:
  - path: "/test"
    type: file
`,
			wantErr: false,
		},
		{
			name: "invalid type",
			yaml: `
entries:
  - path: "/test"
    type: socket
`,
			wantErr: true,
		},
		{
			name: "missing type",
			yaml: `
entries:
  - path: "/test"
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			specPath := filepath.Join(dir, "spec.yaml")
			if err := os.WriteFile(specPath, []byte(tt.yaml), 0o644); err != nil {
				t.Fatal(err)
			}

			_, err := spec.Load(specPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
