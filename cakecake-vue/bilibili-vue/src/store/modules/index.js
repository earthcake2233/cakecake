/**
 * 聚合 Vuex 子模块（Vite 不支持 webpack 的 require.context）
 */
import header from "./header";
import search from "./search";
import video from "./video";
import login from "./login";
import ranking from "./ranking";
import notFound from "./404";

export default {
  header,
  search,
  video,
  login,
  ranking,
  "404": notFound
};
