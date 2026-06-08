package verifier

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/na4ma4/go-permbits"

	"github.com/na4ma4/verify-tree/internal/config"
	"github.com/na4ma4/verify-tree/internal/spec"
)

type CheckType string

const (
	CheckMissing     CheckType = "missing"
	CheckMode        CheckType = "mode"
	CheckOwner       CheckType = "owner"
	CheckGroup       CheckType = "group"
	CheckTypeCheck   CheckType = "type"
	CheckGlobNoMatch CheckType = "glob-no-match"
	CheckTarget      CheckType = "target"
)

type Violation struct {
	Path      string    `json:"path"`
	Check     CheckType `json:"check"`
	Expected  string    `json:"expected,omitempty"`
	Actual    string    `json:"actual,omitempty"`
	SpecIndex int       `json:"spec_index"`
}

type EntryResult struct {
	Path       string      `json:"path"`
	SpecIndex  int         `json:"spec_index"`
	Violations []Violation `json:"violations,omitempty"`
	Passed     bool        `json:"passed"`
	Skipped    bool        `json:"skipped,omitempty"`
}

type Result struct {
	Entries []EntryResult `json:"entries"`
	Passed  bool          `json:"passed"`
}

func Verify(specFile string, vars config.Variables) (*Result, error) {
	s, err := spec.Load(specFile)
	if err != nil {
		return nil, fmt.Errorf("loading spec: %w", err)
	}

	s.ApplyDefaults()

	renderEntry := func(e spec.EntryItem) spec.EntryItem {
		e.Path = vars.Render(e.Path)
		e.Owner = vars.Render(e.Owner)
		e.Group = vars.Render(e.Group)
		e.Mode = vars.Render(e.Mode)
		e.Target = vars.Render(e.Target)
		return e
	}

	var results []EntryResult

	for i, entry := range s.Entries {
		entry = renderEntry(entry)

		ignoreSet := make(map[spec.IgnoreType]struct{})
		for _, ig := range entry.Ignore {
			ignoreSet[ig] = struct{}{}
		}

		if entry.Recursive && entry.Type == "directory" {
			results = append(results, verifyRecursiveDirectory(entry.Path, entry, ignoreSet, i)...)
			continue
		}

		if strings.ContainsAny(entry.Path, "*?[") {
			results = append(results, verifyGlob(entry.Path, entry, ignoreSet, i))
			continue
		}

		results = append(results, verifyPath(entry.Path, entry.Target, entry, ignoreSet, i))
	}

	passed := true
	for _, r := range results {
		if !r.Passed && !r.Skipped {
			passed = false
		}
	}

	return &Result{Entries: results, Passed: passed}, nil
}

func verifyRecursiveDirectory(
	path string,
	entry spec.EntryItem,
	ignoreSet map[spec.IgnoreType]struct{},
	specIndex int,
) []EntryResult {
	var results []EntryResult

	rootResult := verifyPath(path, "", entry, ignoreSet, specIndex)
	results = append(results, rootResult)

	_ = filepath.WalkDir(path, func(walkPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // ignore walk errors
		}
		if walkPath == path {
			return nil
		}

		childEntry := entry
		childEntry.Path = walkPath

		switch {
		case d.IsDir():
			childEntry.Type = spec.TypeDirectory
		case d.Type().IsRegular():
			childEntry.Type = spec.TypeFile
		case d.Type()&os.ModeSymlink != 0:
			childEntry.Type = spec.TypeSymlink
			target, _ := os.Readlink(walkPath)
			childEntry.Target = target
		}

		childResult := verifyPath(walkPath, childEntry.Target, childEntry, ignoreSet, specIndex)
		results = append(results, childResult)

		return nil
	})

	return results
}

func globBaseDir(pattern string) string {
	idx := strings.LastIndexAny(pattern, "*?[")
	if idx < 0 {
		return filepath.Dir(pattern)
	}
	base := pattern[:idx]
	slash := strings.LastIndexAny(base, "/\\")
	if slash < 0 {
		return "."
	}
	return base[:slash]
}

func globFilePattern(pattern string) string {
	idx := strings.LastIndex(pattern, "/")
	if idx < 0 {
		return pattern
	}
	return pattern[idx+1:]
}

