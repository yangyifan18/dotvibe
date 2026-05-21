package recipe

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const RecipeTextDiffMaxBytes = 1024 * 1024

const (
	TextKindText   = "text"
	TextKindBinary = "binary"
	TextKindLarge  = "large"
)

func ClassifyText(sample []byte, size int) string {
	if size > RecipeTextDiffMaxBytes {
		return TextKindLarge
	}
	for _, b := range sample {
		if b == 0 {
			return TextKindBinary
		}
	}
	if !utf8.Valid(sample) {
		return TextKindBinary
	}
	return TextKindText
}

func UnifiedTextDiff(oldName, newName string, oldData, newData []byte) string {
	oldLines := splitDiffLines(oldData)
	newLines := splitDiffLines(newData)
	var b strings.Builder
	fmt.Fprintf(&b, "--- %s\n", oldName)
	fmt.Fprintf(&b, "+++ %s\n", newName)
	max := len(oldLines)
	if len(newLines) > max {
		max = len(newLines)
	}
	for i := 0; i < max; i++ {
		var oldLine, newLine string
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}
		switch {
		case i >= len(oldLines):
			fmt.Fprintf(&b, "+%s\n", newLine)
		case i >= len(newLines):
			fmt.Fprintf(&b, "-%s\n", oldLine)
		case oldLine == newLine:
			fmt.Fprintf(&b, " %s\n", oldLine)
		default:
			fmt.Fprintf(&b, "-%s\n+%s\n", oldLine, newLine)
		}
	}
	return b.String()
}

func splitDiffLines(data []byte) []string {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}
