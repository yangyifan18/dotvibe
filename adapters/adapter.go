package adapters

const (
	CategoryConfig  = "config"
	CategoryMemory  = "memory"
	CategorySkills  = "skills"
	CategoryHistory = "history"
)

type FileEntry struct {
	SourcePath string
	InArchive  string
	Category   string
	Size       int64
}

type ExportOpts struct {
	WithHistory     bool
	ExcludePatterns []string
	OnlyTools       []string
}

type ToolStatus struct {
	Name       string
	Path       string
	Size       int64
	Projects   int
	Skills     int
	Agents     int
	Sessions   int
	ConfigFile string
}

type RestoreOpts struct {
	Force   bool
	Project string
}

type Adapter interface {
	Name() string
	ID() string
	Detect() bool
	ListFiles(opts ExportOpts) []FileEntry
	Status() ToolStatus
	RestoreFiles(entries []FileEntry, archiveDir string, opts RestoreOpts) error
}

func AllAdapters() []Adapter {
	return []Adapter{
		NewClaudeAdapter(),
		NewCodexAdapter(),
		NewOpenCodeAdapter(),
	}
}
