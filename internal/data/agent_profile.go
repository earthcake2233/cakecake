package data

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"minibili/internal/config"
	"minibili/internal/model"
)

var agentSlugRe = regexp.MustCompile(`^[a-z][a-z0-9_]{1,30}$`)

const maxAgentProfiles = 12

// EnsureAgentProfiles migrates legacy singleton settings and guarantees at least one profile.
func EnsureAgentProfiles(db *gorm.DB, cfg *config.C, lg *zap.Logger) error {
	if db == nil {
		return nil
	}
	_ = EnsureDefaultAgentSettings(db, lg)

	var n int64
	_ = db.Model(&model.AgentProfile{}).Count(&n).Error
	if n > 0 {
		return backfillDmAgentProfileIDs(db, lg)
	}

	displayName := defaultAgentDisplayName
	sign := defaultAgentSign
	systemPrompt := defaultAgentSystemPrompt
	welcome := []string{defaultAgentWelcome}
	avatarURL := ""
	enabled := true

	if st, err := GetAgentSettings(db); err == nil && st != nil {
		if v := strings.TrimSpace(st.DisplayName); v != "" {
			displayName = v
		}
		if v := strings.TrimSpace(st.Sign); v != "" {
			sign = v
		}
		if v := strings.TrimSpace(st.SystemPrompt); v != "" {
			systemPrompt = v
		}
		if v := strings.TrimSpace(st.WelcomeMessage); v != "" {
			welcome = []string{v}
		}
		avatarURL = strings.TrimSpace(st.AvatarURL)
		enabled = st.AssistantEnabled
	}

	botID, err := findOrCreateLegacyBotUser(db, cfg, displayName, sign, avatarURL, lg)
	if err != nil {
		return err
	}

	p := model.AgentProfile{
		Slug:                "default",
		BotUserID:           botID,
		DisplayName:         displayName,
		AvatarURL:           avatarURL,
		Sign:                sign,
		SystemPrompt:        systemPrompt,
		WelcomeMessagesJSON: model.EncodeWelcomeMessages(welcome),
		SortOrder:           0,
		Enabled:             enabled,
	}
	if err := db.Create(&p).Error; err != nil {
		return err
	}
	if lg != nil {
		lg.Info("default agent profile created", zap.Uint64("profile_id", p.ID), zap.Uint64("bot_user_id", botID))
	}
	return backfillDmAgentProfileIDs(db, lg)
}

