package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/construct-space/cli/internal/agent"
	"github.com/construct-space/cli/internal/entry"
	"github.com/construct-space/cli/internal/manifest"
	"github.com/construct-space/cli/internal/jsruntime"
	"github.com/construct-space/cli/internal/ui"
	"github.com/spf13/cobra"
)

var entryOnly bool

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the space (generate entry + run Vite)",
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

		// Generate entry.ts
		if err := entry.WriteEntry(root, m); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Failed to generate entry: %s", err)))
			os.Exit(1)
		}

		if entryOnly {
			fmt.Println(ui.Success("Done (entry only)."))
			return nil
		}

		// Detect runtime
		rt, err := jsruntime.Detect()
		if err != nil {
			fmt.Println(ui.Error(err.Error()))
			os.Exit(1)
		}

		fmt.Println(ui.Info(fmt.Sprintf("Using %s %s", rt.Name, rt.Version)))

		// Ensure dependencies are installed
		if err := rt.EnsureDeps(root); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Failed to install dependencies: %s", err)))
			os.Exit(1)
		}

		// Run preBuild hook
		if err := jsruntime.RunHook(m.Hooks, "preBuild", root); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("preBuild hook failed: %s", err)))
			os.Exit(1)
		}

		// Run Vite build with spinner
		err = ui.RunWithSpinner("Building space...", func() error {
			ctx := context.Background()
			return buildSpace(ctx, root, m, rt)
		})

		if err != nil {
			fmt.Println(ui.Error(err.Error()))
			os.Exit(1)
		}

		// Run postBuild hook
		if err := jsruntime.RunHook(m.Hooks, "postBuild", root); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("postBuild hook failed: %s", err)))
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	buildCmd.Flags().BoolVar(&entryOnly, "entry-only", false, "Only generate src/entry.ts")
}

func buildSpace(ctx context.Context, root string, m *manifest.SpaceManifest, rt *jsruntime.Runtime) error {
	// Run Vite build via the runtime
	if err := runViteBuild(ctx, root, rt); err != nil {
		return err
	}

	// Bundle agent/ directory into config.agent at dist root AND project root
	// dist/config.agent — included in published bundle
	// ./config.agent — used by operator in dev mode (picks up from space source dir)
	agentDir := filepath.Join(root, "agent")
	if info, err := os.Stat(agentDir); err == nil && info.IsDir() {
		distDir := filepath.Join(root, "dist")
		if err := agent.BundleAgentDir(agentDir, distDir); err != nil {
			return fmt.Errorf("failed to bundle agent files: %w", err)
		}
		// Also write to project root for dev mode
		if err := agent.BundleAgentDir(agentDir, root); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to write config.agent to root: %v\n", err)
		}
	}

	// Write dist/manifest.json with build metadata
	distDir := filepath.Join(root, "dist")
	expectedBundle := fmt.Sprintf("space-%s.iife.js", m.ID)
	bundlePath := filepath.Join(distDir, expectedBundle)

	// If expected bundle not found, look for any space-*.iife.js and rename it.
	// This handles the case where vite.config.ts uses a different name than manifest ID.
	if _, statErr := os.Stat(bundlePath); os.IsNotExist(statErr) {
		matches, _ := filepath.Glob(filepath.Join(distDir, "space-*.iife.js"))
		if len(matches) == 1 {
			os.Rename(matches[0], bundlePath)
			// Also rename CSS if it exists
			oldCSS := strings.TrimSuffix(matches[0], ".iife.js") + ".css"
			newCSS := filepath.Join(distDir, fmt.Sprintf("space-%s.css", m.ID))
			if _, cssErr := os.Stat(oldCSS); cssErr == nil {
				os.Rename(oldCSS, newCSS)
			}
		}
	}

	bundleData, err := os.ReadFile(bundlePath)
	if err != nil {
		return fmt.Errorf("bundle not found at %s: %w", bundlePath, err)
	}

	hash := sha256.Sum256(bundleData)
	checksum := hex.EncodeToString(hash[:])

	raw, err := manifest.ReadRaw(root)
	if err != nil {
		return err
	}

	buildMeta := &manifest.BuildMeta{
		Checksum:       checksum,
		Size:           int64(len(bundleData)),
		HostAPIVersion: "0.2.0",
		BuiltAt:        time.Now().UTC().Format(time.RFC3339),
	}

	if err := manifest.WriteWithBuild(distDir, raw, buildMeta); err != nil {
		return err
	}

	// Widget metadata is included in dist/manifest.json via the widgets field.
	// No separate dist/widgets/manifest.json needed — components are in the IIFE.

	return nil
}

func runViteBuild(_ context.Context, root string, rt *jsruntime.Runtime) error {
	cmd := rt.BuildCmd(root, "build")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
