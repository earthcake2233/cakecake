package agent

import (
	"cakecake/internal/model/agent"
	"cakecake/internal/model/dm"
	"cakecake/internal/service/servicetest"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/aigateway"
	"cakecake/internal/config"
	"cakecake/internal/pkg/sensitive"
)

// ---------- helpers ----------

func seedAgentProfile(t *testing.T, db *gorm.DB) *agent.AgentProfile {
	t.Helper()
	p := &agent.AgentProfile{
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

func seedAgentConversation(t *testing.T, db *gorm.DB, humanID, botID uint64, profileID uint64) *dm.DmConversation {
	t.Helper()
	low, high := humanID, botID
	if low > high {
		low, high = high, low
	}
	conv := &dm.DmConversation{
		UserLow:        low,
		UserHigh:       high,
		Kind:           dm.DmKindAgent,
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
		mr, rdb := servicetest.NewRedis(t)
		defer mr.Close()
		s := &AgentService{
			Cfg:   &config.C{AgentDailyQuota: 0},
			Redis: rdb,
		}
		require.True(t, s.CheckQuota(context.Background(), 1))
	})

	t.Run("no usage yet returns true", func(t *testing.T) {
		mr, rdb := servicetest.NewRedis(t)
		defer mr.Close()
		s := &AgentService{
			Cfg:   &config.C{AgentDailyQuota: 10},
			Redis: rdb,
		}
		require.True(t, s.CheckQuota(context.Background(), 1))
	})

	t.Run("under quota returns true", func(t *testing.T) {
		mr, rdb := servicetest.NewRedis(t)
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
		mr, rdb := servicetest.NewRedis(t)
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
		mr, rdb := servicetest.NewRedis(t)
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
		mr, rdb := servicetest.NewRedis(t)
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
		db := servicetest.NewDB(t)
		s := &AgentService{Store: NewAgentStore(db)}
		require.NoError(t, s.EnsureForUser(0))
	})

	t.Run("creates conversations", func(t *testing.T) {
		db := servicetest.NewDB(t)
		seedAgentProfile(t, db)
		s := &AgentService{Store: NewAgentStore(db)}
		require.NoError(t, s.EnsureForUser(42))
	})
}

// ---------- IsAgentConversation ----------
func TestAgentService_IsAgentConversation_More(t *testing.T) {
	s := &AgentService{}
	require.False(t, s.IsAgentConversation(nil))
	require.False(t, s.IsAgentConversation(&dm.DmConversation{}))
	require.False(t, s.IsAgentConversation(&dm.DmConversation{Kind: "human"}))
	require.True(t, s.IsAgentConversation(&dm.DmConversation{Kind: dm.DmKindAgent}))
}

// ---------- IsBotUser ----------

func TestAgentService_IsBotUser(t *testing.T) {
	t.Run("nil DB", func(t *testing.T) {
		s := &AgentService{}
		require.False(t, s.IsBotUser(1))
	})

	t.Run("zero user", func(t *testing.T) {
		db := servicetest.NewDB(t)
		s := &AgentService{Store: NewAgentStore(db)}
		require.False(t, s.IsBotUser(0))
	})

	t.Run("not a bot user", func(t *testing.T) {
		db := servicetest.NewDB(t)
		s := &AgentService{Store: NewAgentStore(db)}
		require.False(t, s.IsBotUser(9999))
	})

	t.Run("is a bot user", func(t *testing.T) {
		db := servicetest.NewDB(t)
		p := seedAgentProfile(t, db)
		s := &AgentService{Store: NewAgentStore(db)}
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
		db := servicetest.NewDB(t)
		prof := seedAgentProfile(t, db)
		s := &AgentService{Store: NewAgentStore(db)}
		p, err := s.profileForConversation(&dm.DmConversation{AgentProfileID: prof.ID})
		require.NoError(t, err)
		require.Equal(t, prof.ID, p.ID)
	})

	t.Run("by bot user ID (low)", func(t *testing.T) {
		db := servicetest.NewDB(t)
		prof := seedAgentProfile(t, db)
		s := &AgentService{Store: NewAgentStore(db)}
		conv := seedAgentConversation(t, db, 42, prof.BotUserID, 0)
		db.Model(conv).Update("agent_profile_id", 0)
		p, err := s.profileForConversation(conv)
		require.NoError(t, err)
		require.Equal(t, prof.ID, p.ID)
	})

	t.Run("no matching profile", func(t *testing.T) {
		db := servicetest.NewDB(t)
		s := &AgentService{Store: NewAgentStore(db)}
		p, err := s.profileForConversation(&dm.DmConversation{
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
		s := &AgentService{Store: NewAgentStore(servicetest.NewDB(t))}
		msg, err := s.PostAssistantMessage(nil, 42, "hello")
		require.Error(t, err)
		require.Nil(t, msg)
	})

	t.Run("no matching profile", func(t *testing.T) {
		db := servicetest.NewDB(t)
		s := &AgentService{Store: NewAgentStore(db)}
		msg, err := s.PostAssistantMessage(&dm.DmConversation{ID: 1}, 42, "hello")
		require.Error(t, err)
		require.Nil(t, msg)
	})

	t.Run("empty content", func(t *testing.T) {
		db := servicetest.NewDB(t)
		prof := seedAgentProfile(t, db)
		conv := seedAgentConversation(t, db, 42, prof.BotUserID, prof.ID)
		s := &AgentService{Store: NewAgentStore(db)}
		msg, err := s.PostAssistantMessage(conv, 42, "  ")
		require.Error(t, err)
		require.Nil(t, msg)
	})

	t.Run("successful post", func(t *testing.T) {
		db := servicetest.NewDB(t)
		prof := seedAgentProfile(t, db)
		conv := seedAgentConversation(t, db, 42, prof.BotUserID, prof.ID)
		require.NoError(t, db.Create(&dm.DmParticipant{
			ConversationID: conv.ID,
			UserID:         42,
			UnreadCount:    0,
		}).Error)

		s := &AgentService{Store: NewAgentStore(db)}
		msg, err := s.PostAssistantMessage(conv, 42, "Hello! How can I help?")
		require.NoError(t, err)
		require.NotNil(t, msg)
		require.Equal(t, "assistant", msg.Role)
		require.Equal(t, "Hello! How can I help?", msg.Content)
		require.Equal(t, prof.BotUserID, msg.SenderID)

		var updated dm.DmConversation
		require.NoError(t, db.First(&updated, conv.ID).Error)
		require.Contains(t, updated.LastPreview, "Hello!")
	})

	t.Run("truncates long content to 8000", func(t *testing.T) {
		db := servicetest.NewDB(t)
		prof := seedAgentProfile(t, db)
		conv := seedAgentConversation(t, db, 42, prof.BotUserID, prof.ID)
		require.NoError(t, db.Create(&dm.DmParticipant{
			ConversationID: conv.ID,
			UserID:         42,
		}).Error)

		long := ""
		for i := 0; i < 8500; i++ {
			long += "a"
		}
		s := &AgentService{Store: NewAgentStore(db)}
		msg, err := s.PostAssistantMessage(conv, 42, long)
		require.NoError(t, err)
		require.NotNil(t, msg)
		require.Equal(t, 8000, len([]rune(msg.Content)))
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
		_, err := s.GenerateReply(context.Background(), &dm.DmConversation{ID: 1}, "hello")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not configured")
	})

	t.Run("no matching profile", func(t *testing.T) {
		db := servicetest.NewDB(t)
		s := &AgentService{
			Store: NewAgentStore(db),
			Cfg: &config.C{
				DeepSeekAPIKey:      "sk-test",
				AgentEnabled:        true,
				AgentRequestTimeout: 30 * time.Second,
			},
			Gateway: &aigateway.Gateway{LLM: &aigateway.Client{APIKey: "sk-test"}},
		}
		_, err := s.GenerateReply(context.Background(), &dm.DmConversation{
			ID:       1,
			UserLow:  1,
			UserHigh: 2,
		}, "hello")
		require.Error(t, err)
		require.Contains(t, err.Error(), "profile missing")
	})

	t.Run("profile disabled", func(t *testing.T) {
		db := servicetest.NewDB(t)
		prof := seedAgentProfile(t, db)
		prof.Enabled = false
		require.NoError(t, db.Save(prof).Error)
		conv := seedAgentConversation(t, db, 42, prof.BotUserID, prof.ID)

		s := &AgentService{
			Store: NewAgentStore(db),
			Cfg: &config.C{
				DeepSeekAPIKey:      "sk-test",
				AgentEnabled:        true,
				AgentRequestTimeout: 30 * time.Second,
			},
			Gateway: &aigateway.Gateway{LLM: &aigateway.Client{APIKey: "sk-test"}},
		}
		_, err := s.GenerateReply(context.Background(), conv, "hello")
		require.Error(t, err)
		require.Contains(t, err.Error(), "disabled")
	})

	t.Run("sensitive input rejected", func(t *testing.T) {
		db := servicetest.NewDB(t)
		prof := seedAgentProfile(t, db)
		conv := seedAgentConversation(t, db, 42, prof.BotUserID, prof.ID)

		tmp := t.TempDir()
		wordFile := filepath.Join(tmp, "words.txt")
		require.NoError(t, os.WriteFile(wordFile, []byte("badword\n"), 0o600))

		filter := sensitive.NewFilter(wordFile, zap.NewNop())
		require.NoError(t, filter.Reload())

		s := &AgentService{
			Store: NewAgentStore(db),
			Cfg: &config.C{
				DeepSeekAPIKey:      "sk-test",
				AgentEnabled:        true,
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
		db := servicetest.NewDB(t)
		prof := seedAgentProfile(t, db)
		prof.SystemPrompt = "  "
		require.NoError(t, db.Save(prof).Error)
		conv := seedAgentConversation(t, db, 42, prof.BotUserID, prof.ID)

		s := &AgentService{
			Store: NewAgentStore(db),
			Cfg: &config.C{
				DeepSeekAPIKey:      "sk-test",
				AgentEnabled:        true,
				AgentRequestTimeout: 30 * time.Second,
			},
			Gateway: &aigateway.Gateway{LLM: &aigateway.Client{APIKey: "sk-test"}},
		}
		_, err := s.GenerateReply(context.Background(), conv, "hello")
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty system prompt")
	})
}

func TestFilterReferencedItems(t *testing.T) {
	items := json.RawMessage(`[
		{"id":1,"title":"打上花火治愈心灵","author":"earthcake"},
		{"id":2,"title":"Go 与 MySQL 实战教程","author":"earthcake"}
	]`)

	// Reply cites only the relevant video by title -> only that card survives.
	kept := filterReferencedItems("推荐你看看《Go 与 MySQL 实战教程》", items)
	require.NotNil(t, kept)
	var arr []map[string]interface{}
	require.NoError(t, json.Unmarshal(kept, &arr))
	require.Len(t, arr, 1)
	require.Equal(t, float64(2), arr[0]["id"])

	// Reply mentions an id explicitly -> that item survives.
	kept2 := filterReferencedItems("视频 1 不错", items)
	require.NotNil(t, kept2)
	var arr2 []map[string]interface{}
	require.NoError(t, json.Unmarshal(kept2, &arr2))
	require.Len(t, arr2, 1)
	require.Equal(t, float64(1), arr2[0]["id"])

	// Reply says nothing relevant -> nothing survives.
	require.Nil(t, filterReferencedItems("站内暂时没有相关教程", items))
}

func TestReplyDismissesResults(t *testing.T) {
	require.True(t, replyDismissesResults("我在站里搜了一圈，结果只翻出几个动画区视频，跟编程八竿子打不着，站内暂时没有相关教程。"))
	require.True(t, replyDismissesResults("搜到了《溯》，但和编程无关"))
	require.True(t, replyDismissesResults("没找到相关教程"))
	require.True(t, replyDismissesResults("咱们站确实没有 Go 连 MySQL 的教程视频"))
	require.True(t, replyDismissesResults("结果只有动画和音乐的投稿"))
	require.False(t, replyDismissesResults("推荐你看看《Go 与 MySQL 实战教程》，站内有这个视频。"))
}

// buildReplyResult must drop all result cards when the reply dismisses them,
// even if a result title is mentioned while being dismissed.
func TestBuildReplyResult_DropsDismissedResults(t *testing.T) {
	items := json.RawMessage(`[{"id":7,"title":"曾火遍全网的《溯》，你是否还知道？"}]`)
	coll := &toolActivityCollector{
		acts: []map[string]interface{}{
			{"span_id": "s1", "tool_name": "search_videos", "status": "done"},
		},
		results: map[string]json.RawMessage{"s1": items},
	}
	result, err := buildReplyResult("搜到了《溯》，但和编程无关，站内暂时没有相关教程。", coll)
	require.NoError(t, err)
	require.Empty(t, result.ToolResultData)
	require.NotEmpty(t, result.ToolActivities)
}

func TestParseSuggestionsJSON(t *testing.T) {
	require.Equal(t, []string{"a", "b", "c"}, parseSuggestionsJSON(`["a","b","c"]`))
	require.Equal(t, []string{"a", "b"}, parseSuggestionsJSON("下面是追问：\n```json\n[\"a\",\"b\"]\n```"))
	require.Equal(t, []string{"x", "y", "z"}, parseSuggestionsJSON(`["x","y","z","w"]`))
	require.Nil(t, parseSuggestionsJSON("没有追问"))
	require.Nil(t, parseSuggestionsJSON(""))
}

func TestPlainTextPreview(t *testing.T) {
	got := plainTextPreview("**评论区：**\n- earthcake：我喜欢你\n- 支持！\n```go\nfmt.Println(\"hi\")\n```\n[链接](https://x.com)")
	require.NotContains(t, got, "**")
	require.NotContains(t, got, "```")
	require.NotContains(t, got, "- earthcake")
	require.NotContains(t, got, "[链接]")
	require.Contains(t, got, "评论区")
	require.Contains(t, got, "earthcake")
	require.Contains(t, got, "支持")
}

func TestPartialEndsInsideCodeFence(t *testing.T) {
	require.False(t, partialEndsInsideCodeFence("第一段没有代码"))
	require.True(t, partialEndsInsideCodeFence("先看代码：\n```go\npackage main\nfunc main() {"))
	require.False(t, partialEndsInsideCodeFence("```go\nfmt.Println(1)\n```\n结束"))
}

func TestNormalizeMarkdownFences(t *testing.T) {
	got := normalizeMarkdownFences("开头\n````go\ncode\n```\n结尾")
	require.Equal(t, "开头\n```go\ncode\n```\n结尾", got)
	got2 := normalizeMarkdownFences("```go\nunclosed")
	require.True(t, strings.HasSuffix(got2, "\n```"))
	require.Equal(t, 2, strings.Count(got2, "```"))
	got3 := normalizeMarkdownFences("// 验证```go\n// 验证连接是否成功\n}")
	require.NotContains(t, got3, "```")
	require.Equal(t, "// 验证\n// 验证连接是否成功\n}", got3)
	got4 := normalizeMarkdownFences("```go\ncode line\n```go\nmore code\n```")
	require.Equal(t, "```go\ncode line\n\nmore code\n```", got4)
}

func TestDropSeamDuplicateLines(t *testing.T) {
	got := dropSeamDuplicateLines(
		"    // 格式：用户名:密码@tcp(主机:端口)/数据库名?参数",
		"// 格式：用户名:密码@tcp(主机:端口)/数据库名?参数\n    db, err := sql.Open(\"mysql\", dsn)",
	)
	require.Equal(t, "    db, err := sql.Open(\"mysql\", dsn)", got)

	got2 := dropSeamDuplicateLines(
		"\tdsn := \"root:123456@tcp(127",
		"\tdsn := \"root:123456@tcp(127.0.0.1:3306)/testdb?charset=utf8&parseTime=true&loc=Local\"\n\n\tdb, err := sql.Open(\"mysql\", dsn)",
	)
	require.Equal(t, "\tdsn := \"root:123456@tcp(127.0.0.1:3306)/testdb?charset=utf8&parseTime=true&loc=Local\"\n\n\tdb, err := sql.Open(\"mysql\", dsn)", got2)

	got3 := dropSeamDuplicateLines(
		"\tdefer db.Close",
		"\tdefer db.Close()\n\n\tif err := db.Ping(); err != nil {",
	)
	require.Equal(t, "\tdefer db.Close()\n\n\tif err := db.Ping(); err != nil {", got3)
}

func TestMergeContinuation(t *testing.T) {
	require.Equal(t, "a\nb", mergeContinuation("a", "b"))
	require.Equal(
		t,
		"// 验证连接是否通\nerr := db.Ping()",
		mergeContinuation("// 验证连接是否通", "// 验证连接是否通\nerr := db.Ping()"),
	)
	require.Equal(
		t,
		"// 一定要 Ping 一下，确认数据库真的能连上\n\tif err := db.Ping(); err != nil {",
		mergeContinuation(
			"// 一定要 Ping 一下，确认",
			"\t// 一定要 Ping 一下，确认数据库真的能连上\n\tif err := db.Ping(); err != nil {",
		),
	)
	require.Equal(t, "```go\ncode", mergeContinuation("```go", "```go\ncode"))
	require.Equal(t, "part", mergeContinuation("part", ""))
	require.Equal(t, "tail", mergeContinuation("", "tail"))
}

func TestDedupeConsecutiveLines(t *testing.T) {
	require.Equal(t, "a\nb", dedupeConsecutiveLines("a\na\nb"))
	require.Equal(t, "x\ny", dedupeConsecutiveLines("x\ny\ny"))
	require.Equal(t, "x\ny", dedupeConsecutiveLines("x\ny"))
}

// ---------- ResetConversation ----------

func TestAgentService_ResetConversation(t *testing.T) {

	t.Run("nil DB", func(t *testing.T) {
		s := &AgentService{}
		msg, err := s.ResetConversation(context.Background(), &dm.DmConversation{}, 0)
		require.Error(t, err)
		require.Nil(t, msg)
	})

	t.Run("nil conv", func(t *testing.T) {
		s := &AgentService{Store: NewAgentStore(servicetest.NewDB(t))}
		msg, err := s.ResetConversation(context.Background(), nil, 42)
		require.Error(t, err)
		require.Nil(t, msg)
	})

	t.Run("zero human", func(t *testing.T) {
		db := servicetest.NewDB(t)
		s := &AgentService{Store: NewAgentStore(db)}
		msg, err := s.ResetConversation(context.Background(), &dm.DmConversation{ID: 1}, 0)
		require.Error(t, err)
		require.Nil(t, msg)
	})

	t.Run("no matching profile", func(t *testing.T) {
		db := servicetest.NewDB(t)
		s := &AgentService{Store: NewAgentStore(db)}
		conv := &dm.DmConversation{ID: 1, UserLow: 1, UserHigh: 2}
		msg, err := s.ResetConversation(context.Background(), conv, 1)
		require.Error(t, err)
		require.Nil(t, msg)
	})

	t.Run("successful reset", func(t *testing.T) {
		db := servicetest.NewDB(t)
		prof := seedAgentProfile(t, db)
		conv := seedAgentConversation(t, db, 42, prof.BotUserID, prof.ID)
		require.NoError(t, db.Create(&dm.DmParticipant{
			ConversationID: conv.ID,
			UserID:         42,
			UnreadCount:    5,
		}).Error)

		s := &AgentService{Store: NewAgentStore(db)}
		msg, err := s.ResetConversation(context.Background(), conv, 42)
		require.NoError(t, err)
		require.NotNil(t, msg)
		require.Equal(t, "assistant", msg.Role)
		require.Equal(t, prof.BotUserID, msg.SenderID)
		require.Equal(t, "Hello!", msg.Content)

		var count int64
		db.Model(&dm.DmMessage{}).Where("conversation_id = ?", conv.ID).Count(&count)
		require.Equal(t, int64(1), count)

		var part dm.DmParticipant
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
