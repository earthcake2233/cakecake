<p align="center">
  <a href="README.md">
    <img src="https://img.shields.io/badge/Chinese-999999?style=flat-square" alt="Chinese">
  </a>
  <strong><img src="https://img.shields.io/badge/English-00a1d6?style=flat-square" alt="English"></strong>
</p>
# Handler Package Reference

Quick lookup: which file contains which HTTP/WebSocket handlers.

> Naming convention: files are prefixed by domain (`video_*`, `article_*`, `admin_*`, `user_*`, `search_*`).
> Suffix `_oss` = OSS upload helpers; `_ws` = WebSocket; `_test` = tests.

---

## Video (`video*.go`)

| File | Key handlers |
| :--- | :--- |
| `video.go` | `UploadVideo`, `ListPublishedVideos`, `ListMyVideos`, `GetVideo`, `UpdateMyVideo`, `UpdateVideoCover`, `DeleteMyVideo` |
| `video_draft.go` | `SaveVideoDraft`, `UpdateVideoDraft`, `PublishVideoDraft`, `ReplaceVideoMedia`, `GetMyVideoDraftSource` |
| `video_engagement.go` | `ToggleVideoFavorite`, `GetVideoFavoritePicker`, `SetVideoFavoriteFolders`, `PostVideoCoin`, `ToggleWatchLater`, `ListMyVideoFavorites`, `ListUserVideoFavorites` |
| `video_like.go` | `ToggleVideoLike` |
| `video_zone.go` | Zone constants matching frontend `videoZones.js` |
| `video_zone_catalog.go` | Zone catalog data |
| `video_oss.go` | OSS upload helpers for video and cover images |

## Article (`article*.go`)

| File | Key handlers |
| :--- | :--- |
| `article.go` | `PostArticle`, `PutMyArticle`, `GetArticle`, `PostArticleView`, `ListMyArticles`, `PatchArticlePlayback` |
| `article_comment.go` | `ListArticleComments`, `PostArticleComment`, `DeleteArticleComment`, `PinArticleComment`, `ToggleArticleCommentLike`, `ApproveArticleComment` |
| `article_engagement.go` | `ToggleArticleFavorite`, `PostArticleCoin`, `ListMyArticleFavorites`, `ListUserArticleFavorites` |
| `article_oss.go` | OSS upload helpers for article images |

## Comment & Danmaku

| File | Key handlers |
| :--- | :--- |
| `comment.go` | `ListComments`, `PostComment`, `DeleteComment`, `PinComment`, `ToggleLike`, `ToggleDislike`, `ApproveComment`, `IgnoreCuratedComment` |
| `danmaku.go` | `PostDanmaku`, `ToggleDanmakuLike` |
| `creator_comment.go` | `ListCreatorComments` (creator hub comment management view) |
| `creator_danmaku.go` | `ListCreatorDanmakus`, `DeleteDanmaku` (creator hub danmaku management view) |
| `dynamic_comment.go` | `ListDynamicComments`, `PostDynamicComment`, `DeleteDynamicComment`, `ToggleDynamicCommentLike` |

## Direct Message (`direct_message*.go`)

| File | Key handlers |
| :--- | :--- |
| `direct_message.go` | `ListDmConversations`, `CreateDmConversation`, `DeleteDmConversation`, `ListDmMessages`, `SendDmMessage` |
| `direct_message_ws.go` | `ServeChat` (WebSocket chat endpoint) |
| `agent_direct_message.go` | AI agent conversation helpers (`ensureAgentConversationFor`, `runAgentReply`, `pushAgentFallback`) |

## User (`user*.go`, `space*.go`, `follow*.go`)

