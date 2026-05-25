<template>
  <div class="ai-hub" v-loading="loading">
    <header class="ai-hub__hero">
      <div class="ai-hub__hero-inner">
        <div class="ai-hub__hero-text">
          <p class="ai-hub__eyebrow">Message Center · Agent Studio</p>
          <h1 class="ai-hub__title">AI 角色中心</h1>
          <p class="ai-hub__desc">
            配置多个 AI 助手角色，每个角色在消息中心独立会话；新建会话时从欢迎语库中随机抽取一句。
          </p>
        </div>
        <div class="ai-hub__stats">
          <div class="ai-hub__stat">
            <strong>{{ enabledCount }}</strong>
            <span>已启用</span>
          </div>
          <div class="ai-hub__stat">
            <strong>{{ profiles.length }}</strong>
            <span>总角色</span>
          </div>
          <div
            class="ai-hub__stat"
            :class="{ 'ai-hub__stat--warn': !deepseekConfigured }"
          >
            <strong>{{ deepseekConfigured ? "OK" : "—" }}</strong>
            <span>DeepSeek</span>
          </div>
        </div>
      </div>
    </header>

    <el-alert
      v-if="!deepseekConfigured"
      type="warning"
      :closable="false"
      show-icon
      class="ai-hub__alert"
      title="未配置 DEEPSEEK_API_KEY，用户发消息后将收到未配置提示。"
    />

    <div class="ai-hub__toolbar">
      <span class="ai-hub__toolbar-hint">最多 {{ maxProfiles }} 个角色</span>
      <el-button
        type="primary"
        :disabled="profiles.length >= maxProfiles"
        @click="openCreate"
      >
        + 新建角色
      </el-button>
    </div>

    <div v-if="profiles.length" class="ai-hub__grid">
      <article
        v-for="p in profiles"
        :key="p.id"
        class="ai-card"
        :class="{
          'ai-card--off': !p.enabled,
          'ai-card--active': drawerVisible && editingId === p.id
        }"
        @click="openEdit(p)"
      >
        <div class="ai-card__avatar-wrap">
          <img
            class="ai-card__avatar"
            :src="p.avatar_url || defaultAvatar"
            alt=""
          />
          <span v-if="p.enabled" class="ai-card__badge ai-card__badge--on">启用</span>
          <span v-else class="ai-card__badge">停用</span>
        </div>
        <h3 class="ai-card__name">{{ p.display_name }}</h3>
        <p class="ai-card__slug">@{{ p.slug }}</p>
        <p class="ai-card__sign">{{ p.sign || "暂无签名" }}</p>
        <div class="ai-card__meta">
          <span>{{ (p.welcome_messages || []).length }} 条欢迎语</span>
          <span>排序 {{ p.sort_order }}</span>
        </div>
      </article>
    </div>
    <el-empty v-else description="还没有 AI 角色，点击右上角新建" />

    <section v-if="drawerVisible" ref="editPanelRef" class="ai-edit-panel">
      <header class="ai-edit-panel__head">
        <div>
          <h2 class="ai-edit-panel__title">{{ drawerTitle }}</h2>
          <p class="ai-edit-panel__hint">修改后请点击底部「{{ saveButtonLabel }}」提交</p>
        </div>
        <el-button link type="primary" @click="closeEdit">关闭</el-button>
      </header>

      <el-form label-position="top" class="ai-edit-panel__form">
        <section class="ai-drawer__section">
          <h4>基础信息</h4>
          <el-form-item label="角色标识 (slug)" required>
            <el-input
              v-model="form.slug"
              maxlength="32"
              placeholder="小写英文，如 helper、cs_bot"
            />
            <p v-if="editingId" class="ai-slug-hint">
              修改后将同步更新卡片上的 @标识；已有会话不受影响
            </p>
          </el-form-item>
          <el-form-item label="显示名称" required>
            <el-input v-model="form.display_name" maxlength="64" show-word-limit />
          </el-form-item>
          <el-form-item label="头像">
            <div class="ai-avatar-field">
              <img
                v-if="avatarPreviewUrl"
                class="ai-avatar-field__img"
                :src="avatarPreviewUrl"
                alt=""
              />
              <div v-else class="ai-avatar-field__empty">未设置</div>
              <div class="ai-avatar-field__actions">
                <input
                  ref="avatarFileRef"
                  type="file"
                  accept="image/jpeg,image/png,image/gif,image/webp,image/bmp"
                  class="adm-upload__input"
                  @change="onPickAvatar"
                />
                <div class="ai-avatar-field__btns">
                  <el-button size="small" @click="triggerAvatarPick">
                    选择并裁剪
                  </el-button>
                  <el-button
                    v-if="avatarPreviewUrl || pendingAvatarFile"
                    size="small"
                    link
                    type="danger"
                    @click="clearAvatar"
                  >
                    清除头像
                  </el-button>
                </div>
                <p v-if="pendingAvatarFile" class="ai-avatar-field__pending">
                  已选新头像，点击下方「{{ saveButtonLabel }}」后将上传并删除云端旧文件
                </p>
                <el-input
                  v-model="form.avatar_url"
                  size="small"
                  placeholder="或粘贴图片 URL（保存时生效）"
                  @input="onAvatarUrlInput"
                />
              </div>
            </div>
          </el-form-item>
          <el-form-item label="个性签名">
            <el-input v-model="form.sign" maxlength="500" show-word-limit />
          </el-form-item>
          <el-form-item label="排序">
            <el-input-number v-model="form.sort_order" :min="0" :max="999" />
          </el-form-item>
          <el-form-item label="启用">
            <el-switch v-model="form.enabled" />
          </el-form-item>
        </section>

        <section class="ai-drawer__section">
          <h4>角色人设</h4>
          <el-form-item required>
            <el-input
              v-model="form.system_prompt"
              type="textarea"
              :rows="8"
              maxlength="12000"
              show-word-limit
              placeholder="DeepSeek system 提示词"
            />
          </el-form-item>
        </section>

        <section class="ai-drawer__section">
          <div class="ai-welcome-head">
            <h4>欢迎语库</h4>
            <el-button link type="primary" @click="addWelcomeLine">+ 添加一句</el-button>
          </div>
          <p class="ai-welcome-tip">新建会话时随机抽取其中一句作为首条消息</p>
          <div
            v-for="(line, idx) in form.welcome_messages"
            :key="'w' + idx"
            class="ai-welcome-row"
          >
            <el-input
              v-model="form.welcome_messages[idx]"
              type="textarea"
              :rows="2"
              maxlength="500"
              :placeholder="'欢迎语 ' + (idx + 1)"
            />
            <el-button
              link
              type="danger"
              :disabled="form.welcome_messages.length <= 1"
              @click="removeWelcomeLine(idx)"
            >
              删除
            </el-button>
          </div>
        </section>
      </el-form>

      <footer class="ai-edit-panel__footer">
        <el-button
          v-if="editingId"
          type="danger"
          plain
          :loading="disabling"
          @click="onDisable"
        >
          停用角色
        </el-button>
        <div class="ai-edit-panel__footer-right">
          <el-button @click="closeEdit">取消</el-button>
          <el-button type="primary" :loading="saving" @click="onSave">
            {{ saveButtonLabel }}
          </el-button>
        </div>
      </footer>
    </section>

    <MbAvatarCropDialog
      :visible="avatarCropVisible"
      :src="avatarCropSrc"
      :file-name="avatarCropFileName"
      :circular="true"
      @cancel="onAvatarCropCancel"
      @confirm="onAvatarCropConfirm"
    />
  </div>
