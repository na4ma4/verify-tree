package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var varRe = regexp.MustCompile(`\{\{\s*(\w+)\s*\}\}`)

type Variables map[string]string

func (v Variables) Render(s string) string {
	return varRe.ReplaceAllStringFunc(s, func(match string) string {
		key := strings.TrimSpace(match[2 : len(match)-2])
		if val, ok := v[key]; ok {
			return val
		}
		return match
	})
}

func (v Variables) Add(key, value string) {
	v[key] = value
}

func (v Variables) MergeFrom(other Variables) {
	for k, val := range other {
		if _, exists := v[k]; !exists {
			v[k] = val
		}
	}
}

//nolint:mnd // k=v is 2
func (v Variables) LoadFromEnv() {
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			continue
		}
		if after, ok := strings.CutPrefix(parts[0], "VT_VAR_"); ok {
			key := after
			key = strings.ToLower(key)
			if _, exists := v[key]; !exists {
				v[key] = parts[1]
			}
		}
	}
}

//nolint:mnd // k=v is 2
func (v Variables) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading variables file: %w", err)
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if _, exists := v[key]; !exists {
			v[key] = value
		}
	}

	return nil
}

func (v Variables) CollectUnused(s string) []string {
	used := map[string]struct{}{}
	for _, match := range varRe.FindAllStringSubmatch(s, -1) {
		used[match[1]] = struct{}{}
	}

	var unused []string
	for k := range v {
		if _, ok := used[k]; !ok {
			unused = append(unused, k)
		}
	}
	return unused
}
