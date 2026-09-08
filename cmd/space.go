package cmd

import "github.com/spf13/cobra"

var spaceCmd = &cobra.Command{
	Use:   "space",
	Short: "Space lifecycle commands",
	Long:  "Space lifecycle commands: scaffold, build, dev, run, validate, check, publish, and clean.",
}

var spaceBuildCmd = &cobra.Command{
	Use:   "build",
	Short: buildCmd.Short,
	RunE: func(cmd *cobra.Command, args []string) error {
		return buildCmd.RunE(cmd, args)
	},
}

var spaceDevCmd = &cobra.Command{
	Use:   "dev",
	Short: devCmd.Short,
	RunE: func(cmd *cobra.Command, args []string) error {
		return devCmd.RunE(cmd, args)
	},
}

var spaceValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: validateCmd.Short,
	RunE: func(cmd *cobra.Command, args []string) error {
		return validateCmd.RunE(cmd, args)
	},
}

var spaceScaffoldCmd = &cobra.Command{
	Use:     "scaffold [name]",
	Aliases: []string{"new", "create"},
	Short:   scaffoldCmd.Short,
	RunE: func(cmd *cobra.Command, args []string) error {
		return scaffoldCmd.RunE(cmd, args)
	},
}

var spaceRunCmd = &cobra.Command{
	Use:   "run",
	Short: runCmd.Short,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCmd.RunE(cmd, args)
	},
}

var spaceCheckCmd = &cobra.Command{
	Use:   "check",
	Short: checkCmd.Short,
	RunE: func(cmd *cobra.Command, args []string) error {
		return checkCmd.RunE(cmd, args)
	},
}

var spacePublishCmd = &cobra.Command{
	Use:   "publish",
	Short: publishCmd.Short,
	RunE: func(cmd *cobra.Command, args []string) error {
		return publishCmd.RunE(cmd, args)
	},
}

var spaceCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: cleanCmd.Short,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cleanCmd.RunE(cmd, args)
	},
}

func init() {
	spaceBuildCmd.Flags().BoolVar(&entryOnly, "entry-only", false, "Only generate src/entry.ts")
	spaceScaffoldCmd.Flags().BoolVar(&withTests, "with-tests", false, "Include E2E testing boilerplate (Playwright)")
	spacePublishCmd.Flags().BoolVarP(&publishYes, "yes", "y", false, "Skip all confirmation prompts")
	spacePublishCmd.Flags().StringVar(&publishBump, "bump", "", "Auto-bump version if tag exists (patch, minor, major)")
	spaceCleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Also remove node_modules and lockfiles")

	spaceCmd.AddCommand(spaceScaffoldCmd)
	spaceCmd.AddCommand(spaceBuildCmd)
	spaceCmd.AddCommand(spaceDevCmd)
	spaceCmd.AddCommand(spaceRunCmd)
	spaceCmd.AddCommand(spaceValidateCmd)
	spaceCmd.AddCommand(spaceCheckCmd)
	spaceCmd.AddCommand(spacePublishCmd)
	spaceCmd.AddCommand(spaceCleanCmd)
}
