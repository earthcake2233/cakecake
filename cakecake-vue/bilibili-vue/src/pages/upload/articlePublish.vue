<template>
  <CreatorShell>
    <div class="ap-wrap">
      <h1 class="ap-title">{{ isEdit ? "编辑专栏" : "新建专栏" }}</h1>
      <p v-if="editFailReason" class="ap-reject-hint">
        审核未通过：{{ editFailReason }}。修改后可重新发布。
      </p>
      <div class="ap-field">
        <label class="ap-label">标题</label>
        <input
          v-model="title"
          type="text"
          class="ap-input"
          maxlength="80"
          placeholder="请输入专栏标题"
        />
      </div>
      <div class="ap-field ap-field--cover">
        <label class="ap-label">封面（可选）</label>
        <input
          ref="coverInput"
          type="file"
          accept="image/jpeg,image/png,image/webp,image/gif,image/bmp"
          class="ap-cover-input"
          @change="onCoverFileChange"
        />
        <div
          class="ap-cover-drop"
          :class="{ 'ap-cover-drop--hover': coverDropHover }"
          @click="pickCover"
          @dragenter.prevent="coverDropHover = true"
          @dragover.prevent="coverDropHover = true"
          @dragleave.prevent="coverDropHover = false"
          @drop.prevent="onCoverDrop"
        >
          <img
            v-if="coverPreview"
            class="ap-cover-preview"
            :src="coverPreview"
            alt=""
          />
          <template v-else>
            <p class="ap-cover-hint">拖拽图片到此处或点击上传</p>
            <p class="ap-cover-sub">支持 JPG / PNG / WebP / GIF，最大 10MB</p>
          </template>
          <button
            v-if="coverPreview"
            type="button"
            class="ap-cover-change"
            @click.stop="pickCover"
          >
            更换封面
          </button>
        </div>
      </div>
      <div class="ap-field">
        <label class="ap-label">标签（逗号分隔，可选）</label>
        <input
          v-model="tagsText"
          type="text"
          class="ap-input"
          placeholder="技术, 教程"
        />
      </div>
      <div class="ap-editor-row">
        <div class="ap-field ap-field--grow">
          <label class="ap-label">正文（Markdown）</label>
          <textarea
            v-model="bodyMd"
            class="ap-textarea"
            placeholder="支持 Markdown：## 标题、**加粗**、![图片](url)、代码块等"
          />
        </div>
        <div class="ap-field ap-field--grow">
          <label class="ap-label">预览</label>
          <div class="ap-preview" v-html="previewHtml" />
        </div>
      </div>
      <div class="ap-actions">
        <button
          type="button"
          class="ap-btn ap-btn--ghost"
          :disabled="submitting || coverUploading || articleLoading"
          @click="saveDraft"
        >
          存草稿
        </button>
        <button
          type="button"
          class="ap-btn ap-btn--primary"
          :disabled="submitting || coverUploading || articleLoading"
          @click="publish"
        >
          {{ submitting ? "提交中…" : "发布专栏" }}
        </button>
      </div>
    </div>
  </CreatorShell>
</template>

<script>
import MarkdownIt from "markdown-it";
import DOMPurify from "dompurify";
import { ElMessage, ElMessageBox } from "element-plus";
import CreatorShell from "@/components/creator/CreatorShell.vue";
import {
  mbGetMyArticle,
  mbPostArticle,
  mbPutMyArticle,
  mbUpdateArticleCover
} from "@/api/minibili";
import { minibiliArticleReadRoute } from "@/utils/minibiliRoutes";
import { showCreatorArticleReviewNotice } from "@/utils/creatorArticleReviewNotice";

const md = new MarkdownIt({ html: false, linkify: true, breaks: true });
const LEAVE_MSG =
  "系统可能不会保存您填写的专栏内容噢...(´；ω；`)";

