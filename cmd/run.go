package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/construct-space/cli/internal/agent"
	"github.com/construct-space/cli/internal/appdir"
	"github.com/construct-space/cli/internal/copy"
	"github.com/construct-space/cli/internal/manifest"
	"github.com/construct-space/cli/internal/ui"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Install built space to Construct spaces directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, _ := os.Getwd()

		if !manifest.Exists(root) {
			fmt.Println(ui.Error("No space.manifest.json found in current directory"))
			os.Exit(1)
		}

		m, err := manifest.Read(root)
		if err != nil {
			fmt.Println(ui.Error(err.Error()))
			os.Exit(1)
		}

		distDir := filepath.Join(root, "dist")
		if _, err := os.Stat(distDir); os.IsNotExist(err) {
			fmt.Println(ui.Error("No dist/ directory found. Run 'construct build' first."))
			os.Exit(1)
		}

		bundlePath := filepath.Join(distDir, fmt.Sprintf("space-%s.iife.js", m.ID))
		if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
			fmt.Println(ui.Error(fmt.Sprintf("Bundle not found: space-%s.iife.js — run 'construct build' first.", m.ID)))
			os.Exit(1)
		}

		// Re-bundle agent/ into dist/ before installing (ensures latest agent config)
		agentDir := filepath.Join(root, "agent")
		if info, aErr := os.Stat(agentDir); aErr == nil && info.IsDir() {
			if err := agent.BundleAgentDir(agentDir, distDir); err != nil {
				fmt.Fprintf(os.Stderr, "warning: agent bundle failed: %v\n", err)
			}
		}

		installDir := appdir.SpaceDir(m.ID)

		err = ui.RunWithSpinner(fmt.Sprintf("Installing %s...", m.ID), func() error {
			if err := os.MkdirAll(installDir, 0755); err != nil {
				return err
			}
			return copy.Dir(distDir, installDir)
		})

		if err != nil {
			os.Exit(1)
		}

		fmt.Println()
		fmt.Printf("  Installed \"%s\" to %s\n", m.ID, installDir)
		fmt.Println("  Restart Construct to load the updated space.")
		fmt.Println()

		return nil
	},
}
