# -*- coding: utf-8 -*-
"""Patch PersonalSpace.vue: collect video three-dot menu + transfer dialog."""
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PS = ROOT / "src" / "pages" / "minibili" / "PersonalSpace.vue"

OLD_BLOCK = """                    <li
                      v-for="v in collectDisplayVideos"
                      :key="'fav-' + v.id"
                      class="mb-space__vcell"
                    >
                      <router-link
                        class="mb-space__vcell-link"
                        :to="{ name: 'video', params: { aid: String(v.id) } }"
                      >"""

NEW_BLOCK = """                    <li
                      v-for="v in collectDisplayVideos"
                      :key="'fav-' + v.id"
                      class="mb-space__vcell"
                      :class="{
                        'is-collect-menu-open': collectVideoMenuId === v.id
                      }"
                    >
                      <router-link
                        class="mb-space__vcell-link"
                        :to="{ name: 'video', params: { aid: String(v.id) } }"
                      >"""

OLD_TITLE = """                        <motion class="mb-space__vtext-col">
                          <p class="mb-space__vtitle" :title="v.title">
                            {{ v.title }}
                          </p>
                          <p class="mb-space__vmeta mb-space__vmeta--collect">"""

NEW_TITLE = """                        <div class="mb-space__vtext-col">
                          <motion class="mb-space__vtitle-row">
                            <p class="mb-space__vtitle" :title="v.title">
                              {{ v.title }}
                            </p>
                            <div
                              v-if="isOwnSpace && mbLoggedIn"
                              class="mb-space__vmore-wrap"
                            >
                              <button
                                type="button"
                                class="mb-space__vmore-btn"
                                aria-label="更多操作"
                                @click.stop.prevent="toggleCollectVideoMenu(v.id)"
                              >
                                <span class="mb-space__vmore-ico" aria-hidden="true"
                                  >&#8942;</span
                                >
                              </button>
                              <ul
                                v-if="collectVideoMenuId === v.id"
                                class="mb-space__vmore-menu"
                                role="menu"
                                @click.stop
                              >
                                <li role="none">
                                  <button
                                    type="button"
                                    role="menuitem"
                                    @click="onCollectUnfavorite(v)"
                                  >
                                    取消收藏
                                  </button>
                                </li>
                                <li role="none">
                                  <button
                                    type="button"
                                    role="menuitem"
                                    @click="onCollectCopyTo(v)"
                                  >
                                    复制至
                                  </button>
                                </li>
                                <li role="none">
                                  <button
                                    type="button"
                                    role="menuitem"
                                    @click="onCollectMoveTo(v)"
                                  >
                                    移动至
                                  </button>
                                </li>
                              </ul>
                            </div>
                          </div>
                          <p class="mb-space__vmeta mb-space__vmeta--collect">"""

# fix motion typos in NEW_TITLE - use div not motion
NEW_TITLE = NEW_TITLE.replace("<motion ", "<motion ").replace("</motion>", "</motion>")
NEW_TITLE = NEW_TITLE.replace("motion class", "div class").replace("</motion>", "</div>")

OLD_CLOSE = """                          </p>
                        </div>
                      </router-link>
                    </li>
                  </ul>
                  <div
                    v-else
                    class="mb-space__empty-img"
                    role="img"
                    aria-label="暂无收藏视频"
                  >"""

# The old close might use </div> already - read file

