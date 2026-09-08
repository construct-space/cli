package cmd

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/charmbracelet/huh"
	"github.com/construct-space/cli/internal/ui"
	"github.com/spf13/cobra"
)

// TemplateFS is set by main.go with the embedded templates
var TemplateFS embed.FS

var nameRegex = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var withTests bool

type scaffoldData struct {
	Name               string // directory name (e.g. "space-dino")
	ID                 string // space ID without "space-" prefix (e.g. "dino")
	DisplayName        string
	DisplayNameNoSpace string
}

func toDisplayName(name string) string {
	words := strings.Split(name, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

var scaffoldCmd = &cobra.Command{
	Use:     "scaffold [name]",
	Aliases: []string{"new", "create"},
	Short:   "Create a new Construct space project",
	RunE: func(cmd *cobra.Command, args []string) error {
		var name string

		if len(args) > 0 {
			name = args[0]
		}

		// Interactive form if name not provided or we want more input
		if name == "" {
			form := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Space name").
						Description("Lowercase alphanumeric with hyphens (e.g. my-space)").
						Value(&name).
						Validate(func(s string) error {
							if !nameRegex.MatchString(s) {
								return fmt.Errorf("must be lowercase alphanumeric with hyphens, starting with a letter")
							}
							return nil
						}),
				),
			)

			if err := form.Run(); err != nil {
				return err
			}
		}

		if !nameRegex.MatchString(name) {
			fmt.Println(ui.Error("Invalid name: must be lowercase alphanumeric with hyphens, starting with a letter"))
			os.Exit(1)
		}

		// Check if directory exists
		if _, err := os.Stat(name); err == nil {
			fmt.Println(ui.Error(fmt.Sprintf("Directory '%s' already exists", name)))
			os.Exit(1)
		}

		// Strip "space-" prefix for the space ID (convention: spaces are identified without the prefix)
		id := name
		if strings.HasPrefix(id, "space-") {
			id = strings.TrimPrefix(id, "space-")
		}

		displayName := toDisplayName(id)
		data := scaffoldData{
			Name:               name,
			ID:                 id,
			DisplayName:        displayName,
			DisplayNameNoSpace: strings.ReplaceAll(displayName, " ", ""),
		}

		fmt.Println(ui.Info(fmt.Sprintf("Creating space: %s", displayName)))

		// Create directory structure
		dirs := []string{
			name,
			filepath.Join(name, "src", "pages"),
			filepath.Join(name, "src", "components"),
			filepath.Join(name, "src", "composables"),
			filepath.Join(name, "agent", "skills"),
			filepath.Join(name, "agent", "hooks"),
			filepath.Join(name, "agent", "tools"),
			filepath.Join(name, "widgets", "summary"),
			filepath.Join(name, ".github", "workflows"),
		}
		for _, d := range dirs {
			if err := os.MkdirAll(d, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", d, err)
			}
		}

		// Template files mapping: template name → output path
		files := map[string]string{
			"space.manifest.json.tmpl": filepath.Join(name, "space.manifest.json"),
			"package.json.tmpl":        filepath.Join(name, "package.json"),
			"vite.config.ts.tmpl":      filepath.Join(name, "vite.config.ts"),
			"index.vue.tmpl":           filepath.Join(name, "src", "pages", "index.vue"),
			"config.md.tmpl":           filepath.Join(name, "agent", "config.md"),
			"skill.md.tmpl":            filepath.Join(name, "agent", "skills", "default.md"),
			"safety.json.tmpl":         filepath.Join(name, "agent", "hooks", "safety.json"),
			"build.yml.tmpl":           filepath.Join(name, ".github", "workflows", "build.yml"),
			"tsconfig.json.tmpl":       filepath.Join(name, "tsconfig.json"),
			"gitignore.tmpl":           filepath.Join(name, ".gitignore"),
			"readme.md.tmpl":           filepath.Join(name, "README.md"),
			"widgets/2x1.vue.tmpl":     filepath.Join(name, "widgets", "summary", "2x1.vue"),
			"widgets/4x1.vue.tmpl":     filepath.Join(name, "widgets", "summary", "4x1.vue"),
		}

		for tmplName, outPath := range files {
			if err := renderTemplate(tmplName, outPath, data); err != nil {
				return fmt.Errorf("failed to render %s: %w", tmplName, err)
			}
		}

		// Optionally scaffold E2E test boilerplate
		if withTests {
			e2eDirs := []string{
				filepath.Join(name, "e2e"),
			}
			for _, d := range e2eDirs {
				if err := os.MkdirAll(d, 0755); err != nil {
					return fmt.Errorf("failed to create directory %s: %w", d, err)
				}
			}

			e2eFiles := map[string]string{
				"e2e/playwright.config.ts.tmpl": filepath.Join(name, "playwright.config.ts"),
				"e2e/space.spec.ts.tmpl":        filepath.Join(name, "e2e", "space.spec.ts"),
			}
			for tmplName, outPath := range e2eFiles {
				if err := renderTemplate(tmplName, outPath, data); err != nil {
					return fmt.Errorf("failed to render %s: %w", tmplName, err)
				}
			}
			fmt.Println(ui.Info("E2E testing boilerplate added (Playwright)"))
		}

		fmt.Println(ui.Success(fmt.Sprintf("Space '%s' created!", name)))
		fmt.Println()
		fmt.Printf("  cd %s\n", name)
		fmt.Println("  bun install")
		fmt.Println("  construct space dev")
		fmt.Println()

		return nil
	},
}

func init() {
	scaffoldCmd.Flags().BoolVar(&withTests, "with-tests", false, "Include E2E testing boilerplate (Playwright)")
}

func renderTemplate(tmplName, outPath string, data scaffoldData) error {
	content, err := TemplateFS.ReadFile("templates/space/" + tmplName)
	if err != nil {
		return fmt.Errorf("template not found: %s", tmplName)
	}

	tmpl, err := template.New(tmplName).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", tmplName, err)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}
