<p align="center">
  <strong><img src="https://img.shields.io/badge/🇨🇳中文-00a1d6?style=flat-square" alt="中文"></strong>
  <a href="README_EN.md">
    <img src="https://img.shields.io/badge/🇬🇧English-999999?style=flat-square" alt="English">
  </a>
</p>

# Handler 包参考索引

快速查找：哪个文件包含哪些 HTTP / WebSocket 处理器。

> 命名约定：文件按领域前缀分组（`video_*`、`article_*`、`admin_*`、`user_*`、`search_*`）。
> 后缀 `_oss` = OSS 上传辅助函数；`_ws` = WebSocket；`_test` = 测试。

---

## 视频 (`video*.go`)

| 文件 | 主要处理器 |
| :--- | :--- |
| `video.go` | `UploadVideo`、`ListPublishedVideos`、`ListMyVideos`、`GetVideo`、`UpdateMyVideo`、`UpdateVideoCover`、`DeleteMyVideo` |
| `video_draft.go` | `SaveVideoDraft`、`UpdateVideoDraft`、`PublishVideoDraft`、`ReplaceVideoMedia`、`GetMyVideoDraftSource` |
| `video_engagement.go` | `ToggleVideoFavorite`、`GetVideoFavoritePicker`、`SetVideoFavoriteFolders`、`PostVideoCoin`、`ToggleWatchLater`、`ListMyVideoFavorites`、`ListUserVideoFavorites` |
| `video_like.go` | `ToggleVideoLike` |
| `video_zone.go` | 分区常量，对应前端 `videoZones.js` |
| `video_zone_catalog.go` | 分区目录数据 |
| `video_oss.go` | 视频/封面上传 OSS 辅助函数 |

## 专栏 (`article*.go`)

| 文件 | 主要处理器 |
| :--- | :--- |
| `article.go` | `PostArticle`、`PutMyArticle`、`GetArticle`、`PostArticleView`、`ListMyArticles`、`PatchArticlePlayback` |
| `article_comment.go` | `ListArticleComments`、`PostArticleComment`、`DeleteArticleComment`、`PinArticleComment`、`ToggleArticleCommentLike`、`ApproveArticleComment` |
| `article_engagement.go` | `ToggleArticleFavorite`、`PostArticleCoin`、`ListMyArticleFavorites`、`ListUserArticleFavorites` |
| `article_oss.go` | 专栏图片上传 OSS 辅助函数 |

## 评论与弹幕

| 文件 | 主要处理器 |
| :--- | :--- |
| `comment.go` | `ListComments`、`PostComment`、`DeleteComment`、`PinComment`、`ToggleLike`、`ToggleDislike`、`ApproveComment`、`IgnoreCuratedComment` |
| `danmaku.go` | `PostDanmaku`、`ToggleDanmakuLike` |
| `creator_comment.go` | `ListCreatorComments`（创作中心评论管理视图） |
| `creator_danmaku.go` | `ListCreatorDanmakus`、`DeleteDanmaku`（创作中心弹幕管理视图） |
| `dynamic_comment.go` | `ListDynamicComments`、`PostDynamicComment`、`DeleteDynamicComment`、`ToggleDynamicCommentLike` |

## 私信 (`direct_message*.go`)

| 文件 | 主要处理器 |
| :--- | :--- |
| `direct_message.go` | `ListDmConversations`、`CreateDmConversation`、`DeleteDmConversation`、`ListDmMessages`、`SendDmMessage` |
| `direct_message_ws.go` | `ServeChat`（WebSocket 聊天端点） |
| `agent_direct_message.go` | AI 助手对话辅助函数（`ensureAgentConversationFor`、`runAgentReply`、`pushAgentFallback`） |

## 用户 (`user*.go`、`space*.go`、`follow*.go`)

