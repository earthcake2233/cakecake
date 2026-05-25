/**
 * 个人空间文案（与 PersonalSpace.vue 解耦，统一 UTF-8 维护）
 */
export const personalSpaceZhCN = {
  header: {
    bannerAria: "个人空间头图",
    genderPrefix: "性别：",
  },
  relations: {
    followingHiddenToast: "由于该用户隐私设置，关注列表不可见",
    followersHiddenToast: "由于该用户隐私设置，粉丝列表不可见",
    followingHiddenInline: "由于该用户隐私设置，关注列表不可见",
    followersHiddenInline: "由于该用户隐私设置，粉丝列表不可见",
    sideMyFollowing: "我的关注",
    sideTheirFollowing: "Ta的关注",
    sideMyFans: "我的粉丝",
    sideTheirFans: "Ta的粉丝",
    sideAllFollowing: "全部关注",
    titleAllFollowing: "全部关注",
    titleFans: "Ta的粉丝",
    tagFollowing: "已关注",
    tagMutual: "已互粉",
    tagFans: "粉丝",
    setGroup: "设置分组",
    unfollow: "取消关注",
    unfollowDone: "已取消关注",
    groupEmpty: "暂无分组，可在左侧新建",
    assignHintPrefix: "请为",
    assignHintSuffix: "设置分组",
    assignNewGroup: "新建分组",
    assignCancel: "取消",
    assignConfirm: "确定",
    assignSaving: "保存中…",
    assignSaved: "分组已更新",
  },
  nav: {
    dockAria: "空间主导航",
    searchPlaceholder: "搜索视频、动态",
    searchBtnAria: "搜索",
    statsAria: "空间数据",
    following: "关注",
    fans: "粉丝",
    likes: "获赞",
    home: "主页",
    dynamic: "动态",
    contribute: "投稿",
    collect: "收藏",
    settings: "设置",
  },
  perspective: {
    triggerSelf: "视角：我自己",
    fan: "我的粉丝",
    visitor: "新访客",
    bannerFan: "这是我的空间在粉丝中的样子",
    bannerVisitor: "这是我的空间在新访客眼中的样子",
    closePreview: "关闭预览",
    menuAria: "切换预览视角",
  },
  settings: {
    accountLink: "前往个人中心修改资料与账号设置",
    guestEmptyAria: "仅可在自己的空间内管理账号设置",
  },
  notice: {
    placeholder: "编辑我的公告",
    toast: "公告保存成功",
    emptyAria: "暂无公告",
  },
  deleteVideo: {
    closeAria: "关闭",
    title: "删除视频？",
    message: "删除后不可恢复，是否继续？",
    cancel: "取消",
    confirm: "删除",
  },
  commentLoginSuffix: "后参与评论",
  videoManage: "视频管理",
  playAll: "播放全部",
  titleDot: "·",
  dynMbStation: {
    pick_comment: {
      title: "开启评论精选",
      message:
        "开启精选评论后，所有评论都需经过我的确认后再向所有用户展示。可前往PC端创作中心",
    },
    close_comments: {
      title: "关闭评论",
      message:
        "关闭评论将会禁止任何在此评论区发表内容，且已有评论在关闭期间将不可见",
    },
    restore_comments: {
      title: "关闭评论",
      message:
        "恢复评论后，用户可正常发表评论、参与评论互动，已有的评论也恢复正常展示",
    },
    close_danmaku: {
      title: "关闭评论",
      message:
        "关闭评论将会禁止任何在此评论区发表内容，且已有评论在关闭期间将不可见",
    },
    restore_danmaku: {
      title: "关闭评论",
      message:
        "恢复评论后，用户可正常发表评论、参与评论互动，已有的评论也恢复正常展示",
    },
  },
} as const;

export type SpacePerspectiveMode = "self" | "fan" | "visitor";
