package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yangyifan18/dotvibe/backup"
)

var listCmd = &cobra.Command{
	Use:   "list <archive>",
	Short: "Show backup contents",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ar, err := backup.ReadArchive(args[0])
		if err != nil {
			return fmt.Errorf("failed to read archive: %w", err)
		}
		defer ar.Close()

		m := ar.Manifest
		archiveLabel := "backup"
		if m.ArchiveKind == backup.ArchiveKindRecipe && m.Recipe != nil {
			archiveLabel = "recipe"
			fmt.Printf("dotvibe recipe — %s\n", m.Recipe.Name)
			if m.Recipe.Description != "" {
				fmt.Printf("Description: %s\n", m.Recipe.Description)
			}
			if m.Recipe.Author != "" {
				fmt.Printf("Author: %s\n", m.Recipe.Author)
			}
			fmt.Printf("Created: %s\n", m.Created.Format("2006-01-02"))
		} else {
			fmt.Printf("dotvibe backup — %s\n", m.Created.Format("2006-01-02"))
		}
		fmt.Printf("From: %s\n\n", m.Hostname)

		files := ar.ListFiles()
		toolFiles := map[string]int{}
		for _, f := range files {
			toolID := ""
			for i, c := range f {
				if c == '/' {
					toolID = f[:i]
					break
				}
			}
			toolFiles[toolID]++
		}

		for toolID, tm := range m.Tools {
			count := toolFiles[toolID]
			fmt.Printf("  %s: %v (%d files)\n", toolID, tm.Included, count)
		}

		fmt.Printf("\nTotal files in %s: %d\n", archiveLabel, len(files))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
