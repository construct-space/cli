package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/construct-space/cli/internal/appdir"
)

const (
	credentialsFile = "credentials.json"
	// appAuthFile is written by the Construct desktop app when the user
	// signs in from the UI. When `credentials.json` is missing we fall back
	// to reading this so `construct login` isn't needed after the user has
	// already logged into the app — they share the data dir anyway.
	appAuthFile   = "auth.json"
	DefaultPortal = "https://developer.construct.space"
)

type Credentials struct {
	Token  string `json:"token"`
	Portal string `json:"portal"`
	User   *User  `json:"user,omitempty"`
}

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func credentialsPath() (string, error) {
	return filepath.Join(appdir.DataDir(), credentialsFile), nil
}

func StoreCredentials(creds *Credentials) error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}

	// Write with restrictive permissions (owner read/write only)
	return os.WriteFile(path, append(data, '\n'), 0600)
}

func LoadCredentials() (*Credentials, error) {
	path, err := credentialsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// No CLI credentials — try the desktop app's auth.json so the user
		// doesn't have to log in twice.
		if creds, ok := loadAppAuthFallback(); ok {
			return creds, nil
		}
		return nil, fmt.Errorf("not logged in — run 'construct login' first (or sign in to the Construct app)")
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("corrupted credentials file: %w", err)
	}

	if creds.Token == "" {
		if fallback, ok := loadAppAuthFallback(); ok {
			return fallback, nil
		}
		return nil, fmt.Errorf("not logged in — run 'construct login' first (or sign in to the Construct app)")
	}

	return &creds, nil
}

// loadAppAuthFallback reads the desktop app's auth.json (written by
// stores/auth.ts on login) and converts it to the CLI's Credentials shape.
// Returns (nil, false) when the file is missing, malformed, or has no token.
func loadAppAuthFallback() (*Credentials, bool) {
	dataDir := filepath.Dir(mustCredentialsPath())
	raw, err := os.ReadFile(filepath.Join(dataDir, appAuthFile))
	if err != nil {
		return nil, false
	}

	// Shape mirrors the app's persist payload in stores/auth.ts:
	//   { user, token, oauth_token, authenticated, updated_at }
	var shape struct {
		User *struct {
			ID    any    `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"user"`
		Token         string `json:"token"`
		OAuthToken    string `json:"oauth_token"`
		Authenticated bool   `json:"authenticated"`
	}
	if err := json.Unmarshal(raw, &shape); err != nil {
		return nil, false
	}

	// Prefer the oauth_token (accounts service) over the API token — Graph
	// and other services authenticate against oauth.
	token := shape.OAuthToken
	if token == "" {
		token = shape.Token
	}
	if token == "" || !shape.Authenticated {
		return nil, false
	}

	creds := &Credentials{Token: token, Portal: DefaultPortal}
	if shape.User != nil {
		id := ""
		switch v := shape.User.ID.(type) {
		case string:
			id = v
		case float64:
			id = fmt.Sprintf("%.0f", v)
		}
		creds.User = &User{ID: id, Name: shape.User.Name, Email: shape.User.Email}
	}
	return creds, true
}

// mustCredentialsPath is credentialsPath() without the error return — the
// underlying function never actually returns an error today, and this keeps
// loadAppAuthFallback readable.
func mustCredentialsPath() string {
	p, _ := credentialsPath()
	return p
}

func IsAuthenticated() bool {
	creds, err := LoadCredentials()
	return err == nil && creds.Token != ""
}

func ClearCredentials() error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func GetPortalURL() string {
	creds, err := LoadCredentials()
	if err != nil || creds.Portal == "" {
		return DefaultPortal
	}
	return creds.Portal
}
