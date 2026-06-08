package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/na4ma4/verify-tree/internal/spec"
)

func TestSpecLoad_yamlSyntaxError_contextLines(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	content := "a: 1\nb: 2\nc: 3\nd: 4\nbad: [unclosed\ne: 5\nf: 6\ng: 7\n"
	specPath := filepath.Join(dir, "spec.yaml")
	if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := spec.Load(specPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	msg := err.Error()

	if !strings.Contains(msg, specPath+":") {
		t.Errorf("expected file path in error, got: %v", msg)
	}

	if !strings.Contains(msg, ">    4 |") {
		t.Errorf("expected > marker on line 4, got: %v", msg)
	}
}

func TestSpecLoad_yamlSyntaxError_nearStart(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	content := "bad: [unclosed\nsecond: line\n"
	specPath := filepath.Join(dir, "spec.yaml")
	if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := spec.Load(specPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	msg := err.Error()

	if !strings.Contains(msg, ">    1 |") {
		t.Errorf("expected > marker on line 1, got: %v", msg)
	}
}

func TestSpecLoad_yamlSyntaxError_nearEnd(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var lines []string
	for i := 1; i <= 12; i++ {
		lines = append(lines, "line")
	}
	lines[11] = "bad: [unclosed"
	content := strings.Join(lines, "\n")

	specPath := filepath.Join(dir, "spec.yaml")
	if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := spec.Load(specPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	msg := err.Error()

	if !strings.Contains(msg, ">   12 |") {
		t.Errorf("expected > marker on line 12, got: %v", msg)
	}
}

func TestSpecLoad_yamlErrorIncludesOriginal(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	content := "bad: [unclosed"
	specPath := filepath.Join(dir, "spec.yaml")
	if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := spec.Load(specPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "yaml:") {
		t.Errorf("expected underlying yaml error in message, got: %v", err)
	}
}

func TestSpecLoad_yamlErrorNoLineNumber(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	content := ": value"
	specPath := filepath.Join(dir, "spec.yaml")
	if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := spec.Load(specPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	msg := err.Error()

	if !strings.HasPrefix(msg, specPath+": ") {
		t.Errorf("expected path prefix without line number, got: %v", msg)
	}
}

func TestSpecLoad_didNotFindExpectedKey_noHintWithColon(t *testing.T) {
	t.Parallel()

	// This YAML fails at yaml.Unmarshal with "did not find expected key"
	// The error points to line 2 which has a colon, so no hint is added.
	dir := t.TempDir()
	content := "entries:\n  - path: /foo\n type: file"
	specPath := filepath.Join(dir, "spec.yaml")
	if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := spec.Load(specPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	msg := err.Error()

	if !strings.Contains(msg, "did not find expected key") {
		t.Errorf("expected 'did not find expected key' in error, got: %v", msg)
	}

	if strings.Contains(msg, "hint:") {
		t.Errorf("expected no hint for line with colon, got: %v", msg)
	}

	if !strings.Contains(msg, ">    2 |") {
		t.Errorf("expected > marker on line 2, got: %v", msg)
	}
}
