package adapters

const (
	CategoryConfig   = "config"
	CategoryMemory   = "memory"
	CategorySkills   = "skills"
	CategoryHistory  = "history"
	CategorySettings = "settings"
	CategoryRules    = "rules"
	CategoryAgents   = "agents"
	CategoryCommands = "commands"
)

type FileEntry struct {
	SourcePath string
	InArchive  string
	Category   string
	Size       int64
}

const (
	RestoreWrite     = "write"
	RestoreSkip      = "skip"
	RestoreOverwrite = "overwrite"
)

type ExportOpts struct {
	WithHistory     bool
	ExcludePatterns []string
	OnlyTools       []string
}

type RecipeOpts struct {
	IncludeSettings bool
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

type RestorePlanEntry struct {
	FileEntry
	TargetPath string
	Action     string
	Reason     string
}

type RestoreSummary struct {
	Written     int
	Skipped     int
	Overwritten int
	Failed      int
}

type Adapter interface {
	Name() string
	ID() string
	Detect() bool
	ListFiles(opts ExportOpts) []FileEntry
	ListRecipeFiles(opts RecipeOpts) []FileEntry
	Status() ToolStatus
	FilterRestoreEntries(entries []FileEntry, opts RestoreOpts) []FileEntry
	PlanRestore(entries []FileEntry, opts RestoreOpts) ([]RestorePlanEntry, error)
	RestoreFiles(entries []FileEntry, archiveDir string, opts RestoreOpts) (RestoreSummary, error)
}

func AllAdapters() []Adapter {
	return []Adapter{
		NewClaudeAdapter(),
		NewCodexAdapter(),
		NewOpenCodeAdapter(),
	}
}
