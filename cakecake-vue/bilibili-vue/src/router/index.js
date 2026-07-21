import { createRouter, createWebHashHistory } from "vue-router";
import { nextTick } from "vue";
import { ElMessageBox } from "element-plus";
import { clearStuckPageOverlays } from "@/utils/clearPageOverlays";
import { getAccessToken, getRefreshToken } from "@/utils/authTokens";
import { isAdminLoggedIn } from "@/utils/adminAuth";
import {
  isAccessTokenExpired,
  refreshMinibiliAccessToken
} from "@/utils/minibiliTokenRefresh";
import { shouldRedirectVideoToNotFound } from "@/utils/notFoundRedirect";
import {
  SITE_HOME_TITLE,
  buildDocumentTitle
} from "@/constants/siteTitle";

const minibiliEnv =
  import.meta.env.VITE_MINIBILI_API === "true" ||
  import.meta.env.VITE_MINIBILI_API === "1";

const routes = [
  {
    name: "home",
    path: "/",
    component: () => import("@/pages/home/index.vue"),
    meta: {
      title: SITE_HOME_TITLE
    }
  },
  {
    name: "Ranking",
    path: "/ranking",
    component: () => import("@/pages/ranking/ranking.vue"),
    redirect: "/ranking/all/0/0/0",
    children: [
      {
        name: "rankingDetail",
        path: ":type/:rid/:rankselect/:rankselect2",
        component: () => import("@/components/ranking/allList.vue"),
        meta: {
          title: buildDocumentTitle("热门视频排行榜")
        }
      }
    ]
  },
  {
    path: "/search",
    component: () => import("@/pages/search/search.vue"),
    redirect: "/search/all",
    children: [
      {
        name: "searchAll",
        path: "all",
        component: () => import("@/components/search/searchList.vue"),
        meta: {
          title: buildDocumentTitle("搜索结果")
        }
      },
      {
        name: "searchVideo",
        path: "video",
        component: () => import("@/components/search/searchList.vue"),
        meta: {
          title: buildDocumentTitle("搜索结果")
        }
      },
      {
        name: "searchBangumi",
        path: "bangumi",
        component: () => import("@/components/search/searchList.vue"),
        meta: {
          title: buildDocumentTitle("搜索结果")
        }
      },
      {
        name: "searchPgc",
        path: "pgc",
        component: () => import("@/components/search/searchList.vue"),
        meta: {
          title: buildDocumentTitle("搜索结果")
        }
      },
      {
        name: "searchLive",
        path: "live",
        component: () => import("@/components/search/searchList.vue"),
        meta: {
          title: buildDocumentTitle("搜索结果")
        }
      },
      {
        name: "searchArticle",
        path: "article",
        component: () => import("@/components/search/searchList.vue"),
        meta: {
          title: buildDocumentTitle("搜索结果")
        }
      },
      {
        name: "searchTopic",
        path: "topic",
        component: () => import("@/components/search/searchList.vue"),
        meta: {
          title: buildDocumentTitle("搜索结果")
        }
      },
      {
        name: "upuser",
        path: "upuser",
        component: () => import("@/components/search/searchList.vue"),
        meta: {
          title: buildDocumentTitle("搜索结果")
        }
      },
      {
        name: "photo",
        path: "photo",
        component: () => import("@/components/search/searchList.vue"),
        meta: {
          title: buildDocumentTitle("搜索结果")
        }
      }
    ]
  },
  {
    name: "video",
    path: "/video/:aid",
    component: () => import("@/pages/video/video.vue"),
    meta: {
      title: ":aid - " + SITE_HOME_TITLE
    }
  },
  {
    name: "upload",
    path: "/upload",
    component: () => import("@/pages/upload/upload.vue"),
    meta: { title: buildDocumentTitle("创作中心") }
  },
  {
    name: "videoPublish",
    path: "/upload/publish",
    component: () => import("@/pages/upload/videoPublish.vue"),
    meta: { title: "投稿视频 - 创作中心" }
  },
  {
    name: "videoEdit",
    path: "/upload/edit/:id",
    component: () => import("@/pages/upload/videoPublish.vue"),
    meta: { title: "编辑视频 - 创作中心" }
  },
  {
    name: "articlePublish",
    path: "/upload/article/publish",
    component: () => import("@/pages/upload/articlePublish.vue"),
    meta: { title: "专栏投稿 - 创作中心" }
  },
  {
    name: "articleEdit",
    path: "/upload/article/edit/:id",
    component: () => import("@/pages/upload/articlePublish.vue"),
    meta: { title: "编辑专栏 - 创作中心" }
  },
  {
    name: "manuscript",
    path: "/upload/manuscript",
    component: () => import("@/pages/upload/manuscript.vue"),
    meta: { title: "稿件管理 - 创作中心" }
  },
  {
    name: "appeal",
    path: "/upload/appeal",
    component: () => import("@/pages/upload/appeal.vue"),
    meta: { title: "申诉管理 - 创作中心" }
  },
  {
    name: "creatorComments",
    path: "/upload/comments",
    component: () => import("@/pages/upload/commentManage.vue"),
    meta: { title: "评论管理 - 创作中心" }
  },
  {
    name: "creatorDanmakus",
    path: "/upload/danmakus",
    component: () => import("@/pages/upload/danmakuManage.vue"),
    meta: { title: "弹幕管理 - 创作中心" }
  },
  {
    path: "/minibili/login",
    name: "minibiliLogin",
    component: () => import("@/pages/minibili/Login.vue"),
    meta: { title: "cakecake 登录" }
  },
  {
    path: "/minibili/register",
    name: "minibiliRegister",
    component: () => import("@/pages/minibili/Register.vue"),
    meta: { title: "cakecake 注册" }
  },
  {
    name: "minibiliMessages",
    path: "/minibili/messages",
    component: () => import("@/pages/minibili/Messages.vue"),
    meta: { title: "cakecake 消息", requireMinibiliAuth: true }
  },
  {
    name: "minibiliPersonalCenter",
    path: "/minibili/account",
    component: () => import("@/pages/minibili/PersonalCenter.vue"),
    meta: { title: "个人中心 - cakecake", requireMinibiliAuth: true }
  },
  {
    name: "minibiliUserSpace",
    path: "/minibili/up/:userId",
    component: () => import("@/pages/minibili/PersonalSpace.vue"),
    meta: { title: "个人空间 - cakecake" }
  },
  {
    name: "minibiliUserSpaceRelations",
    path: "/minibili/up/:userId/relations",
    component: () => import("@/pages/minibili/SpaceRelations.vue"),
    meta: { title: "关注与粉丝 - cakecake" }
  },
  {
    name: "minibiliWatchLater",
    path: "/minibili/watch-later",
    component: () => import("@/pages/minibili/WatchLater.vue"),
    meta: { title: "稍后再看 - cakecake" }
  },
  {
    name: "minibiliDynamics",
    path: "/minibili/dynamics",
    component: () => import("@/pages/minibili/Dynamics.vue"),
    meta: { title: "动态 - cakecake", requireMinibiliAuth: true }
  },
  {
    name: "minibiliArticleRead",
    path: "/minibili/article/:id",
    component: () => import("@/pages/minibili/ArticleRead.vue"),
    meta: { title: "专栏 - cakecake" }
  },
  {
    name: "minibiliDynamicRead",
    path: "/minibili/dynamic/:id",
    component: () => import("@/pages/minibili/ArticleRead.vue"),
    meta: { title: "动态 - cakecake" }
  },
  {
    name: "minibiliViewHistory",
    path: "/minibili/history",
    component: () => import("@/pages/minibili/ViewHistory.vue"),
    meta: { title: "历史记录 - cakecake", requireMinibiliAuth: true }
  },
  {
    name: "minibiliUpload",
    path: "/minibili/upload",
    component: () => import("@/pages/minibili/Upload.vue"),
    meta: { title: "cakecake 上传", requireMinibiliAuth: true }
  },
  {
    path: "/admin/login",
    name: "adminLogin",
    component: () => import("@/pages/admin/AdminLogin.vue"),
    meta: { title: "运营后台登录", hideGlobalChrome: true }
  },
  {
    path: "/admin",
    component: () => import("@/pages/admin/AdminLayout.vue"),
    meta: { hideGlobalChrome: true, requireAdminAuth: true },
    children: [
      { path: "", redirect: { name: "adminBanners" } },
      {
        path: "banners",
        name: "adminBanners",
        component: () => import("@/pages/admin/BannerManage.vue"),
        meta: { title: "首页轮播 - 运营后台" }
      },
      {
        path: "hot-search",
        name: "adminHotSearch",
        component: () => import("@/pages/admin/HotSearchManage.vue"),
        meta: { title: "热搜运营 - 运营后台" }
      },
      {
        path: "video-review",
        name: "adminVideoReview",
        component: () => import("@/pages/admin/VideoReview.vue"),
        meta: { title: "视频审核 - 运营后台" }
      },
      {
        path: "article-review",
        name: "adminArticleReview",
        component: () => import("@/pages/admin/ArticleReview.vue"),
        meta: { title: "专栏审核 - 运营后台" }
      },
      {
        path: "dynamic-manage",
        name: "adminDynamicManage",
        component: () => import("@/pages/admin/DynamicManage.vue"),
        meta: { title: "动态管理 - 运营后台" }
      },
      {
        path: "agent",
        name: "adminAgent",
        component: () => import("@/pages/admin/AgentManage.vue"),
        meta: { title: "AI 角色 - 运营后台" }
      },
      {
        path: "system-configs",
        name: "adminSystemConfigs",
        component: () => import("@/pages/admin/SystemConfigManage.vue"),
        meta: { title: "运行时配置 - 运营后台" }
      }
    ]
  },
  {
    path: "/404",
    name: "notFound",
    component: () => import("@/pages/notFound/404.vue"),
    meta: {
      title: "页面不存在 - cakecake"
    }
  },
  {
    path: "/:pathMatch(.*)*",
    redirect: "/404"
  }
];

