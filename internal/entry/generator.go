package entry

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/construct-space/cli/internal/manifest"
)

func capitalize(s string) string {
	if s == "" {
		return s
	}
	result := []rune(s)
	result[0] = unicode.ToUpper(result[0])

	// Convert kebab-case to CamelCase: "my-page" → "MyPage"
	var out []rune
	upper := false
	for i, r := range result {
		if r == '-' {
			upper = true
			continue
		}
		if upper || i == 0 {
			out = append(out, unicode.ToUpper(r))
			upper = false
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}

type pageInfo struct {
	VarName   string
	Import    string
	Path      string
	Component string
}

func resolvePages(m *manifest.SpaceManifest, pagePrefix string) []pageInfo {
	var pages []pageInfo
	for _, p := range m.Pages {
		// Route path as-declared in the manifest (preserved for the generated
		// `pages` map). The filename/identifier derivations below normalize
		// away leading/trailing slashes so we never emit things like
		// `pages//foo.vue` or identifiers like `/EditorPage`.
		route := strings.Trim(p.Path, "/")

		component := p.Component
		if component == "" {
			if route == "" {
				component = "pages/index.vue"
			} else {
				component = fmt.Sprintf("pages/%s.vue", route)
			}
		}

		// Derive a valid JS identifier from either the component filename or
		// the route path. Strip slashes, colons, brackets, and anything else
		// that can't appear in a JS identifier. Falls back to "Page" when
		// there's nothing usable (shouldn't happen in practice).
		var basis string
		if p.Component != "" {
			base := filepath.Base(p.Component)
			base = strings.TrimSuffix(base, filepath.Ext(base))
			base = strings.Trim(base, "[]")
			basis = base
		} else if route != "" {
			basis = route
		}
		varName := pascalCaseIdent(basis) + "Page"

		importPath := pagePrefix + component

		pages = append(pages, pageInfo{
			VarName:   varName,
			Import:    importPath,
			Path:      p.Path,
			Component: component,
		})
	}
	return pages
}

// pascalCaseIdent builds a PascalCase JS identifier from a string, splitting
// on anything that's not a letter or digit. Routes like "/editor/:id" →
// "EditorId"; "settings" → "Settings"; "" → "Index".
func pascalCaseIdent(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	if len(parts) == 0 {
		return "Index"
	}
	var out strings.Builder
	for _, seg := range parts {
		out.WriteString(capitalize(seg))
	}
	return out.String()
}

type widgetImport struct {
	VarName    string
	ImportPath string
	WidgetID   string
	SizeKey    string
}

func resolveWidgets(m *manifest.SpaceManifest, prefix string) []widgetImport {
	var imports []widgetImport
	for _, w := range m.Widgets {
		// Sort size keys for deterministic output
		var sizeKeys []string
		for k := range w.Sizes {
			sizeKeys = append(sizeKeys, k)
		}
		sort.Strings(sizeKeys)

		for _, sizeKey := range sizeKeys {
			componentPath := w.Sizes[sizeKey]
			varName := capitalize(w.ID) + "Widget" + sizeKey
			importPath := prefix + componentPath
			imports = append(imports, widgetImport{
				VarName:    varName,
				ImportPath: importPath,
				WidgetID:   w.ID,
				SizeKey:    sizeKey,
			})
		}
	}
	return imports
}

func Generate(root string, m *manifest.SpaceManifest) string {
	// Determine page prefix based on directory structure
	pagePrefix := "../"
	if info, err := os.Stat(filepath.Join(root, "src", "pages")); err == nil && info.IsDir() {
		pagePrefix = "./"
	}

	pages := resolvePages(m, pagePrefix)
	// Widgets always live at project root (widgets/), entry.ts is in src/
	widgets := resolveWidgets(m, "../")

	var sb strings.Builder
	sb.WriteString("// Auto-generated entry — do not edit manually\n")
	sb.WriteString("// Generated from space.manifest.json\n")

	for _, p := range pages {
		sb.WriteString(fmt.Sprintf("import %s from '%s'\n", p.VarName, p.Import))
	}
	for _, w := range widgets {
		sb.WriteString(fmt.Sprintf("import %s from '%s'\n", w.VarName, w.ImportPath))
	}

	sb.WriteString("\nconst spaceExport = {\n")
	sb.WriteString("  pages: {\n")
	for _, p := range pages {
		sb.WriteString(fmt.Sprintf("    '%s': %s,\n", p.Path, p.VarName))
	}
	sb.WriteString("  },\n")

	if len(widgets) > 0 {
		sb.WriteString("  widgets: {\n")
		// Group by widget ID
		widgetsByID := make(map[string][]widgetImport)
		var widgetOrder []string
		for _, w := range widgets {
			if _, exists := widgetsByID[w.WidgetID]; !exists {
				widgetOrder = append(widgetOrder, w.WidgetID)
			}
			widgetsByID[w.WidgetID] = append(widgetsByID[w.WidgetID], w)
		}
		for _, wid := range widgetOrder {
			sb.WriteString(fmt.Sprintf("    '%s': {\n", wid))
			for _, w := range widgetsByID[wid] {
				sb.WriteString(fmt.Sprintf("      '%s': %s,\n", w.SizeKey, w.VarName))
			}
			sb.WriteString("    },\n")
		}
		sb.WriteString("  },\n")
	}

	sb.WriteString("}\n\n")
	sb.WriteString("export default spaceExport\n")

	return sb.String()
}

func WriteEntry(root string, m *manifest.SpaceManifest) error {
	content := Generate(root, m)

	srcDir := filepath.Join(root, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return fmt.Errorf("failed to create src directory: %w", err)
	}

	entryPath := filepath.Join(srcDir, "entry.ts")
	return os.WriteFile(entryPath, []byte(content), 0644)
}
