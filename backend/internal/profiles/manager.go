package profiles

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Source indicates where a profile comes from.
type Source string

const (
	SourceSystem Source = "system"
	SourceCustom Source = "custom"
)

type Profile struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	SessionToken    string `json:"sessionToken,omitempty"`
	Region          string `json:"region,omitempty"`
	Source          Source `json:"source"`
}

// PublicProfile is a redacted view of a Profile sent to the frontend.
type PublicProfile struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Source Source `json:"source"`
}

// Status summarizes the profile state for the frontend.
type Status struct {
	SystemAvailable bool            `json:"systemAvailable"`
	ActiveID        string          `json:"activeId"`
	Profiles        []PublicProfile `json:"profiles"`
}

// Manager keeps track of profiles and the active selection.
type Manager struct {
	mu              sync.RWMutex
	profiles        map[string]Profile
	activeID        string
	systemAvailable bool
	nextID          int64
	storePath       string
}

// NewManager creates a Manager and probes whether system AWS credentials
// are available (using aws sts get-caller-identity).
func NewManager(ctx context.Context) *Manager {
	storePath := os.Getenv("PROFILE_STORE_PATH")
	if storePath == "" {
		// Default to a project-local file so profiles persist across restarts
		// as long as the working directory is preserved or mounted.
		storePath = filepath.Join(".", ".aws-local-dashboard-profiles.json")
	}

	m := &Manager{
		profiles:  make(map[string]Profile),
		nextID:    1,
		storePath: storePath,
	}

	if ok := checkCredentialsWithEnv(ctx, nil); ok {
		m.systemAvailable = true
		if m.activeID == "" {
			m.activeID = "system"
		}
	}

	// Best-effort load of any previously saved custom profiles.
	_ = m.loadFromDisk()

	return m
}

// Status returns a snapshot of profile state.
func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var pubs []PublicProfile
	for _, p := range m.profiles {
		pubs = append(pubs, PublicProfile{
			ID:     p.ID,
			Name:   p.Name,
			Source: p.Source,
		})
	}

	active := m.activeID
	if active == "" && m.systemAvailable {
		active = "system"
	}

	return Status{
		SystemAvailable: m.systemAvailable,
		ActiveID:        active,
		Profiles:        pubs,
	}
}

// ActiveID returns the identifier of the currently active profile
// ("system" or a custom profile id).
func (m *Manager) ActiveID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.activeID == "" && m.systemAvailable {
		return "system"
	}
	return m.activeID
}

// ActiveEnv returns environment variable overrides for the active profile.
// If the system profile is active (or none), it returns nil.
func (m *Manager) ActiveEnv() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.activeID == "" || m.activeID == "system" {
		return nil
	}

	p, ok := m.profiles[m.activeID]
	if !ok {
		return nil
	}

	var env []string
	if p.AccessKeyID != "" {
		env = append(env, "AWS_ACCESS_KEY_ID="+p.AccessKeyID)
	}
	if p.SecretAccessKey != "" {
		env = append(env, "AWS_SECRET_ACCESS_KEY="+p.SecretAccessKey)
	}
	if p.SessionToken != "" {
		env = append(env, "AWS_SESSION_TOKEN="+p.SessionToken)
	}
	if p.Region != "" {
		env = append(env, "AWS_DEFAULT_REGION="+p.Region)
	}
	// Disable IMDS to avoid slow lookups when using explicit keys.
	env = append(env, "AWS_EC2_METADATA_DISABLED=true")
	return env
}

// AddAndActivateProfile validates credentials by calling sts get-caller-identity,
// then stores and activates the profile if valid.
func (m *Manager) AddAndActivateProfile(ctx context.Context, name, accessKey, secretKey, sessionToken, region string) (Profile, error) {
	if strings.TrimSpace(name) == "" {
		return Profile{}, fmt.Errorf("profile name is required")
	}
	if accessKey == "" || secretKey == "" {
		return Profile{}, fmt.Errorf("access key id and secret access key are required")
	}

	env := []string{
		"AWS_ACCESS_KEY_ID=" + accessKey,
		"AWS_SECRET_ACCESS_KEY=" + secretKey,
	}
	if sessionToken != "" {
		env = append(env, "AWS_SESSION_TOKEN="+sessionToken)
	}
	if region != "" {
		env = append(env, "AWS_DEFAULT_REGION="+region)
	}
	env = append(env, "AWS_EC2_METADATA_DISABLED=true")

	if ok := checkCredentialsWithEnv(ctx, env); !ok {
		return Profile{}, fmt.Errorf("unable to validate credentials with AWS (sts get-caller-identity failed)")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	id := strconv.FormatInt(m.nextID, 10)
	m.nextID++

	p := Profile{
		ID:              id,
		Name:            name,
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		SessionToken:    sessionToken,
		Region:          region,
		Source:          SourceCustom,
	}

	m.profiles[id] = p
	m.activeID = id

	m.saveLocked()

	return p, nil
}

// SetActiveProfile switches the active profile. Use id "system" to use
// the process / host default credentials (if available).
func (m *Manager) SetActiveProfile(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if id == "system" {
		if !m.systemAvailable {
			return fmt.Errorf("system AWS credentials are not available")
		}
		m.activeID = "system"
		return nil
	}

	if _, ok := m.profiles[id]; !ok {
		return fmt.Errorf("profile %q not found", id)
	}
	m.activeID = id
	m.saveLocked()
	return nil
}

// loadFromDisk restores profiles and activeId from the store file, if present.
func (m *Manager) loadFromDisk() error {
	if m.storePath == "" {
		return nil
	}

	data, err := os.ReadFile(m.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var state struct {
		NextID   int64     `json:"nextId"`
		ActiveID string    `json:"activeId"`
		Profiles []Profile `json:"profiles"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if state.NextID > 0 {
		m.nextID = state.NextID
	}
	if state.ActiveID != "" {
		m.activeID = state.ActiveID
	}
	m.profiles = make(map[string]Profile, len(state.Profiles))
	for _, p := range state.Profiles {
		// Skip any legacy entries that don't have credentials; they can't be used.
		if p.AccessKeyID == "" || p.SecretAccessKey == "" {
			continue
		}
		m.profiles[p.ID] = p
	}

	return nil
}

// saveLocked persists profiles and activeId to disk. Caller must hold m.mu.
func (m *Manager) saveLocked() {
	if m.storePath == "" {
		return
	}

	var profiles []Profile
	for _, p := range m.profiles {
		profiles = append(profiles, p)
	}

	state := struct {
		NextID   int64     `json:"nextId"`
		ActiveID string    `json:"activeId"`
		Profiles []Profile `json:"profiles"`
	}{
		NextID:   m.nextID,
		ActiveID: m.activeID,
		Profiles: profiles,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}

	_ = os.WriteFile(m.storePath, data, 0o600)
}

// checkCredentialsWithEnv runs a lightweight AWS CLI call to verify whether
// credentials are usable. If envOverrides is nil, it uses the current process env.
func checkCredentialsWithEnv(ctx context.Context, envOverrides []string) bool {
	args := []string{"sts", "get-caller-identity", "--output", "json"}
	cmd := exec.CommandContext(ctx, "aws", args...)

	if envOverrides != nil {
		cmd.Env = append(os.Environ(), envOverrides...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// We intentionally don't log here to avoid leaking credentials in logs.
		return false
	}

	// Basic sanity-check that the output looks like JSON.
	var tmp map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &tmp); err != nil {
		return false
	}
	return true
}
