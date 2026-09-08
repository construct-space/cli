package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/construct-space/cli/internal/operator"
	"github.com/spf13/cobra"
)

var (
	vibeModel   string
	vibeProject string
)

var vibeCmd = &cobra.Command{
	Use:   "vibe [goal]",
	Short: "Start a Vibe session with the operator",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		goal := strings.Join(args, " ")
		addr := operatorAddr()
		clientID := fmt.Sprintf("construct-%d", os.Getpid())

		client, err := operator.NewClient(addr, clientID)
		if err != nil {
			return fmt.Errorf("connect to operator: %w", err)
		}
		defer client.Close()
		syncAuth(client)

		state := operator.NewSessionState()
		state.Mode = operator.ModeVibe
		if vibeModel != "" {
			state.Model = vibeModel
		}
		if vibeProject != "" {
			state.Project.Path = vibeProject
			state.Project.Name = strings.TrimSuffix(vibeProject, "/")
			if idx := strings.LastIndex(state.Project.Name, "/"); idx >= 0 {
				state.Project.Name = state.Project.Name[idx+1:]
			}
		}

		_, events, err := operator.StartVibeStream(client, state, operator.VibeParams{
			Goal:   goal,
			Source: "vibe",
		})
		if err != nil {
			return err
		}

		for evt := range events {
			switch evt.Kind {
			case operator.EventAssistantText:
				fmt.Print(evt.Text)
			case operator.EventProgress:
				fmt.Fprintf(os.Stderr, "[progress] %s\n", evt.Text)
			case operator.EventToolCall:
				fmt.Fprintf(os.Stderr, "[tool] %s %s\n", evt.ToolName, operator.Truncate(evt.Input, 80))
			case operator.EventToolResult:
				prefix := "[result]"
				if evt.IsError {
					prefix = "[error]"
				}
				fmt.Fprintf(os.Stderr, "%s %s %s\n", prefix, evt.ToolName, operator.Truncate(evt.Content, 120))
			case operator.EventVibeSession:
				if evt.Session != nil {
					fmt.Fprintf(os.Stderr, "[vibe] session=%s status=%s\n", evt.Session.SessionID, evt.Session.Status)
				}
			case operator.EventDone:
				if evt.Session != nil && evt.Session.SessionID != "" {
					fmt.Fprintf(os.Stderr, "\nsession_id: %s\n", evt.Session.SessionID)
				}
			case operator.EventError:
				fmt.Fprintf(os.Stderr, "\nerror: %s\n", evt.Text)
				return fmt.Errorf("%s", evt.Text)
			}
		}
		fmt.Println()
		return nil
	},
}

func init() {
	vibeCmd.Flags().StringVar(&vibeModel, "model", "", "LLM model to use")
	vibeCmd.Flags().StringVar(&vibeProject, "project", "", "Project directory path")
}
