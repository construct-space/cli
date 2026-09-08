package cmd

import (
	"fmt"
	"os"

	"github.com/construct-space/cli/internal/operator"
	"github.com/construct-space/cli/internal/shell"
	"github.com/spf13/cobra"
)

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List configured providers from the operator",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr := operatorAddr()
		clientID := fmt.Sprintf("construct-%d", os.Getpid())

		client, err := operator.NewClient(addr, clientID)
		if err != nil {
			return fmt.Errorf("connect to operator: %w", err)
		}
		defer client.Close()

		resp, err := client.Send("providers.list", nil)
		if err != nil {
			return err
		}
		if !resp.Success {
			return fmt.Errorf("%s", resp.Error)
		}
		fmt.Println(shell.PrettyJSON(resp.Data))
		return nil
	},
}
