import { h } from "vue";
import { ElMessageBox } from "element-plus";
import "@/styles/cm-del-msgbox.scss";

export const CREATOR_VIDEO_REVIEW_HINT =
  "视频审核中，马上就能和大家见面啦~";

const DEFER_MS = 280;

/** 投稿成功后的审核提示弹窗（创作中心）；延迟展示，避免路由 afterEach 误关弹窗 */
export function showCreatorVideoReviewNotice() {
  return new Promise((resolve) => {
    window.setTimeout(() => {
      ElMessageBox.alert(
        h("div", { class: "cm-del-msgbox-body" }, [
          h("p", { class: "cm-del-msgbox-msg" }, CREATOR_VIDEO_REVIEW_HINT)
        ]),
        "投稿成功",
        {
          confirmButtonText: "我知道了",
          customClass: "cm-del-msgbox",
          confirmButtonClass: "cm-del-msgbox__ok",
          showClose: true,
          closeOnClickModal: false,
          closeOnPressEscape: true
        }
      )
        .then(resolve)
        .catch(resolve);
    }, DEFER_MS);
  });
}