export default {
  name: "ArticlePublishPage",
  components: { CreatorShell },
  data() {
    return {
      title: "",
      coverUrl: "",
      pendingCoverFile: null,
      coverObjectUrl: "",
      coverDropHover: false,
      coverUploading: false,
      tagsText: "",
      bodyMd: "",
      submitting: false,
      articleLoading: false,
      editArticleStatus: "",
      editFailReason: "",
      /** 切换路由时递增，避免异步 load 回写上一篇稿件 */
      articleLoadGen: 0,
      leaveGuardEnabled: false,
      saveCommitted: false,
      baselineSnapshot: "",
      leavePromptOpen: false,
      leaveBackTrapInstalled: false,
      suppressLeavePopstateOnce: false,
      leaveWindowListenersBound: false
    };
  },
  computed: {
    coverPreview() {
      if (this.coverObjectUrl) return this.coverObjectUrl;
      const u = String(this.coverUrl || "").trim();
      return u || "";
    },
    isEdit() {
      return !!this.$route.params.id;
    },
    editId() {
      const n = parseInt(String(this.$route.params.id || ""), 10);
      return Number.isFinite(n) && n > 0 ? n : 0;
    },
    previewHtml() {
      const raw = md.render(this.bodyMd || "");
      return DOMPurify.sanitize(raw);
    },
    shouldConfirmLeave() {
      if (!this.leaveGuardEnabled || this.saveCommitted || this.articleLoading) {
        return false;
      }
      return this.baselineSnapshot !== this.currentSnapshot();
    }
  },
  watch: {
    "$route.fullPath"() {
      void this.bootstrapArticlePage();
    },
    title() {
      this.$nextTick(() => this.syncLeaveBackTrap());
    },
    bodyMd() {
      this.$nextTick(() => this.syncLeaveBackTrap());
    },
    tagsText() {
      this.$nextTick(() => this.syncLeaveBackTrap());
    },
    coverUrl() {
      this.$nextTick(() => this.syncLeaveBackTrap());
    }
  },
  mounted() {
    void this.bootstrapArticlePage();
    this.bindLeaveWindowListeners();
  },
  activated() {
    void this.bootstrapArticlePage();
    this.bindLeaveWindowListeners();
  },
  deactivated() {
    this.unbindLeaveWindowListeners();
    ElMessageBox.close();
    this.leavePromptOpen = false;
  },
  beforeUnmount() {
    this.unbindLeaveWindowListeners();
    ElMessageBox.close();
    this.leavePromptOpen = false;
    this.revokeCoverObjectUrl();
  },
  beforeRouteLeave(_to, _from, next) {
    if (!this.shouldConfirmLeave) {
      next();
      return;
    }
    if (this.leavePromptOpen) {
      next(false);
      return;
    }
    this.promptLeave()
      .then(() => {
        ElMessageBox.close();
        this.clearLeaveBackTrapOnly();
        this.saveCommitted = true;
        next();
      })
      .catch(() => next(false));
  },
  methods: {
    currentSnapshot() {
      const coverPending = this.pendingCoverFile
        ? `f:${this.pendingCoverFile.name}:${this.pendingCoverFile.size}`
        : "";
      const coverLocal = this.coverObjectUrl ? `u:${this.coverObjectUrl}` : "";
      return [
        String(this.title || "").trim(),
        String(this.bodyMd || ""),
        String(this.tagsText || "").trim(),
        String(this.coverUrl || "").trim(),
        coverPending,
        coverLocal
      ].join("\n");
    },
    bindLeaveWindowListeners() {
      if (this.leaveWindowListenersBound) return;
      this.leaveWindowListenersBound = true;
      window.addEventListener("beforeunload", this.onWindowBeforeUnload);
      window.addEventListener("popstate", this.onLeavePopstate);
    },
    unbindLeaveWindowListeners() {
      if (!this.leaveWindowListenersBound) return;
      this.leaveWindowListenersBound = false;
      window.removeEventListener("beforeunload", this.onWindowBeforeUnload);
      window.removeEventListener("popstate", this.onLeavePopstate);
    },
    promptLeave() {
      if (this.leavePromptOpen) {
        return new Promise(() => {});
      }
      this.leavePromptOpen = true;
      return ElMessageBox.confirm(LEAVE_MSG, "确定要离开吗？", {
        confirmButtonText: "确定",
        cancelButtonText: "取消",
        center: true,
        showClose: false,
        closeOnClickModal: false,
        customClass: "vp-leave-msgbox",
        distinguishCancelAndClose: true
      }).finally(() => {
        this.leavePromptOpen = false;
      });
    },
    syncLeaveBackTrap() {
      if (typeof window === "undefined" || !window.history) return;
      if (!this.leaveGuardEnabled || !this.shouldConfirmLeave) {
        this.clearLeaveBackTrapOnly();
        return;
      }
      if (this.leaveBackTrapInstalled) return;
      window.history.pushState({ __apLeaveTrap: true }, "", window.location.href);
      this.leaveBackTrapInstalled = true;
    },
    clearLeaveBackTrapOnly() {
      if (this.leaveBackTrapInstalled) this.leaveBackTrapInstalled = false;
    },
    removeLeaveBackTrapSilent() {
      if (typeof window === "undefined" || !window.history) return;
      if (!this.leaveBackTrapInstalled) return;
      this.leaveBackTrapInstalled = false;
      this.suppressLeavePopstateOnce = true;
      window.history.back();
    },
    onLeavePopstate() {
      if (this.suppressLeavePopstateOnce) {
        this.suppressLeavePopstateOnce = false;
        return;
      }
      if (!this.leaveGuardEnabled || !this.shouldConfirmLeave) return;
      if (this.leavePromptOpen) return;
      this.promptLeave()
        .then(() => {
          ElMessageBox.close();
          this.saveCommitted = true;
          this.leaveBackTrapInstalled = false;
          window.history.back();
        })
        .catch(() => {
          window.history.pushState({ __apLeaveTrap: true }, "", window.location.href);
          this.leaveBackTrapInstalled = true;
        });
    },
    onWindowBeforeUnload(e) {
      if (!this.shouldConfirmLeave) return;
      e.preventDefault();
      e.returnValue = "";
    },
    resetArticleForm() {
      this.articleLoadGen += 1;
      this.title = "";
      this.coverUrl = "";
      this.pendingCoverFile = null;
      this.revokeCoverObjectUrl();
      this.coverDropHover = false;
      this.tagsText = "";
      this.bodyMd = "";
      this.editArticleStatus = "";
      this.editFailReason = "";
      this.submitting = false;
      this.coverUploading = false;
      this.articleLoading = false;
    },
    async bootstrapArticlePage() {
      this.leaveGuardEnabled = false;
      this.saveCommitted = false;
      this.clearLeaveBackTrapOnly();
      this.resetArticleForm();
      const gen = this.articleLoadGen;
      if (this.isEdit) {
        await this.loadEdit();
      }
      if (gen !== this.articleLoadGen) return;
      this.baselineSnapshot = this.currentSnapshot();
      this.leaveGuardEnabled = true;
      this.$nextTick(() => this.syncLeaveBackTrap());
    },
    revokeCoverObjectUrl() {
      if (this.coverObjectUrl) {
        URL.revokeObjectURL(this.coverObjectUrl);
        this.coverObjectUrl = "";
      }
    },
    pickCover() {
      this.$refs.coverInput?.click();
    },
    applyCoverFile(file) {
      if (!file || !file.type.startsWith("image/")) {
        ElMessage.warning("请选择图片文件");
        return;
      }
      if (file.size > 10 * 1024 * 1024) {
        ElMessage.warning("封面不能超过 10MB");
        return;
      }
      this.revokeCoverObjectUrl();
      this.pendingCoverFile = file;
      this.coverObjectUrl = URL.createObjectURL(file);
      this.$nextTick(() => this.syncLeaveBackTrap());
    },
    onCoverFileChange(e) {
      const file = e.target.files?.[0];
      e.target.value = "";
      if (!file) return;
      this.applyCoverFile(file);
      if (this.isEdit && this.editId) {
        void this.uploadCoverNow(this.editId, file);
      }
    },
    onCoverDrop(e) {
      this.coverDropHover = false;
      const file = e.dataTransfer?.files?.[0];
      if (!file) return;
      this.applyCoverFile(file);
      if (this.isEdit && this.editId) {
        void this.uploadCoverNow(this.editId, file);
      }
    },
    async uploadCoverNow(articleId, file) {
      this.coverUploading = true;
      try {
        const res = await mbUpdateArticleCover(articleId, file);
        this.coverUrl = res.cover_url || "";
        this.pendingCoverFile = null;
        this.revokeCoverObjectUrl();
        ElMessage.success("封面上传成功");
      } catch (err) {
        ElMessage.error((err && err.message) || "封面上传失败");
      } finally {
        this.coverUploading = false;
      }
    },
    parseTags() {
      return this.tagsText
        .split(/[,，]/)
        .map(s => s.trim())
        .filter(Boolean)
        .slice(0, 10);
    },
    async loadEdit() {
      const id = this.editId;
      if (!id) return;
      const gen = this.articleLoadGen;
      this.articleLoading = true;
      try {
        const art = await mbGetMyArticle(id);
        if (gen !== this.articleLoadGen || id !== this.editId) return;
        this.title = art.title || "";
        this.coverUrl = art.cover_url || "";
        this.bodyMd = art.body_md || "";
        this.tagsText = (art.tags || []).join(", ");
        this.editArticleStatus = String(art.status || "").trim();
        this.editFailReason = String(art.fail_reason || "").trim();
      } catch (e) {
        if (gen !== this.articleLoadGen) return;
        ElMessage.error((e && e.message) || "加载失败");
        this.$router.replace({ name: "manuscript" });
      } finally {
        if (gen === this.articleLoadGen) {
          this.articleLoading = false;
        }
      }
    },
    async submit(publish) {
      const title = this.title.trim();
      const body_md = this.bodyMd.trim();
      if (publish) {
        if (!title || !body_md) {
          ElMessage.warning("发布前请填写标题和正文");
          return;
        }
      } else if (!title && !body_md) {
        ElMessage.warning("草稿请至少填写标题或正文");
        return;
      }
      this.submitting = true;
      try {
        const payload = {
          title,
          body_md,
          cover_url: this.pendingCoverFile ? "" : this.coverUrl.trim(),
          tags: this.parseTags(),
          publish
        };
        let id = this.editId;
        let resultStatus = "";
        if (this.isEdit) {
          const res = await mbPutMyArticle(this.editId, payload);
          resultStatus = String(res.status || "").trim();
          id = res.id || this.editId;
        } else {
          const res = await mbPostArticle(payload);
          id = res.id;
          resultStatus = String(res.status || "").trim();
        }
        if (this.pendingCoverFile && id) {
          try {
            const cov = await mbUpdateArticleCover(id, this.pendingCoverFile);
            this.coverUrl = cov.cover_url || "";
            this.pendingCoverFile = null;
            this.revokeCoverObjectUrl();
          } catch (coverErr) {
            ElMessage.warning(
              (coverErr && coverErr.message) ||
                "专栏已保存，但封面上传失败，可在编辑页重新上传"
            );
          }
        }
        const pendingReview = publish && resultStatus === "pending_review";
        ElMessage.success(
          publish
            ? pendingReview
              ? "专栏已提交审核"
              : "专栏已发布"
            : "草稿已保存"
        );
        this.saveCommitted = true;
        this.leaveGuardEnabled = false;
        this.clearLeaveBackTrapOnly();
        const readRoute = minibiliArticleReadRoute(id);
        if (pendingReview) {
          await this.$router.replace({
            name: "manuscript",
            query: { tab: "article", status: "processing" }
          });
          void showCreatorArticleReviewNotice();
        } else if (publish && readRoute) {
          await this.$router.push(readRoute);
        } else if (id) {
          await this.$router.replace({
            name: "manuscript",
            query: { tab: "article", articleSub: "draft" }
          });
        } else {
          await this.$router.replace({ name: "manuscript" });
        }
      } catch (e) {
        ElMessage.error((e && e.message) || "保存失败");
      } finally {
        this.submitting = false;
      }
    },
    saveDraft() {
      void this.submit(false);
    },
    publish() {
      void this.submit(true);
    }
  }
};
</script>

