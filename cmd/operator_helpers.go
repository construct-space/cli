package cmd

import (
	"github.com/construct-space/cli/internal/auth"
	"github.com/construct-space/cli/internal/operator"
)

// syncAuth sends the construct login token to the operator if the user is authenticated.
func syncAuth(client *operator.Client) {
	if creds, err := auth.LoadCredentials(); err == nil {
		operator.SyncAuthToken(client, creds.Token)
	}
}
