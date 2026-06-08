# verify-tree — agent notes

## Build & test

```sh
mage TestLocal          # lint + test
go build ./...          # compile check
mage golang:lint        # golangci-lint
mage golang:test        # tests with race detector
```

## Key packages

| Package | Purpose |
|---------|---------|
| `cmd/verify-tree/` | Cobra CLI, flag parsing, variable collection |
| `internal/config/` | Template variable resolution (CLI > env > file) |
| `internal/spec/` | YAML spec types, loading via go-yamladv, defaults |
| `internal/verifier/` | Core verification logic (type/mode/owner/group/target) |
| `internal/output/` | Output formatting: text (color/plain), JSON, YAML via `output.Format` |

## Dependencies

- `github.com/na4ma4/go-yamladv` — YAML with `!include` tag support
- `github.com/na4ma4/go-permbits` — Permission bit parsing and comparison
- `github.com/spf13/cobra` — CLI framework
- `go.yaml.in/yaml/v3` — YAML v3 library (used by go-yamladv)

## Spec semantics

- Defaults apply only when an entry does **not** specify its own value
- Owner/group checked by name first, falls back to UID/GID
- `recursive: true` on a directory walks the tree and checks all children
- `recursive: true` on a glob recursively searches subdirectories
- Glob expansion uses `filepath.Glob` for exact matches
- Recursive glob walks the base directory and matches each entry's name against the file pattern