</template>

<script>
import {
  adminCreateAgentProfile,
  adminDeleteAgentProfile,
  adminListAgentProfiles,
  adminUpdateAgentProfile,
  adminUploadAgentProfileAvatar
} from "@/api/admin";
import MbAvatarCropDialog from "@/components/minibili/MbAvatarCropDialog.vue";
import defaultAvatar from "@/assets/akari.jpg";
import { ElMessage, ElMessageBox } from "element-plus";

const MAX_AVATAR_BYTES = 10 * 1024 * 1024;

export default {
  components: { MbAvatarCropDialog },
  data() {
    return {
      defaultAvatar,
      loading: false,
      saving: false,
      disabling: false,
      profiles: [],
      maxProfiles: 12,
      deepseekConfigured: true,
      drawerVisible: false,
      editingId: null,
      savedAvatarUrl: "",
      pendingAvatarFile: null,
      pendingAvatarPreviewUrl: null,
      avatarCropVisible: false,
      avatarCropSrc: "",
      avatarCropFileName: "agent-avatar.jpg",
      _avatarCropObjectUrl: null,
      form: this.emptyForm()
    };
  },
  computed: {
    enabledCount() {
      return (this.profiles || []).filter(p => p.enabled).length;
    },
    drawerTitle() {
      return this.editingId ? "编辑 AI 角色" : "新建 AI 角色";
    },
    avatarPreviewUrl() {
      if (this.pendingAvatarPreviewUrl) return this.pendingAvatarPreviewUrl;
      return String(this.form.avatar_url || "").trim();
    },
    saveButtonLabel() {
      if (this.editingId) {
        return this.pendingAvatarFile ? "保存并上传" : "保存更改";
      }
      return this.pendingAvatarFile ? "创建并上传" : "创建角色";
    }
  },
  beforeUnmount() {
    this.closeAvatarCrop();
    this.clearPendingAvatar();
  },
  created() {
    this.load();
  },
  methods: {
    closeEdit() {
      this.drawerVisible = false;
      this.editingId = null;
      this.clearPendingAvatar();
    },
    emptyForm() {
      return {
        slug: "",
        display_name: "",
        avatar_url: "",
        sign: "",
        system_prompt: "",
        welcome_messages: ["你好，我是你的 AI 助手，有什么可以帮你的？"],
        sort_order: 0,
        enabled: true
      };
    },
    async load() {
      this.loading = true;
      try {
        const body = await adminListAgentProfiles();
        const d = (body && body.data) || {};
        this.profiles = d.items || [];
        this.maxProfiles = d.max_profiles || 12;
        this.deepseekConfigured = d.deepseek_configured !== false;
      } catch (e) {
        ElMessage.error((e && e.message) || "加载失败");
      } finally {
        this.loading = false;
      }
    },
    openCreate() {
      this.editingId = null;
      this.savedAvatarUrl = "";
      this.clearPendingAvatar();
      this.form = this.emptyForm();
      this.form.sort_order = this.profiles.length;
      this.drawerVisible = true;
      this.$nextTick(() => this.scrollToEditPanel());
    },
    openEdit(p) {
      this.editingId = p.id;
      this.savedAvatarUrl = p.avatar_url || "";
      this.clearPendingAvatar();
      this.form = {
        slug: p.slug,
        display_name: p.display_name || "",
        avatar_url: p.avatar_url || "",
        sign: p.sign || "",
        system_prompt: p.system_prompt || "",
        welcome_messages: (p.welcome_messages && p.welcome_messages.length
          ? [...p.welcome_messages]
          : ["你好！"]),
        sort_order: p.sort_order || 0,
        enabled: p.enabled !== false
      };
      this.drawerVisible = true;
      this.$nextTick(() => this.scrollToEditPanel());
    },
    scrollToEditPanel() {
      const el = this.$refs.editPanelRef;
      if (el && el.scrollIntoView) {
        el.scrollIntoView({ behavior: "smooth", block: "start" });
      }
    },
    addWelcomeLine() {
      this.form.welcome_messages.push("");
    },
    removeWelcomeLine(idx) {
      if (this.form.welcome_messages.length <= 1) return;
      this.form.welcome_messages.splice(idx, 1);
    },
    buildPayload() {
      const welcome = (this.form.welcome_messages || [])
        .map(s => String(s).trim())
        .filter(Boolean);
      return {
        slug: String(this.form.slug || "").trim(),
        display_name: String(this.form.display_name || "").trim(),
        avatar_url: String(this.form.avatar_url || "").trim(),
        sign: String(this.form.sign || "").trim(),
        system_prompt: String(this.form.system_prompt || "").trim(),
        welcome_messages: welcome,
        sort_order: Number(this.form.sort_order) || 0,
        enabled: !!this.form.enabled
      };
    },
    validateForm() {
      const p = this.buildPayload();
      if (!/^[a-z][a-z0-9_]{1,30}$/.test(p.slug)) {
        ElMessage.warning("slug 需为小写字母开头，仅含 a-z、0-9、_");
        return false;
      }
      if (!p.display_name) {
        ElMessage.warning("请填写显示名称");
        return false;
      }
      if (p.system_prompt.length < 10) {
        ElMessage.warning("角色人设至少 10 个字");
        return false;
      }
      if (!p.welcome_messages.length) {
        ElMessage.warning("至少保留一条欢迎语");
        return false;
      }
      return true;
    },
    async onSave() {
      if (!this.validateForm()) return;
      this.saving = true;
      try {
        const payload = this.buildPayload();
        const hadPendingAvatar = !!this.pendingAvatarFile;
        if (this.editingId) {
          if (this.pendingAvatarFile) {
            const body = await adminUploadAgentProfileAvatar(
              this.editingId,
              this.pendingAvatarFile
            );
            payload.avatar_url =
              (body.data && body.data.avatar_url) || payload.avatar_url;
            this.clearPendingAvatar();
          }
          await adminUpdateAgentProfile(this.editingId, payload);
          ElMessage.success(
            hadPendingAvatar ? "已保存，旧头像已从云端删除" : "已更新"
          );
        } else {
          const body = await adminCreateAgentProfile(payload);
          const created = body.data || {};
          this.editingId = created.id;
          if (this.pendingAvatarFile) {
            await adminUploadAgentProfileAvatar(
              this.editingId,
              this.pendingAvatarFile
            );
            this.clearPendingAvatar();
          }
          ElMessage.success(hadPendingAvatar ? "角色已创建并上传头像" : "角色已创建");
        }
        this.closeEdit();
        await this.load();
      } catch (e) {
        ElMessage.error((e && e.message) || "保存失败");
      } finally {
        this.saving = false;
      }
    },
    async onDisable() {
      if (!this.editingId) return;
      try {
        await ElMessageBox.confirm("停用后用户将无法与该角色新建对话，已有会话保留。", "停用角色");
      } catch {
        return;
      }
      this.disabling = true;
      try {
        await adminDeleteAgentProfile(this.editingId);
        ElMessage.success("已停用");
        this.closeEdit();
        await this.load();
      } catch (e) {
        ElMessage.error((e && e.message) || "操作失败");
      } finally {
        this.disabling = false;
      }
    },
    triggerAvatarPick() {
      const el = this.$refs.avatarFileRef;
      if (el) el.click();
    },
    clearPendingAvatar() {
      if (this.pendingAvatarPreviewUrl) {
        URL.revokeObjectURL(this.pendingAvatarPreviewUrl);
      }
      this.pendingAvatarPreviewUrl = null;
      this.pendingAvatarFile = null;
    },
    clearAvatar() {
      this.clearPendingAvatar();
      this.form.avatar_url = "";
    },
    onAvatarUrlInput() {
      if (this.pendingAvatarFile) {
        this.clearPendingAvatar();
      }
    },
    closeAvatarCrop() {
      if (this._avatarCropObjectUrl) {
        URL.revokeObjectURL(this._avatarCropObjectUrl);
        this._avatarCropObjectUrl = null;
      }
      this.avatarCropVisible = false;
      this.avatarCropSrc = "";
    },
    onAvatarCropCancel() {
      this.closeAvatarCrop();
    },
    onAvatarCropConfirm(file) {
      this.closeAvatarCrop();
      if (!file) return;
      this.clearPendingAvatar();
      this.pendingAvatarFile = file;
      this.pendingAvatarPreviewUrl = URL.createObjectURL(file);
    },
    onPickAvatar(ev) {
      const input = ev.target;
      const file = input && input.files && input.files[0];
      if (input) input.value = "";
      if (!file) return;
      if (file.size > MAX_AVATAR_BYTES) {
        ElMessage.warning("图片不能超过 10MB");
        return;
      }
      const name = String(file.name || "");
      const dot = name.lastIndexOf(".");
      const ext = dot >= 0 ? name.slice(dot).toLowerCase() : "";
      const okExt = new Set([".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp"]);
      if (!okExt.has(ext)) {
        ElMessage.warning("仅支持 jpg、png、gif、bmp、webp");
        return;
      }
      this.closeAvatarCrop();
      const url = URL.createObjectURL(file);
      this._avatarCropObjectUrl = url;
      this.avatarCropSrc = url;
      this.avatarCropFileName = name || "agent-avatar.jpg";
      this.avatarCropVisible = true;
    }
  }
};
</script>

