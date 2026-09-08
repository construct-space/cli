package jsruntime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Runtime struct {
	Name    string // "bun", "deno", or "node"
	Path    string
	Version string
}

func Detect() (*Runtime, error) {
	// Prefer bun
	if path, err := exec.LookPath("bun"); err == nil {
		version := getVersion(path)
		return &Runtime{Name: "bun", Path: path, Version: version}, nil
	}

	// Then deno
	if path, err := exec.LookPath("deno"); err == nil {
		version := getVersion(path)
		return &Runtime{Name: "deno", Path: path, Version: version}, nil
	}

	// Fall back to node
	if path, err := exec.LookPath("node"); err == nil {
		version := getVersion(path)
		return &Runtime{Name: "node", Path: path, Version: version}, nil
	}

	return nil, fmt.Errorf("no JS runtime found in PATH — install bun, deno, or node")
}

func getVersion(path string) string {
	out, err := exec.Command(path, "--version").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// Exec returns an exec.Cmd that runs a package binary via the detected runtime.
// e.g. rt.Exec("vite", "build") → "bun run vite build" / "deno run -A npm:vite build" / "npx vite build"
func (r *Runtime) Exec(bin string, args ...string) *exec.Cmd {
	switch r.Name {
	case "bun":
		return exec.Command("bun", append([]string{"run", bin}, args...)...)
	case "deno":
		return exec.Command("deno", append([]string{"run", "-A", "npm:" + bin}, args...)...)
	default:
		return exec.Command("npx", append([]string{bin}, args...)...)
	}
}

// InstallCmd returns the package install command for the detected runtime
func (r *Runtime) InstallCmd() string {
	switch r.Name {
	case "bun":
		return "bun install"
	case "deno":
		return "deno install"
	default:
		return "npm install"
	}
}

// EnsureDeps checks for node_modules and runs install if missing
func (r *Runtime) EnsureDeps(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "node_modules")); err == nil {
		return nil
	}

	fmt.Printf("  Installing dependencies (%s)...\n", r.InstallCmd())
	var cmd *exec.Cmd
	switch r.Name {
	case "bun":
		cmd = exec.Command("bun", "install")
	case "deno":
		cmd = exec.Command("deno", "install")
	default:
		cmd = exec.Command("npm", "install")
	}
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// BuildCmd returns an exec.Cmd for running vite build in the given directory
func (r *Runtime) BuildCmd(dir string, viteArgs ...string) *exec.Cmd {
	cmd := r.Exec("vite", viteArgs...)
	cmd.Dir = dir
	return cmd
}

// WatchCmd returns an exec.Cmd for running vite in watch mode
func (r *Runtime) WatchCmd(dir string) *exec.Cmd {
	return r.BuildCmd(dir, "build", "--watch")
}

// GlobalInstall returns an exec.Cmd to install a package globally
func (r *Runtime) GlobalInstall(pkg string) *exec.Cmd {
	switch r.Name {
	case "bun":
		return exec.Command("bun", "install", "-g", pkg)
	case "deno":
		return exec.Command("deno", "install", "-A", "npm:"+pkg)
	default:
		return exec.Command("npm", "install", "-g", pkg)
	}
}
