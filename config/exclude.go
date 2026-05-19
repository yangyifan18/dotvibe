package config

import (
	"path/filepath"
	"strings"
)

var defaultExcludeDirs = []string{
	"auth.json",
	"telemetry",
	"cache",
	"session-env",
	"shell-snapshots",
}

type Excluder struct {
	customPatterns []string
}

func NewExcluder(customPatterns []string) *Excluder {
	return &Excluder{customPatterns: customPatterns}
}

func (e *Excluder) IsExcluded(path string) bool {
	segments := strings.Split(filepath.ToSlash(path), "/")
	for _, seg := range segments {
		for _, exc := range defaultExcludeDirs {
			if strings.HasPrefix(seg, exc) {
				return true
			}
		}
	}

	for _, pattern := range e.customPatterns {
		if matchGlob(pattern, path) {
			return true
		}
	}

	return false
}

func matchGlob(pattern, path string) bool {
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(filepath.ToSlash(path), "/")

	// Try matching against all suffixes of the path
	for i := 0; i <= len(pathParts); i++ {
		if matchGlobParts(patternParts, pathParts[i:]) {
			return true
		}
	}
	return false
}

func matchGlobParts(patterns, paths []string) bool {
	if len(patterns) == 0 {
		return len(paths) == 0
	}

	if patterns[0] == "*" {
		// * matches zero or more path parts
		for i := 0; i <= len(paths); i++ {
			if matchGlobParts(patterns[1:], paths[i:]) {
				return true
			}
		}
		return false
	}

	if len(paths) == 0 {
		return false
	}

	// Match pattern part against path part using filepath.Match
	pat := patterns[0]
	matched, err := filepath.Match(pat, paths[0])
	if err != nil {
		return false
	}

	if !matched {
		// Also try with * prefix and suffix for substring matching
		wrapped := "*" + pat + "*"
		matched, err = filepath.Match(wrapped, paths[0])
		if err != nil {
			return false
		}
	}

	if matched {
		return matchGlobParts(patterns[1:], paths[1:])
	}

	return false
}
