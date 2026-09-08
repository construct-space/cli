package jsruntime

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/construct-space/cli/internal/manifest"
)

// RunHook executes a lifecycle hook command if defined.
// The command is run via sh -c in the given working directory.
func RunHook(hooks *manifest.Hooks, phase string, dir string) error {
	if hooks == nil {
		return nil
	}

	var cmd string
	switch phase {
	case "preBuild":
		cmd = hooks.PreBuild
	case "postBuild":
		cmd = hooks.PostBuild
	case "preDev":
		cmd = hooks.PreDev
	case "postDev":
		cmd = hooks.PostDev
	default:
		return fmt.Errorf("unknown hook phase: %s", phase)
	}

	if strings.TrimSpace(cmd) == "" {
		return nil
	}

	fmt.Printf("  Running %s hook: %s\n", phase, cmd)
	proc := exec.Command("sh", "-c", cmd)
	proc.Dir = dir
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	return proc.Run()
}