func verifyGlob(
	pattern string,
	entry spec.EntryItem,
	ignoreSet map[spec.IgnoreType]struct{},
	specIndex int,
) EntryResult {
	baseDir := globBaseDir(pattern)
	filePat := globFilePattern(pattern)

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return EntryResult{
			Path:      pattern,
			SpecIndex: specIndex,
			Violations: []Violation{{
				Path:     pattern,
				Check:    CheckGlobNoMatch,
				Expected: "glob match",
				Actual:   fmt.Sprintf("invalid glob: %v", err),
			}},
			Passed: false,
		}
	}

	if entry.Recursive {
		_ = filepath.WalkDir(baseDir, func(walkPath string, _ fs.DirEntry, err error) error {
			if err != nil {
				return nil //nolint:nilerr // ignore walk errors
			}
			if walkPath == baseDir {
				return nil
			}
			name := filepath.Base(walkPath)
			match, matchErr := filepath.Match(filePat, name)
			if matchErr == nil && match {
				matches = append(matches, walkPath)
			}
			return nil
		})
	}

	if len(matches) == 0 {
		if _, ignored := ignoreSet[spec.IgnoreGlobNoMatch]; ignored {
			return EntryResult{
				Path:      pattern,
				SpecIndex: specIndex,
				Skipped:   true,
				Passed:    true,
			}
		}
		return EntryResult{
			Path:      pattern,
			SpecIndex: specIndex,
			Violations: []Violation{{
				Path:     pattern,
				Check:    CheckGlobNoMatch,
				Expected: "at least one match",
				Actual:   "no files matched",
			}},
			Passed: false,
		}
	}

	var childResults []EntryResult
	for _, m := range matches {
		childResult := verifyPath(m, entry.Target, entry, ignoreSet, specIndex)
		childResults = append(childResults, childResult)
	}

	return mergeEntryResults(pattern, specIndex, childResults)
}

func mergeEntryResults(pattern string, specIndex int, results []EntryResult) EntryResult {
	var allViolations []Violation
	passed := true

	for _, r := range results {
		if !r.Passed && !r.Skipped {
			passed = false
		}
		allViolations = append(allViolations, r.Violations...)
	}

	return EntryResult{
		Path:       pattern,
		SpecIndex:  specIndex,
		Violations: allViolations,
		Passed:     passed,
	}
}

func verifyPath(
	path, renderedTarget string,
	entry spec.EntryItem,
	ignoreSet map[spec.IgnoreType]struct{},
	specIndex int,
) EntryResult {
	var violations []Violation

	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if _, ignored := ignoreSet[spec.IgnoreMissing]; ignored {
				return EntryResult{
					Path:      path,
					SpecIndex: specIndex,
					Skipped:   true,
					Passed:    true,
				}
			}
			return EntryResult{
				Path:      path,
				SpecIndex: specIndex,
				Violations: []Violation{{
					Path:     path,
					Check:    CheckMissing,
					Expected: "exists",
					Actual:   "not found",
				}},
				Passed: false,
			}
		}
		return EntryResult{
			Path:      path,
			SpecIndex: specIndex,
			Violations: []Violation{{
				Path:   path,
				Check:  CheckMissing,
				Actual: err.Error(),
			}},
			Passed: false,
		}
	}

	violations = checkType(path, info, entry.Type, ignoreSet, violations)
	violations = checkMode(path, info, entry.Mode, ignoreSet, violations)

	if entry.Type == "symlink" {
		violations = checkTarget(path, info, renderedTarget, ignoreSet, violations)
	}

	if entry.Owner != "" {
		violations = checkOwner(path, info, entry.Owner, ignoreSet, violations)
	}

	if entry.Group != "" {
		violations = checkGroup(path, info, entry.Group, ignoreSet, violations)
	}

	passed := len(violations) == 0
	return EntryResult{
		Path:       path,
		SpecIndex:  specIndex,
		Violations: violations,
		Passed:     passed,
	}
}

func checkType(
	path string,
	info os.FileInfo,
	expectedType spec.Type,
	ignoreSet map[spec.IgnoreType]struct{},
	violations []Violation,
) []Violation {
	if _, ignored := ignoreSet[spec.IgnoreTypeCheck]; ignored {
		return violations
	}

	var actualType spec.Type
	switch {
	case info.Mode().IsRegular():
		actualType = spec.TypeFile
	case info.IsDir():
		actualType = spec.TypeDirectory
	case info.Mode()&os.ModeSymlink != 0:
		actualType = spec.TypeSymlink
	default:
		actualType = spec.Type(fmt.Sprintf("other (%v)", info.Mode()))
	}

	if actualType != expectedType {
		violations = append(violations, Violation{
			Path:     path,
			Check:    CheckTypeCheck,
			Expected: expectedType.String(),
			Actual:   actualType.String(),
		})
	}

	return violations
}

