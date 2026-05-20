package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/yangyifan18/dotvibe/bootstrap"
)

var (
	setupInstall bool
	setupYes     bool
	setupOnly    string
	setupBases   []string
)

var setupCmd = &cobra.Command{
	Use:   "setup [archive]",
	Short: "Bootstrap supported coding agents and optionally restore a backup",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		archive := ""
		if len(args) == 1 {
			archive = args[0]
		}

		results := bootstrap.DetectTools(bootstrap.DefaultToolSpecs())
		printSetupPlan(cmd.OutOrStdout(), results, archive)
		if !setupInstall {
			return nil
		}

		commands := buildInstallCommandPlan(results)
		if len(commands) > 0 {
			if !setupYes && !confirmSetupInstall(commands) {
				fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
				return nil
			}
			for _, installCommand := range commands {
				if err := runInstallCommand(cmd.OutOrStdout(), installCommand); err != nil {
					return err
				}
			}
		}

		if archive != "" {
			return runSetupRestore(archive)
		}
		return nil
	},
}

func init() {
	setupCmd.Flags().BoolVar(&setupInstall, "install", false, "run safe install commands after confirmation")
	setupCmd.Flags().BoolVarP(&setupYes, "yes", "y", false, "skip setup confirmation")
	setupCmd.Flags().StringVar(&setupOnly, "only", "", "restore only specified tools after setup")
	setupCmd.Flags().StringSliceVar(&setupBases, "base", nil, "base archive for incremental restore")
	rootCmd.AddCommand(setupCmd)
}

func printSetupPlan(w io.Writer, results []bootstrap.ToolCheckResult, archive string) {
	fmt.Fprintln(w, "dotvibe setup plan")
	for _, result := range results {
		if result.Installed {
			fmt.Fprintf(w, "  %s: installed (%s)\n", result.Name, result.FoundBinary)
			continue
		}

		fmt.Fprintf(w, "  %s: missing\n", result.Name)
		for _, installCommand := range result.InstallCommands {
			label := installCommand.Manager
			if !isAutoRunnableInstallCommand(installCommand) {
				label += ", manual-review"
			}
			fmt.Fprintf(w, "    [%s] %s\n", label, installCommand.Command)
		}
	}
	if archive != "" {
		fmt.Fprintf(w, "Restore after setup: %s\n", archive)
	}
}

func buildInstallCommandPlan(results []bootstrap.ToolCheckResult) []bootstrap.InstallCommand {
	var commands []bootstrap.InstallCommand
	for _, result := range results {
		if result.Installed {
			continue
		}
		for _, installCommand := range result.InstallCommands {
			if isAutoRunnableInstallCommand(installCommand) {
				commands = append(commands, installCommand)
				break
			}
		}
	}
	return commands
}

func isAutoRunnableInstallCommand(installCommand bootstrap.InstallCommand) bool {
	return installCommand.SafeRun &&
		!installCommand.ManualOnly &&
		!installCommand.UsesShell &&
		installCommand.Executable != ""
}

func confirmSetupInstall(commands []bootstrap.InstallCommand) bool {
	fmt.Println("\nRun install commands?")
	for _, installCommand := range commands {
		fmt.Printf("  %s\n", installCommand.Command)
	}
	fmt.Print("Proceed? [Y/n] ")
	var answer string
	fmt.Scanln(&answer)
	return answer == "" || answer == "y" || answer == "Y"
}

func runInstallCommand(w io.Writer, installCommand bootstrap.InstallCommand) error {
	if !isAutoRunnableInstallCommand(installCommand) {
		return fmt.Errorf("install command requires manual review: %s", installCommand.Command)
	}
	fmt.Fprintf(w, "Running: %s\n", installCommand.Command)
	cmd := exec.Command(installCommand.Executable, installCommand.Args...)
	cmd.Stdout = w
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func runSetupRestore(archive string) error {
	oldYes, oldOnly, oldProject, oldForce, oldDryRun, oldBases := importYes, importOnly, importProject, importForce, importDryRun, importBases
	defer func() {
		importYes = oldYes
		importOnly = oldOnly
		importProject = oldProject
		importForce = oldForce
		importDryRun = oldDryRun
		importBases = oldBases
	}()

	importYes = true
	importOnly = setupOnly
	importProject = ""
	importForce = false
	importDryRun = false
	importBases = setupBases
	return importCmd.RunE(importCmd, []string{archive})
}
