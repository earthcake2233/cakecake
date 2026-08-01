import store from "@/store/index";
import { setCakecakePostLoginRedirect } from "@/utils/authTokens";

/**
 * 打开顶栏 cakecake 登录/注册弹窗（替代独立 /cakecake/login 页）。
 * @param {{ tab?: number, redirect?: string }} [opts]
 * @param {number} [opts.tab] 0 登录，1 注册
 * @param {string} [opts.redirect] 登录成功后跳转的 path（须以 / 开头）
 */
export function openCakecakeLoginModal(opts = {}) {
  const tab = opts.tab === 1 ? 1 : 0;
  const redir = opts.redirect && String(opts.redirect).trim();
  if (redir && redir.startsWith("/")) {
    setCakecakePostLoginRedirect(redir);
  }
  store.commit("login/SET_LOGIN_TAB", tab);
  store.commit("login/OPEN_LOGIN_MODAL");
}
