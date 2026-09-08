package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	goRuntime "runtime"
	"strings"
	"time"

	"github.com/construct-space/cli/internal/auth"
	"github.com/construct-space/cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	loginPortal string
	loginAPIKey string
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Construct",
	Long: `Log in to your Construct account via browser to publish spaces.

You can also skip the browser flow by passing an existing publisher API
key (csk_live_…) with --api-key. Useful when the desktop app has already
enrolled you as a developer and you just want the CLI to reuse the same
credentials:

  construct login --api-key csk_live_abc123…`,
	RunE: func(cmd *cobra.Command, args []string) error {
		portalURL := loginPortal
		if portalURL == "" {
			portalURL = auth.DefaultPortal
		}

		// Check if already logged in
		if creds, err := auth.LoadCredentials(); err == nil && creds.Token != "" {
			userName := "unknown"
			if creds.User != nil && creds.User.Name != "" {
				userName = creds.User.Name
			}
			fmt.Println(ui.Info(fmt.Sprintf("Already logged in as %s", userName)))
			fmt.Println(ui.DimStyle.Render("  Run 'construct logout' to sign out first."))
			return nil
		}

		// --api-key short-circuit: validate the key against the portal and
		// save it as credentials. No browser, no local server — the key
		// IS the identity.
		if loginAPIKey != "" {
			return loginWithAPIKey(portalURL, loginAPIKey)
		}

		// Find a free port for the callback server
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			fmt.Println(ui.Error("Failed to start local server: " + err.Error()))
			os.Exit(1)
		}
		port := listener.Addr().(*net.TCPAddr).Port
		callbackURL := fmt.Sprintf("http://localhost:%d/callback", port)

		// Channel to receive the token
		tokenCh := make(chan string, 1)
		errCh := make(chan string, 1)

		// Start local HTTP server to catch the callback
		mux := http.NewServeMux()
		mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
			token := r.URL.Query().Get("token")
			errMsg := r.URL.Query().Get("error")

			if errMsg != "" {
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprintf(w, `<html><body style="font-family:system-ui;text-align:center;padding:60px">
					<h2 style="color:#EF4444">Login Failed</h2>
					<p>%s</p>
					<p style="color:#6B7280">You can close this window.</p>
				</body></html>`, errMsg)
				errCh <- errMsg
				return
			}

			if token == "" {
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprint(w, `<html><body style="font-family:system-ui;text-align:center;padding:60px">
					<h2 style="color:#EF4444">Login Failed</h2>
					<p>No token received.</p>
					<p style="color:#6B7280">You can close this window.</p>
				</body></html>`)
				errCh <- "no token received"
				return
			}

			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><body style="font-family:system-ui;text-align:center;padding:60px">
				<h2 style="color:#10B981">Logged In!</h2>
				<p>You can close this window and return to your terminal.</p>
			</body></html>`)
			tokenCh <- token
		})

		server := &http.Server{Handler: mux}
		go server.Serve(listener)
		defer server.Close()

		// Build the login URL — portal's CLI login endpoint redirects to accounts OAuth
		loginURL := fmt.Sprintf("%s/api/auth/cli-login?callback=%s",
			portalURL, url.QueryEscape(callbackURL))

		fmt.Println(ui.Info("Opening browser to log in..."))
		fmt.Println(ui.DimStyle.Render("  If the browser doesn't open, visit:"))
		fmt.Println(ui.DimStyle.Render("  " + loginURL))
		fmt.Println()

		// Open browser
		openBrowser(loginURL)

		// Wait for callback (timeout after 5 minutes)
		fmt.Println(ui.DimStyle.Render("  Waiting for authentication..."))

		select {
		case token := <-tokenCh:
			// Verify token and get user info
			user, err := verifyToken(portalURL, token)
			if err != nil {
				fmt.Println(ui.Error("Failed to verify token: " + err.Error()))
				os.Exit(1)
			}

			creds := &auth.Credentials{
				Token:  token,
				Portal: portalURL,
				User:   user,
			}
			if err := auth.StoreCredentials(creds); err != nil {
				fmt.Println(ui.Error("Failed to save credentials: " + err.Error()))
				os.Exit(1)
			}

			fmt.Println()
			name := "there"
			if user != nil && user.Name != "" {
				name = user.Name
			}
			fmt.Println(ui.Success(fmt.Sprintf("Logged in as %s", name)))

		case errMsg := <-errCh:
			fmt.Println(ui.Error("Login failed: " + errMsg))
			os.Exit(1)

		case <-time.After(5 * time.Minute):
			fmt.Println(ui.Error("Login timed out. Try again."))
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	loginCmd.Flags().StringVar(&loginPortal, "portal", "", "Portal URL (default: https://developer.construct.space)")
	loginCmd.Flags().StringVar(&loginAPIKey, "api-key", "", "Log in with an existing publisher API key (csk_live_…) instead of the browser flow")
}

// loginWithAPIKey validates a publisher API key against the portal and
// saves it as the CLI's credentials. Expects the caller to have already
// verified no existing login is in place.
func loginWithAPIKey(portalURL, key string) error {
	key = strings.TrimSpace(key)
	if !strings.HasPrefix(key, "csk_") {
		fmt.Println(ui.Error("Invalid key — expected a csk_live_… publisher key."))
		os.Exit(1)
	}

	user, err := verifyToken(portalURL, key)
	if err != nil {
		fmt.Println(ui.Error("Could not verify key: " + err.Error()))
		os.Exit(1)
	}

	creds := &auth.Credentials{
		Token:  key,
		Portal: portalURL,
		User:   user,
	}
	if err := auth.StoreCredentials(creds); err != nil {
		fmt.Println(ui.Error("Failed to save credentials: " + err.Error()))
		os.Exit(1)
	}

	name := "there"
	if user != nil && user.Name != "" {
		name = user.Name
	}
	fmt.Println(ui.Success(fmt.Sprintf("Logged in as %s", name)))
	return nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch goRuntime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		cmd.Start()
	}
}

func verifyToken(portalURL, token string) (*auth.User, error) {
	req, err := http.NewRequest("GET", portalURL+"/api/auth/cli-verify", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		User *auth.User `json:"user"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result.User, nil
}
