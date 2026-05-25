import { createApp } from "vue";
import App from "./App.vue";
import router from "./router/index";
import { formatVideoBvid, parseVideoIdFromRoute } from "./utils/videoBvid";
import { buildDocumentTitle } from "./constants/siteTitle";
import store from "./store/index";
import VueLazyload from "vue-lazyload";
import axios from "./utils/http";
import { registerMinibiliSessionInvalidate } from "./utils/minibiliAuthSync";
import {
  getAccessToken,
  getRefreshToken
} from "./utils/authTokens";
import {
  isAccessTokenExpired,
  refreshMinibiliAccessToken
} from "./utils/minibiliTokenRefresh";
import loadingImg from "./assets/loading.png";
import { clearStuckPageOverlays } from "./utils/clearPageOverlays";
import { installElasticLayout } from "./utils/elasticLayout";

/* Message + 各页用到的 Element 组件（此前只引 Message，Button/Form 等会无主题、看起来像灰色不可点） */
import "element-plus/dist/index.css";
import "./styles/mb-dark-toast.scss";
import { ElMessage } from "element-plus";
import { setupElementPlus } from "./plugins/elementPlus";

/** 产品要求：成功操作不弹出绿色 Toast，仅静默更新界面（错误、警告仍保留） */
ElMessage.success = () => {};

const app = createApp(App);
setupElementPlus(app);

registerMinibiliSessionInvalidate(() => {
  store.commit("login/SET_SIGNIN", { signIn: "0" });
  store.commit("login/SYNC_MINIBILI_ME", null);
  store.commit("login/SET_USER_INFO", { proInfo: [] });
});

const minibiliEnv =
  import.meta.env.VITE_MINIBILI_API === "true" ||
  import.meta.env.VITE_MINIBILI_API === "1";

app.use(store);
app.use(router);

/** 有 refresh 时先续 access，避免首屏请求因 2h 过期 access 直接登出 */
if (minibiliEnv && localStorage.getItem("signIn") === "1") {
  const hasRefresh = !!getRefreshToken();
  const needRefresh = !getAccessToken() || isAccessTokenExpired();
  if (hasRefresh && needRefresh) {
    void refreshMinibiliAccessToken().then(ok => {
      if (ok) {
        void store.dispatch("login/refreshMinibiliMe");
      }
    });
  } else if (getAccessToken()) {
    void store.dispatch("login/refreshMinibiliMe");
  }
}
app.use(VueLazyload, {
  preLoad: 1.3,
  error: loadingImg,
  loading: loadingImg,
  attempt: 1
});
app.config.globalProperties.$http = axios;

router.beforeEach((to, _from, next) => {
  if (to.name === "video" && to.params.aid) {
    const vid = parseVideoIdFromRoute(to.params.aid);
    const label = vid != null ? formatVideoBvid(vid) : String(to.params.aid);
    document.title = buildDocumentTitle(label);
  } else if (to.meta.title) {
    document.title = to.meta.title;
  }
  next();
});

clearStuckPageOverlays();
installElasticLayout();
app.mount("#app");
