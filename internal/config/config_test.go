package config_test

import (
	"os"
	"testing"

	"github.com/na4ma4/verify-tree/internal/config"
)

func TestVariables_Render(t *testing.T) {
	t.Parallel()
	v := config.Variables{
		"root": "/opt/app",
		"user": "admin",
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"{{ root }}", "/opt/app"},
		{"{{ root }}/data", "/opt/app/data"},
		{"{{ root }}/{{ user }}/config", "/opt/app/admin/config"},
		{"no-template", "no-template"},
		{"{{ unknown }}", "{{ unknown }}"},
	}

	for _, tt := range tests {
		result := v.Render(tt.input)
		if result != tt.expected {
			t.Errorf("Render(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestVariables_MergeFrom(t *testing.T) {
	t.Parallel()
	base := config.Variables{"a": "1", "b": "2"}
	override := config.Variables{"b": "3", "c": "4"}

	base.MergeFrom(override)

	if base["a"] != "1" {
		t.Errorf("a = %q, want 1", base["a"])
	}
	if base["b"] != "2" {
		t.Errorf("b = %q, want 2 (should not be overridden)", base["b"])
	}
	if base["c"] != "4" {
		t.Errorf("c = %q, want 4", base["c"])
	}
}

func TestVariables_LoadFromEnv(t *testing.T) {
	t.Setenv("VT_VAR_TEST_KEY", "test-value")

	v := config.Variables{}
	v.LoadFromEnv()

	if v["test_key"] != "test-value" {
		t.Errorf("test_key = %q, want %q", v["test_key"], "test-value")
	}
}

func TestVariables_LoadFromFile(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	path := tmp + "/vars.txt"
	_ = os.WriteFile(path, []byte("key1=value1\nkey2=value2\n"), 0o644)

	v := config.Variables{}
	if err := v.LoadFromFile(path); err != nil {
		t.Fatal(err)
	}

	if v["key1"] != "value1" {
		t.Errorf("key1 = %q, want value1", v["key1"])
	}
	if v["key2"] != "value2" {
		t.Errorf("key2 = %q, want value2", v["key2"])
	}
}

func TestVariables_CollectUnused(t *testing.T) {
	t.Parallel()
	v := config.Variables{
		"root":   "/opt",
		"unused": "nope",
	}

	unused := v.CollectUnused("hello {{ root }} world")

	if len(unused) != 1 || unused[0] != "unused" {
		t.Errorf("unused = %v, want [unused]", unused)
	}
}
