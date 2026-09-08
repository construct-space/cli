package appdir

import (
	"os"
	"path/filepath"
	"runtime"
)

// DataDir returns the Construct data directory, matching the Tauri app + operator.
//
// Resolution order:
//  1. CONSTRUCT_DATA_DIR env var (set by Tauri when launching operator)
//  2. OS-specific path:
//     - macOS:   ~/Library/Application Support/Construct/
//     - Windows: %APPDATA%/Construct/
//     - Linux:   $XDG_DATA_HOME/construct/ (or ~/.local/share/construct/)
func DataDir() string {
	if dir := os.Getenv("CONSTRUCT_DATA_DIR"); dir != "" {
		return dir
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Construct")
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "Construct")
		}
		return filepath.Join(home, "AppData", "Roaming", "Construct")
	default:
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			return filepath.Join(xdg, "construct")
		}
		return filepath.Join(home, ".local", "share", "construct")
	}
}

// DevDataDir returns the data directory for dev instances.
func DevDataDir() string {
	if dir := os.Getenv("CONSTRUCT_DATA_DIR"); dir != "" {
		return dir
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Construct Dev")
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "Construct Dev")
		}
		return filepath.Join(home, "AppData", "Roaming", "Construct Dev")
	default:
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			return filepath.Join(xdg, "construct-dev")
		}
		return filepath.Join(home, ".local", "share", "construct-dev")
	}
}

// SpacesDir returns the directory where built spaces are installed.
func SpacesDir() string {
	return filepath.Join(DataDir(), "spaces")
}

// DevSpacesDir returns the directory where dev-mode spaces are installed.
func DevSpacesDir() string {
	return filepath.Join(DevDataDir(), "spaces")
}

// SpaceDir returns the install directory for a specific space.
func SpaceDir(spaceID string) string {
	return filepath.Join(SpacesDir(), spaceID)
}

// DevSpaceDir returns the dev install directory for a specific space.
func DevSpaceDir(spaceID string) string {
	return filepath.Join(DevSpacesDir(), spaceID)
}
