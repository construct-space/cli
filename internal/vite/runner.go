package vite

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/construct-space/cli/internal/jsruntime"
)

// Build runs a Vite build in the given directory
func Build(ctx context.Context, dir string, rt *jsruntime.Runtime) error {
	cmd := buildCommand(rt, dir, "build")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vite build failed: %w", err)
	}
	return nil
}

// Watch runs Vite in watch mode and returns the command for cleanup
func Watch(ctx context.Context, dir string, rt *jsruntime.Runtime, stdout, stderr io.Writer) (*exec.Cmd, error) {
	cmd := buildCommand(rt, dir, "build", "--watch")
	cmd.Dir = dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start vite watch: %w", err)
	}

	return cmd, nil
}

func buildCommand(rt *jsruntime.Runtime, dir string, args ...string) *exec.Cmd {
	cmd := rt.Exec("vite", args...)
	cmd.Dir = dir
	return cmd
}
