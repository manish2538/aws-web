package awscli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/local/aws-local-dashboard/internal/profiles"
)

// Executor abstracts running AWS CLI commands.
type Executor interface {
	RunJSON(ctx context.Context, args ...string) ([]byte, error)
}

type CLIExecutor struct {
	profileManager *profiles.Manager
}

// NewCLIExecutor creates a new CLIExecutor.
func NewCLIExecutor(profileManager *profiles.Manager) *CLIExecutor {
	return &CLIExecutor{
		profileManager: profileManager,
	}
}

// RunJSON runs an aws CLI command and returns the JSON output.
func (e *CLIExecutor) RunJSON(ctx context.Context, args ...string) ([]byte, error) {
	// Ensure we always request JSON
	args = append(args, "--output", "json")

	cmd := exec.CommandContext(ctx, "aws", args...)

	// Apply active profile environment, without mutating system configuration.
	if e.profileManager != nil {
		if envOverrides := e.profileManager.ActiveEnv(); len(envOverrides) > 0 {
			cmd.Env = append(os.Environ(), envOverrides...)
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, fmt.Errorf("aws cli error: %s", errMsg)
	}

	return stdout.Bytes(), nil
}


