package agentapi

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yangyifan18/dotvibe/backup"
)

type StageOptions struct {
	ArchiveDir string
	StageDir   string
	Plan       ImportPlan
	Manifest   *backup.Manifest
}

type StageResult struct {
	StageDir     string `json:"stage_dir"`
	FilesStaged  int    `json:"files_staged"`
	LocalCopies  int    `json:"local_copies"`
	PlanPath     string `json:"plan_path"`
	ManifestPath string `json:"manifest_path,omitempty"`
}

func StageImport(opts StageOptions) (StageResult, error) {
	if opts.ArchiveDir == "" {
		return StageResult{}, fmt.Errorf("archive dir is required")
	}
	if opts.StageDir == "" {
		return StageResult{}, fmt.Errorf("stage dir is required")
	}
	result := StageResult{StageDir: opts.StageDir, PlanPath: filepath.Join(opts.StageDir, "import-plan.json")}
	if err := createStageRoot(opts.StageDir); err != nil {
		return result, err
	}
	if err := os.MkdirAll(filepath.Join(opts.StageDir, "local"), 0755); err != nil {
		return result, err
	}
	if err := os.MkdirAll(filepath.Join(opts.StageDir, "files"), 0755); err != nil {
		return result, err
	}
	for _, entry := range opts.Plan.Entries {
		if entry.Path == "" {
			continue
		}
		src, err := safeStageJoin(opts.ArchiveDir, entry.Path)
		if err != nil {
			return result, err
		}
		dst, err := safeStageJoin(filepath.Join(opts.StageDir, "files"), entry.Path)
		if err != nil {
			return result, err
		}
		if err := copyStageFile(src, dst); err != nil {
			return result, err
		}
		result.FilesStaged++
		if entry.TargetPath != "" && entry.NeedsReview {
			localLogicalPath := entry.Path
			if entry.LocalStagePath != "" {
				localLogicalPath = entry.LocalStagePath
			}
			localDst, err := safeStageJoin(filepath.Join(opts.StageDir, "local"), localLogicalPath)
			if err != nil {
				return result, err
			}
			if err := copyStageFile(entry.TargetPath, localDst); err == nil {
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
	if err := writeNewFile(result.PlanPath, data, 0644); err != nil {
		return result, err
	}
	if opts.Manifest != nil {
		result.ManifestPath = filepath.Join(opts.StageDir, "manifest.json")
		manifestData, err := json.MarshalIndent(opts.Manifest, "", "  ")
		if err != nil {
			return result, err
		}
		if err := writeNewFile(result.ManifestPath, manifestData, 0644); err != nil {
			return result, err
		}
	}
	return result, nil
}

func createStageRoot(stageDir string) error {
	if stageDir == "." || stageDir == string(filepath.Separator) {
		return fmt.Errorf("stage dir must not be %q", stageDir)
	}
	parent := filepath.Dir(stageDir)
	if parent != "." {
		if err := os.MkdirAll(parent, 0755); err != nil {
			return err
		}
	}
	if err := os.Mkdir(stageDir, 0755); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("stage dir already exists: %s", stageDir)
		}
		return err
	}
	return nil
}

func safeStageJoin(root, logicalPath string) (string, error) {
	if err := validateStageLogicalPath(logicalPath); err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(filepath.Join(rootAbs, filepath.FromSlash(logicalPath)))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return "", err
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", fmt.Errorf("stage path escapes root: %s", logicalPath)
	}
	return targetAbs, nil
}

func validateStageLogicalPath(path string) error {
	if path == "" || path == "." || strings.HasPrefix(path, "/") || strings.Contains(path, "\\") {
		return fmt.Errorf("unsafe stage path: %s", path)
	}
	for _, part := range strings.Split(path, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("unsafe stage path: %s", path)
		}
	}
	return nil
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
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func writeNewFile(path string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