<style lang="scss" scoped>
$blue: #00a1d6;
$text: #18191c;
$line: #e3e5e7;

.ap-wrap {
  max-width: 1200px;
}
.ap-title {
  margin: 0 0 24px;
  font-size: 20px;
  font-weight: 600;
}
.ap-reject-hint {
  margin: -12px 0 20px;
  padding: 10px 12px;
  border-radius: 6px;
  background: #fff1f0;
  border: 1px solid #ffccc7;
  color: #cf1322;
  font-size: 13px;
  line-height: 1.5;
}
.ap-field {
  margin-bottom: 16px;
  &--cover {
    max-width: 420px;
  }
  &--grow {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
}
.ap-label {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  color: #61666d;
}
.ap-input {
  width: 100%;
  box-sizing: border-box;
  padding: 8px 12px;
  border: 1px solid $line;
  border-radius: 6px;
  font-size: 14px;
}
.ap-cover-input {
  position: absolute;
  width: 0;
  height: 0;
  opacity: 0;
  pointer-events: none;
}
.ap-cover-drop {
  position: relative;
  min-height: 160px;
  border: 1px dashed $line;
  border-radius: 8px;
  background: #fafafa;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  overflow: hidden;
  transition: border-color 0.15s;
  &--hover {
    border-color: $blue;
    background: #f0f9fc;
  }
}
.ap-cover-preview {
  width: 100%;
  max-height: 220px;
  object-fit: cover;
  display: block;
}
.ap-cover-hint {
  margin: 0 0 6px;
  font-size: 14px;
  color: $text;
}
.ap-cover-sub {
  margin: 0;
  font-size: 12px;
  color: #9499a0;
}
.ap-cover-change {
  position: absolute;
  right: 10px;
  bottom: 10px;
  padding: 6px 12px;
  border: none;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 12px;
  cursor: pointer;
  &:hover {
    background: rgba(0, 0, 0, 0.7);
  }
}
.ap-editor-row {
  display: flex;
  gap: 20px;
  align-items: stretch;
}
.ap-textarea {
  flex: 1;
  min-height: 420px;
  width: 100%;
  box-sizing: border-box;
  padding: 12px;
  border: 1px solid $line;
  border-radius: 6px;
  font-size: 14px;
  line-height: 1.6;
  font-family: Consolas, Monaco, "Courier New", monospace;
  resize: vertical;
}
.ap-preview {
  flex: 1;
  min-height: 420px;
  padding: 12px 14px;
  border: 1px solid $line;
  border-radius: 6px;
  background: #fafafa;
  overflow-y: auto;
  font-size: 14px;
  line-height: 1.75;
  :deep(img) {
    max-width: 100%;
  }
  :deep(pre) {
    background: #f1f2f3;
    padding: 10px;
    border-radius: 4px;
    overflow-x: auto;
  }
}
.ap-actions {
  display: flex;
  gap: 12px;
  margin-top: 20px;
}
.ap-btn {
  padding: 10px 28px;
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
  border: 1px solid $line;
  background: #fff;
  &--primary {
    background: $blue;
    border-color: $blue;
    color: #fff;
  }
  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
}
</style>
