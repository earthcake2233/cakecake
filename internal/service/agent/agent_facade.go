package agent

import (
	"cakecake/internal/model/agent"
	"cakecake/internal/model/dm"
	"context"
	"encoding/json"
	"sync"
	"time"
)

// AgentService facade: domain methods are delegated to the profile, generation
// and feedback services. Client contracts, handlers and tests keep depending
// on AgentService; the facade splits into per-domain proto services at migration.
// All state stays on AgentService (single source of truth) for now.

// MaxProfiles delegates to the corresponding domain service.
func (s *AgentService) MaxProfiles() int {
	return (&AgentProfileService{svc: s}).MaxProfiles()
}

// UnmarshalWelcomeList delegates to the corresponding domain service.
func (s *AgentService) UnmarshalWelcomeList(raw json.RawMessage, fallback []string) ([]string, error) {
	return (&AgentProfileService{svc: s}).UnmarshalWelcomeList(raw, fallback)
}

// NormalizeSlug delegates to the corresponding domain service.
func (s *AgentService) NormalizeSlug(slug string) (string, error) {
	return (&AgentProfileService{svc: s}).NormalizeSlug(slug)
}

// ReloadProfiles delegates to the corresponding domain service.
func (s *AgentService) ReloadProfiles() {
	(&AgentProfileService{svc: s}).ReloadProfiles()
}

// ListAgentProfiles delegates to the corresponding domain service.
func (s *AgentService) ListAgentProfiles(ctx context.Context) ([]agent.AgentProfile, error) {
	return (&AgentProfileService{svc: s}).ListAgentProfiles(ctx)
}

// GetAgentProfile delegates to the corresponding domain service.
func (s *AgentService) GetAgentProfile(ctx context.Context, id uint64) (*agent.AgentProfile, error) {
	return (&AgentProfileService{svc: s}).GetAgentProfile(ctx, id)
}

// CreateAgentProfile delegates to the corresponding domain service.
func (s *AgentService) CreateAgentProfile(ctx context.Context, p *agent.AgentProfile) error {
	return (&AgentProfileService{svc: s}).CreateAgentProfile(ctx, p)
}

// UpdateAgentProfile delegates to the corresponding domain service.
func (s *AgentService) UpdateAgentProfile(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return (&AgentProfileService{svc: s}).UpdateAgentProfile(ctx, id, updates)
}

// DeleteAgentProfile delegates to the corresponding domain service.
func (s *AgentService) DeleteAgentProfile(ctx context.Context, id uint64) error {
	return (&AgentProfileService{svc: s}).DeleteAgentProfile(ctx, id)
}

// CountActiveAgentProfiles delegates to the corresponding domain service.
func (s *AgentService) CountActiveAgentProfiles(ctx context.Context) (int64, error) {
	return (&AgentProfileService{svc: s}).CountActiveAgentProfiles(ctx)
}

// CheckAgentSlugExists delegates to the corresponding domain service.
func (s *AgentService) CheckAgentSlugExists(ctx context.Context, slug string) (bool, error) {
	return (&AgentProfileService{svc: s}).CheckAgentSlugExists(ctx, slug)
}

// UpdateAgentAvatar delegates to the corresponding domain service.
func (s *AgentService) UpdateAgentAvatar(ctx context.Context, id uint64, avatarURL string) error {
	return (&AgentProfileService{svc: s}).UpdateAgentAvatar(ctx, id, avatarURL)
}

// GetGlobalSystemPrompt delegates to the corresponding domain service.
func (s *AgentService) GetGlobalSystemPrompt(ctx context.Context) string {
	return (&AgentProfileService{svc: s}).GetGlobalSystemPrompt(ctx)
}

// UpdateGlobalSystemPrompt delegates to the corresponding domain service.
func (s *AgentService) UpdateGlobalSystemPrompt(ctx context.Context, prompt string) error {
	return (&AgentProfileService{svc: s}).UpdateGlobalSystemPrompt(ctx, prompt)
}

// ProfileCount delegates to the corresponding domain service.
func (s *AgentService) ProfileCount(ctx context.Context) (int64, error) {
	return (&AgentProfileService{svc: s}).ProfileCount(ctx)
}

// CreateAgentBotUser delegates to the corresponding domain service.
func (s *AgentService) CreateAgentBotUser(ctx context.Context, slug, displayName, sign, avatarURL string) (uint64, error) {
	return (&AgentProfileService{svc: s}).CreateAgentBotUser(ctx, slug, displayName, sign, avatarURL)
}

