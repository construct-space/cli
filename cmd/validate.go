package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/construct-space/cli/internal/manifest"
	"github.com/construct-space/cli/internal/ui"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate space.manifest.json",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, _ := os.Getwd()

		if !manifest.Exists(root) {
			fmt.Println(ui.Error("No space.manifest.json found in current directory"))
			os.Exit(1)
		}

		m, err := manifest.Read(root)
		if err != nil {
			fmt.Println(ui.Error(err.Error()))
			os.Exit(1)
		}

		errors := manifest.Validate(m)
		if len(errors) > 0 {
			fmt.Println(ui.Error("Manifest validation failed:"))
			for _, e := range errors {
				fmt.Printf("  - %s\n", e)
			}
			os.Exit(1)
		}

		fmt.Println(ui.Success(fmt.Sprintf(`Valid! Space "%s" v%s`, m.ID, m.Version)))

		// Additional warnings
		pkgPath := filepath.Join(root, "package.json")
		if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
			fmt.Println(ui.Warn("package.json not found"))
		} else {
			type packageMeta struct {
				Version string `json:"version"`
			}
			rawPkg, err := os.ReadFile(pkgPath)
			if err != nil {
				fmt.Println(ui.Warn(fmt.Sprintf("Failed to read package.json: %s", err)))
			} else {
				var pkg packageMeta
				if err := json.Unmarshal(rawPkg, &pkg); err != nil {
					fmt.Println(ui.Warn(fmt.Sprintf("Failed to parse package.json: %s", err)))
				} else if pkg.Version != "" && pkg.Version != m.Version {
					fmt.Println(ui.Warn(fmt.Sprintf("package.json version (%s) does not match manifest version (%s)", pkg.Version, m.Version)))
				}
			}
		}

		if len(m.Pages) == 0 {
			fmt.Println(ui.Warn("No manifest.pages entries found"))
		} else {
			pageRoot := root
			if info, err := os.Stat(filepath.Join(root, "src", "pages")); err == nil && info.IsDir() {
				pageRoot = filepath.Join(root, "src")
			}

			for _, page := range m.Pages {
				componentPath := page.Component
				if componentPath == "" {
					if page.Path == "" {
						componentPath = filepath.Join("pages", "index.vue")
					} else {
						componentPath = filepath.Join("pages", page.Path+".vue")
					}
				}

				if _, err := os.Stat(filepath.Join(pageRoot, componentPath)); os.IsNotExist(err) {
					pageName := page.Path
					if pageName == "" {
						pageName = "/"
					}
					fmt.Println(ui.Warn(fmt.Sprintf(`manifest page "%s" maps to "%s" but file not found`, pageName, componentPath)))
				}
			}
		}

		// Validate agent file
		if m.Agent != "" {
			agentPath := filepath.Join(root, m.Agent)
			content, err := os.ReadFile(agentPath)
			if err != nil {
				fmt.Println(ui.Warn(fmt.Sprintf("Agent file not found: %s", m.Agent)))
			} else {
				validateFrontmatter(string(content), m.Agent)
			}
		}

		// Validate skill files
		for _, skill := range m.Skills {
			skillPath := filepath.Join(root, skill)
			if _, err := os.Stat(skillPath); os.IsNotExist(err) {
				fmt.Println(ui.Warn(fmt.Sprintf("Skill file not found: %s", skill)))
			}
		}

		return nil
	},
}

var frontmatterRegex = regexp.MustCompile(`(?s)^---\n(.*?)\n---`)

func validateFrontmatter(content, filename string) {
	match := frontmatterRegex.FindStringSubmatch(content)
	if match == nil {
		fmt.Println(ui.Warn(fmt.Sprintf("No frontmatter found in %s", filename)))
		return
	}

	fm := match[1]
	if !strings.Contains(fm, "id:") {
		fmt.Println(ui.Warn(fmt.Sprintf("Missing 'id:' in frontmatter of %s", filename)))
	}
	if !strings.Contains(fm, "name:") {
		fmt.Println(ui.Warn(fmt.Sprintf("Missing 'name:' in frontmatter of %s", filename)))
	}
}
