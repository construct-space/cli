package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/construct-space/cli/internal/ui"
	"github.com/spf13/cobra"
)

var tuiAddr string

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive operator TUI",
	Long:  "Connect to the Construct operator and launch a full-screen interactive terminal UI for chat, vibe, and agent interaction.",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr := strings.TrimSpace(tuiAddr)
		if addr == "" {
			addr = envOrDefault("CONSTRUCT_OPERATOR_ADDR", "127.0.0.1:60100")
		}
		clientID := fmt.Sprintf("construct-%d", os.Getpid())
		return ui.RunTUI(addr, clientID)
	},
}

func init() {
	tuiCmd.Flags().StringVar(&tuiAddr, "addr", "", "Operator TCP address (default: $CONSTRUCT_OPERATOR_ADDR or 127.0.0.1:60100)")
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
