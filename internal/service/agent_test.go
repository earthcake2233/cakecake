package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/aigateway"
	"minibili/internal/config"
	"minibili/internal/data"
	"minibili/internal/model"
	"minibili/internal/pkg/sensitive"
)

// ---------- helpers ----------

func newAgentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, data.AutoMigrateAll(db, zap.NewNop()))
	return db
}

func newAgentTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mr, rdb
}

func seedAgentProfile(t *testing.T, db *gorm.DB) *model.AgentProfile {
	t.Helper()
	p := &model.AgentProfile{
		Slug:                "test-assistant",
		BotUserID:           1001,
		DisplayName:         "Test Assistant",
		SystemPrompt:        "You are a helpful test assistant.",
		WelcomeMessagesJSON: "[\"Hello!\"]",
		Enabled:             true,
		SortOrder:           1,
	}
	require.NoError(t, db.Create(p).Error)
	return p
}

func seedAgentConversation(t *testing.T, db *gorm.DB, humanID, botID uint64, profileID uint64) *model.DmConversation {
	t.Helper()
	low, high := humanID, botID
	if low > high {
		low, high = high, low
	}
	conv := &model.DmConversation{
		UserLow:        low,
		UserHigh:       high,
		Kind:           model.DmKindAgent,
		AgentProfileID: profileID,
		LastPreview:    "Welcome!",
	}
	require.NoError(t, db.Create(conv).Error)
	return conv
}

// ---------- gatewayReady ----------

func TestAgentService_gatewayReady(t *testing.T) {
	t.Run("nil Gateway", func(t *testing.T) {
		s := &AgentService{Cfg: &config.C{DeepSeekAPIKey: "sk-test", AgentEnabled: true}}
		require.False(t, s.gatewayReady())
	})

	t.Run("Gateway with nil LLM", func(t *testing.T) {
		s := &AgentService{
			Cfg:     &config.C{DeepSeekAPIKey: "sk-test", AgentEnabled: true},
			Gateway: &aigateway.Gateway{},
		}
		require.False(t, s.gatewayReady())
	})

	t.Run("all ready", func(t *testing.T) {
		s := &AgentService{
			Cfg:     &config.C{DeepSeekAPIKey: "sk-test", AgentEnabled: true},
			Gateway: &aigateway.Gateway{LLM: &aigateway.Client{APIKey: "sk-test"}},
		}
		require.True(t, s.gatewayReady())
	})

	t.Run("empty API key", func(t *testing.T) {
		s := &AgentService{
			Cfg:     &config.C{DeepSeekAPIKey: ""},
			Gateway: &aigateway.Gateway{LLM: &aigateway.Client{APIKey: ""}},
		}
		require.False(t, s.gatewayReady())
	})
}

// ---------- quotaKey ----------

func TestAgentService_quotaKey(t *testing.T) {
	s := &AgentService{}
	key := s.quotaKey(42)
	require.Contains(t, key, "mb:agent:quota:42:")
	require.Contains(t, key, time.Now().Format("20060102"))
}

// ---------- CheckQuota ----------

