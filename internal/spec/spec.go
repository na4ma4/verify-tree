package spec

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/na4ma4/go-yamladv"
	"go.yaml.in/yaml/v3"
)

func Load(path string) (*Spec, error) {
	var data []byte
	{
		var err error
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading spec file: %w", err)
		}
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parsing spec YAML: %w", err)
	}

	baseDir := filepath.Dir(path)
	if err := yamladv.Resolve(&root, baseDir); err != nil {
		return nil, fmt.Errorf("resolving includes: %w", err)
	}

	var spec Spec
	if err := root.Decode(&spec); err != nil {
		return nil, fmt.Errorf("decoding spec: %w", err)
	}

	if err := spec.validate(); err != nil {
		return nil, err
	}

	return &spec, nil
}

func (s *Spec) validate() error {
	for i, entry := range s.Entries {
		if entry.Path == "" {
			return fmt.Errorf("entry %d: path is required", i)
		}
		switch entry.Type {
		case TypeFile, TypeDirectory, TypeSymlink:
		case "":
			return fmt.Errorf("entry %d (%q): type is required", i, entry.Path)
		default:
			return fmt.Errorf("entry %d (%q): unsupported type %q", i, entry.Path, entry.Type)
		}
	}
	return nil
}

func (s *Spec) ApplyDefaults() {
	if s.Defaults == nil {
		return
	}

	for i := range s.Entries {
		e := &s.Entries[i]

		if e.Owner == "" && s.Defaults.Owner != "" {
			e.Owner = s.Defaults.Owner
		}
		if e.Group == "" && s.Defaults.Group != "" {
			e.Group = s.Defaults.Group
		}

		if e.Mode == "" {
			if e.Type == TypeDirectory && s.Defaults.DirMode != "" {
				e.Mode = s.Defaults.DirMode
			} else if e.Type == TypeFile && s.Defaults.FileMode != "" {
				e.Mode = s.Defaults.FileMode
			}
		}
	}
}
