package rollback

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	OperationRecipeApply = "recipe-apply"

	ActionWrite     = "write"
	ActionOverwrite = "overwrite"
	ActionSave      = "save"

	StatusPending = "pending"
	StatusApplied = "applied"
	StatusFailed  = "failed"

	BeforeMissing = "missing"
	BeforeFile    = "file"
)

type RollbackRecord struct {
	ID           string          `json:"id"`
	Operation    string          `json:"operation"`
	Created      time.Time       `json:"created"`
	RecipePath   string          `json:"recipe_path"`
	RecipeDigest string          `json:"recipe_digest"`
	RecipeName   string          `json:"recipe_name"`
	Entries      []RollbackEntry `json:"entries"`
}

type RollbackEntry struct {
	LogicalPath  string `json:"logical_path"`
	TargetPath   string `json:"target_path"`
	Action       string `json:"action"`
	Status       string `json:"status"`
	Error        string `json:"error,omitempty"`
	BeforeState  string `json:"before_state"`
	BeforeSHA256 string `json:"before_sha256,omitempty"`
	BeforeBlob   string `json:"before_blob,omitempty"`
	AfterSHA256  string `json:"after_sha256,omitempty"`
	SavedCopy    string `json:"saved_copy,omitempty"`
}

type Store struct {
	root string
}

func NewStore(root string) Store {
	return Store{root: root}
}

func DefaultStateRoot() string {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "dotvibe")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "dotvibe")
}

func (s Store) RollbacksDir() string         { return filepath.Join(s.root, "rollbacks") }
func (s Store) IncomingDir(id string) string { return filepath.Join(s.root, "incoming", id) }
func (s Store) RecordDir(id string) string   { return filepath.Join(s.RollbacksDir(), id) }
func (s Store) RecordPath(id string) string  { return filepath.Join(s.RecordDir(id), "rollback.json") }

func (s Store) Save(record RollbackRecord) error {
	if record.ID == "" {
		return fmt.Errorf("rollback record id is empty")
	}
	recordDir := s.RecordDir(record.ID)
	if err := os.MkdirAll(recordDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(recordDir, ".rollback-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0644); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, s.RecordPath(record.ID))
}

func (s Store) Load(id string) (RollbackRecord, error) {
	data, err := os.ReadFile(s.RecordPath(id))
	if err != nil {
		return RollbackRecord{}, err
	}
	var record RollbackRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return RollbackRecord{}, err
	}
	return record, nil
}

func (s Store) List() ([]RollbackRecord, error) {
	entries, err := os.ReadDir(s.RollbacksDir())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var records []RollbackRecord
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		record, err := s.Load(entry.Name())
		if err == nil {
			records = append(records, record)
		}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Created.After(records[j].Created) })
	return records, nil
}

func (r RollbackRecord) FilterEntries(path string) []RollbackEntry {
	if path == "" {
		return r.Entries
	}
	clean := strings.TrimRight(filepath.Clean(path), string(os.PathSeparator))
	logical := strings.TrimRight(path, "/")
	var out []RollbackEntry
	for _, entry := range r.Entries {
		if entry.LogicalPath == path || strings.HasPrefix(entry.LogicalPath, logical+"/") || entry.TargetPath == path || strings.HasPrefix(filepath.Clean(entry.TargetPath), clean+string(os.PathSeparator)) {
			out = append(out, entry)
		}
	}
	return out
}

func BlobPath(recordDir string, sum string) string {
	if len(sum) < 2 {
		return filepath.Join(recordDir, "files", "sha256", sum)
	}
	return filepath.Join(recordDir, "files", "sha256", sum[:2], sum[2:])
}

func WriteBlob(recordDir string, data []byte) (string, string, error) {
	sumBytes := sha256.Sum256(data)
	sum := fmt.Sprintf("%x", sumBytes)
	path := BlobPath(recordDir, sum)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", "", err
	}
	rel, err := filepath.Rel(recordDir, path)
	if err != nil {
		return "", "", err
	}
	return sum, rel, nil
}

type PruneOptions struct {
	Keep      int
	OlderThan time.Duration
	DryRun    bool
	Now       time.Time
}

type PrunePlan struct {
	DeletedIDs []string
}

func (s Store) Prune(opts PruneOptions) (PrunePlan, error) {
	records, err := s.List()
	if err != nil {
		return PrunePlan{}, err
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	deleteSet := map[string]bool{}
	if opts.Keep > 0 && len(records) > opts.Keep {
		for _, record := range records[opts.Keep:] {
			deleteSet[record.ID] = true
		}
	}
	if opts.OlderThan > 0 {
		for _, record := range records {
			if opts.Now.Sub(record.Created) > opts.OlderThan {
				deleteSet[record.ID] = true
			}
		}
	}
	var plan PrunePlan
	for id := range deleteSet {
		plan.DeletedIDs = append(plan.DeletedIDs, id)
	}
	sort.Strings(plan.DeletedIDs)
	if opts.DryRun {
		return plan, nil
	}
	for _, id := range plan.DeletedIDs {
		if err := os.RemoveAll(s.RecordDir(id)); err != nil {
			return plan, err
		}
		if err := os.RemoveAll(s.IncomingDir(id)); err != nil {
			return plan, err
		}
	}
	return plan, nil
}