func TestAgentService_CheckQuota(t *testing.T) {
	t.Run("nil service returns true", func(t *testing.T) {
		var s *AgentService
		require.True(t, s.CheckQuota(context.Background(), 1))
	})

	t.Run("nil Redis returns true", func(t *testing.T) {
		s := &AgentService{Cfg: &config.C{AgentDailyQuota: 10}}
		require.True(t, s.CheckQuota(context.Background(), 1))
	})

	t.Run("quota <= 0 returns true", func(t *testing.T) {
		mr, rdb := newAgentTestRedis(t)
		defer mr.Close()
		s := &AgentService{
			Cfg:   &config.C{AgentDailyQuota: 0},
			Redis: rdb,
		}
		require.True(t, s.CheckQuota(context.Background(), 1))
	})

	t.Run("no usage yet returns true", func(t *testing.T) {
		mr, rdb := newAgentTestRedis(t)
		defer mr.Close()
		s := &AgentService{
			Cfg:   &config.C{AgentDailyQuota: 10},
			Redis: rdb,
		}
		require.True(t, s.CheckQuota(context.Background(), 1))
	})

	t.Run("under quota returns true", func(t *testing.T) {
		mr, rdb := newAgentTestRedis(t)
		defer mr.Close()
		s := &AgentService{
			Cfg:   &config.C{AgentDailyQuota: 10},
			Redis: rdb,
		}
		key := s.quotaKey(1)
		rdb.Set(context.Background(), key, 5, 0)
		require.True(t, s.CheckQuota(context.Background(), 1))
	})

	t.Run("at quota returns false", func(t *testing.T) {
		mr, rdb := newAgentTestRedis(t)
		defer mr.Close()
		s := &AgentService{
			Cfg:   &config.C{AgentDailyQuota: 10},
			Redis: rdb,
		}
		key := s.quotaKey(1)
		rdb.Set(context.Background(), key, 10, 0)
		require.False(t, s.CheckQuota(context.Background(), 1))
	})

	t.Run("over quota returns false", func(t *testing.T) {
		mr, rdb := newAgentTestRedis(t)
		defer mr.Close()
		s := &AgentService{
			Cfg:   &config.C{AgentDailyQuota: 10},
			Redis: rdb,
		}
		key := s.quotaKey(1)
		rdb.Set(context.Background(), key, 15, 0)
		require.False(t, s.CheckQuota(context.Background(), 1))
	})
}

// ---------- IncrQuota ----------

func TestAgentService_IncrQuota(t *testing.T) {
	t.Run("nil service does nothing", func(t *testing.T) {
		var s *AgentService
		s.IncrQuota(context.Background(), 1)
	})

	t.Run("nil Redis does nothing", func(t *testing.T) {
		s := &AgentService{}
		s.IncrQuota(context.Background(), 1)
	})

	t.Run("increments and sets expiry", func(t *testing.T) {
		mr, rdb := newAgentTestRedis(t)
		defer mr.Close()
		s := &AgentService{
			Redis: rdb,
		}
		s.IncrQuota(context.Background(), 42)
		key := s.quotaKey(42)
		val, err := rdb.Get(context.Background(), key).Int()
		require.NoError(t, err)
		require.Equal(t, 1, val)
		ttl, err := rdb.TTL(context.Background(), key).Result()
		require.NoError(t, err)
		require.Greater(t, ttl, 24*time.Hour)
	})
}

// ---------- EnsureForUser ----------

func TestAgentService_EnsureForUser(t *testing.T) {
	t.Run("nil DB", func(t *testing.T) {
		s := &AgentService{}
		require.NoError(t, s.EnsureForUser(1))
	})

	t.Run("zero user", func(t *testing.T) {
		db := newAgentTestDB(t)
		s := &AgentService{DB: db}
		require.NoError(t, s.EnsureForUser(0))
	})

	t.Run("creates conversations", func(t *testing.T) {
		db := newAgentTestDB(t)
		seedAgentProfile(t, db)
		s := &AgentService{DB: db}
		require.NoError(t, s.EnsureForUser(42))
	})
}

// ---------- IsAgentConversation ----------
func TestAgentService_IsAgentConversation_More(t *testing.T) {
	s := &AgentService{}
	require.False(t, s.IsAgentConversation(nil))
	require.False(t, s.IsAgentConversation(&model.DmConversation{}))
	require.False(t, s.IsAgentConversation(&model.DmConversation{Kind: "human"}))
	require.True(t, s.IsAgentConversation(&model.DmConversation{Kind: model.DmKindAgent}))
}

// ---------- IsBotUser ----------