func parseMode(s string) (fs.FileMode, error) {
	if s == "" {
		return 0, errors.New("empty mode string")
	}

	if s[0] >= '0' && s[0] <= '9' {
		v, err := strconv.ParseUint(s, 8, 32)
		if err != nil {
			return 0, fmt.Errorf("parsing octal mode %q: %w", s, err)
		}
		return fs.FileMode(v), nil
	}

	return permbits.FromString(s)
}

func checkMode(
	path string,
	info os.FileInfo,
	expectedMode string,
	ignoreSet map[spec.IgnoreType]struct{},
	violations []Violation,
) []Violation {
	if expectedMode == "" {
		return violations
	}

	if _, ignored := ignoreSet[spec.IgnoreMode]; ignored {
		return violations
	}

	want, err := parseMode(expectedMode)
	if err != nil {
		violations = append(violations, Violation{
			Path:     path,
			Check:    CheckMode,
			Expected: expectedMode,
			Actual:   fmt.Sprintf("invalid mode: %v", err),
		})
		return violations
	}

	perm := info.Mode().Perm()
	if perm != want {
		violations = append(violations, Violation{
			Path:     path,
			Check:    CheckMode,
			Expected: fmt.Sprintf("%o", want),
			Actual:   fmt.Sprintf("%o", perm),
		})
	}

	return violations
}

func checkOwner(
	path string,
	info os.FileInfo,
	expectedOwner string,
	ignoreSet map[spec.IgnoreType]struct{},
	violations []Violation,
) []Violation {
	if _, ignored := ignoreSet[spec.IgnoreOwner]; ignored {
		return violations
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return violations
	}

	uid := stat.Uid
	expectedUID := lookupUser(expectedOwner)

	if uid != expectedUID {
		actualName := uidToString(uid)
		violations = append(violations, Violation{
			Path:     path,
			Check:    CheckOwner,
			Expected: expectedOwner,
			Actual:   actualName,
		})
	}

	return violations
}

func checkGroup(
	path string,
	info os.FileInfo,
	expectedGroup string,
	ignoreSet map[spec.IgnoreType]struct{},
	violations []Violation,
) []Violation {
	if _, ignored := ignoreSet[spec.IgnoreGroup]; ignored {
		return violations
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return violations
	}

	gid := stat.Gid
	expectedGID := lookupGroup(expectedGroup)

	if gid != expectedGID {
		actualName := gidToString(gid)
		violations = append(violations, Violation{
			Path:     path,
			Check:    CheckGroup,
			Expected: expectedGroup,
			Actual:   actualName,
		})
	}

	return violations
}

func checkTarget(
	path string,
	_ os.FileInfo,
	expectedTarget string,
	ignoreSet map[spec.IgnoreType]struct{},
	violations []Violation,
) []Violation {
	if _, ignored := ignoreSet[spec.IgnoreTarget]; ignored {
		return violations
	}

	actual, err := os.Readlink(path)
	if err != nil {
		violations = append(violations, Violation{
			Path:     path,
			Check:    CheckTarget,
			Expected: expectedTarget,
			Actual:   fmt.Sprintf("readlink error: %v", err),
		})
		return violations
	}

	if actual != expectedTarget {
		violations = append(violations, Violation{
			Path:     path,
			Check:    CheckTarget,
			Expected: expectedTarget,
			Actual:   actual,
		})
	}

	return violations
}

func lookupUser(name string) uint32 {
	if u, uErr := user.Lookup(name); uErr == nil {
		if uid, err := strconv.ParseUint(u.Uid, 10, 32); err == nil {
			return uint32(uid)
		}
	}

	if uid, err := strconv.ParseUint(name, 10, 32); err == nil {
		return uint32(uid)
	}
	return 0
}

func lookupGroup(name string) uint32 {
	if g, gErr := user.LookupGroup(name); gErr == nil {
		if gid, err := strconv.ParseUint(g.Gid, 10, 32); err == nil {
			return uint32(gid)
		}
	}

	if gid, err := strconv.ParseUint(name, 10, 32); err == nil {
		return uint32(gid)
	}
	return 0
}

func uidToString(uid uint32) string {
	if u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10)); err == nil {
		return u.Username
	}
	return strconv.FormatUint(uint64(uid), 10)
}

func gidToString(gid uint32) string {
	if g, err := user.LookupGroupId(strconv.FormatUint(uint64(gid), 10)); err == nil {
		return g.Name
	}
	return strconv.FormatUint(uint64(gid), 10)
}

func (r *Result) ExitCode() int {
	if r.Passed {
		return 0
	}
	return 1
}
