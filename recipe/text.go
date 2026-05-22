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
	common := lcsLines(oldLines, newLines)
	oldIndex, newIndex := 0, 0
	for _, line := range common {
		for oldIndex < len(oldLines) && oldLines[oldIndex] != line {
			fmt.Fprintf(&b, "-%s\n", oldLines[oldIndex])
			oldIndex++
		}
		for newIndex < len(newLines) && newLines[newIndex] != line {
			fmt.Fprintf(&b, "+%s\n", newLines[newIndex])
			newIndex++
		}
		fmt.Fprintf(&b, " %s\n", line)
		oldIndex++
		newIndex++
	}
	for oldIndex < len(oldLines) {
		fmt.Fprintf(&b, "-%s\n", oldLines[oldIndex])
		oldIndex++
	}
	for newIndex < len(newLines) {
		fmt.Fprintf(&b, "+%s\n", newLines[newIndex])
		newIndex++
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

func lcsLines(a, b []string) []string {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	if len(a) == 1 {
		for _, line := range b {
			if line == a[0] {
				return []string{a[0]}
			}
		}
		return nil
	}

	mid := len(a) / 2
	leftScores := lcsLengths(a[:mid], b)
	rightScores := lcsLengths(reverseLines(a[mid:]), reverseLines(b))
	split := 0
	best := -1
	for i := 0; i <= len(b); i++ {
		score := leftScores[i] + rightScores[len(b)-i]
		if score > best {
			best = score
			split = i
		}
	}

	left := lcsLines(a[:mid], b[:split])
	right := lcsLines(a[mid:], b[split:])
	return append(left, right...)
}

func lcsLengths(a, b []string) []int {
	prev := make([]int, len(b)+1)
	for _, aLine := range a {
		curr := make([]int, len(b)+1)
		for j, bLine := range b {
			if aLine == bLine {
				curr[j+1] = prev[j] + 1
				continue
			}
			if curr[j] > prev[j+1] {
				curr[j+1] = curr[j]
			} else {
				curr[j+1] = prev[j+1]
			}
		}
		prev = curr
	}
	return prev
}

func reverseLines(lines []string) []string {
	out := make([]string, len(lines))
	for i := range lines {
		out[len(lines)-1-i] = lines[i]
	}
	return out
}
