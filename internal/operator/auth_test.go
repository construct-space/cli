package operator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProviderKeysFromCodexAuthReadsTopLevelAndNestedKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	content := `{
		"DEEPSEEK_API_KEY": "sk-deepseek",
		"providers": {
			"mimo": "sk-mimo"
		},
		"api_keys": {
			"ZAI_API_KEY": "sk-zai"
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	imported, err := LoadProviderKeysFromCodexAuth(path)
	if err != nil {
		t.Fatalf("LoadProviderKeysFromCodexAuth: %v", err)
	}
	if imported.Path != path {
		t.Fatalf("expected path %q, got %q", path, imported.Path)
	}
	if imported.Keys["deepseek"] != "sk-deepseek" {
		t.Fatalf("expected deepseek key, got %#v", imported.Keys)
	}
	if imported.Keys["mimo"] != "sk-mimo" {
		t.Fatalf("expected mimo key, got %#v", imported.Keys)
	}
	if imported.Keys["zai"] != "sk-zai" {
		t.Fatalf("expected zai key, got %#v", imported.Keys)
	}
}
