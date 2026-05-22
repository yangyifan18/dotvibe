package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunRecipeDiffHumanAndJSON(t *testing.T) {
	left := buildRecipeCommandFixture(t, map[string]string{"codex-cli/agents/a.md": "old\n"})
	right := buildRecipeCommandFixture(t, map[string]string{"codex-cli/agents/a.md": "new\n", "codex-cli/agents/b.md": "added\n"})
	var human bytes.Buffer
	if err := runRecipeDiff(left, right, recipeDiffOptions{Content: true}, &human); err != nil {
		t.Fatalf("runRecipeDiff human: %v", err)
	}
	if !strings.Contains(human.String(), "Changed: 1") || !strings.Contains(human.String(), "+new") {
		t.Fatalf("unexpected human diff: %s", human.String())
	}
	var jsonOut bytes.Buffer
	if err := runRecipeDiff(left, right, recipeDiffOptions{JSON: true}, &jsonOut); err != nil {
		t.Fatalf("runRecipeDiff json: %v", err)
	}
	if !strings.Contains(jsonOut.String(), `"added"`) || !strings.Contains(jsonOut.String(), `"changed"`) {
		t.Fatalf("unexpected JSON diff: %s", jsonOut.String())
	}
}