<style lang="scss" scoped>
@import "@/style/mixin";

.ai-hub {
  min-height: 480px;
}
.ai-hub__hero {
  margin: 0 0 20px;
  padding: 24px 24px 20px;
  border-radius: 8px;
  background: $white;
  border: 1px solid #e3e5e7;
  color: #18191c;
  box-shadow: none;
}
.ai-hub__hero-inner {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  flex-wrap: wrap;
}
.ai-hub__eyebrow {
  margin: 0 0 8px;
  font-size: 12px;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: #9499a0;
}
.ai-hub__title {
  margin: 0 0 10px;
  font-size: 22px;
  font-weight: 600;
  color: #18191c;
}
.ai-hub__desc {
  margin: 0;
  max-width: 520px;
  font-size: 14px;
  line-height: 1.6;
  color: #61666d;
}
.ai-hub__stats {
  display: flex;
  gap: 10px;
}
.ai-hub__stat {
  min-width: 72px;
  padding: 10px 14px;
  border-radius: 6px;
  background: #f6f7f8;
  border: 1px solid #e3e5e7;
  text-align: center;
  strong {
    display: block;
    font-size: 18px;
    font-weight: 600;
    color: $blue;
  }
  span {
    font-size: 12px;
    color: #9499a0;
  }
  &--warn strong {
    color: #e6a23c;
  }
}
.ai-hub__alert {
  margin-bottom: 16px;
}
.ai-hub__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
.ai-hub__toolbar-hint {
  font-size: 13px;
  color: #9499a0;
}
.ai-hub__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 16px;
}
.ai-card {
  padding: 20px 18px 16px;
  border-radius: 12px;
  background: #fff;
  border: 1px solid #e8eaed;
  cursor: pointer;
  transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s;
  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 16px rgba(0, 0, 0, 0.06);
    border-color: #00a1d6;
  }
  &--off {
    opacity: 0.72;
    background: #fafafa;
  }
  &--active {
    border-color: $blue;
    box-shadow: 0 0 0 2px rgba(0, 174, 236, 0.15);
  }
}
.ai-card__avatar-wrap {
  position: relative;
  width: 64px;
  height: 64px;
  margin-bottom: 12px;
}
.ai-card__avatar {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid #e3e5e7;
}
.ai-card__badge {
  position: absolute;
  right: -6px;
  bottom: -2px;
  padding: 2px 7px;
  border-radius: 999px;
  font-size: 10px;
  background: #e5e7eb;
  color: #6b7280;
  &--on {
    background: #dcfce7;
    color: #15803d;
  }
}
.ai-card__name {
  margin: 0 0 4px;
  font-size: 16px;
  font-weight: 600;
  color: #18191c;
}
.ai-card__slug {
  margin: 0 0 8px;
  font-size: 12px;
  color: $blue;
}
.ai-card__sign {
  margin: 0 0 12px;
  font-size: 13px;
  color: #61666d;
  line-height: 1.45;
  min-height: 38px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.ai-card__meta {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  color: #9499a0;
}
.ai-edit-panel {
  margin-top: 20px;
  background: $white;
  border: 1px solid #e3e5e7;
  border-radius: 8px;
  overflow: hidden;
}
.ai-edit-panel__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 20px 24px 0;
}
.ai-edit-panel__title {
  margin: 0 0 6px;
  font-size: 18px;
  font-weight: 600;
  color: #18191c;
}
.ai-edit-panel__hint {
  margin: 0;
  font-size: 13px;
  color: #9499a0;
}
.ai-edit-panel__form {
  padding: 16px 24px 8px;
}
.ai-edit-panel__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 24px 20px;
  border-top: 1px solid #e3e5e7;
  background: #fafbfc;
  position: sticky;
  bottom: 0;
  z-index: 2;
}
.ai-edit-panel__footer-right {
  display: flex;
  gap: 8px;
  margin-left: auto;
}
.ai-slug-hint {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.5;
  color: #9499a0;
}
.ai-drawer__section {
  margin-bottom: 24px;
  h4 {
    margin: 0 0 12px;
    font-size: 14px;
    font-weight: 600;
    color: #18191c;
  }
}
.ai-welcome-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
  h4 {
    margin: 0;
  }
}
.ai-welcome-tip {
  margin: 0 0 12px;
  font-size: 12px;
  color: #9499a0;
}
.ai-welcome-row {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  margin-bottom: 10px;
  .el-input {
    flex: 1;
  }
}
.ai-avatar-field {
  display: flex;
  gap: 14px;
  align-items: flex-start;
  width: 100%;
}
.ai-avatar-field__img {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
  border: 2px solid #e5e7eb;
}
.ai-avatar-field__empty {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  background: #f3f4f6;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: #9ca3af;
  flex-shrink: 0;
}
.ai-avatar-field__actions {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.ai-avatar-field__btns {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}
.ai-avatar-field__pending {
  margin: 0;
  font-size: 12px;
  line-height: 1.5;
  color: #fb7299;
}
.adm-upload__input {
  display: none;
}
</style>
