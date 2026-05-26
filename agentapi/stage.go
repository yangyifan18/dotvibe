package agentapi

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type StageOptions struct {
	ArchiveDir string
	StageDir   string
	Plan       ImportPlan
}

type StageResult struct {
	StageDir    string `json:"stage_dir"`
	FilesStaged int    `json:"files_staged"`
	LocalCopies int    `json:"local_copies"`
	PlanPath    string `json:"plan_path"`
}

func StageImport(opts StageOptions) (StageResult, error) {
	if opts.ArchiveDir == "" {
		return StageResult{}, fmt.Errorf("archive dir is required")
	}
	if opts.StageDir == "" {
		return StageResult{}, fmt.Errorf("stage dir is required")
	}
	result := StageResult{StageDir: opts.StageDir, PlanPath: filepath.Join(opts.StageDir, "import-plan.json")}
	if err := os.MkdirAll(filepath.Join(opts.StageDir, "files"), 0755); err != nil {
		return result, err
	}
	if err := os.MkdirAll(filepath.Join(opts.StageDir, "local"), 0755); err != nil {
		return result, err
	}
	for _, entry := range opts.Plan.Entries {
		if entry.Path == "" {
			continue
		}
		if err := copyStageFile(filepath.Join(opts.ArchiveDir, filepath.FromSlash(entry.Path)), filepath.Join(opts.StageDir, "files", filepath.FromSlash(entry.Path))); err != nil {
			return result, err
		}
		result.FilesStaged++
		if entry.TargetPath != "" && entry.NeedsReview {
			if err := copyStageFile(entry.TargetPath, filepath.Join(opts.StageDir, "local", filepath.FromSlash(entry.Path))); err == nil {
				result.LocalCopies++
			} else if !os.IsNotExist(err) {
				return result, err
			}
		}
	}
	data, err := json.MarshalIndent(opts.Plan, "", "  ")
	if err != nil {
		return result, err
	}
	if err := os.WriteFile(result.PlanPath, data, 0644); err != nil {
		return result, err
	}
	return result, nil
}

func copyStageFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