// RenameAgentProfileSlug delegates to the corresponding domain service.
func (s *AgentService) RenameAgentProfileSlug(ctx context.Context, p *agent.AgentProfile, newSlug string) error {
	return (&AgentProfileService{svc: s}).RenameAgentProfileSlug(ctx, p, newSlug)
}

// SyncAgentProfile delegates to the corresponding domain service.
func (s *AgentService) SyncAgentProfile(ctx context.Context, p *agent.AgentProfile) error {
	return (&AgentProfileService{svc: s}).SyncAgentProfile(ctx, p)
}

// EnsureAgentProfiles delegates to the corresponding domain service.
func (s *AgentService) EnsureAgentProfiles(ctx context.Context) error {
	return (&AgentProfileService{svc: s}).EnsureAgentProfiles(ctx)
}

// SetMessageFeedback delegates to the corresponding domain service.
func (s *AgentService) SetMessageFeedback(ctx context.Context, messageID uint64, userID uint64, feedback string) error {
	return (&AgentFeedbackService{svc: s}).SetMessageFeedback(ctx, messageID, userID, feedback)
}

// ListAgentFeedbacks delegates to the corresponding domain service.
func (s *AgentService) ListAgentFeedbacks(ctx context.Context, limit int, offset int) ([]agent.AgentFeedback, error) {
	return (&AgentFeedbackService{svc: s}).ListAgentFeedbacks(ctx, limit, offset)
}

// ListAgentFeedbacksWithContent delegates to the corresponding domain service.
func (s *AgentService) ListAgentFeedbacksWithContent(ctx context.Context, limit int, offset int) ([]AgentFeedbackRow, error) {
	return (&AgentFeedbackService{svc: s}).ListAgentFeedbacksWithContent(ctx, limit, offset)
}

// RunReply delegates to the corresponding domain service.
func (s *AgentService) RunReply(humanID uint64, conv *dm.DmConversation, userText string) {
	(&AgentGenerationService{svc: s}).RunReply(humanID, conv, userText)
}

// ResumeReply delegates to the corresponding domain service.
func (s *AgentService) ResumeReply(uid uint64, convID uint64) {
	(&AgentGenerationService{svc: s}).ResumeReply(uid, convID)
}

func (s *AgentService) resumeReplyLocal(uid uint64, convID uint64) {
	(&AgentGenerationService{svc: s}).resumeReplyLocal(uid, convID)
}

func (s *AgentService) handleControl(uid uint64, ctrl map[string]interface{}) {
	(&AgentGenerationService{svc: s}).handleControl(uid, ctrl)
}

// RegenerateReply delegates to the corresponding domain service.
func (s *AgentService) RegenerateReply(uid uint64, convID uint64) {
	(&AgentGenerationService{svc: s}).RegenerateReply(uid, convID)
}

func (s *AgentService) regenerateWelcome(uid uint64, conv *dm.DmConversation, current string) {
	(&AgentGenerationService{svc: s}).regenerateWelcome(uid, conv, current)
}

func (s *AgentService) continueReply(uid uint64, convID uint64, partial string) {
	(&AgentGenerationService{svc: s}).continueReply(uid, convID, partial)
}

func (s *AgentService) continueFromDraft(ctx context.Context, conv *dm.DmConversation, partial string, genID uint64) (string, error) {
	return (&AgentGenerationService{svc: s}).continueFromDraft(ctx, conv, partial, genID)
}

func (s *AgentService) streamTail(humanID uint64, genID uint64, tail string) {
	(&AgentGenerationService{svc: s}).streamTail(humanID, genID, tail)
}

func (s *AgentService) pushAgentMessage(ctx context.Context, humanID uint64, conv *dm.DmConversation, msg *dm.DmMessage) {
	(&AgentGenerationService{svc: s}).pushAgentMessage(ctx, humanID, conv, msg)
}

func (s *AgentService) attachSuggestions(humanID uint64, messageID uint64, reply string) {
	(&AgentGenerationService{svc: s}).attachSuggestions(humanID, messageID, reply)
}

func (s *AgentService) pushFallback(ctx context.Context, humanID uint64, conv *dm.DmConversation, text string) {
	(&AgentGenerationService{svc: s}).pushFallback(ctx, humanID, conv, text)
}

func (s *AgentService) pushContinueMode(uid uint64, mode string) {
	(&AgentGenerationService{svc: s}).pushContinueMode(uid, mode)
}

