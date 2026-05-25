/**
 * 视口弹性布局：仅通过运行时注入布局规则，不修改项目内既有 .scss / .vue <style>。
 * 只覆盖 width / max-width / min-width / float / flex / position / overflow 等布局属性。
 */

const STYLE_ID = "mb-elastic-layout";

const LAYOUT_CSS = `
:root {
  --mb-e-gutter: 16px;
  --mb-e-main: 1160px;
  --mb-e-sub: 980px;
  --mb-e-dynamics: 1400px;
}

html {
  overflow-x: clip;
}

#app.app {
  width: 100%;
  max-width: 100%;
  overflow-x: clip;
  box-sizing: border-box;
}

.app-body {
  width: 100%;
  max-width: 100%;
  min-width: 0;
}

.bili-wrapper,
.main-inner,
.error-container,
.primary-menu,
.contain,
.footer-cnt {
  box-sizing: border-box !important;
  margin-left: auto !important;
  margin-right: auto !important;
}

.bili-wrapper,
.main-inner,
.error-container,
.primary-menu {
  width: min(var(--mb-e-main), calc(100vw - 2 * var(--mb-e-gutter))) !important;
  max-width: 100% !important;
}

/* 动态页三栏布局需更宽，避免中间 feed 被 1160 总宽压窄 */
.bili-wrapper.mb-dyn-wrap {
  width: min(var(--mb-e-dynamics), calc(100vw - 2 * var(--mb-e-gutter))) !important;
  max-width: 100% !important;
}

.contain,
.footer-cnt {
  width: min(var(--mb-e-sub), calc(100vw - 2 * var(--mb-e-gutter))) !important;
  max-width: 100% !important;
}

.msg-layout,
.adm-body {
  max-width: min(var(--mb-e-main), calc(100vw - 2 * var(--mb-e-gutter))) !important;
  width: 100% !important;
  box-sizing: border-box !important;
  margin-left: auto !important;
  margin-right: auto !important;
}

.vp-page {
  max-width: min(1040px, calc(100vw - 2 * var(--mb-e-gutter))) !important;
  width: 100% !important;
  box-sizing: border-box !important;
  margin-left: auto !important;
  margin-right: auto !important;
}

.mm-wrap {
  max-width: min(1120px, calc(100vw - 2 * var(--mb-e-gutter))) !important;
  width: 100% !important;
  box-sizing: border-box !important;
  margin-left: auto !important;
  margin-right: auto !important;
}

.ap-wrap {
  max-width: min(1200px, calc(100vw - 2 * var(--mb-e-gutter))) !important;
  width: 100% !important;
  box-sizing: border-box !important;
  margin-left: auto !important;
  margin-right: auto !important;
}

.creator-panel {
  max-width: min(920px, calc(100vw - 2 * var(--mb-e-gutter))) !important;
  width: 100% !important;
  box-sizing: border-box !important;
  margin-left: auto !important;
  margin-right: auto !important;
}

@media (max-width: 1199px) {
  .bili-wrapper,
  .main-inner,
  .error-container,
  .primary-menu,
  .contain,
  .msg-layout,
  .vp-page,
  .mm-wrap,
  .ap-wrap,
  .creator-panel,
  .adm-body,
  .bili-wrapper.mb-dyn-wrap {
    padding-left: var(--mb-e-gutter);
    padding-right: var(--mb-e-gutter);
  }
}

@media (max-width: 959px) {
  .bili-wrapper:has(.l-con):has(.r-con) {
    display: flex !important;
    flex-direction: column !important;
    gap: 16px;
  }

  .bili-wrapper .l-con,
  .bili-wrapper .r-con,
  .popularize-module > .r-con {
    float: none !important;
    width: 100% !important;
    max-width: 100% !important;
  }
}

@media (min-width: 960px) and (max-width: 1199px) {
  .bili-wrapper .l-con {
    width: calc(100% - 276px) !important;
    max-width: 900px !important;
    min-width: 0 !important;
  }

  .bili-wrapper .r-con {
    width: 260px !important;
    min-width: 0 !important;
  }
}

@media (max-width: 1189px) {
  .video-page:not(.video-page--wide) .video-body-stack {
    width: 100% !important;
    max-width: 100% !important;
  }

  .video-page:not(.video-page--wide) .video-side-dock {
    position: static !important;
    left: auto !important;
    top: auto !important;
    width: 100% !important;
    height: auto !important;
    margin-top: 16px;
  }
}

.app-header .head-content .search,
.app-header .search {
  max-width: min(420px, calc(100vw - 120px));
}

.adm-table-wrap {
  max-width: 100%;
}
`;

/** 在 document 就绪后注入弹性布局样式表 */
export function installElasticLayout() {
  if (typeof document === "undefined") return;
  if (document.getElementById(STYLE_ID)) return;

  const style = document.createElement("style");
  style.id = STYLE_ID;
  style.setAttribute("data-source", "mb-elastic-layout");
  style.textContent = LAYOUT_CSS;
  document.head.appendChild(style);

  const syncViewport = () => {
    const w = window.innerWidth;
    document.documentElement.classList.toggle("mb-e-narrow", w < 960);
    document.documentElement.classList.toggle("mb-e-compact", w < 1200);
  };

  syncViewport();
  window.addEventListener("resize", syncViewport, { passive: true });
}
