package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "dotvibe",
	Short:   "Backup and restore vibe coding agent data",
	Version: version,
	Long: `dotvibe backs up and restores data from AI coding tools
(Claude Code, Codex CLI, OpenCode) for migration between machines.

Usage:
  dotvibe status              Show detected tools
  dotvibe export              Create a backup
  dotvibe list <archive>      Show backup contents
  dotvibe diff <a> <b>        Compare two backups
  dotvibe import <archive>    Restore from backup`,
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
