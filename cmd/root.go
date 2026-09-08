package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const Version = "0.5.4"

var rootCmd = &cobra.Command{
	Use:   "construct",
	Short: "Construct CLI — build spaces, interact with the operator",
	Long:  "Construct CLI — scaffold, build, develop, and publish spaces; chat, vibe, and interact with the Construct operator.",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("construct v%s\n", Version)
	},
}

func Execute() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(spaceCmd)

	// Top-level aliases for common space commands (backwards compat)
	rootCmd.AddCommand(scaffoldCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(devCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(publishCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)

	// Operator interaction commands
	rootCmd.AddCommand(tuiCmd)
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(vibeCmd)
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(modelsCmd)
	rootCmd.AddCommand(agentsCmd)
	rootCmd.AddCommand(providersCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
