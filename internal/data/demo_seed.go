package data

import (
	"encoding/json"
	"fmt"

	"cakecake/internal/config"
	"cakecake/internal/model/danmaku"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// demoVideoSeed describes one public demo video inserted when SEED_DEMO_DATA=true.
// Media files are public URLs on the project demo bucket (read-only, no credentials).
type demoVideoSeed struct {
	Title       string
	Description string
	DurationSec float64
	VideoURL    string
	CoverURL    string
	Zone        string
	Tags        []string
	PlayCount   uint64
	Uploader    string
	AvatarURL   string
}

// demoDanmakuSeed is one danmaku inserted for the first demo video so the
// realtime danmaku effect is visible immediately.
type demoDanmakuSeed struct {
	VideoTime float64
	Content   string
	Color     string
}

// demoSeedFailAfterVideos is a test-only hook: when >= 0, seedDemoVideos fails
// after creating that many videos so tests can exercise all-or-nothing rollback.
var demoSeedFailAfterVideos = -1

var demoVideoSeeds = []demoVideoSeed{
	{
		Title:       "【炮姐/AMV】我永远都会守护在你的身边！",
		Description: "自制 本人的第二个AMV作品，从妹妹篇结束后便开始构思了，具体九月开始挖的坑，于2013年10月26日完稿。\n顺便联动一下我的魔禁/超炮AMV：av4545451\n记得让弹幕多样化一些噢~喜欢的话点个关注，大感谢~\n\n本项目所展示的视频均为B站搬运，仅用于学习交流！ 原作者：暗猫の祝福\n原视频链接：https://www.bilibili.com/video/av810872/",
		DurationSec: 323.707,
		VideoURL:    "https://earthcake.oss-cn-beijing.aliyuncs.com/videos/4.mp4",
		CoverURL:    "https://earthcake.oss-cn-beijing.aliyuncs.com/covers/6.jpg",
		Zone:        "动画-MAD·AMV",
		Tags:        []string{"弹幕之神第一季冠军", "见证弹幕的力量", "御坂美琴", "某科学的超电磁炮", "炮姐", "AMV"},
		PlayCount:   392,
		Uploader:    "暗猫の祝福",
		AvatarURL:   "https://earthcake.oss-cn-beijing.aliyuncs.com/avatars/6.png",
	},
	{
		Title:       "【你的名字ED/4K/60FPS】-なんでもないや/没什么大不了",
		Description: "素材：你的名字\nBGM：なんでもないや\n软件：\npremiere\nSVFI\nTopaz Video Enhance AI\n\n本项目所展示的视频均为B站搬运，仅用于学习交流！\n原作者：加载超时请稍后\n原视频链接：https://www.bilibili.com/video/av40113026/",
		DurationSec: 340.949,
		VideoURL:    "https://earthcake.oss-cn-beijing.aliyuncs.com/videos/1.mp4",
		CoverURL:    "https://earthcake.oss-cn-beijing.aliyuncs.com/covers/1.jpg",
		Zone:        "音乐-OP/ED/OST",
		Tags:        []string{"新海诚", "你的名字", "宫水三叶", "ED", "治愈"},
		PlayCount:   63,
		Uploader:    "加载超时请稍后",
		AvatarURL:   "https://earthcake.oss-cn-beijing.aliyuncs.com/avatars/3.png",
	},
	{
		Title:       "红 魔 一 族",
		Description: "累了~累了\n素材：为美好的世界献上祝福\n部分手书来自UP@皓际工造 已授权\n\n 本项目所展示的视频均为B站搬运，仅用于学习交流！\n原作者：Baka恶魔\n原视频链接：https://www.bilibili.com/video/av583773395/",
		DurationSec: 122.709,
		VideoURL:    "https://earthcake.oss-cn-beijing.aliyuncs.com/videos/6.mp4",
		CoverURL:    "https://earthcake.oss-cn-beijing.aliyuncs.com/covers/8.jpg",
		Zone:        "动画-MAD·AMV",
		Tags:        []string{"为美好的世界献上祝福", "惠惠", "MAD.AMV", "洗脑", "搞笑"},
		PlayCount:   42,
		Uploader:    "Baka恶魔",
		AvatarURL:   "https://earthcake.oss-cn-beijing.aliyuncs.com/avatars/7.png",
	},
	{
		Title:       "曾火遍全网的《溯》，你是否还知道？",
		Description: "视频制作  三栗lil 精灵 千珏 隐紫 傻不理 英梨梨 老平凡 星海 御魔君 程情 酱一\n希望大家喜欢的话三连关注支持一下\n\n本项目所展示的视频均为B站搬运，仅用于学习交流！ 原作者：三栗lil  \n原视频链接：https://www.bilibili.com/video/av85054372/",
		DurationSec: 189.648833,
		VideoURL:    "https://earthcake.oss-cn-beijing.aliyuncs.com/videos/2.mp4",
		CoverURL:    "https://earthcake.oss-cn-beijing.aliyuncs.com/covers/4.jpg",
		Zone:        "动画-MAD·AMV",
		Tags:        []string{"治愈向", "MAD", "论BGM的重要性", "BGM", "动画"},
		PlayCount:   35,
		Uploader:    "三栗lili",
		AvatarURL:   "https://earthcake.oss-cn-beijing.aliyuncs.com/avatars/4.png",
	},
	{
		Title:       "洱海边的 Love Potion",
		Description: "时间实在不够了啊啊啊啊啊啊啊只录了两边啊啊啊啊啊啊啊啊根本没复习啊啊啊啊啊啊啊动作全忘了啊啊啊啊啊啊啊啊\n\n 本项目所展示的视频均为B站搬运，仅用于学习交流！ 原作者：-Yeuoly-\n 原视频链接：https://www.bilibili.com/video/av116126302341011/",
		DurationSec: 80.294376,
		VideoURL:    "https://earthcake.oss-cn-beijing.aliyuncs.com/videos/3.mp4",
		CoverURL:    "https://earthcake.oss-cn-beijing.aliyuncs.com/covers/5.jpg",
		Zone:        "舞蹈-宅舞",
		Tags:        []string{"宅舞", "云南", "洱海", "舞蹈翻跳", "UP主的旅行日记"},
		PlayCount:   30,
		Uploader:    "Yeuoly",
		AvatarURL:   "https://earthcake.oss-cn-beijing.aliyuncs.com/avatars/5.jpg",
	},
	{
		Title:       "170秒带你回味从前的B站动画区",
		Description: "比起高燃打斗闪现流之类的，感觉当时那些啥治愈催泪抒情致郁更值得回味~一定要整一首日语歌，男女混唱更好，最好还要带一段rap，紧接着评论弹幕各种情感抒发各种表白\n不过其实以前的MMD才是深得人心，哈哈哈哈 老绅士了~\nPS：连片尾都没有的，怎么可能是传统艺能~还没到时间不急\n\n 本项目所展示的视频均为B站搬运，仅用于学习交流！ \n原作者：科学超电磁炮F \n原视频链接：https://www.bilibili.com/video/av421579975/",
		DurationSec: 170.623667,
		VideoURL:    "https://earthcake.oss-cn-beijing.aliyuncs.com/videos/8.mp4",
		CoverURL:    "https://earthcake.oss-cn-beijing.aliyuncs.com/covers/10.jpg",
		Zone:        "动画-MAD·AMV",
		Tags:        []string{"动画", "论BGM的重要性", "多素材", "治愈向", "催泪向"},
		PlayCount:   25,
		Uploader:    "科学超电磁炮F",
		AvatarURL:   "https://earthcake.oss-cn-beijing.aliyuncs.com/avatars/9.png",
	},
	{
		Title:       "【泛式/剧情MAD】原来你是我最想留住的幸运 「春物完结篇」",
		Description: "前年想用《小幸运》做一个MAD，去年想用「团子素材」做一个MAD，今年我完成了这个梦想。\n\n 本项目所展示的视频均为B站搬运，仅用于学习交流！ \n原作者：泛式\n原视频链接：https://www.bilibili.com/video/av372617316/",
		DurationSec: 297.898333,
		VideoURL:    "https://earthcake.oss-cn-beijing.aliyuncs.com/videos/10.mp4",
		CoverURL:    "https://earthcake.oss-cn-beijing.aliyuncs.com/covers/12.jpg",
		Zone:        "动画-MAD·AMV",
		Tags:        []string{"由比滨结衣", "比企谷八幡", "泛式", "春物", "我的青春恋爱物语果然有问题", "MAD"},
		PlayCount:   12,
		Uploader:    "泛式大大",
		AvatarURL:   "https://earthcake.oss-cn-beijing.aliyuncs.com/avatars/11.png",
	},
}

var demoDanmakuSeeds = []demoDanmakuSeed{
	{VideoTime: 3.5, Content: "前排围观", Color: "#FFFFFF"},
	{VideoTime: 8.2, Content: "画质太顶了", Color: "#FFFFFF"},
	{VideoTime: 15.0, Content: "为御坂美琴打call", Color: "#FFD700"},
	{VideoTime: 22.7, Content: "233333", Color: "#FFFFFF"},
	{VideoTime: 30.1, Content: "见证弹幕的力量", Color: "#FFFFFF"},
	{VideoTime: 45.5, Content: "泪目", Color: "#FFFFFF"},
	{VideoTime: 60.0, Content: "梦开始的地方", Color: "#FFFFFF"},
	{VideoTime: 88.8, Content: "前方高能", Color: "#FF4E6A"},
}

// SeedDemoData inserts demo users/videos/danmaku on a fresh database when
// SEED_DEMO_DATA=true. It is a no-op when the flag is off or the videos table
// already has content, so restarts never duplicate data.
func SeedDemoData(db *gorm.DB, cfg *config.C, lg *zap.Logger) error {
	if db == nil || cfg == nil || !cfg.SeedDemoData {
		return nil
	}
	var n int64
	if err := db.Model(&video.Video{}).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		if lg != nil {
			lg.Info("seed demo data skipped: videos table already has content")
		}
		return nil
	}

	// All-or-nothing: users, videos and danmaku are inserted in one transaction so a
	// mid-seed failure rolls everything back instead of leaving a partial demo that
	// would be skipped forever by the "videos table non-empty" guard above.
	return db.Transaction(func(tx *gorm.DB) error {
		userIDs, err := seedDemoUsers(tx, cfg.DemoUserPassword)
		if err != nil {
			return fmt.Errorf("seed demo users: %w", err)
		}
		videoIDs, err := seedDemoVideos(tx, userIDs, lg)
		if err != nil {
			return fmt.Errorf("seed demo videos: %w", err)
		}
		if len(videoIDs) > 0 && len(demoDanmakuSeeds) > 0 {
			firstVideoID := videoIDs[0]
			firstVideoUserID := userIDs[demoVideoSeeds[0].Uploader]
			if err := seedDemoDanmaku(tx, firstVideoID, firstVideoUserID, lg); err != nil {
				return fmt.Errorf("seed demo danmaku: %w", err)
			}
		}
		return nil
	})
}