func (s *AgentService) latestUserTurnHasAssistantReply(uid uint64, convID uint64) bool {
	return (&AgentGenerationService{svc: s}).latestUserTurnHasAssistantReply(uid, convID)
}

func (s *AgentService) runLock(uid uint64) *sync.Mutex {
	return (&AgentGenerationService{svc: s}).runLock(uid)
}

func (s *AgentService) deltaSender(humanID uint64, genID uint64) func(string) {
	return (&AgentGenerationService{svc: s}).deltaSender(humanID, genID)
}

func (s *AgentService) publishEvent(ctx context.Context, uid uint64, payload interface{}) {
	(&AgentGenerationService{svc: s}).publishEvent(ctx, uid, payload)
}

func (s *AgentService) publishControl(ctx context.Context, uid uint64, payload interface{}) {
	(&AgentGenerationService{svc: s}).publishControl(ctx, uid, payload)
}

func (s *AgentService) draftText(uid uint64) string {
	return (&AgentGenerationService{svc: s}).draftText(uid)
}

func (s *AgentService) currentGenID(uid uint64) uint64 {
	return (&AgentGenerationService{svc: s}).currentGenID(uid)
}

func (s *AgentService) generationState(uid uint64) *agentGenState {
	return (&AgentGenerationService{svc: s}).generationState(uid)
}

func (s *AgentService) beginGeneration(uid uint64, cancel context.CancelFunc) uint64 {
	return (&AgentGenerationService{svc: s}).beginGeneration(uid, cancel)
}

func (s *AgentService) endGeneration(uid uint64, genID uint64) {
	(&AgentGenerationService{svc: s}).endGeneration(uid, genID)
}

func (s *AgentService) saveDraft(uid uint64, st *agentGenState) {
	(&AgentGenerationService{svc: s}).saveDraft(uid, st)
}

func (s *AgentService) clearDraft(uid uint64) {
	(&AgentGenerationService{svc: s}).clearDraft(uid)
}

func (s *AgentService) supersedeGeneration(uid uint64) {
	(&AgentGenerationService{svc: s}).supersedeGeneration(uid)
}

func (s *AgentService) dropCurrentGeneration(uid uint64) {
	(&AgentGenerationService{svc: s}).dropCurrentGeneration(uid)
}

func (s *AgentService) hasRunningGeneration(uid uint64) bool {
	return (&AgentGenerationService{svc: s}).hasRunningGeneration(uid)
}

func (s *AgentService) storePendingReply(uid uint64, genID uint64, conv *dm.DmConversation, result *GenerateReplyResult) {
	(&AgentGenerationService{svc: s}).storePendingReply(uid, genID, conv, result)
}

func (s *AgentService) takePendingReply(uid uint64) (*dm.DmConversation, *GenerateReplyResult, uint64, bool) {
	return (&AgentGenerationService{svc: s}).takePendingReply(uid)
}

// PauseGeneration delegates to the corresponding domain service.
func (s *AgentService) PauseGeneration(uid uint64) {
	(&AgentGenerationService{svc: s}).PauseGeneration(uid)
}

func (s *AgentService) pauseGeneration(uid uint64) {
	(&AgentGenerationService{svc: s}).pauseGeneration(uid)
}

func (s *AgentService) resumeGeneration(uid uint64) {
	(&AgentGenerationService{svc: s}).resumeGeneration(uid)
}

func (s *AgentService) isGenerationPaused(uid uint64) bool {
	return (&AgentGenerationService{svc: s}).isGenerationPaused(uid)
}

func (s *AgentService) clearGenerationState(uid uint64) {
	(&AgentGenerationService{svc: s}).clearGenerationState(uid)
}

func (s *AgentService) writeSnapshot(uid uint64, snap *genSnapshot) {
	(&AgentGenerationService{svc: s}).writeSnapshot(uid, snap)
}

func (s *AgentService) readSnapshot(uid uint64) *genSnapshot {
	return (&AgentGenerationService{svc: s}).readSnapshot(uid)
}

func (s *AgentService) clearSnapshot(uid uint64, genID uint64) {
	(&AgentGenerationService{svc: s}).clearSnapshot(uid, genID)
}

func (s *AgentService) updateSnapshotPaused(uid uint64, genID uint64, paused bool, pauseSeq uint64) {
	(&AgentGenerationService{svc: s}).updateSnapshotPaused(uid, genID, paused, pauseSeq)
}

