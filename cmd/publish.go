package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/construct-space/cli/internal/auth"
	"github.com/construct-space/cli/internal/entry"
	"github.com/construct-space/cli/internal/manifest"
	"github.com/construct-space/cli/internal/publish"
	"github.com/construct-space/cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	publishYes  bool
	publishBump string
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish a space to the Construct registry",
	Long: `Upload space source code to the Construct registry for review and publishing.

The server builds the space from source to ensure safety. Requires authentication
via 'construct login'.

Use --yes (-y) to skip all confirmation prompts.
Use --bump to auto-bump version (patch, minor, major) when a tag already exists.`,
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

		// Regenerate src/entry.ts so the packed source includes widget exports
		if err := entry.WriteEntry(root, m); err != nil {
			fmt.Println(ui.Warn(fmt.Sprintf("Could not regenerate entry: %s", err)))
		}

		// Check authentication
		creds, err := auth.LoadCredentials()
		if err != nil {
			fmt.Println(ui.Error(err.Error()))
			fmt.Println(ui.DimStyle.Render("  Run 'construct login' to authenticate."))
			os.Exit(1)
		}

		// Check for uncommitted changes
		out, err := execGit(root, "status", "--porcelain")
		if err == nil && strings.TrimSpace(out) != "" {
			fmt.Println(ui.Warn("You have uncommitted changes."))

			if !publishYes {
				var proceed bool
				huh.NewForm(
					huh.NewGroup(
						huh.NewConfirm().
							Title("Publish anyway?").
							Affirmative("Yes").
							Negative("No").
							Value(&proceed),
					),
				).Run()

				if !proceed {
					fmt.Println("Cancelled.")
					return nil
				}
			} else {
				fmt.Println(ui.DimStyle.Render("  Continuing (--yes)"))
			}
		}

		// Version management
		currentVersion := m.Version
		tag := "v" + currentVersion

		// Check if this version was already published
		tagExists := false
		if _, err := execGit(root, "rev-parse", tag); err == nil {
			tagExists = true
		}

		if tagExists {
			fmt.Println(ui.Warn(fmt.Sprintf("Version %s already tagged.", currentVersion)))
			fmt.Println()

			bumpChoice := ""
			newVersion := ""

			if publishBump != "" {
				// Use --bump flag
				bumpChoice = publishBump
			} else if publishYes {
				// Default to patch when --yes is used
				bumpChoice = "patch"
			} else {
				err := huh.NewForm(
					huh.NewGroup(
						huh.NewSelect[string]().
							Title(fmt.Sprintf("Version %s exists. What do you want to do?", currentVersion)).
							Options(
								huh.NewOption("Bump patch  ("+bumpPatch(currentVersion)+")", "patch"),
								huh.NewOption("Bump minor  ("+bumpMinor(currentVersion)+")", "minor"),
								huh.NewOption("Bump major  ("+bumpMajor(currentVersion)+")", "major"),
								huh.NewOption("Enter custom version", "custom"),
								huh.NewOption("Cancel", "cancel"),
							).
							Value(&bumpChoice),
					),
				).Run()
				if err != nil {
					return err
				}
			}

			switch bumpChoice {
			case "patch":
				newVersion = bumpPatch(currentVersion)
			case "minor":
				newVersion = bumpMinor(currentVersion)
			case "major":
				newVersion = bumpMajor(currentVersion)
			case "custom":
				err := huh.NewForm(
					huh.NewGroup(
						huh.NewInput().
							Title("Enter version").
							Placeholder("e.g. 1.0.0").
							Value(&newVersion).
							Validate(func(s string) error {
								parts := strings.Split(s, ".")
								if len(parts) != 3 {
									return fmt.Errorf("must be semver (x.y.z)")
								}
								for _, p := range parts {
									if _, err := strconv.Atoi(p); err != nil {
										return fmt.Errorf("must be semver (x.y.z)")
									}
								}
								return nil
							}),
					),
				).Run()
				if err != nil {
					return err
				}
			case "cancel":
				fmt.Println("Cancelled.")
				return nil
			}

			// Bump version in files
			if err := setVersionInFiles(root, currentVersion, newVersion); err != nil {
				fmt.Println(ui.Error(err.Error()))
				os.Exit(1)
			}

			// Commit version bump
			execGit(root, "add", "package.json", "space.manifest.json")
			execGit(root, "commit", "-m", fmt.Sprintf("release: v%s", newVersion))
			execGit(root, "push")

			// Re-read manifest
			m, _ = manifest.Read(root)
			fmt.Println(ui.Success(fmt.Sprintf("Version bumped to %s", m.Version)))
		}

		// Show summary
		fmt.Println()
		fmt.Printf("  Space:   %s\n", ui.AccentStyle.Render(m.Name))
		fmt.Printf("  Version: %s\n", ui.AccentStyle.Render("v"+m.Version))
		fmt.Printf("  Server:  %s\n", ui.DimStyle.Render(creds.Portal))
		if creds.User != nil {
			fmt.Printf("  Author:  %s\n", ui.DimStyle.Render(creds.User.Name))
		}
		fmt.Println()

		if !publishYes {
			var confirm bool
			err = huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title("Publish this space?").
						Description("Source code will be uploaded and built on the server.").
						Affirmative("Yes, publish").
						Negative("Cancel").
						Value(&confirm),
				),
			).Run()
			if err != nil {
				return err
			}
			if !confirm {
				fmt.Println("Cancelled.")
				return nil
			}
		} else {
			fmt.Println(ui.DimStyle.Render("  Publishing (--yes)..."))
		}

		// Pack source
		var tarballPath string
		err = ui.RunWithSpinner("Packing source...", func() error {
			var packErr error
			tarballPath, packErr = publish.PackSource(root)
			return packErr
		})
		if err != nil {
			fmt.Println(ui.Error(err.Error()))
			os.Exit(1)
		}
		defer os.Remove(tarballPath)

		tarInfo, _ := os.Stat(tarballPath)
		fmt.Println(ui.Success(fmt.Sprintf("Source packed (%s)", formatBytes(tarInfo.Size()))))

		// Upload to server
		var result *PublishResult
		err = ui.RunWithSpinner("Uploading & building...", func() error {
			var uploadErr error
			result, uploadErr = uploadSource(creds.Portal, creds.Token, tarballPath, m)
			return uploadErr
		})
		if err != nil {
			fmt.Println(ui.Error(err.Error()))
			os.Exit(1)
		}

		// Tag locally
		tag = "v" + m.Version
		execGit(root, "tag", tag)
		execGit(root, "push", "origin", tag)

		// Show result
		fmt.Println()
		if result.Status == "approved" || result.Status == "pending_review" {
			fmt.Println(ui.Success(fmt.Sprintf("Published %s v%s", m.Name, m.Version)))
			if result.Status == "pending_review" {
				fmt.Println(ui.DimStyle.Render("  Status: pending review — your space will be available after approval."))
			}
		} else if result.Status == "build_failed" {
			fmt.Println(ui.Error("Build failed on server:"))
			if result.Log != "" {
				fmt.Println(result.Log)
			}
		} else {
			fmt.Println(ui.Info(fmt.Sprintf("Status: %s", result.Status)))
		}

		if result.Build != nil {
			fmt.Printf("  Checksum: %s\n", ui.DimStyle.Render(result.Build.Checksum))
			fmt.Printf("  Size:     %s\n", ui.DimStyle.Render(formatBytes(result.Build.Size)))
			fmt.Printf("  Duration: %s\n", ui.DimStyle.Render(result.Build.Duration))
		}
		fmt.Println()

		return nil
	},
}

