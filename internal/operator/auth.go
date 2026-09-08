package operator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var authProviderKeyEnvNames = map[string]string{
	"deepseek": "DEEPSEEK_API_KEY",
	"mimo":     "MIMO_API_KEY",
	"xai":      "XAI_API_KEY",
	"zai":      "ZAI_API_KEY",
}

type ProviderAuthImport struct {
	Path string
	Keys map[string]string
}

func LoadProviderKeysFromCodexAuth(path string) (ProviderAuthImport, error) {
	if strings.TrimSpace(path) == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ProviderAuthImport{}, err
		}
		path = filepath.Join(home, ".codex", "auth.json")
	}
	path = filepath.Clean(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return ProviderAuthImport{}, err
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return ProviderAuthImport{}, fmt.Errorf("parse auth json: %w", err)
	}

	keys := map[string]string{}
	for providerID, envName := range authProviderKeyEnvNames {
		collectProviderKey(keys, providerID, raw[providerID])
		collectProviderKey(keys, providerID, raw[envName])
		collectProviderKey(keys, providerID, raw[strings.ToLower(envName)])
	}
	for _, nestedKey := range []string{"providers", "provider_keys", "api_keys"} {
		nested, _ := raw[nestedKey].(map[string]any)
		for providerID, envName := range authProviderKeyEnvNames {
			collectProviderKey(keys, providerID, nested[providerID])
			collectProviderKey(keys, providerID, nested[envName])
			collectProviderKey(keys, providerID, nested[strings.ToLower(envName)])
		}
	}

	return ProviderAuthImport{
		Path: path,
		Keys: keys,
	}, nil
}

// SyncAuthToken sends the construct login token to the operator if available.
// This allows operator commands to use the same credentials as space commands.
func SyncAuthToken(client *Client, token string) {
	if strings.TrimSpace(token) == "" {
		return
	}
	go func() {
		_, _ = client.Send("settings.set", map[string]any{
			"key":   "auth_token",
			"value": strings.TrimSpace(token),
		})
	}()
}

func collectProviderKey(dst map[string]string, providerID string, value any) {
	if dst == nil {
		return
	}
	key, _ := value.(string)
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	dst[providerID] = key
}
