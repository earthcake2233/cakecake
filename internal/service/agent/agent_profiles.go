package agent

import (
	"encoding/json"

	"cakecake/internal/data"
	"cakecake/internal/model/agent"
	"context"
)

// AgentProfileService owns the agent-profile domain (CRUD, bot user, slug, prompt).
type AgentProfileService struct {
	svc *AgentService
}

// MaxProfiles returns the maximum number of agent profiles allowed.
func (ps *AgentProfileService) MaxProfiles() int {
	return data.MaxAgentProfilesLimit()
}

// UnmarshalWelcomeList parses welcome messages from JSON, falling back when needed.
func (ps *AgentProfileService) UnmarshalWelcomeList(raw json.RawMessage, fallback []string) ([]string, error) {
	return data.UnmarshalWelcomeList(raw, fallback)
}

// NormalizeSlug normalizes an agent profile slug.
func (ps *AgentProfileService) NormalizeSlug(slug string) (string, error) {
	return data.NormalizeAgentSlug(slug)
}

// ReloadProfiles reloads agent profiles (no-op; profiles are ensured lazily).
func (ps *AgentProfileService) ReloadProfiles() {}

// ListAgentProfiles returns all agent profiles.
func (ps *AgentProfileService) ListAgentProfiles(ctx context.Context) ([]agent.AgentProfile, error) {
	return ps.svc.Store.ListAgentProfiles(ctx)
}

// GetAgentProfile returns an agent profile by ID.
func (ps *AgentProfileService) GetAgentProfile(ctx context.Context, id uint64) (*agent.AgentProfile, error) {
	return ps.svc.Store.GetAgentProfile(id)
}

// CreateAgentProfile creates a new agent profile.
func (ps *AgentProfileService) CreateAgentProfile(ctx context.Context, p *agent.AgentProfile) error {
	return ps.svc.Store.CreateAgentProfile(ctx, p)
}

// UpdateAgentProfile updates an agent profile.
func (ps *AgentProfileService) UpdateAgentProfile(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return ps.svc.Store.UpdateAgentProfile(ctx, id, updates)
}

// DeleteAgentProfile deletes an agent profile by ID.
func (ps *AgentProfileService) DeleteAgentProfile(ctx context.Context, id uint64) error {
	return ps.svc.Store.DeleteAgentProfile(ctx, id)
}

// CountActiveAgentProfiles returns the count of enabled agent profiles.
func (ps *AgentProfileService) CountActiveAgentProfiles(ctx context.Context) (int64, error) {
	return ps.svc.Store.CountActiveAgentProfiles(ctx)
}

// CheckAgentSlugExists checks if a slug is already taken.
func (ps *AgentProfileService) CheckAgentSlugExists(ctx context.Context, slug string) (bool, error) {
	return ps.svc.Store.CheckAgentSlugExists(ctx, slug)
}

// UpdateAgentAvatar updates the avatar_url of an agent profile.
func (ps *AgentProfileService) UpdateAgentAvatar(ctx context.Context, id uint64, avatarURL string) error {
	return ps.svc.Store.UpdateAgentAvatar(ctx, id, avatarURL)
}

// GetGlobalSystemPrompt returns the global system prompt from agent settings.
func (ps *AgentProfileService) GetGlobalSystemPrompt(ctx context.Context) string {
	return ps.svc.Store.GetGlobalSystemPrompt()
}

// ProfileCount returns total number of agent profiles.
func (ps *AgentProfileService) ProfileCount(ctx context.Context) (int64, error) {
	return ps.svc.Store.ProfileCount(ctx)
}

// CreateAgentBotUser creates a non-login system user for a new profile.
func (ps *AgentProfileService) CreateAgentBotUser(ctx context.Context, slug, displayName, sign, avatarURL string) (uint64, error) {
	return ps.svc.Store.CreateAgentBotUser(ctx, slug, displayName, sign, avatarURL)
}

// RenameAgentProfileSlug updates a profile slug and the linked bot user username.
func (ps *AgentProfileService) RenameAgentProfileSlug(ctx context.Context, p *agent.AgentProfile, newSlug string) error {
	return ps.svc.Store.RenameAgentProfileSlug(ctx, p, newSlug)
}

// SyncAgentProfile copies profile display fields onto the bot user row.
func (ps *AgentProfileService) SyncAgentProfile(ctx context.Context, p *agent.AgentProfile) error {
	return ps.svc.Store.SyncAgentProfile(ctx, p)
}

// EnsureAgentProfiles migrates legacy settings and guarantees at least one profile.
func (ps *AgentProfileService) EnsureAgentProfiles(ctx context.Context) error {
	return ps.svc.Store.EnsureAgentProfiles(ps.svc.Cfg, ps.svc.Log)
}

// AgentBotUsername returns the internal username for a profile slug.
func AgentBotUsername(slug string) string {
	return data.AgentBotUsername(slug)
}
