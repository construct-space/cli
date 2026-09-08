package cmd

import (
	"fmt"
	"os"

	"github.com/construct-space/cli/internal/manifest"
	"github.com/construct-space/cli/internal/jsruntime"
	"github.com/construct-space/cli/internal/ui"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Type-check and lint the space",
	Long:  "Runs vue-tsc --noEmit for type checking and eslint for linting.",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, _ := os.Getwd()

		if !manifest.Exists(root) {
			fmt.Println(ui.Error("No space.manifest.json found — run this from a space directory"))
			os.Exit(1)
		}

		rt, err := jsruntime.Detect()
		if err != nil {
			fmt.Println(ui.Error(err.Error()))
			os.Exit(1)
		}

		if err := rt.EnsureDeps(root); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Failed to install dependencies: %s", err)))
			os.Exit(1)
		}

		// Step 1: Type check with vue-tsc
		fmt.Println(ui.Info("Running type check (vue-tsc --noEmit)..."))
		tscCmd := rt.Exec("vue-tsc", "--noEmit")
		tscCmd.Dir = root
		tscCmd.Stdout = os.Stdout
		tscCmd.Stderr = os.Stderr

		if err := tscCmd.Run(); err != nil {
			fmt.Println(ui.Error("Type check failed"))
			os.Exit(1)
		}
		fmt.Println(ui.Success("Type check passed"))

		// Step 2: Lint with eslint
		fmt.Println(ui.Info("Running lint (eslint .)..."))
		lintCmd := rt.Exec("eslint", ".")
		lintCmd.Dir = root
		lintCmd.Stdout = os.Stdout
		lintCmd.Stderr = os.Stderr

		if err := lintCmd.Run(); err != nil {
			fmt.Println(ui.Error("Lint failed"))
			os.Exit(1)
		}
		fmt.Println(ui.Success("Lint passed"))

		fmt.Println()
		fmt.Println(ui.Success("All checks passed!"))
		return nil
	},
}
