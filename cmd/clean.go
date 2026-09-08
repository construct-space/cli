package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/construct-space/cli/internal/ui"
	"github.com/spf13/cobra"
)

var cleanAll bool

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove build artifacts (dist/, .vite/, node_modules/)",
	Long:  "Remove build artifacts and optionally node_modules to start fresh.",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, _ := os.Getwd()

		targets := []string{"dist", ".vite"}
		if cleanAll {
			targets = append(targets, "node_modules")
		}

		// Also clean common lockfiles when --all is set
		lockfiles := []string{"bun.lockb", "package-lock.json", "yarn.lock", "pnpm-lock.yaml"}

		removed := 0
		for _, dir := range targets {
			p := filepath.Join(root, dir)
			info, err := os.Stat(p)
			if err != nil || !info.IsDir() {
				continue
			}
			fmt.Println(ui.Info(fmt.Sprintf("Removing %s/", dir)))
			if err := os.RemoveAll(p); err != nil {
				fmt.Println(ui.Error(fmt.Sprintf("Failed to remove %s: %s", dir, err)))
				continue
			}
			removed++
		}

		if cleanAll {
			for _, lf := range lockfiles {
				p := filepath.Join(root, lf)
				if _, err := os.Stat(p); err == nil {
					fmt.Println(ui.Info(fmt.Sprintf("Removing %s", lf)))
					if err := os.Remove(p); err != nil {
						fmt.Println(ui.Error(fmt.Sprintf("Failed to remove %s: %s", lf, err)))
						continue
					}
					removed++
				}
			}
		}

		if removed == 0 {
			fmt.Println(ui.Warn("Nothing to clean."))
		} else {
			fmt.Println(ui.Success(fmt.Sprintf("Cleaned %d item(s).", removed)))
		}

		return nil
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Also remove node_modules and lockfiles")
}
