package main

import (
	"os"

	"github.com/yangyifan18/dotvibe/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
