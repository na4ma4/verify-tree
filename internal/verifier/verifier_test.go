package verifier_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/na4ma4/verify-tree/internal/config"
	"github.com/na4ma4/verify-tree/internal/verifier"
)

func createTestDir(t *testing.T) (string, map[string]string) {
	t.Helper()
	dir := t.TempDir()

	root := filepath.Join(dir, "opt", "app")
	must(t, os.MkdirAll(root, 0o755))
	must(t, os.WriteFile(filepath.Join(root, "config.yml"), []byte("test"), 0o644))
	must(t, os.MkdirAll(filepath.Join(root, "subdir"), 0o755))
	must(t, os.WriteFile(filepath.Join(root, "subdir", "data.txt"), []byte("data"), 0o644))
	must(t, os.Symlink("/var/log/app", filepath.Join(root, "logs")))

	return dir, map[string]string{
		"root": root,
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestVerify_Pass(t *testing.T) {
	t.Parallel()
	dir, varsMap := createTestDir(t)
	vars := config.Variables(varsMap)

	specContent := `
entries:
  - path: "{{ root }}"
    type: directory
    mode: "0755"

  - path: "{{ root }}/config.yml"
    type: file
    mode: "0644"

  - path: "{{ root }}/subdir"
    type: directory
    mode: "0755"

  - path: "{{ root }}/subdir/data.txt"
    type: file
    mode: "0644"

  - path: "{{ root }}/logs"
    type: symlink
    target: "/var/log/app"
    ignore: [missing]
`

	specFile := filepath.Join(dir, "spec.yaml")
	must(t, os.WriteFile(specFile, []byte(specContent), 0o644))

	result, err := verifier.Verify(specFile, vars)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if !result.Passed {
		t.Errorf("result.Passed = false, want true")
		for _, e := range result.Entries {
			if !e.Passed {
				t.Logf("  failed entry: %s", e.Path)
				for _, v := range e.Violations {
					t.Logf("    %s: expected %s, got %s", v.Check, v.Expected, v.Actual)
				}
			}
		}
	}
}

func TestVerify_ModeMismatch(t *testing.T) {
	t.Parallel()
	dir, varsMap := createTestDir(t)
	vars := config.Variables(varsMap)

	specContent := `
entries:
  - path: "{{ root }}/config.yml"
    type: file
    mode: "0600"
`
	specFile := filepath.Join(dir, "spec.yaml")
	must(t, os.WriteFile(specFile, []byte(specContent), 0o644))

	result, err := verifier.Verify(specFile, vars)
	if err != nil {
		t.Fatal(err)
	}

	if result.Passed {
		t.Error("result.Passed = true, want false (mode should mismatch)")
	}
}

func TestVerify_MissingWithIgnore(t *testing.T) {
	t.Parallel()
	dir, varsMap := createTestDir(t)
	vars := config.Variables(varsMap)

	specContent := `
entries:
  - path: "{{ root }}/does-not-exist"
    type: file
    ignore: [missing]
`
	specFile := filepath.Join(dir, "spec.yaml")
	must(t, os.WriteFile(specFile, []byte(specContent), 0o644))

	result, err := verifier.Verify(specFile, vars)
	if err != nil {
		t.Fatal(err)
	}

	if !result.Passed {
		t.Error("result.Passed = false, want true (missing ignored)")
	}
}

func TestVerify_Defaults(t *testing.T) {
	t.Parallel()
	dir, varsMap := createTestDir(t)
	vars := config.Variables(varsMap)

	specContent := `
defaults:
  file_mode: "0600"
  dir_mode: "0750"

entries:
  - path: "{{ root }}"
    type: directory
    mode: "0755"
`
	specFile := filepath.Join(dir, "spec.yaml")
	must(t, os.WriteFile(specFile, []byte(specContent), 0o644))

	result, err := verifier.Verify(specFile, vars)
	if err != nil {
		t.Fatal(err)
	}

	if !result.Passed {
		t.Errorf("result.Passed = false, want true")
		for _, e := range result.Entries {
			if !e.Passed {
				for _, v := range e.Violations {
					t.Logf("  %s: expected %s, got %s", v.Check, v.Expected, v.Actual)
				}
			}
		}
	}
}

func TestVerify_GlobPass(t *testing.T) {
	t.Parallel()
	dir, _ := createTestDir(t)

	vars := config.Variables{
		"root": filepath.Join(dir, "opt", "app"),
	}

	specContent := `
entries:
  - path: "{{ root }}/*.yml"
    type: file
    mode: "0644"
`
	specFile := filepath.Join(dir, "spec.yaml")
	must(t, os.WriteFile(specFile, []byte(specContent), 0o644))

	result, err := verifier.Verify(specFile, vars)
	if err != nil {
		t.Fatal(err)
	}

	if !result.Passed {
		t.Errorf("result.Passed = false, want true (glob should match config.yml)")
		for _, e := range result.Entries {
			if !e.Passed {
				for _, v := range e.Violations {
					t.Logf("  %s: expected %s, got %s", v.Check, v.Expected, v.Actual)
				}
			}
		}
	}
}

func TestVerify_GlobNoMatchIgnored(t *testing.T) {
	t.Parallel()
	dir, varsMap := createTestDir(t)
	vars := config.Variables(varsMap)

	specContent := `
entries:
  - path: "{{ root }}/*.py"
    type: file
    ignore: [glob-no-match]
`
	specFile := filepath.Join(dir, "spec.yaml")
	must(t, os.WriteFile(specFile, []byte(specContent), 0o644))

	result, err := verifier.Verify(specFile, vars)
	if err != nil {
		t.Fatal(err)
	}

	if !result.Passed {
		t.Error("result.Passed = false, want true (glob-no-match ignored)")
	}
}
