package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunRecipeInspectHumanAndJSON(t *testing.T) {
	path := buildRecipeCommandFixture(t, map[string]string{"codex-cli/agents/reviewer.md": "# Reviewer\n"})
	var human bytes.Buffer
	if err := runRecipeInspect(path, false, &human); err != nil {
		t.Fatalf("runRecipeInspect human: %v", err)
	}
	for _, want := range []string{"Recipe:", "Created:", "Risks: errors=", "Tools:", "Files:", "codex-cli"} {
		if !strings.Contains(human.String(), want) {
			t.Fatalf("human output missing %q: %s", want, human.String())
		}
	}
	if strings.Contains(human.String(), "Author: \n") {
		t.Fatalf("unexpected human output: %s", human.String())
	}
	var jsonOut bytes.Buffer
	if err := runRecipeInspect(path, true, &jsonOut); err != nil {
		t.Fatalf("runRecipeInspect json: %v", err)
	}
	if !strings.Contains(jsonOut.String(), `"name"`) || !strings.Contains(jsonOut.String(), `"files"`) {
		t.Fatalf("unexpected JSON output: %s", jsonOut.String())
	}
}