func seedDemoUsers(db *gorm.DB, password string) (map[string]uint64, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	ids := make(map[string]uint64, len(demoVideoSeeds))
	for _, s := range demoVideoSeeds {
		var existing user.User
		err := db.Where("username = ?", s.Uploader).First(&existing).Error
		if err == nil {
			ids[s.Uploader] = existing.ID
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
		u := user.User{
			Username:     s.Uploader,
			PasswordHash: string(hash),
			AvatarURL:    s.AvatarURL,
			Nickname:     s.Uploader,
			Sign:         "cakecake 演示数据用户",
			Gender:       "secret",
		}
		if err := db.Create(&u).Error; err != nil {
			return nil, err
		}
		cid := user.FormatCakeID(u.ID)
		if err := db.Model(&u).Update("cake_id", cid).Error; err != nil {
			return nil, err
		}
		ids[s.Uploader] = u.ID
	}
	return ids, nil
}

func seedDemoVideos(db *gorm.DB, userIDs map[string]uint64, lg *zap.Logger) ([]uint64, error) {
	ids := make([]uint64, 0, len(demoVideoSeeds))
	for i, s := range demoVideoSeeds {
		uid, ok := userIDs[s.Uploader]
		if !ok {
			return nil, fmt.Errorf("missing demo user for uploader %q", s.Uploader)
		}
		tags, err := json.Marshal(s.Tags)
		if err != nil {
			return nil, err
		}
		v := video.Video{
			UserID:      uid,
			Title:       s.Title,
			Description: s.Description,
			DurationSec: s.DurationSec,
			Status:      video.StatusPublished,
			VideoURL:    s.VideoURL,
			CoverURL:    s.CoverURL,
			PlayCount:   s.PlayCount,
			TagsJSON:    string(tags),
			Zone:        s.Zone,
		}
		// Only the first video has seeded danmaku; keep the counter in sync with
		// the actual rows so the UI never shows numbers without backing data.
		if i == 0 {
			v.DanmakuCount = uint64(len(demoDanmakuSeeds))
		}
		if err := db.Create(&v).Error; err != nil {
			return nil, err
		}
		if demoSeedFailAfterVideos >= 0 && i+1 >= demoSeedFailAfterVideos {
			return nil, fmt.Errorf("injected demo seed failure after %d videos", i+1)
		}
		ids = append(ids, v.ID)
		if lg != nil {
			lg.Info("seed demo video", zap.Uint64("video_id", v.ID), zap.String("title", v.Title))
		}
	}
	return ids, nil
}

func seedDemoDanmaku(db *gorm.DB, videoID, userID uint64, lg *zap.Logger) error {
	for _, s := range demoDanmakuSeeds {
		d := danmaku.Danmaku{
			VideoID:   videoID,
			UserID:    userID,
			Content:   s.Content,
			Color:     s.Color,
			Type:      "scroll",
			FontSize:  "md",
			VideoTime: s.VideoTime,
		}
		if err := db.Create(&d).Error; err != nil {
			return err
		}
	}
	if lg != nil {
		lg.Info("seed demo danmaku", zap.Uint64("video_id", videoID), zap.Int("count", len(demoDanmakuSeeds)))
	}
	return nil
}
