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
	for _, want := range []string{"Changed: 1", "[codex-cli/agents]", "->", "+new"} {
		if !strings.Contains(human.String(), want) {
			t.Fatalf("human diff missing %q: %s", want, human.String())
		}
	}
	var jsonOut bytes.Buffer
	if err := runRecipeDiff(left, right, recipeDiffOptions{JSON: true}, &jsonOut); err != nil {
		t.Fatalf("runRecipeDiff json: %v", err)
	}
	if !strings.Contains(jsonOut.String(), `"added"`) || !strings.Contains(jsonOut.String(), `"changed"`) {
		t.Fatalf("unexpected JSON diff: %s", jsonOut.String())
	}
}

func TestRunRecipeDiffJSONContentIncludesHunk(t *testing.T) {
	left := buildRecipeCommandFixture(t, map[string]string{"codex-cli/agents/a.md": "old\n"})
	right := buildRecipeCommandFixture(t, map[string]string{"codex-cli/agents/a.md": "new\n"})
	var jsonOut bytes.Buffer
	if err := runRecipeDiff(left, right, recipeDiffOptions{JSON: true, Content: true}, &jsonOut); err != nil {
		t.Fatalf("runRecipeDiff json content: %v", err)
	}
	output := jsonOut.String()
	if !strings.Contains(output, `"content_diff_status": "text"`) || !strings.Contains(output, `+new`) {
		t.Fatalf("JSON content diff missing text hunk: %s", output)
	}
}