const router = createRouter({
  history: createWebHashHistory(),
  routes,
  scrollBehavior(_to, _from, savedPosition) {
    if (savedPosition) {
      return savedPosition;
    }
    return { top: 0, left: 0 };
  }
});

/** cakecake：未登录访问需鉴权页时跳转首页 */
router.beforeEach(async (to, _from, next) => {
  if (!document.querySelector(".mm-del-overlay")) {
    clearStuckPageOverlays();
  }
  const needAdmin = to.matched.some(
    r => r.meta && r.meta.requireAdminAuth === true
  );
  if (needAdmin && !isAdminLoggedIn()) {
    next({ name: "adminLogin", replace: true });
    return;
  }
  if (to.name === "adminLogin" && isAdminLoggedIn()) {
    next({ name: "adminBanners", replace: true });
    return;
  }
  if (shouldRedirectVideoToNotFound(to)) {
    next({ name: "notFound", replace: true });
    return;
  }
  if (!minibiliEnv) {
    next();
    return;
  }
  const need = to.matched.some(
    r => r.meta && r.meta.requireMinibiliAuth === true
  );
  if (need && (!getAccessToken() || isAccessTokenExpired())) {
    if (getRefreshToken()) {
      const ok = await refreshMinibiliAccessToken();
      if (ok) {
        next();
        return;
      }
    }
    next({ path: "/", replace: true });
    return;
  }
  next();
});

/** 离开发布/编辑页：关对话框 + 清理 MessageBox 滚动锁（勿关掉投稿成功审核提示） */
router.afterEach((to, from) => {
  if (from.name === "videoPublish" || from.name === "videoEdit") {
    const keepReviewNotice =
      (to.name === "upload" &&
        String(to.query.success || "").toLowerCase() === "publish") ||
      (to.name === "manuscript" && String(to.query.reviewNotice) === "1");
    if (!keepReviewNotice) {
      ElMessageBox.close();
      nextTick(() => {
        document.body.classList.remove("el-popup-parent--hidden");
        document.body.style.removeProperty("width");
      });
    }
  }
  if (!document.querySelector(".mm-del-overlay")) {
    clearStuckPageOverlays();
  }
});

export default router;
