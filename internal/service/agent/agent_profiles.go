package agent

import (
	"encoding/json"

	"cakecake/internal/data"
	"cakecake/internal/model/agent"
	"context"
)

// MaxProfiles returns the maximum number of agent profiles allowed.
func (s *AgentService) MaxProfiles() int {
	return data.MaxAgentProfilesLimit()
}

// UnmarshalWelcomeList parses welcome messages from JSON, falling back when needed.
func (s *AgentService) UnmarshalWelcomeList(raw json.RawMessage, fallback []string) ([]string, error) {
	return data.UnmarshalWelcomeList(raw, fallback)
}

// NormalizeSlug normalizes an agent profile slug.
func (s *AgentService) NormalizeSlug(slug string) (string, error) {
	return data.NormalizeAgentSlug(slug)
}

// ReloadProfiles reloads agent profiles (no-op; profiles are ensured lazily).
func (s *AgentService) ReloadProfiles() {}

// ListAgentProfiles returns all agent profiles.
func (s *AgentService) ListAgentProfiles(ctx context.Context) ([]agent.AgentProfile, error) {
	return s.Store.ListAgentProfiles(ctx)
}

// GetAgentProfile returns an agent profile by ID.
func (s *AgentService) GetAgentProfile(ctx context.Context, id uint64) (*agent.AgentProfile, error) {
	return s.Store.GetAgentProfile(id)
}

// CreateAgentProfile creates a new agent profile.
func (s *AgentService) CreateAgentProfile(ctx context.Context, p *agent.AgentProfile) error {
	return s.Store.CreateAgentProfile(ctx, p)
}

// UpdateAgentProfile updates an agent profile.
func (s *AgentService) UpdateAgentProfile(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return s.Store.UpdateAgentProfile(ctx, id, updates)
}

// DeleteAgentProfile deletes an agent profile by ID.
func (s *AgentService) DeleteAgentProfile(ctx context.Context, id uint64) error {
	return s.Store.DeleteAgentProfile(ctx, id)
}

// CountActiveAgentProfiles returns the count of enabled agent profiles.
func (s *AgentService) CountActiveAgentProfiles(ctx context.Context) (int64, error) {
	return s.Store.CountActiveAgentProfiles(ctx)
}

// CheckAgentSlugExists checks if a slug is already taken.
func (s *AgentService) CheckAgentSlugExists(ctx context.Context, slug string) (bool, error) {
	return s.Store.CheckAgentSlugExists(ctx, slug)
}

// UpdateAgentAvatar updates the avatar_url of an agent profile.
func (s *AgentService) UpdateAgentAvatar(ctx context.Context, id uint64, avatarURL string) error {
	return s.Store.UpdateAgentAvatar(ctx, id, avatarURL)
}

// GetGlobalSystemPrompt returns the global system prompt from agent settings.
func (s *AgentService) GetGlobalSystemPrompt(ctx context.Context) string {
	return s.Store.GetGlobalSystemPrompt()
}

// ProfileCount returns total number of agent profiles.
func (s *AgentService) ProfileCount(ctx context.Context) (int64, error) {
	return s.Store.ProfileCount(ctx)
}

// CreateAgentBotUser creates a non-login system user for a new profile.
func (s *AgentService) CreateAgentBotUser(ctx context.Context, slug, displayName, sign, avatarURL string) (uint64, error) {
	return s.Store.CreateAgentBotUser(ctx, slug, displayName, sign, avatarURL)
}

// RenameAgentProfileSlug updates a profile slug and the linked bot user username.
func (s *AgentService) RenameAgentProfileSlug(ctx context.Context, p *agent.AgentProfile, newSlug string) error {
	return s.Store.RenameAgentProfileSlug(ctx, p, newSlug)
}

// SyncAgentProfile copies profile display fields onto the bot user row.
func (s *AgentService) SyncAgentProfile(ctx context.Context, p *agent.AgentProfile) error {
	return s.Store.SyncAgentProfile(ctx, p)
}

// EnsureAgentProfiles migrates legacy settings and guarantees at least one profile.
func (s *AgentService) EnsureAgentProfiles(ctx context.Context) error {
	return s.Store.EnsureAgentProfiles(s.Cfg, s.Log)
}

// AgentBotUsername returns the internal username for a profile slug.
func AgentBotUsername(slug string) string {
	return data.AgentBotUsername(slug)
}
