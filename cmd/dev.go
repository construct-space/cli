package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/construct-space/cli/internal/appdir"
	"github.com/construct-space/cli/internal/copy"
	"github.com/construct-space/cli/internal/entry"
	"github.com/construct-space/cli/internal/jsruntime"
	"github.com/construct-space/cli/internal/manifest"
	"github.com/construct-space/cli/internal/ui"
	"github.com/construct-space/cli/internal/watcher"
	"github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start dev mode with file watching and live reload",
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

		// Detect runtime
		rt, err := jsruntime.Detect()
		if err != nil {
			fmt.Println(ui.Error(err.Error()))
			os.Exit(1)
		}

		fmt.Println(ui.Info(fmt.Sprintf("Dev mode — %s (%s %s)", m.ID, rt.Name, rt.Version)))

		// Ensure dependencies are installed
		if err := rt.EnsureDeps(root); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Failed to install dependencies: %s", err)))
			os.Exit(1)
		}

		// Generate initial entry
		if err := entry.WriteEntry(root, m); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Failed to generate entry: %s", err)))
			os.Exit(1)
		}

		// Install to the main Construct spaces dir — Construct reads from here.
		// Also install to DEV dir if it exists, so both instances pick up changes.
		installDir := appdir.SpaceDir(m.ID)

		// Write .dev marker so SpaceLoader enables hot-reload polling
		os.MkdirAll(installDir, 0755)
		devMarker := filepath.Join(installDir, ".dev")
		os.WriteFile(devMarker, []byte("dev"), 0644)
		defer os.Remove(devMarker) // clean up when dev mode stops

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Set up file watcher for manifest changes
		w, err := watcher.New(200 * time.Millisecond)
		if err != nil {
			return fmt.Errorf("failed to create watcher: %w", err)
		}
		defer w.Close()

		// Watch the manifest file
		if err := w.Add(filepath.Join(root, manifest.ManifestFile)); err != nil {
			return fmt.Errorf("failed to watch manifest: %w", err)
		}

		// Set up dist watcher for auto-install
		distDir := filepath.Join(root, "dist")
		distWatcher, err := watcher.New(200 * time.Millisecond)
		if err != nil {
			return fmt.Errorf("failed to create dist watcher: %w", err)
		}
		defer distWatcher.Close()

		// Start TUI
		devModel := ui.NewDevModel(m.ID)
		p := tea.NewProgram(devModel)

		// Start Vite watch via runtime
		viteCmd := rt.WatchCmd(root)
		viteCmd.Dir = root

		vitePipe, _ := viteCmd.StdoutPipe()
		viteCmd.Stderr = viteCmd.Stdout

		if err := viteCmd.Start(); err != nil {
			return fmt.Errorf("failed to start vite watch: %w", err)
		}

		// Read Vite output and send to TUI
		go func() {
			scanner := bufio.NewScanner(vitePipe)
			for scanner.Scan() {
				line := scanner.Text()
				p.Send(ui.ViteOutputMsg{Line: line})
			}
		}()

		// Watch manifest changes → regenerate entry + update dist manifest
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case evt := <-w.Events:
					// Re-read manifest and regenerate entry
					if newM, err := manifest.Read(root); err == nil {
						entry.WriteEntry(root, newM)
					}
					// Update dist/manifest.json if dist exists
					if info, statErr := os.Stat(distDir); statErr == nil && info.IsDir() {
						if raw, readErr := manifest.ReadRaw(root); readErr == nil {
							// Preserve existing build metadata if present
							existingDist := filepath.Join(distDir, manifest.ManifestFile)
							if existingData, rdErr := os.ReadFile(existingDist); rdErr == nil {
								var existingRaw map[string]any
								if json.Unmarshal(existingData, &existingRaw) == nil {
									if build, ok := existingRaw["build"]; ok {
										raw["build"] = build
									}
								}
							}
							data, _ := json.MarshalIndent(raw, "", "  ")
							os.WriteFile(existingDist, append(data, '\n'), 0644)
							// Also copy directly to install dir
							os.MkdirAll(installDir, 0755)
							copy.Dir(distDir, installDir)
						}
					}
					p.Send(ui.FileChangeMsg{Path: evt.Path, Time: time.Now()})
				}
			}
		}()

		// Watch dist changes → install to spaces dir
		go func() {
			// Wait for dist to exist
			for {
				if info, err := os.Stat(distDir); err == nil && info.IsDir() {
					break
				}
				time.Sleep(500 * time.Millisecond)
			}
			distWatcher.Add(distDir)
			// Also watch subdirs (e.g. dist/agent/)
			entries, _ := os.ReadDir(distDir)
			for _, e := range entries {
				if e.IsDir() {
					distWatcher.Add(filepath.Join(distDir, e.Name()))
				}
			}

			devInstallDir := appdir.DevSpaceDir(m.ID)
			for {
				select {
				case <-ctx.Done():
					return
				case <-distWatcher.Events:
					if info, err := os.Stat(distDir); err == nil && info.IsDir() {
						// Install to main Construct
						os.MkdirAll(installDir, 0755)
						copy.Dir(distDir, installDir)
						// Also install to DEV instance if its dir exists
						if devParent := filepath.Dir(devInstallDir); dirExists(devParent) {
							os.MkdirAll(devInstallDir, 0755)
							copy.Dir(distDir, devInstallDir)
						}
					}
				}
			}
		}()

		// Run TUI
		if _, err := p.Run(); err != nil {
			cancel()
			viteCmd.Process.Kill()
			return err
		}

		// Cleanup
		cancel()
		if viteCmd.Process != nil {
			viteCmd.Process.Kill()
		}

		fmt.Println(ui.Success("Dev mode stopped."))
		return nil
	},
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