func init() {
	publishCmd.Flags().BoolVarP(&publishYes, "yes", "y", false, "Skip all confirmation prompts")
	publishCmd.Flags().StringVar(&publishBump, "bump", "", "Auto-bump version if tag exists (patch, minor, major)")
}

type PublishResult struct {
	Status string       `json:"status"`
	Space  *SpaceResult `json:"space,omitempty"`
	Build  *BuildResult `json:"build,omitempty"`
	Error  string       `json:"error,omitempty"`
	Errors []string     `json:"errors,omitempty"`
	Log    string       `json:"log,omitempty"`
}

type SpaceResult struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Status  string `json:"status"`
}

type BuildResult struct {
	Checksum string `json:"checksum"`
	Size     int64  `json:"size"`
	Duration string `json:"duration"`
}

func uploadSource(portalURL, token, tarballPath string, m *manifest.SpaceManifest) (*PublishResult, error) {
	// Create multipart form
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer writer.Close()

		// Add manifest JSON
		manifestJSON, _ := json.Marshal(m)
		fw, err := writer.CreateFormField("manifest")
		if err != nil {
			pw.CloseWithError(fmt.Errorf("failed to create manifest field: %w", err))
			return
		}
		fw.Write(manifestJSON)

		// Add source tarball
		fw, err = writer.CreateFormFile("source", filepath.Base(tarballPath))
		if err != nil {
			pw.CloseWithError(fmt.Errorf("failed to create source field: %w", err))
			return
		}
		f, err := os.Open(tarballPath)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		defer f.Close()
		io.Copy(fw, f)
	}()

	req, err := http.NewRequest("POST", portalURL+"/api/publish", pr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("authentication failed — run 'construct login' to re-authenticate")
	}

	var result PublishResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unexpected response: %s", string(body))
	}

	if resp.StatusCode >= 400 {
		errMsg := result.Error
		if errMsg == "" && len(result.Errors) > 0 {
			errMsg = strings.Join(result.Errors, "; ")
		}
		if errMsg == "" {
			errMsg = fmt.Sprintf("server returned %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("%s", errMsg)
	}

	return &result, nil
}

func execGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func setVersionInFiles(root, oldVer, newVer string) error {
	oldStr := fmt.Sprintf(`"version": "%s"`, oldVer)
	newStr := fmt.Sprintf(`"version": "%s"`, newVer)

	for _, file := range []string{"package.json", "space.manifest.json"} {
		path := filepath.Join(root, file)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		updated := strings.Replace(string(data), oldStr, newStr, 1)
		if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
			return fmt.Errorf("failed to update %s: %w", file, err)
		}
	}
	return nil
}

func bumpPatch(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return version
	}
	patch, _ := strconv.Atoi(parts[2])
	return fmt.Sprintf("%s.%s.%d", parts[0], parts[1], patch+1)
}

func bumpMinor(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return version
	}
	minor, _ := strconv.Atoi(parts[1])
	return fmt.Sprintf("%s.%d.0", parts[0], minor+1)
}

func bumpMajor(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return version
	}
	major, _ := strconv.Atoi(parts[0])
	return fmt.Sprintf("%d.0.0", major+1)
}

func formatBytes(b int64) string {
	const kb = 1024
	const mb = kb * 1024
	if b >= mb {
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	}
	if b >= kb {
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	}
	return fmt.Sprintf("%d B", b)
}
