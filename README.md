# verify-tree

Verify a file tree matches a YAML specification of paths, types, ownership, and permissions — with template variable support.

## Usage

```sh
verify-tree --spec spec.yaml \
  --var root=/opt/app \
  --var app_owner=myuser \
  --var app_group=mygroup \
  --var storage=/data \
  --var program=myapp
```

Flags:

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--spec` | `-s` | `spec.yaml` | Path to specification YAML |
| `--var` | `-v` | — | Template variable (`key=value`, repeatable) |
| `--var-file` | — | — | Variables file (`key=value` per line) |
| `--format` | `-f` | `text` | Output format: `text`, `json`, or `yaml` |
| `--no-colour` | — | `false` | Disable colored output |
| `--no-color` | — | `false` | Disable colored output (alias) |

Variable precedence (first wins): CLI `--var` > env `VERIFY_TREE_VAR_*` > `--var-file`.

Exit code: `0` if all entries pass, `1` if any violations are found.

## Spec format

```yaml
defaults:
  owner: "{{ app_owner }}"
  group: "{{ app_group }}"
  file_mode: "0640"
  dir_mode: "0750"

entries:
  - path: "{{ root }}"
    type: directory
    mode: "0755"

  - path: "{{ root }}/bin/something"
    type: file
    mode: "0755"

  - path: "{{ root }}/etc/config.yml"
    type: file

  - path: "{{ root }}/logs"
    type: symlink
    target: "{{ storage }}/{{ program }}/logs"
    ignore: [missing]

  - path: "{{ root }}/etc/*.toml"
    type: file
    mode: "0600"
```

### Entry fields

| Field | Required | Description |
|-------|----------|-------------|
| `path` | yes | File path, supports `{{ var }}` templates and globs (`*`, `?`, `[...]`) |
| `type` | yes | `file`, `directory`, or `symlink` |
| `mode` | no | Octal permission string (e.g. `0640`, `0755`, `2770`). Falls back to `defaults.file_mode` / `defaults.dir_mode` |
| `owner` | no | Username or UID. Falls back to `defaults.owner` |
| `group` | no | Group name or GID. Falls back to `defaults.group` |
| `target` | for symlinks | Expected symlink target path |
| `recursive` | no | Walk directory tree and verify all children against this entry's spec |
| `ignore` | no | List of check types to skip: `missing`, `mode`, `owner`, `group`, `type`, `glob-no-match`, `target` |

### Ignore types

| Value | When to use |
|-------|-------------|
| `missing` | Path is optional and may not exist |
| `glob-no-match` | Glob pattern may legitimately match zero files |
| `mode` | Skip permission check for this path |
| `owner` | Skip owner check |
| `group` | Skip group check |
| `type` | Skip file type check |
| `target` | Skip symlink target check |

### Recursive

When `recursive: true` on a `directory` entry, the tool walks the entire tree
under that path and verifies every file, directory, and symlink against the
entry's mode/owner/group settings. The `type` check only applies to the root
path; children inherit the remaining checks.

### YAML error context

When the spec file has YAML syntax errors, the tool displays the error with a
window of surrounding lines and a `>` marker on the offending line:

```
spec.yaml:4: did not find expected ',' or '}'
  >    4 |   bad: [unclosed
       5 |   e: 5
       6 |   f: 6
       7 |   g: 7

yaml: line 4: did not find expected ',' or '}'
```

If the error is `did not find expected key`, a hint is also shown:

```
spec.yaml:2: did not find expected key
  hint: did you mean `- path: /some/path`?
  >    2 |   - /some/path
```

## Installation

```sh
go install github.com/na4ma4/verify-tree/cmd/verify-tree@latest
```
