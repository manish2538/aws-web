package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/local/aws-local-dashboard/internal/awscli"
)

// Command represents a safe, read-only AWS CLI command that can be executed
// via the dashboard.
type Command struct {
	ID             string   `json:"id"`
	Label          string   `json:"label"`
	Description    string   `json:"description"`
	Service        string   `json:"service"`
	Args           []string `json:"args"`
	SupportsRegion bool     `json:"supportsRegion"`
}

// PublicCommand is what we send to the frontend (no raw args).
type PublicCommand struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	Description    string `json:"description"`
	Service        string `json:"service"`
	SupportsRegion bool   `json:"supportsRegion"`
}

type Manager struct {
	exec     awscli.Executor
	commands map[string]Command
}

// LoadManager loads commands from a JSON config file (if present). If the file
// is missing, we fall back to the baked-in default set in command-config.json.
func LoadManager(exec awscli.Executor, configPath string) (*Manager, error) {
	if configPath == "" {
		configPath = filepath.Join(".", "command-config.json")
	}

	commands := map[string]Command{}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read command config: %w", err)
		}
		// If the file doesn't exist we just start with an empty set.
	} else {
		var list []Command
		if err := json.Unmarshal(data, &list); err != nil {
			return nil, fmt.Errorf("failed to parse command config: %w", err)
		}
		for _, c := range list {
			if c.ID == "" || len(c.Args) == 0 {
				continue
			}
			commands[c.ID] = c
		}
	}

	return &Manager{
		exec:     exec,
		commands: commands,
	}, nil
}

// List returns public metadata for all configured commands.
func (m *Manager) List() []PublicCommand {
	var out []PublicCommand
	for _, c := range m.commands {
		out = append(out, PublicCommand{
			ID:             c.ID,
			Label:          c.Label,
			Description:    c.Description,
			Service:        c.Service,
			SupportsRegion: c.SupportsRegion,
		})
	}
	return out
}

// Execute runs a configured command by id and returns its raw JSON output and the
// concrete arguments used.
func (m *Manager) Execute(ctx context.Context, id string, region string) ([]byte, []string, error) {
	cmd, ok := m.commands[id]
	if !ok {
		return nil, nil, fmt.Errorf("unknown command id %q", id)
	}

	args := append([]string{}, cmd.Args...)
	if cmd.SupportsRegion && strings.TrimSpace(region) != "" {
		args = append(args, "--region", region)
	}

	out, err := m.exec.RunJSON(ctx, args...)
	if err != nil {
		return nil, nil, err
	}
	return out, args, nil
}

// ExecuteRaw runs an arbitrary aws CLI command (still using --output json under
// the hood). The caller is responsible for validating that the args are safe
// (read-only).
func (m *Manager) ExecuteRaw(ctx context.Context, args []string) ([]byte, []string, error) {
	if len(args) == 0 {
		return nil, nil, fmt.Errorf("no arguments provided")
	}
	out, err := m.exec.RunJSON(ctx, args...)
	if err != nil {
		return nil, nil, err
	}
	return out, args, nil
}


