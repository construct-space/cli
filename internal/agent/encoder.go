package agent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const AgentKey = "construct-agent-obfuscate-v1"

// BundledAgent is a single encoded file containing the full agent directory.
// config.agent at the space root contains this JSON (XOR+base64 encoded).
type BundledAgent struct {
	Config string            `json:"config"`          // config.md / agent.md content
	Tools  map[string]string `json:"tools,omitempty"` // tool-id → tool .md content
	Skills map[string]string `json:"skills,omitempty"`// skill-id → skill .md content
	Hooks  map[string]string `json:"hooks,omitempty"` // hook-id → hook .json content
}

func encode(content []byte) string {
	key := []byte(AgentKey)
	xored := make([]byte, len(content))
	for i := range content {
		xored[i] = content[i] ^ key[i%len(key)]
	}
	return base64.StdEncoding.EncodeToString(xored)
}

// BundleAgentDir reads the agent/ directory and produces a single encoded
// config.agent file in distDir (the space's dist/ root, not dist/agent/).
func BundleAgentDir(srcDir, distDir string) error {
	bundle := BundledAgent{
		Tools:  make(map[string]string),
		Skills: make(map[string]string),
		Hooks:  make(map[string]string),
	}

	// Read agent config (config.md or agent.md)
	configPath := filepath.Join(srcDir, "config.md")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = filepath.Join(srcDir, "agent.md")
	}
	if data, err := os.ReadFile(configPath); err == nil {
		bundle.Config = string(data)
	}

	// Read tools/*.md
	toolsDir := filepath.Join(srcDir, "tools")
	if entries, err := os.ReadDir(toolsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(toolsDir, e.Name()))
			if err != nil {
				continue
			}
			id := strings.TrimSuffix(e.Name(), ".md")
			bundle.Tools[id] = string(data)
		}
	}

	// Read skills/*.md
	skillsDir := filepath.Join(srcDir, "skills")
	if entries, err := os.ReadDir(skillsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(skillsDir, e.Name()))
			if err != nil {
				continue
			}
			id := strings.TrimSuffix(e.Name(), ".md")
			bundle.Skills[id] = string(data)
		}
	}

	// Read hooks/*.json
	hooksDir := filepath.Join(srcDir, "hooks")
	if entries, err := os.ReadDir(hooksDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(hooksDir, e.Name()))
			if err != nil {
				continue
			}
			id := strings.TrimSuffix(e.Name(), ".json")
			bundle.Hooks[id] = string(data)
		}
	}

	// Skip if no config found
	if bundle.Config == "" {
		return nil
	}

	// Marshal to JSON, then encode
	jsonData, err := json.Marshal(bundle)
	if err != nil {
		return fmt.Errorf("failed to marshal agent bundle: %w", err)
	}

	encoded := encode(jsonData)
	outPath := filepath.Join(distDir, "config.agent")
	return os.WriteFile(outPath, []byte(encoded), 0644)
}