| 文件 | 主要处理器 |
| :--- | :--- |
| `user_me.go` | `GetMe`、`UpdateMeProfile`、`UpdateMeUsername`、`UpdateMePassword`、`UpdateMeAvatar`、`UpdateMeAnnouncement` |
| `user_space.go` | `GetUserPublic`、`ListUserPublishedVideos` |
| `user_follow.go` | `ListUserFollowing`、`ListUserFollowers`、`ToggleFollowUser` |
| `user_block.go` | `BlockUser` |
| `user_dynamic.go` | `PostUserDynamic`、`GetUserDynamic`、`ListMyDynamics`、`DeleteMyDynamic`、`ToggleDynamicLike` |
| `user_daily_reward.go` | `GetMeDailyRewards`、`PostMeDailyRewardWatch` |
| `user_avatar.go` | 头像 URL 辅助函数 |
| `account_deletion.go` | `RequestAccountDeletion`、`RevokeAccountDeletion` |
| `space_privacy.go` | `GetMeSpacePrivacy`、`UpdateMeSpacePrivacy` |
| `follow_group.go` | `ListMyFollowGroups`、`CreateFollowGroup`、`UpdateFollowGroup`、`DeleteFollowGroup`、`AddFollowGroupMember` |
| `follow_helpers.go` | 共享辅助函数：`getFollowCounts`、`isFollowing` |

## 管理后台 (`admin*.go`)

| 文件 | 主要处理器 |
| :--- | :--- |
| `admin_auth.go` | `AdminLogin`、`AdminRefresh`、`AdminMe` |
| `admin_video.go` | `AdminListVideos`、`AdminGetVideo`、`AdminApproveVideo`、`AdminRejectVideo`、`AdminDeleteVideo` |
| `admin_article.go` | `AdminListArticles`、`AdminGetArticle`、`AdminApproveArticle`、`AdminRejectArticle`、`AdminDeleteArticle` |
| `admin_dynamic.go` | `AdminListDynamics`、`AdminGetDynamic`、`AdminDeleteDynamic` |
| `admin_banner.go` | `AdminListBanners`、`AdminCreateBanner`、`AdminUpdateBanner`、`AdminDeleteBanner` |
| `admin_banner_upload.go` | `AdminUploadBannerImage`、`AdminUploadBannerImageByID` |
| `admin_agent.go` | `AdminListAgentProfiles`、`AdminCreateAgentProfile`、`AdminUpdateAgentProfile`、`AdminDeleteAgentProfile` |
| `admin_hot_search.go` | `AdminListHotSearchOps`、`AdminCreateHotSearchOp`、`AdminUpdateHotSearchOp`、`AdminDeleteHotSearchOp` |
| `admin_hot_search_dashboard.go` | `AdminHotSearchDashboard`、`AdminRemoveHotSearchRedis`、`AdminBoostHotSearchRedis`、`AdminReorderHotSearch` |
| `admin_system_config.go` | `AdminListSystemConfigs`、`AdminUpdateSystemConfig` |

## 搜索

| 文件 | 主要处理器 |
| :--- | :--- |
| `search.go` | `SearchAll`、`InitHotRecorder` |
| `search_history.go` | `GetMySearchHistory`、`PutMySearchHistory`、`PostMySearchHistory` |
| `search_suggest.go` | `SearchSuggest` |
| `hot_search.go` | `HotSearchList` |

## 其他

| 文件 | 主要处理器 |
| :--- | :--- |
| `auth.go` | `Register`、`Login`、`Refresh` |
| `favorite_folder.go` | `ListMyFavoriteFolders`、`CreateFavoriteFolder`、`UpdateFavoriteFolder`、`DeleteFavoriteFolder` 及收藏夹增删查 |
| `view_history.go` | `PostVideoViewHistory`、`RecordArticleViewHistory`、`ListMyViewHistory` |
| `coin_ledger.go` | `ListMeCoinLedger` |
| `health.go` | `Health` |
| `stats.go` | `HomeStats` |
| `ws.go` | `ServeDanmaku`（WebSocket 弹幕端点） |
| `home_banner.go` | `ListHomeBanners` |
| `sensitive_ugc.go` | 敏感内容检测辅助函数 |
| `es_sync.go` | Elasticsearch 索引同步辅助函数 |
| `notification_helpers.go` | 通知格式化与聚合辅助函数 |
| `swagger.go` | `RegisterSwaggerRoutes` |
| `router.go` | `RegisterRoutes`（主路由表） |
| `deps.go` | `Dependencies` 结构体、`API` 类型 |

## OSS 辅助函数（后缀 `_oss.go`）

| 文件 | 用途 |
| :--- | :--- |
| `agent_oss.go` | Agent 头像上传至 OSS |
| `article_oss.go` | 专栏图片上传至 OSS |
| `banner_oss.go` | Banner 图片上传至 OSS |
| `dynamic_oss.go` | 动态图片上传至 OSS |
| `favorite_folder_oss.go` | 收藏夹封面上传至 OSS |
| `video_oss.go` | 视频/封面上传至 OSS |
