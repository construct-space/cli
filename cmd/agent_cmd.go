package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/construct-space/cli/internal/operator"
	"github.com/spf13/cobra"
)

var (
	agentCmdModel   string
	agentCmdAgentID string
)

var agentCmd = &cobra.Command{
	Use:   "agent [task]",
	Short: "Dispatch a task to an operator agent",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		task := strings.Join(args, " ")
		addr := operatorAddr()
		clientID := fmt.Sprintf("construct-%d", os.Getpid())

		client, err := operator.NewClient(addr, clientID)
		if err != nil {
			return fmt.Errorf("connect to operator: %w", err)
		}
		defer client.Close()
		syncAuth(client)

		state := operator.NewSessionState()
		state.Mode = operator.ModeAgent
		if agentCmdModel != "" {
			state.Model = agentCmdModel
		}
		if agentCmdAgentID != "" {
			state.AgentID = agentCmdAgentID
		}

		_, events, err := operator.StartAgentStream(client, state, operator.AgentParams{Task: task})
		if err != nil {
			return err
		}

		for evt := range events {
			switch evt.Kind {
			case operator.EventAssistantText:
				fmt.Print(evt.Text)
			case operator.EventToolCall:
				fmt.Fprintf(os.Stderr, "[tool] %s %s\n", evt.ToolName, operator.Truncate(evt.Input, 80))
			case operator.EventToolResult:
				prefix := "[result]"
				if evt.IsError {
					prefix = "[error]"
				}
				fmt.Fprintf(os.Stderr, "%s %s %s\n", prefix, evt.ToolName, operator.Truncate(evt.Content, 120))
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
	agentCmd.Flags().StringVar(&agentCmdModel, "model", "", "LLM model to use")
	agentCmd.Flags().StringVar(&agentCmdAgentID, "agent-id", "general", "Agent ID to dispatch to")
}