func findOrCreateLegacyBotUser(db *gorm.DB, cfg *config.C, displayName, sign, avatarURL string, lg *zap.Logger) (uint64, error) {
	username := "minibili_ai"
	if cfg != nil && strings.TrimSpace(cfg.AgentBotUsername) != "" {
		username = strings.TrimSpace(cfg.AgentBotUsername)
	}
	var u model.User
	err := db.Where("username = ?", username).First(&u).Error
	if err == nil {
		_ = syncAgentProfileToUser(db, &model.AgentProfile{
			DisplayName: displayName,
			AvatarURL:   avatarURL,
			Sign:        sign,
		}, u.ID)
		return u.ID, nil
	}
	if err != gorm.ErrRecordNotFound {
		return 0, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(fmt.Sprintf("bot-%s-not-for-login", username)), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	u = model.User{
		Username:     username,
		PasswordHash: string(hash),
		Nickname:     displayName,
		AvatarURL:    avatarURL,
		Sign:         sign,
	}
	if err := db.Create(&u).Error; err != nil {
		return 0, err
	}
	cid := model.FormatCakeID(u.ID)
	_ = db.Model(&u).Update("cake_id", cid).Error
	if lg != nil {
		lg.Info("legacy agent bot user created", zap.String("username", username), zap.Uint64("user_id", u.ID))
	}
	return u.ID, nil
}


func backfillDmAgentProfileIDs(db *gorm.DB, lg *zap.Logger) error {
	var profiles []model.AgentProfile
	if err := db.Find(&profiles).Error; err != nil {
		return err
	}
	byBot := map[uint64]uint64{}
	for i := range profiles {
		byBot[profiles[i].BotUserID] = profiles[i].ID
	}
	var convs []model.DmConversation
	_ = db.Where("kind = ?", model.DmKindAgent).Find(&convs).Error
	for i := range convs {
		c := &convs[i]
		if c.AgentProfileID > 0 {
			continue
		}
		pid := byBot[c.UserLow]
		if pid == 0 {
			pid = byBot[c.UserHigh]
		}
		if pid > 0 {
			_ = db.Model(c).Update("agent_profile_id", pid).Error
		}
	}
	return nil
}

// ListAgentProfiles returns all profiles for admin (newest sort_order first in UI handled by handler).
func ListAgentProfiles(db *gorm.DB) ([]model.AgentProfile, error) {
	var list []model.AgentProfile
	if err := db.Order("sort_order ASC, id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ListEnabledAgentProfiles returns enabled personas for user-facing ensure.
func ListEnabledAgentProfiles(db *gorm.DB) ([]model.AgentProfile, error) {
	var list []model.AgentProfile
	if err := db.Where("enabled = ?", true).Order("sort_order ASC, id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// GetAgentProfile loads one profile by id.
func GetAgentProfile(db *gorm.DB, id uint64) (*model.AgentProfile, error) {
	var p model.AgentProfile
	if err := db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// GetAgentProfileByBotUserID finds profile for a system bot account.
func GetAgentProfileByBotUserID(db *gorm.DB, botUserID uint64) (*model.AgentProfile, error) {
	var p model.AgentProfile
	if err := db.Where("bot_user_id = ?", botUserID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// PickWelcomeMessage chooses one welcome line at random.
func PickWelcomeMessage(p *model.AgentProfile) string {
	if p == nil {
		return defaultAgentWelcome
	}
	list := model.ParseWelcomeMessages(p.WelcomeMessagesJSON)
	if len(list) == 0 {
		return defaultAgentWelcome
	}
	if len(list) == 1 {
		return list[0]
	}
	max := big.NewInt(int64(len(list)))
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return list[0]
	}
	return list[n.Int64()]
}

// EnsureAgentConversationForProfile creates a user鈫攂ot thread for one persona.
func EnsureAgentConversationForProfile(db *gorm.DB, humanID uint64, profile *model.AgentProfile) (*model.DmConversation, bool, error) {
	if db == nil || profile == nil || humanID == 0 || profile.BotUserID == 0 || humanID == profile.BotUserID {
		return nil, false, fmt.Errorf("invalid agent conversation")
	}
	if !profile.Enabled {
		return nil, false, nil
	}
	low, high := humanID, profile.BotUserID
	if low > high {
		low, high = high, low
	}
	var conv model.DmConversation
	err := db.Where("user_low = ? AND user_high = ?", low, high).First(&conv).Error
	if err == nil {
		updates := map[string]interface{}{
			"kind":             model.DmKindAgent,
			"agent_profile_id": profile.ID,
		}
		_ = db.Model(&conv).Updates(updates).Error
		conv.Kind = model.DmKindAgent
		conv.AgentProfileID = profile.ID
		ensureDmParticipants(db, conv.ID, humanID, profile.BotUserID)
		return &conv, false, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, false, err
	}
	welcome := PickWelcomeMessage(profile)
	now := db.NowFunc()
	conv = model.DmConversation{
		UserLow:          low,
		UserHigh:         high,
		Kind:             model.DmKindAgent,
		AgentProfileID:   profile.ID,
		LastMessageAt:    now,
		LastPreview:      welcome,
	}
	if err := db.Create(&conv).Error; err != nil {
		return nil, false, err
	}
	ensureDmParticipants(db, conv.ID, humanID, profile.BotUserID)
	msg := model.DmMessage{
		ConversationID: conv.ID,
		SenderID:       profile.BotUserID,
		Role:           "assistant",
		Content:        welcome,
		CreatedAt:      now,
	}
	_ = db.Create(&msg).Error
	return &conv, true, nil
}

// EnsureAllAgentConversationsForUser ensures threads for each enabled profile.
func EnsureAllAgentConversationsForUser(db *gorm.DB, humanID uint64) error {
	profiles, err := ListEnabledAgentProfiles(db)
	if err != nil {
		return err
	}
	for i := range profiles {
		_, _, err := EnsureAgentConversationForProfile(db, humanID, &profiles[i])
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateAgentBotUser creates a non-login system user for a new profile.
func CreateAgentBotUser(db *gorm.DB, slug, displayName, sign, avatarURL string) (uint64, error) {
	username := AgentBotUsername(slug)
	hash, err := bcrypt.GenerateFromPassword([]byte(fmt.Sprintf("bot-%s-not-for-login", username)), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	u := model.User{
		Username:     username,
		PasswordHash: string(hash),
		Nickname:     displayName,
		AvatarURL:    avatarURL,
		Sign:         sign,
	}
	if err := db.Create(&u).Error; err != nil {
		return 0, err
	}
	cid := model.FormatCakeID(u.ID)
	_ = db.Model(&u).Update("cake_id", cid).Error
	return u.ID, nil
}

// AgentBotUsername builds the internal username for a profile slug.
func AgentBotUsername(slug string) string {
	return "ai_" + strings.TrimSpace(slug)
}

// NormalizeAgentSlug validates and normalizes slug input.
func NormalizeAgentSlug(slug string) (string, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if !agentSlugRe.MatchString(slug) {
		return "", fmt.Errorf("invalid slug")
	}
	return slug, nil
}

// RenameAgentProfileSlug updates profile slug and the linked bot user's username.
func RenameAgentProfileSlug(db *gorm.DB, p *model.AgentProfile, newSlug string) error {
	if db == nil || p == nil {
		return fmt.Errorf("invalid profile")
	}
	newSlug, err := NormalizeAgentSlug(newSlug)
	if err != nil {
		return err
	}
	if newSlug == p.Slug {
		return nil
	}
	var taken int64
	if err := db.Model(&model.AgentProfile{}).
		Where("slug = ? AND id <> ?", newSlug, p.ID).
		Count(&taken).Error; err != nil {
		return err
	}
	if taken > 0 {
		return fmt.Errorf("slug taken")
	}
	newUsername := AgentBotUsername(newSlug)
	if err := db.Model(&model.User{}).
		Where("username = ? AND id <> ?", newUsername, p.BotUserID).
		Count(&taken).Error; err != nil {
		return err
	}
	if taken > 0 {
		return fmt.Errorf("username taken")
	}
	tx := db.Begin()
	if err := tx.Model(p).Update("slug", newSlug).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Model(&model.User{}).Where("id = ?", p.BotUserID).
		Update("username", newUsername).Error; err != nil {
		tx.Rollback()
		return err
	}
	p.Slug = newSlug
	return tx.Commit().Error
}

// SyncAgentProfile copies profile display fields onto the bot user row.
func SyncAgentProfile(db *gorm.DB, p *model.AgentProfile) error {
	if p == nil {
		return nil
	}
	return syncAgentProfileToUser(db, p, p.BotUserID)
}

func syncAgentProfileToUser(db *gorm.DB, p *model.AgentProfile, botUserID uint64) error {
	if db == nil || p == nil || botUserID == 0 {
		return nil
	}
	name := strings.TrimSpace(p.DisplayName)
	if name == "" {
		name = defaultAgentDisplayName
	}
	return db.Model(&model.User{}).Where("id = ?", botUserID).Updates(map[string]interface{}{
		"nickname":   name,
		"avatar_url": strings.TrimSpace(p.AvatarURL),
		"sign":       strings.TrimSpace(p.Sign),
	}).Error
}

// ProfileCount returns total profiles (for create limit).
func ProfileCount(db *gorm.DB) (int64, error) {
	var n int64
	err := db.Model(&model.AgentProfile{}).Count(&n).Error
	return n, err
}

// MaxAgentProfilesLimit is the ops-configurable cap.
func MaxAgentProfilesLimit() int {
	return maxAgentProfiles
}

// MarshalWelcomeList helper for handlers.
func MarshalWelcomeList(list []string) (string, error) {
	if len(list) == 0 {
		return "", fmt.Errorf("empty welcome list")
	}
	for i, s := range list {
		if strings.TrimSpace(s) == "" {
			return "", fmt.Errorf("empty welcome at %d", i)
		}
	}
	return model.EncodeWelcomeMessages(list), nil
}

// UnmarshalWelcomeList from API request []string.
func UnmarshalWelcomeList(raw json.RawMessage, fallback []string) ([]string, error) {
	if len(raw) == 0 {
		return fallback, nil
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(list))
	for _, s := range list {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty welcome list")
	}
	return out, nil
}

func ensureDmParticipants(db *gorm.DB, convID, humanID, botID uint64) {
	for _, uid := range []uint64{humanID, botID} {
		var p model.DmParticipant
		if db.Where("conversation_id = ? AND user_id = ?", convID, uid).First(&p).Error == gorm.ErrRecordNotFound {
			_ = db.Create(&model.DmParticipant{
				ConversationID: convID,
				UserID:         uid,
				UnreadCount:    0,
			}).Error
		}
	}
}




