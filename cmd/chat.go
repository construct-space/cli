package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/construct-space/cli/internal/operator"
	"github.com/spf13/cobra"
)

var chatModel string

var chatCmd = &cobra.Command{
	Use:   "chat [message]",
	Short: "Send a one-shot chat message to the operator",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		message := strings.Join(args, " ")
		addr := operatorAddr()
		clientID := fmt.Sprintf("construct-%d", os.Getpid())

		client, err := operator.NewClient(addr, clientID)
		if err != nil {
			return fmt.Errorf("connect to operator: %w", err)
		}
		defer client.Close()
		syncAuth(client)

		state := operator.NewSessionState()
		if chatModel != "" {
			state.Model = chatModel
		}

		_, events, err := operator.StartChatStream(client, state, operator.ChatParams{Message: message})
		if err != nil {
			return err
		}

		for evt := range events {
			switch evt.Kind {
			case operator.EventAssistantText:
				fmt.Print(evt.Text)
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
	chatCmd.Flags().StringVar(&chatModel, "model", "", "LLM model to use")
}

func operatorAddr() string {
	return envOrDefault("CONSTRUCT_OPERATOR_ADDR", "127.0.0.1:60100")
}