| File | Key handlers |
| :--- | :--- |
| `user_me.go` | `GetMe`, `UpdateMeProfile`, `UpdateMeUsername`, `UpdateMePassword`, `UpdateMeAvatar`, `UpdateMeAnnouncement` |
| `user_space.go` | `GetUserPublic`, `ListUserPublishedVideos` |
| `user_follow.go` | `ListUserFollowing`, `ListUserFollowers`, `ToggleFollowUser` |
| `user_block.go` | `BlockUser` |
| `user_dynamic.go` | `PostUserDynamic`, `GetUserDynamic`, `ListMyDynamics`, `DeleteMyDynamic`, `ToggleDynamicLike` |
| `user_daily_reward.go` | `GetMeDailyRewards`, `PostMeDailyRewardWatch` |
| `user_avatar.go` | Avatar URL helpers |
| `account_deletion.go` | `RequestAccountDeletion`, `RevokeAccountDeletion` |
| `space_privacy.go` | `GetMeSpacePrivacy`, `UpdateMeSpacePrivacy` |
| `follow_group.go` | `ListMyFollowGroups`, `CreateFollowGroup`, `UpdateFollowGroup`, `DeleteFollowGroup`, `AddFollowGroupMember` |
| `follow_helpers.go` | Shared helpers: `getFollowCounts`, `isFollowing` |

## Admin (`admin*.go`)

| File | Key handlers |
| :--- | :--- |
| `admin_auth.go` | `AdminLogin`, `AdminRefresh`, `AdminMe` |
| `admin_video.go` | `AdminListVideos`, `AdminGetVideo`, `AdminApproveVideo`, `AdminRejectVideo`, `AdminDeleteVideo` |
| `admin_article.go` | `AdminListArticles`, `AdminGetArticle`, `AdminApproveArticle`, `AdminRejectArticle`, `AdminDeleteArticle` |
| `admin_dynamic.go` | `AdminListDynamics`, `AdminGetDynamic`, `AdminDeleteDynamic` |
| `admin_banner.go` | `AdminListBanners`, `AdminCreateBanner`, `AdminUpdateBanner`, `AdminDeleteBanner` |
| `admin_banner_upload.go` | `AdminUploadBannerImage`, `AdminUploadBannerImageByID` |
| `admin_agent.go` | `AdminListAgentProfiles`, `AdminCreateAgentProfile`, `AdminUpdateAgentProfile`, `AdminDeleteAgentProfile` |
| `admin_hot_search.go` | `AdminListHotSearchOps`, `AdminCreateHotSearchOp`, `AdminUpdateHotSearchOp`, `AdminDeleteHotSearchOp` |
| `admin_hot_search_dashboard.go` | `AdminHotSearchDashboard`, `AdminRemoveHotSearchRedis`, `AdminBoostHotSearchRedis`, `AdminReorderHotSearch` |
| `admin_system_config.go` | `AdminListSystemConfigs`, `AdminUpdateSystemConfig` |

## Search

| File | Key handlers |
| :--- | :--- |
| `search.go` | `SearchAll`, `InitHotRecorder` |
| `search_history.go` | `GetMySearchHistory`, `PutMySearchHistory`, `PostMySearchHistory` |
| `search_suggest.go` | `SearchSuggest` |
| `hot_search.go` | `HotSearchList` |

## Other

| File | Key handlers |
| :--- | :--- |
| `auth.go` | `Register`, `Login`, `Refresh` |
| `favorite_folder.go` | `ListMyFavoriteFolders`, `CreateFavoriteFolder`, `UpdateFavoriteFolder`, `DeleteFavoriteFolder`, list/add/remove video favorites |
| `view_history.go` | `PostVideoViewHistory`, `RecordArticleViewHistory`, `ListMyViewHistory` |
| `coin_ledger.go` | `ListMeCoinLedger` |
| `health.go` | `Health` |
| `stats.go` | `HomeStats` |
| `ws.go` | `ServeDanmaku` (WebSocket danmaku endpoint) |
| `home_banner.go` | `ListHomeBanners` |
| `sensitive_ugc.go` | Sensitive content check helpers |
| `es_sync.go` | Elasticsearch index sync helpers |
| `notification_helpers.go` | Notification formatting and aggregation helpers |
| `swagger.go` | `RegisterSwaggerRoutes` |
| `router.go` | `RegisterRoutes` (master route table) |
| `deps.go` | `Dependencies` struct, `API` type |

## OSS Helpers (suffix `_oss.go`)

| File | Purpose |
| :--- | :--- |
| `agent_oss.go` | Agent profile avatar upload to OSS |
| `article_oss.go` | Article image upload to OSS |
| `banner_oss.go` | Banner image upload to OSS |
| `dynamic_oss.go` | Dynamic post image upload to OSS |
| `favorite_folder_oss.go` | Favorite folder cover upload to OSS |
| `video_oss.go` | Video and cover upload to OSS |
