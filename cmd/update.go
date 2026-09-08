package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/construct-space/cli/internal/jsruntime"
	"github.com/construct-space/cli/internal/ui"
	"github.com/spf13/cobra"
)

type npmVersionResponse struct {
	Version string `json:"version"`
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the Construct CLI to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(ui.Info(fmt.Sprintf("Current version: %s", Version)))

		// Check latest version from npm
		fmt.Println(ui.Info("Checking for updates..."))
		resp, err := http.Get("https://registry.npmjs.org/@construct-space/cli/latest")
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Failed to check for updates: %s", err)))
			os.Exit(1)
		}
		defer resp.Body.Close()

		var pkg npmVersionResponse
		if err := json.NewDecoder(resp.Body).Decode(&pkg); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Failed to parse version info: %s", err)))
			os.Exit(1)
		}

		latest := pkg.Version
		if latest == Version {
			fmt.Println(ui.Success("Already up to date!"))
			return nil
		}

		fmt.Println(ui.Info(fmt.Sprintf("New version available: %s → %s", Version, latest)))

		// Detect how CLI was installed and update accordingly
		constructPath, _ := exec.LookPath("construct")

		if strings.Contains(constructPath, "go/bin") {
			// Installed via `go install`
			fmt.Println(ui.Info("Updating via go install..."))
			goCmd := exec.Command("go", "install", "github.com/construct-space/cli@latest")
			goCmd.Stdout = os.Stdout
			goCmd.Stderr = os.Stderr
			if err := goCmd.Run(); err != nil {
				fmt.Println(ui.Error(fmt.Sprintf("Update failed: %s", err)))
				os.Exit(1)
			}
		} else {
			// Use detected JS runtime
			rt, rtErr := jsruntime.Detect()
			if rtErr != nil {
				fmt.Println(ui.Error(rtErr.Error()))
				os.Exit(1)
			}
			fmt.Println(ui.Info(fmt.Sprintf("Updating via %s...", rt.Name)))
			updateCmd := rt.GlobalInstall("@construct-space/cli@latest")
			updateCmd.Stdout = os.Stdout
			updateCmd.Stderr = os.Stderr
			if err := updateCmd.Run(); err != nil {
				fmt.Println(ui.Error(fmt.Sprintf("Update failed: %s", err)))
				os.Exit(1)
			}
		}

		fmt.Println(ui.Success(fmt.Sprintf("Updated to %s!", latest)))
		return nil
	},
}
