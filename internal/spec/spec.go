package spec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/na4ma4/go-yamladv"
	"go.yaml.in/yaml/v3"
)

const contextLines = 5

var yamlLineRe = regexp.MustCompile(`yaml: line (\d+): `)

var yamlExpectedKeyRe = regexp.MustCompile(`yaml: line (\d+): did not find expected key`)

func hintForLine(lines []string, lineNum int) string {
	line := strings.TrimSpace(lines[lineNum-1])

	if line == "" || line == "-" {
		return "each entry needs at least `path:` and `type:` keys"
	}

	if !strings.Contains(line, ":") {
		value := strings.TrimLeft(strings.TrimPrefix(line, "-"), " ")
		return fmt.Sprintf("did you mean `- path: %s`?", value)
	}

	return ""
}

func yamlErrorWithContext(path string, data []byte, err error) error {
	matches := yamlLineRe.FindStringSubmatch(err.Error())
	if matches == nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	lineNum, _ := strconv.Atoi(matches[1])
	lines := strings.Split(string(data), "\n")

	start := min(max(lineNum-contextLines, 1), len(lines))
	end := min(lineNum+contextLines, len(lines))

	var b strings.Builder
	fmt.Fprintf(&b, "%s:%d: ", path, lineNum)

	msg := strings.TrimPrefix(err.Error(), matches[0])
	fmt.Fprintf(&b, "%s\n", msg)

	if yamlExpectedKeyRe.MatchString(err.Error()) {
		if hint := hintForLine(lines, lineNum); hint != "" {
			fmt.Fprintf(&b, "  hint: %s\n", hint)
		}
	}

	for n := start; n <= end; n++ {
		marker := "  "
		if n == lineNum {
			marker = "> "
		}
		fmt.Fprintf(&b, "  %s%4d | %s\n", marker, n, lines[n-1])
	}

	return errors.Join(errors.New(b.String()), err)
}

func Load(path string) (*Spec, error) {
	var data []byte
	{
		var err error
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, yamlErrorWithContext(path, data, err)
	}

	baseDir := filepath.Dir(path)
	if err := yamladv.Resolve(&root, baseDir); err != nil {
		return nil, fmt.Errorf("%s: include: %w", path, err)
	}

	var spec Spec
	if err := root.Decode(&spec); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
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
