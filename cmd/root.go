package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dotvibe",
	Short: "Backup and restore vibe coding agent data",
	Long: `dotvibe backs up and restores data from AI coding tools
(Claude Code, Codex CLI, OpenCode) for migration between machines.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
}

func printErr(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", a...)
}
