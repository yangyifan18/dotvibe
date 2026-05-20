package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yangyifan18/dotvibe/adapters"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show detected vibe coding tools",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Detected vibe coding tools:")

		found := false
		for _, adapter := range adapters.AllAdapters() {
			if !adapter.Detect() {
				continue
			}
			found = true
			s := adapter.Status()
			fmt.Printf("  %-12s %-20s %s\n", s.Name, s.Path, formatSize(s.Size))

			if s.Projects > 0 {
				fmt.Printf("    %d projects with memory\n", s.Projects)
			}
			if s.Agents > 0 {
				fmt.Printf("    %d custom agents\n", s.Agents)
			}
			if s.Skills > 0 {
				fmt.Printf("    %d skills\n", s.Skills)
			}
			if s.Sessions > 0 {
				fmt.Printf("    %d sessions\n", s.Sessions)
			}
			if s.ConfigFile != "" {
				fmt.Printf("    config: %s\n", s.ConfigFile)
			}
			fmt.Println()
		}

		if !found {
			fmt.Println("  No vibe coding tools detected.")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.0f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.0f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
