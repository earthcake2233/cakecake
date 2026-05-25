import { h } from "vue";
import { ElMessageBox } from "element-plus";
import "@/styles/cm-del-msgbox.scss";

export const CREATOR_ARTICLE_REVIEW_HINT =
  "专栏审核中，马上就能和大家见面啦~";

const DEFER_MS = 280;

/** 专栏投稿成功后的审核提示弹窗（创作中心） */
export function showCreatorArticleReviewNotice() {
  return new Promise((resolve) => {
    window.setTimeout(() => {
      ElMessageBox.alert(
        h("div", { class: "cm-del-msgbox-body" }, [
          h("p", { class: "cm-del-msgbox-msg" }, CREATOR_ARTICLE_REVIEW_HINT)
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
