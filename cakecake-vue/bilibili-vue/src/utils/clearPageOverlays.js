/**
 * 清理 Teleport 到 body 的残留遮罩（动态编辑、Element 滚动锁等），避免创作中心整页无法点击。
 */
export function clearStuckPageOverlays() {
  if (typeof document === "undefined") return;

  document.body.classList.remove(
    "el-popup-parent--hidden",
    "mb-dyn-edit-leave-open"
  );
  document.body.style.removeProperty("width");
  document.body.style.removeProperty("overflow");
  document.body.style.removeProperty("padding-right");

  document
    .querySelectorAll(".mb-dyn-edit-overlay, .mb-dyn-leave-overlay")
    .forEach(el => el.remove());

  /* 勿移除稿件删除确认层 */
  if (document.querySelector(".mm-del-overlay")) {
    return;
  }

  /** 无可见 MessageBox / Dialog 时移除 Element 残留全屏遮罩 */
  const msgBox = document.querySelector(".el-message-box__wrapper");
  const msgVisible = msgBox && msgBox.offsetParent !== null;
  if (!msgVisible) {
    document.querySelectorAll(".el-overlay").forEach(el => {
      if (el.querySelector(".el-message-box")) return;
      if (el.querySelector(".el-dialog")) return;
      el.remove();
    });
  }
}