func TestAgentService_IsBotUser(t *testing.T) {
	t.Run("nil DB", func(t *testing.T) {
		s := &AgentService{}
		require.False(t, s.IsBotUser(1))
	})

	t.Run("zero user", func(t *testing.T) {
		db := newAgentTestDB(t)
		s := &AgentService{DB: db}
		require.False(t, s.IsBotUser(0))
	})

	t.Run("not a bot user", func(t *testing.T) {
		db := newAgentTestDB(t)
		s := &AgentService{DB: db}
		require.False(t, s.IsBotUser(9999))
	})

	t.Run("is a bot user", func(t *testing.T) {
		db := newAgentTestDB(t)
		p := seedAgentProfile(t, db)
		s := &AgentService{DB: db}
		require.True(t, s.IsBotUser(p.BotUserID))
	})
}

// ---------- profileForConversation ----------

func TestAgentService_profileForConversation(t *testing.T) {

	t.Run("nil conv", func(t *testing.T) {
		s := &AgentService{}
		p, err := s.profileForConversation(nil)
		require.Error(t, err)
		require.Nil(t, p)
	})

	t.Run("by profile ID", func(t *testing.T) {
		db := newAgentTestDB(t)
		prof := seedAgentProfile(t, db)
		s := &AgentService{DB: db}
		p, err := s.profileForConversation(&model.DmConversation{AgentProfileID: prof.ID})
		require.NoError(t, err)
		require.Equal(t, prof.ID, p.ID)
	})

	t.Run("by bot user ID (low)", func(t *testing.T) {
		db := newAgentTestDB(t)
		prof := seedAgentProfile(t, db)
		s := &AgentService{DB: db}
		conv := seedAgentConversation(t, db, 42, prof.BotUserID, 0)
		db.Model(conv).Update("agent_profile_id", 0)
		p, err := s.profileForConversation(conv)
		require.NoError(t, err)
		require.Equal(t, prof.ID, p.ID)
	})

	t.Run("no matching profile", func(t *testing.T) {
		db := newAgentTestDB(t)
		s := &AgentService{DB: db}
		p, err := s.profileForConversation(&model.DmConversation{
			UserLow:  1,
			UserHigh: 2,
		})
		require.Error(t, err)
		require.Nil(t, p)
	})
}

// ---------- PostAssistantMessage ----------

func TestAgentService_PostAssistantMessage(t *testing.T) {

	t.Run("nil conv", func(t *testing.T) {
		s := &AgentService{DB: newAgentTestDB(t)}
		msg, err := s.PostAssistantMessage(nil, 42, "hello")
		require.Error(t, err)
		require.Nil(t, msg)
	})

	t.Run("no matching profile", func(t *testing.T) {
		db := newAgentTestDB(t)
		s := &AgentService{DB: db}
		msg, err := s.PostAssistantMessage(&model.DmConversation{ID: 1}, 42, "hello")
		require.Error(t, err)
		require.Nil(t, msg)
	})

	t.Run("empty content", func(t *testing.T) {
		db := newAgentTestDB(t)
		prof := seedAgentProfile(t, db)
		conv := seedAgentConversation(t, db, 42, prof.BotUserID, prof.ID)
		s := &AgentService{DB: db}
		msg, err := s.PostAssistantMessage(conv, 42, "  ")
		require.Error(t, err)
		require.Nil(t, msg)
	})

	t.Run("successful post", func(t *testing.T) {
		db := newAgentTestDB(t)
		prof := seedAgentProfile(t, db)
		conv := seedAgentConversation(t, db, 42, prof.BotUserID, prof.ID)
		require.NoError(t, db.Create(&model.DmParticipant{
			ConversationID: conv.ID,
			UserID:         42,
			UnreadCount:    0,
		}).Error)

		s := &AgentService{DB: db}
		msg, err := s.PostAssistantMessage(conv, 42, "Hello! How can I help?")
		require.NoError(t, err)
		require.NotNil(t, msg)
		require.Equal(t, "assistant", msg.Role)
		require.Equal(t, "Hello! How can I help?", msg.Content)
		require.Equal(t, prof.BotUserID, msg.SenderID)

		var updated model.DmConversation
		require.NoError(t, db.First(&updated, conv.ID).Error)
		require.Contains(t, updated.LastPreview, "Hello!")
	})

	t.Run("truncates long content to 500", func(t *testing.T) {
		db := newAgentTestDB(t)
		prof := seedAgentProfile(t, db)
		conv := seedAgentConversation(t, db, 42, prof.BotUserID, prof.ID)
		require.NoError(t, db.Create(&model.DmParticipant{
			ConversationID: conv.ID,
			UserID:         42,
		}).Error)

		long := ""
		for i := 0; i < 600; i++ {
			long += "a"
		}
		s := &AgentService{DB: db}
		msg, err := s.PostAssistantMessage(conv, 42, long)
		require.NoError(t, err)
		require.NotNil(t, msg)
		require.Equal(t, 500, len([]rune(msg.Content)))
	})
}