func (s *AgentService) snapshotRunning(uid uint64, genID uint64, convID uint64) {
	(&AgentGenerationService{svc: s}).snapshotRunning(uid, genID, convID)
}

func (s *AgentService) snapshotPending(uid uint64, genID uint64, convID uint64, result *GenerateReplyResult) {
	(&AgentGenerationService{svc: s}).snapshotPending(uid, genID, convID, result)
}

func (s *AgentService) gatewayReady() bool {
	return (&AgentGenerationService{svc: s}).gatewayReady()
}

func (s *AgentService) quotaKey(userID uint64) string {
	return (&AgentGenerationService{svc: s}).quotaKey(userID)
}

// CheckQuota delegates to the corresponding domain service.
func (s *AgentService) CheckQuota(ctx context.Context, userID uint64) bool {
	return (&AgentGenerationService{svc: s}).CheckQuota(ctx, userID)
}

// IncrQuota delegates to the corresponding domain service.
func (s *AgentService) IncrQuota(ctx context.Context, userID uint64) {
	(&AgentGenerationService{svc: s}).IncrQuota(ctx, userID)
}

// EnsureForUser delegates to the corresponding domain service.
func (s *AgentService) EnsureForUser(humanID uint64) error {
	return (&AgentProfileService{svc: s}).EnsureForUser(humanID)
}

// IsAgentConversation delegates to the corresponding domain service.
func (s *AgentService) IsAgentConversation(conv *dm.DmConversation) bool {
	return (&AgentProfileService{svc: s}).IsAgentConversation(conv)
}

// IsBotUser delegates to the corresponding domain service.
func (s *AgentService) IsBotUser(uid uint64) bool {
	return (&AgentProfileService{svc: s}).IsBotUser(uid)
}

func (s *AgentService) profileForConversation(conv *dm.DmConversation) (*agent.AgentProfile, error) {
	return (&AgentProfileService{svc: s}).profileForConversation(conv)
}

// PostAssistantMessage delegates to the corresponding domain service.
func (s *AgentService) PostAssistantMessage(conv *dm.DmConversation, humanID uint64, content string, extra ...string) (*dm.DmMessage, error) {
	return (&AgentGenerationService{svc: s}).PostAssistantMessage(conv, humanID, content, extra...)
}

func (s *AgentService) applyDynamicGatewayConfig() {
	(&AgentGenerationService{svc: s}).applyDynamicGatewayConfig()
}

// GenerateReply delegates to the corresponding domain service.
func (s *AgentService) GenerateReply(ctx context.Context, conv *dm.DmConversation, userText string) (*GenerateReplyResult, error) {
	return (&AgentGenerationService{svc: s}).GenerateReply(ctx, conv, userText)
}

// GenerateSuggestions delegates to the corresponding domain service.
func (s *AgentService) GenerateSuggestions(ctx context.Context, reply string) []string {
	return (&AgentGenerationService{svc: s}).GenerateSuggestions(ctx, reply)
}

// UpdateMessageSuggestions delegates to the corresponding domain service.
func (s *AgentService) UpdateMessageSuggestions(ctx context.Context, messageID uint64, suggestions []string) error {
	return (&AgentGenerationService{svc: s}).UpdateMessageSuggestions(ctx, messageID, suggestions)
}

func (s *AgentService) generateSuggestions(ctx context.Context, reply string) []string {
	return (&AgentGenerationService{svc: s}).generateSuggestions(ctx, reply)
}

func (s *AgentService) agentReplyTimeout() time.Duration {
	return (&AgentGenerationService{svc: s}).agentReplyTimeout()
}

func (s *AgentService) agentSystemPrompt(profile *agent.AgentProfile) func() {
	return (&AgentGenerationService{svc: s}).agentSystemPrompt(profile)
}

// ResetConversation delegates to the corresponding domain service.
func (s *AgentService) ResetConversation(ctx context.Context, conv *dm.DmConversation, humanID uint64) (*dm.DmMessage, error) {
	return (&AgentGenerationService{svc: s}).ResetConversation(ctx, conv, humanID)
}

func (s *AgentService) enabledTools() map[string]bool {
	return (&AgentGenerationService{svc: s}).enabledTools()
}

func (s *AgentService) setupToolCallbacks(traceID string, humanID uint64) {
	(&AgentGenerationService{svc: s}).setupToolCallbacks(traceID, humanID)
}

func (s *AgentService) clearToolCallbacks() {
	(&AgentGenerationService{svc: s}).clearToolCallbacks()
}
