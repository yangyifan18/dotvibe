package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yangyifan18/dotvibe/agentapi"
)

func TestRunAgentDoctorJSON(t *testing.T) {
	var out bytes.Buffer
	err := runAgentDoctor(agentDoctorOptions{JSON: true, Version: "test"}, &out)
	if err != nil {
		t.Fatalf("runAgentDoctor: %v", err)
	}
	var report agentapi.DoctorReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if report.DotvibeVersion != "test" || !report.Capabilities.AgentDoctor {
		t.Fatalf("report = %#v", report)
	}
}

func TestRunAgentDoctorHuman(t *testing.T) {
	var out bytes.Buffer
	if err := runAgentDoctor(agentDoctorOptions{JSON: false, Version: "test"}, &out); err != nil {
		t.Fatalf("runAgentDoctor human: %v", err)
	}
	if !strings.Contains(out.String(), "dotvibe agent doctor") || !strings.Contains(out.String(), "Capabilities") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}
