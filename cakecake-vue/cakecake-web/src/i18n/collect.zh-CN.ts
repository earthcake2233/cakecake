/**
 * 个人空间 - 收藏 tab 文案（独立 UTF-8 模块，避免大 Vue 文件编码损坏）
 */
export const collectZhCN = {
  sidenavAria: "收藏分类",
  foldersNav: "我创建的收藏夹",
  watchLaterNav: "稍后再看",
  public: "公开",
  private: "私密",
  videoCount: "视频数:",
  playAll: "播放全部",
  batchOp: "批量操作",
  sortAria: "收藏排序",
  sortRecent: "最近收藏",
  sortPlay: "最多播放",
  sortSubmit: "最近投稿",
  searchScopeCurrent: "当前收藏夹",
  searchScopeAll: "全部收藏夹",
  searchPlaceholder: "请输入关键词",
  searchBtnAria: "搜索",
  feedAria: "收藏视频列表",
  loading: "加载中…",
  emptyFavAria: "暂无收藏视频",
  watchLaterTitle: "稍后再看",
  watchLaterListAria: "稍后再看视频列表",
  watchLaterEmptyAria: "稍后再看暂无内容",
  watchLaterEmptyOwn: "还没有稍后再看的视频",
  watchLaterEmptyGuest: "仅可在自己的空间查看稍后再看",
  defaultFolderTitle: "默认收藏夹",
  favoritedAt: "收藏于",
  metaSep: "·",
  watchLaterOnCover: "稍后再看",
} as const;

export type CollectZhCN = typeof collectZhCN;