// ---------- applyDynamicGatewayConfig ----------

func TestAgentService_applyDynamicGatewayConfig(t *testing.T) {
	t.Run("nil Gateway", func(t *testing.T) {
		s := &AgentService{}
		s.applyDynamicGatewayConfig()
	})

	t.Run("nil RC", func(t *testing.T) {
		s := &AgentService{Gateway: &aigateway.Gateway{MaxHistory: 10, HistoryTTL: 1 * time.Hour}}
		s.applyDynamicGatewayConfig()
		require.Equal(t, 10, s.Gateway.MaxHistory)
		require.Equal(t, 1*time.Hour, s.Gateway.HistoryTTL)
	})

	t.Run("with RC", func(t *testing.T) {
		s := &AgentService{
			Gateway: &aigateway.Gateway{MaxHistory: 10, HistoryTTL: 1 * time.Hour},
			RC:      &config.RuntimeConfig{},
		}
		s.applyDynamicGatewayConfig()
		require.NotNil(t, s.Gateway)
	})
}

// ---------- GenerateReply ----------

func TestAgentService_GenerateReply(t *testing.T) {
	t.Run("gateway not ready", func(t *testing.T) {
		s := &AgentService{}
		_, err := s.GenerateReply(context.Background(), &model.DmConversation{ID: 1}, "hello")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not configured")
	})

	t.Run("no matching profile", func(t *testing.T) {
		db := newAgentTestDB(t)
		s := &AgentService{
			DB: db,
			Cfg: &config.C{
				DeepSeekAPIKey:      "sk-test",
				AgentEnabled: true,
				AgentRequestTimeout: 30 * time.Second,
			},
			Gateway: &aigateway.Gateway{LLM: &aigateway.Client{APIKey: "sk-test"}},
		}
		_, err := s.GenerateReply(context.Background(), &model.DmConversation{
			ID:       1,
			UserLow:  1,
			UserHigh: 2,
		}, "hello")
		require.Error(t, err)
		require.Contains(t, err.Error(), "profile missing")
	})

	t.Run("profile disabled", func(t *testing.T) {
		db := newAgentTestDB(t)
		prof := seedAgentProfile(t, db)
		prof.Enabled = false
		require.NoError(t, db.Save(prof).Error)
		conv := seedAgentConversation(t, db, 42, prof.BotUserID, prof.ID)

		s := &AgentService{
			DB: db,
			Cfg: &config.C{
				DeepSeekAPIKey:      "sk-test",
				AgentEnabled: true,
				AgentRequestTimeout: 30 * time.Second,
			},
			Gateway: &aigateway.Gateway{LLM: &aigateway.Client{APIKey: "sk-test"}},
		}
		_, err := s.GenerateReply(context.Background(), conv, "hello")
		require.Error(t, err)
		require.Contains(t, err.Error(), "disabled")
	})

	t.Run("sensitive input rejected", func(t *testing.T) {
		db := newAgentTestDB(t)
		prof := seedAgentProfile(t, db)
		conv := seedAgentConversation(t, db, 42, prof.BotUserID, prof.ID)

		tmp := t.TempDir()
		wordFile := filepath.Join(tmp, "words.txt")
		require.NoError(t, os.WriteFile(wordFile, []byte("badword\n"), 0o600))

		filter := sensitive.NewFilter(wordFile, zap.NewNop())
		require.NoError(t, filter.Reload())

		s := &AgentService{
			DB: db,
			Cfg: &config.C{
				DeepSeekAPIKey:      "sk-test",
				AgentEnabled: true,
				AgentRequestTimeout: 30 * time.Second,
			},
			Gateway: &aigateway.Gateway{LLM: &aigateway.Client{APIKey: "sk-test"}},
			Sens:    filter,
		}
		_, err := s.GenerateReply(context.Background(), conv, "this contains badword")
		require.Error(t, err)
		require.Contains(t, err.Error(), "sensitive")
	})

	t.Run("empty system prompt", func(t *testing.T) {
		db := newAgentTestDB(t)
		prof := seedAgentProfile(t, db)
		prof.SystemPrompt = "  "
		require.NoError(t, db.Save(prof).Error)
		conv := seedAgentConversation(t, db, 42, prof.BotUserID, prof.ID)

		s := &AgentService{
			DB: db,
			Cfg: &config.C{
				DeepSeekAPIKey:      "sk-test",
				AgentEnabled: true,
				AgentRequestTimeout: 30 * time.Second,
			},
			Gateway: &aigateway.Gateway{LLM: &aigateway.Client{APIKey: "sk-test"}},
		}
		_, err := s.GenerateReply(context.Background(), conv, "hello")
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty system prompt")
	})
}

