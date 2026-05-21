package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yangyifan18/dotvibe/bootstrap"
)

var (
	setupInstall bool
	setupYes     bool
	setupOnly    string
	setupBases   []string
)

var (
	detectSetupTools = func() []bootstrap.ToolCheckResult {
		return bootstrap.DetectTools(bootstrap.DefaultToolSpecs())
	}
	setupRestore  = runSetupRestore
	setupLookPath = exec.LookPath
)

var setupCmd = &cobra.Command{
	Use:   "setup [archive]",
	Short: "Bootstrap supported coding agents and optionally restore a backup",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSetup,
}

func init() {
	setupCmd.Flags().BoolVar(&setupInstall, "install", false, "run safe install commands after confirmation")
	setupCmd.Flags().BoolVarP(&setupYes, "yes", "y", false, "skip setup confirmation")
	setupCmd.Flags().StringVar(&setupOnly, "only", "", "restore only specified tools after setup")
	setupCmd.Flags().StringSliceVar(&setupBases, "base", nil, "base archive for incremental restore")
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	archive := ""
	if len(args) == 1 {
		archive = args[0]
	}

	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()
	results := detectSetupTools()
	printSetupPlan(out, results, archive)
	if !setupInstall {
		return nil
	}

	commands := buildInstallCommandPlan(results)
	if !setupYes && (len(commands) > 0 || archive != "") {
		if !confirmSetupActions(inputReader(cmd), out, commands, archive) {
			fmt.Fprintln(out, "Cancelled.")
			return nil
		}
	}
	for _, installCommand := range commands {
		if err := runInstallCommand(out, errOut, inputReader(cmd), installCommand); err != nil {
			return err
		}
	}

	if archive != "" {
		return setupRestore(archive)
	}
	return nil
}

func inputReader(cmd *cobra.Command) io.Reader {
	if cmd != nil {
		return cmd.InOrStdin()
	}
	return os.Stdin
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
			if isAvailableAutoInstallCommand(installCommand) {
				commands = append(commands, installCommand)
				break
			}
		}
	}
	return commands
}

func isAvailableAutoInstallCommand(installCommand bootstrap.InstallCommand) bool {
	if !isAutoRunnableInstallCommand(installCommand) {
		return false
	}
	_, err := setupLookPath(installCommand.Executable)
	return err == nil
}

func isAutoRunnableInstallCommand(installCommand bootstrap.InstallCommand) bool {
	return installCommand.SafeRun &&
		!installCommand.ManualOnly &&
		!installCommand.UsesShell &&
		installCommand.Executable != ""
}

func confirmSetupActions(r io.Reader, w io.Writer, commands []bootstrap.InstallCommand, archive string) bool {
	fmt.Fprintln(w, "\nRun setup actions?")
	for _, installCommand := range commands {
		fmt.Fprintf(w, "  Install: %s\n", installCommand.Command)
	}
	if archive != "" {
		fmt.Fprintf(w, "  Restore backup: %s\n", archive)
	}
	fmt.Fprint(w, "Proceed? [Y/n] ")
	answer, err := bufio.NewReader(r).ReadString('\n')
	if err != nil {
		return false
	}
	answer = strings.TrimSpace(answer)
	return answer == "" || answer == "y" || answer == "Y"
}

func runInstallCommand(w io.Writer, errW io.Writer, in io.Reader, installCommand bootstrap.InstallCommand) error {
	if !isAutoRunnableInstallCommand(installCommand) {
		return fmt.Errorf("install command requires manual review: %s", installCommand.Command)
	}
	fmt.Fprintf(w, "Running: %s\n", installCommand.Command)
	cmd := exec.Command(installCommand.Executable, installCommand.Args...)
	cmd.Stdout = w
	cmd.Stderr = errW
	cmd.Stdin = in
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
