package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

type Author struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

type Navigation struct {
	Label string `json:"label"`
	Icon  string `json:"icon"`
	To    string `json:"to"`
	Order int    `json:"order"`
}

type ToolbarItem struct {
	ID     string `json:"id"`
	Icon   string `json:"icon"`
	Label  string `json:"label"`
	Action string `json:"action,omitempty"`
	To     string `json:"to,omitempty"`
}

type ContextMenuAction struct {
	Type        string            `json:"type"`
	SpaceID     string            `json:"spaceId,omitempty"`
	Page        string            `json:"page,omitempty"`
	Mode        string            `json:"mode,omitempty"`
	RequestType string            `json:"requestType,omitempty"`
	Params      map[string]any    `json:"params,omitempty"`
	Query       map[string]string `json:"query,omitempty"`
}

type ContextMenuItem struct {
	ID       string            `json:"id,omitempty"`
	Label    string            `json:"label,omitempty"`
	Icon     string            `json:"icon,omitempty"`
	Type     string            `json:"type,omitempty"`
	Disabled bool              `json:"disabled,omitempty"`
	Shortcut string            `json:"shortcut,omitempty"`
	Children []ContextMenuItem `json:"children,omitempty"`
	Action   *ContextMenuAction `json:"action,omitempty"`
}

type Page struct {
	Path            string        `json:"path"`
	Label           string        `json:"label"`
	Icon            string        `json:"icon,omitempty"`
	Default         bool          `json:"default,omitempty"`
	RequiresContext bool          `json:"requiresContext,omitempty"`
	Component       string        `json:"component,omitempty"`
	Toolbar         []ToolbarItem `json:"toolbar,omitempty"`
}

type Dependencies struct {
	Spaces []string `json:"spaces,omitempty"`
	Skills []string `json:"skills,omitempty"`
}

type Permissions struct {
	CanAccessNetwork    bool `json:"canAccessNetwork,omitempty"`
	CanAccessFileSystem bool `json:"canAccessFileSystem,omitempty"`
	CanRunCommands      bool `json:"canRunCommands,omitempty"`
}

type Theme struct {
	Color string `json:"color"`
	Bg    string `json:"bg"`
}

type Hooks struct {
	PreBuild  string `json:"preBuild,omitempty"`
	PostBuild string `json:"postBuild,omitempty"`
	PreDev    string `json:"preDev,omitempty"`
	PostDev   string `json:"postDev,omitempty"`
}

type Widget struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Icon        string            `json:"icon,omitempty"`
	DefaultSize string            `json:"defaultSize"`
	Sizes       map[string]string `json:"sizes"`
}

type SpaceManifest struct {
	ID                  string        `json:"id"`
	Name                string        `json:"name"`
	Version             string        `json:"version"`
	Description         string        `json:"description"`
	Author              Author        `json:"author"`
	Icon                string        `json:"icon"`
	Scope               string        `json:"scope"`
	MinConstructVersion string        `json:"minConstructVersion,omitempty"`
	Navigation          Navigation    `json:"navigation"`
	Pages               []Page        `json:"pages"`
	Toolbar             []ToolbarItem `json:"toolbar,omitempty"`
	ContextMenus        map[string][]ContextMenuItem `json:"contextMenus,omitempty"`
	Agent               string        `json:"agent,omitempty"`
	Skills              []string      `json:"skills,omitempty"`
	Dependencies        *Dependencies `json:"dependencies,omitempty"`
	Permissions         *Permissions  `json:"permissions,omitempty"`
	Screenshots         []string      `json:"screenshots,omitempty"`
	Keywords            []string      `json:"keywords,omitempty"`
	Recommended         bool          `json:"recommended,omitempty"`
	Theme               *Theme        `json:"theme,omitempty"`
	Hooks               *Hooks        `json:"hooks,omitempty"`
	Widgets             []Widget      `json:"widgets,omitempty"`
}

type BuildMeta struct {
	Checksum       string `json:"checksum"`
	Size           int64  `json:"size"`
	HostAPIVersion string `json:"hostApiVersion"`
	BuiltAt        string `json:"builtAt"`
}

type ManifestWithBuild struct {
	SpaceManifest
	Build *BuildMeta `json:"build,omitempty"`
}

const ManifestFile = "space.manifest.json"

var idRegex = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var versionRegex = regexp.MustCompile(`^\d+\.\d+\.\d+`)

func Validate(m *SpaceManifest) []string {
	var errors []string

	if m.ID == "" || !idRegex.MatchString(m.ID) {
		errors = append(errors, "id: must be lowercase alphanumeric with hyphens, starting with a letter")
	}
	if m.Name == "" {
		errors = append(errors, "name: must be a non-empty string")
	}
	if !versionRegex.MatchString(m.Version) {
		errors = append(errors, "version: must be a valid semver (e.g. 1.0.0)")
	}
	if m.Description == "" {
		errors = append(errors, "description: must be a string")
	}
	if m.Author.Name == "" {
		errors = append(errors, "author: must be an object with a name")
	}
	if m.Icon == "" {
		errors = append(errors, "icon: must be a string")
	}
	if m.Scope != "company" && m.Scope != "project" && m.Scope != "both" {
		errors = append(errors, `scope: must be "company", "project", or "both"`)
	}
	if len(m.Pages) == 0 {
		errors = append(errors, "pages: must be a non-empty array")
	}
	if m.Navigation.Label == "" {
		errors = append(errors, "navigation: must be an object")
	}

	return errors
}

func Read(dir string) (*SpaceManifest, error) {
	path := filepath.Join(dir, ManifestFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", ManifestFile, err)
	}

	var m SpaceManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", ManifestFile, err)
	}

	return &m, nil
}

func ReadRaw(dir string) (map[string]any, error) {
	path := filepath.Join(dir, ManifestFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", ManifestFile, err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", ManifestFile, err)
	}

	return raw, nil
}

func Write(dir string, m *SpaceManifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ManifestFile), append(data, '\n'), 0644)
}

func WriteWithBuild(dir string, raw map[string]any, build *BuildMeta) error {
	raw["build"] = build
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "manifest.json"), append(data, '\n'), 0644)
}

func Exists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ManifestFile))
	return err == nil
}
