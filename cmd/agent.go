package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/yangyifan18/dotvibe/adapters"
	"github.com/yangyifan18/dotvibe/agentapi"
	"github.com/yangyifan18/dotvibe/bootstrap"
)

type agentDoctorOptions struct {
	JSON    bool
	Version string
}

var agentDoctorOpts agentDoctorOptions

var agentCmd = &cobra.Command{Use: "agent", Short: "Agent-facing dotvibe automation helpers"}
var agentDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Report agent-facing migration capabilities",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := agentDoctorOpts
		if opts.Version == "" {
			opts.Version = version
		}
		return runAgentDoctor(opts, cmd.OutOrStdout())
	},
}

func init() {
	agentDoctorCmd.Flags().BoolVar(&agentDoctorOpts.JSON, "json", false, "print JSON")
	agentCmd.AddCommand(agentDoctorCmd)
	rootCmd.AddCommand(agentCmd)
}

func runAgentDoctor(opts agentDoctorOptions, w io.Writer) error {
	report := agentapi.BuildDoctorReport(agentapi.DoctorOptions{
		Version:    opts.Version,
		Adapters:   adapters.AllAdapters(),
		ToolChecks: bootstrap.DetectTools(bootstrap.DefaultToolSpecs()),
	})
	if opts.JSON {
		return writeAgentJSON(w, report)
	}
	fmt.Fprintln(w, "dotvibe agent doctor")
	fmt.Fprintf(w, "OK: %v\n", report.OK)
	fmt.Fprintln(w, "Capabilities: export, import, setup, recipe, rollback, agent plans, stage import")
	for _, tool := range report.Tools {
		fmt.Fprintf(w, "  %s: detected=%v installed=%v\n", tool.ID, tool.Detected, tool.Install.Installed)
	}
	return nil
}

func writeAgentJSON(w io.Writer, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(w, string(data))
	return nil
}

type agentInventoryOptions struct{ JSON bool }

var agentInventoryOpts agentInventoryOptions

var agentInventoryCmd = &cobra.Command{
	Use:   "inventory",
	Short: "Print agent-facing migration inventory",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAgentInventory(agentInventoryOpts, cmd.OutOrStdout())
	},
}

func init() {
	agentInventoryCmd.Flags().BoolVar(&agentInventoryOpts.JSON, "json", false, "print JSON")
	agentCmd.AddCommand(agentInventoryCmd)
}

func runAgentInventory(opts agentInventoryOptions, w io.Writer) error {
	report := agentapi.BuildInventory(agentapi.InventoryOptions{Adapters: adapters.AllAdapters()})
	if opts.JSON {
		return writeAgentJSON(w, report)
	}
	fmt.Fprintln(w, "dotvibe agent inventory")
	for _, tool := range report.Tools {
		fmt.Fprintf(w, "  %s: detected=%v categories=%d\n", tool.ID, tool.Detected, len(tool.Categories))
	}
	fmt.Fprintln(w, "Profiles: full, project-memory, recipe")
	return nil
}