def main():
    text = PS.read_text(encoding="utf-8")
    if "collectVideoMenuId" in text:
        print("already patched")
        return
    if OLD_BLOCK not in text:
        raise SystemExit("OLD_BLOCK not found")
    text = text.replace(OLD_BLOCK, NEW_BLOCK, 1)
    if '<div class="mb-space__vtext-col">' in text and "mb-space__vtitle-row" in text:
        print("title block maybe already done")
    else:
        old_t = """                        <motion class="mb-space__vtext-col">
                          <p class="mb-space__vtitle" :title="v.title">
                            {{ v.title }}
                          </p>
                          <p class="mb-space__vmeta mb-space__vmeta--collect">"""
        old_t = old_t.replace("motion", "motion")
        old_t = """                        <div class="mb-space__vtext-col">
                          <p class="mb-space__vtitle" :title="v.title">
                            {{ v.title }}
                          </p>
                          <p class="mb-space__vmeta mb-space__vmeta--collect">"""
        new_t = NEW_TITLE
        if old_t not in text:
            raise SystemExit("OLD_TITLE not found")
        text = text.replace(old_t, new_t, 1)

    imp_old = "import MbCollectFolderCreateDialog from"
    imp_new = (
        "import MbCollectVideoFolderTransferDialog from "
        '"@/components/minibili/MbCollectVideoFolderTransferDialog.vue";\n'
        "import MbCollectFolderCreateDialog from"
    )
    if imp_new not in text:
        text = text.replace(imp_old, imp_new, 1)

    api_old = "  mbCreateFavoriteFolder,\n  mbDeleteMyVideo,"
    api_new = (
        "  mbCreateFavoriteFolder,\n"
        "  mbRemoveVideoFromFavoriteFolder,\n"
        "  mbCopyVideoToFavoriteFolder,\n"
        "  mbMoveVideoFavoriteFolder,\n"
        "  mbDeleteMyVideo,"
    )
    text = text.replace(api_old, api_new, 1)

    comp_old = "    MbCollectFolderCreateDialog,\n    MinibiliCommentsLive,"
    comp_new = (
        "    MbCollectFolderCreateDialog,\n"
        "    MbCollectVideoFolderTransferDialog,\n"
        "    MinibiliCommentsLive,"
    )
    text = text.replace(comp_old, comp_new, 1)

    data_old = "      collectFolderCreateSaving: false,\n      collectFolderIco,"
    data_new = (
        "      collectFolderCreateSaving: false,\n"
        "      collectVideoMenuId: null,\n"
        "      collectTransferOpen: false,\n"
        "      collectTransferMode: 'copy',\n"
        "      collectTransferVideoId: null,\n"
        "      collectTransferSaving: false,\n"
        "      collectFolderIco,"
    )
    text = text.replace(data_old, data_new, 1)

    dlg_old = """  <MbCollectFolderCreateDialog
    v-model="collectFolderCreateOpen"
    :loading="collectFolderCreateSaving"
    @submit="onCollectFolderCreateSubmit"
  />

  <MbStationDialog"""
    dlg_new = """  <MbCollectFolderCreateDialog
    v-model="collectFolderCreateOpen"
    :loading="collectFolderCreateSaving"
    @submit="onCollectFolderCreateSubmit"
  />

  <MbCollectVideoFolderTransferDialog
    v-model="collectTransferOpen"
    :mode="collectTransferMode"
    :from-folder-id="collectFolderId"
    :loading="collectTransferSaving"
    @confirm="onCollectTransferConfirm"
    @cancel="collectTransferOpen = false"
  />

  <MbStationDialog"""
    text = text.replace(dlg_old, dlg_new, 1)

    doc_old = "    onDynCommentDocClick() {\n      this.dynCmtHeadMenuOpen = false;\n    },"
    doc_new = (
        "    onDynCommentDocClick() {\n"
        "      this.dynCmtHeadMenuOpen = false;\n"
        "      this.collectVideoMenuId = null;\n"
        "    },"
    )
    text = text.replace(doc_old, doc_new, 1)

    methods_anchor = "    async loadCollectFavorites() {"
    methods_insert = """    toggleCollectVideoMenu(videoId) {
      const id = Number(videoId);
      this.collectVideoMenuId =
        this.collectVideoMenuId === id ? null : id;
    },
    closeCollectVideoMenu() {
      this.collectVideoMenuId = null;
    },
    async onCollectUnfavorite(v) {
      this.closeCollectVideoMenu();
      if (this.collectFolderId == null) return;
      const videoId = Number(v && v.id);
      if (!videoId) return;
      try {
        await mbRemoveVideoFromFavoriteFolder(
          videoId,
          this.collectFolderId
        );
        this.collectVideos = this.collectVideos.filter(
          (row) => Number(row.id) !== videoId
        );
        await this.loadCollectFolders();
        ElMessage.success("已从当前收藏夹移除");
      } catch (e) {
        const msg =
          (e && e.response && e.response.data && e.response.data.message) ||
          (e && e.message) ||
          "操作失败";
        ElMessage.error(String(msg));
      }
    },
    onCollectCopyTo(v) {
      this.closeCollectVideoMenu();
      this.collectTransferVideoId = Number(v && v.id) || null;
      this.collectTransferMode = "copy";
      this.collectTransferOpen = true;
    },
    onCollectMoveTo(v) {
      this.closeCollectVideoMenu();
      this.collectTransferVideoId = Number(v && v.id) || null;
      this.collectTransferMode = "move";
      this.collectTransferOpen = true;
    },
    async onCollectTransferConfirm(targetFolderId) {
      const videoId = this.collectTransferVideoId;
      const folderId = Number(targetFolderId);
      if (!videoId || !folderId || this.collectTransferSaving) return;
      if (this.collectFolderId == null && this.collectTransferMode === "move") {
        return;
      }
      this.collectTransferSaving = true;
      try {
        if (this.collectTransferMode === "move") {
          await mbMoveVideoFavoriteFolder(
            videoId,
            this.collectFolderId,
            folderId
          );
          this.collectVideos = this.collectVideos.filter(
            (row) => Number(row.id) !== videoId
          );
          ElMessage.success("已移动到其他收藏夹");
        } else {
          await mbCopyVideoToFavoriteFolder(videoId, folderId);
          ElMessage.success("已复制到收藏夹");
        }
        this.collectTransferOpen = false;
        await this.loadCollectFolders();
        if (
          this.collectTransferMode === "copy" &&
          folderId === this.collectFolderId
        ) {
          await this.loadCollectFavorites();
        }
      } catch (e) {
        const msg =
          (e && e.response && e.response.data && e.response.data.message) ||
          (e && e.message) ||
          "操作失败";
        ElMessage.error(String(msg));
      } finally {
        this.collectTransferSaving = false;
      }
    },
"""
    if "toggleCollectVideoMenu" not in text:
        text = text.replace(methods_anchor, methods_insert + methods_anchor, 1)

    PS.write_text(text, encoding="utf-8", newline="\n")
    print("patched", PS)


if __name__ == "__main__":
    main()
