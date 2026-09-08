package cmd

import (
	"fmt"
	"os"

	"github.com/construct-space/cli/internal/auth"
	"github.com/construct-space/cli/internal/ui"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of Construct",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !auth.IsAuthenticated() {
			fmt.Println(ui.Warn("Not logged in."))
			return nil
		}

		if err := auth.ClearCredentials(); err != nil {
			fmt.Println(ui.Error("Failed to remove credentials: " + err.Error()))
			os.Exit(1)
		}

		fmt.Println(ui.Success("Logged out."))
		return nil
	},
}