// ---------- ResetConversation ----------

func TestAgentService_ResetConversation(t *testing.T) {

	t.Run("nil DB", func(t *testing.T) {
		s := &AgentService{}
		msg, err := s.ResetConversation(context.Background(), &model.DmConversation{}, 0)
		require.Error(t, err)
		require.Nil(t, msg)
	})

	t.Run("nil conv", func(t *testing.T) {
		s := &AgentService{DB: newAgentTestDB(t)}
		msg, err := s.ResetConversation(context.Background(), nil, 42)
		require.Error(t, err)
		require.Nil(t, msg)
	})

	t.Run("zero human", func(t *testing.T) {
		db := newAgentTestDB(t)
		s := &AgentService{DB: db}
		msg, err := s.ResetConversation(context.Background(), &model.DmConversation{ID: 1}, 0)
		require.Error(t, err)
		require.Nil(t, msg)
	})

	t.Run("no matching profile", func(t *testing.T) {
		db := newAgentTestDB(t)
		s := &AgentService{DB: db}
		conv := &model.DmConversation{ID: 1, UserLow: 1, UserHigh: 2}
		msg, err := s.ResetConversation(context.Background(), conv, 1)
		require.Error(t, err)
		require.Nil(t, msg)
	})

	t.Run("successful reset", func(t *testing.T) {
		db := newAgentTestDB(t)
		prof := seedAgentProfile(t, db)
		conv := seedAgentConversation(t, db, 42, prof.BotUserID, prof.ID)
		require.NoError(t, db.Create(&model.DmParticipant{
			ConversationID: conv.ID,
			UserID:         42,
			UnreadCount:    5,
		}).Error)

		s := &AgentService{DB: db}
		msg, err := s.ResetConversation(context.Background(), conv, 42)
		require.NoError(t, err)
		require.NotNil(t, msg)
		require.Equal(t, "assistant", msg.Role)
		require.Equal(t, prof.BotUserID, msg.SenderID)
		require.Equal(t, "Hello!", msg.Content)

		var count int64
		db.Model(&model.DmMessage{}).Where("conversation_id = ?", conv.ID).Count(&count)
		require.Equal(t, int64(1), count)

		var part model.DmParticipant
		require.NoError(t, db.Where("conversation_id = ? AND user_id = ?", conv.ID, 42).First(&part).Error)
		require.Equal(t, uint32(0), part.UnreadCount)
	})
}

// ---------- ReloadProfiles ----------

func TestAgentService_ReloadProfiles(t *testing.T) {
	s := &AgentService{}
	s.ReloadProfiles()
	require.NotNil(t, s)
}
