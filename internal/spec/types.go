package spec

type IgnoreType string

const (
	IgnoreMissing     IgnoreType = "missing"
	IgnoreMode        IgnoreType = "mode"
	IgnoreOwner       IgnoreType = "owner"
	IgnoreGroup       IgnoreType = "group"
	IgnoreTypeCheck   IgnoreType = "type"
	IgnoreGlobNoMatch IgnoreType = "glob-no-match"
	IgnoreTarget      IgnoreType = "target"
)

type Entry struct {
	Owner    string `yaml:"owner,omitempty"`
	Group    string `yaml:"group,omitempty"`
	Mode     string `yaml:"mode,omitempty"`
	DirMode  string `yaml:"dir_mode,omitempty"`
	FileMode string `yaml:"file_mode,omitempty"`
}

type EntryItem struct {
	Path      string       `yaml:"path"`
	Type      Type         `yaml:"type"`
	Mode      string       `yaml:"mode,omitempty"`
	Owner     string       `yaml:"owner,omitempty"`
	Group     string       `yaml:"group,omitempty"`
	Target    string       `yaml:"target,omitempty"`
	Recursive bool         `yaml:"recursive,omitempty"`
	Ignore    []IgnoreType `yaml:"ignore,omitempty"`
}

type Spec struct {
	Defaults *Entry      `yaml:"defaults"`
	Entries  []EntryItem `yaml:"entries"`
}

type Type string

func (t Type) String() string {
	return string(t)
}

const (
	TypeFile      Type = "file"
	TypeDirectory Type = "directory"
	TypeSymlink   Type = "symlink"
)
