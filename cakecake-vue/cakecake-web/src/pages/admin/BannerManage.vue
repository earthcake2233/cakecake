<template>
  <div class="adm-panel">
    <div class="adm-panel__head">
      <h2>首页轮播</h2>
      <el-button type="primary" @click="openCreate">新增轮播</el-button>
    </div>
    <el-table v-loading="loading" :data="rows" border stripe>
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column label="封面" width="120">
        <template #default="{ row }">
          <img v-if="row.image_url" :src="row.image_url" class="adm-thumb" alt="" />
        </template>
      </el-table-column>
      <el-table-column prop="title" label="标题" min-width="140" />
      <el-table-column label="跳转" min-width="120">
        <template #default="{ row }">
          {{ row.link_type }} / {{ row.link_target || "—" }}
        </template>
      </el-table-column>
      <el-table-column prop="sort_order" label="排序" width="70" />
      <el-table-column label="启用" width="70">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'">{{
            row.enabled ? "是" : "否"
          }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="140" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog
      v-model="dialogVisible"
      :title="editingId ? '编辑轮播' : '新增轮播'"
      width="520px"
      destroy-on-close
    >
      <el-form label-width="88px">
        <el-form-item label="标题" required>
          <el-input v-model="form.title" maxlength="120" />
        </el-form-item>
        <el-form-item label="轮播图" required>
          <div class="adm-upload">
            <div v-if="form.image_url" class="adm-upload__preview">
              <img :src="form.image_url" alt="预览" />
            </div>
            <div v-else class="adm-upload__placeholder">支持 jpg / png / webp 等，最大 10MB</div>
            <div class="adm-upload__actions">
              <input
                ref="imageFileRef"
                type="file"
                accept="image/jpeg,image/png,image/gif,image/webp,image/bmp"
                class="adm-upload__input"
                @change="onPickImage"
              />
              <el-button :loading="imageUploading" @click="triggerImagePick">
                {{ form.image_url ? "更换图片" : "上传图片" }}
              </el-button>
              <el-button
                v-if="form.image_url"
                link
                type="danger"
                :disabled="imageUploading"
                @click="clearImage"
              >
                清除
              </el-button>
            </div>
            <el-input
              v-model="form.image_url"
              class="adm-upload__url"
              placeholder="或粘贴已有图片 URL"
            />
          </div>
        </el-form-item>
        <el-form-item label="跳转类型">
          <el-select v-model="form.link_type" style="width: 100%">
            <el-option label="无" value="none" />
            <el-option label="站内视频" value="video" />
            <el-option label="外链" value="url" />
          </el-select>
        </el-form-item>
        <el-form-item label="跳转目标">
          <el-input
            v-model="form.link_target"
            :placeholder="
              form.link_type === 'video'
                ? '视频数字 ID，如 7'
                : form.link_type === 'url'
                  ? 'https://...'
                  : '留空'
            "
          />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort_order" :min="0" :max="9999" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="onSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import {
  adminCreateBanner,
  adminDeleteBanner,
  adminListBanners,
  adminUpdateBanner,
  adminUploadBannerImage
} from "@/api/admin";
import { ElMessage, ElMessageBox } from "element-plus";

const MAX_BANNER_IMAGE_BYTES = 10 * 1024 * 1024;

export default {
  data() {
    return {
      loading: false,
      saving: false,
      imageUploading: false,
      rows: [],
      dialogVisible: false,
      editingId: null,
      form: this.emptyForm()
    };
  },
  created() {
    this.load();
  },
  methods: {
    emptyForm() {
      return {
        title: "",
        image_url: "",
        link_type: "video",
        link_target: "",
        sort_order: 0,
        enabled: true
      };
    },
    async load() {
      this.loading = true;
      try {
        const body = await adminListBanners();
        this.rows = (body.data && body.data.items) || [];
      } finally {
        this.loading = false;
      }
    },
    triggerImagePick() {
      const el = this.$refs.imageFileRef;
      if (el) el.click();
    },
    clearImage() {
      this.form.image_url = "";
    },
    async onPickImage(ev) {
      const input = ev.target;
      const file = input && input.files && input.files[0];
      if (input) input.value = "";
      if (!file) return;
      if (file.size > MAX_BANNER_IMAGE_BYTES) {
        ElMessage.warning("图片不能超过 10MB");
        return;
      }
      this.imageUploading = true;
      try {
        const body = await adminUploadBannerImage(file, this.editingId || undefined);
        this.form.image_url = (body.data && body.data.image_url) || "";
        if (!this.form.image_url) {
          ElMessage.error("上传失败");
          return;
        }
        ElMessage.success("图片已上传");
      } catch (e) {
        ElMessage.error((e && e.message) || "上传失败");
      } finally {
        this.imageUploading = false;
      }
    },
    openCreate() {
      this.editingId = null;
      this.form = this.emptyForm();
      this.dialogVisible = true;
    },
    openEdit(row) {
      this.editingId = row.id;
      this.form = {
        title: row.title,
        image_url: row.image_url,
        link_type: row.link_type || "none",
        link_target: row.link_target || "",
        sort_order: row.sort_order || 0,
        enabled: !!row.enabled
      };
      this.dialogVisible = true;
    },
    async onSave() {
      if (!String(this.form.title).trim() || !String(this.form.image_url).trim()) {
        ElMessage.warning("请填写标题并上传轮播图");
        return;
      }
      this.saving = true;
      try {
        const payload = { ...this.form };
        if (this.editingId) {
          await adminUpdateBanner(this.editingId, payload);
          ElMessage.success("已更新");
        } else {
          await adminCreateBanner(payload);
          ElMessage.success("已创建");
        }
        this.dialogVisible = false;
        await this.load();
      } finally {
        this.saving = false;
      }
    },
    async onDelete(row) {
      await ElMessageBox.confirm(`确定删除「${row.title}」？`, "确认");
      await adminDeleteBanner(row.id);
      ElMessage.success("已删除");
      await this.load();
    }
  }
};
</script>

<style lang="scss" scoped>
@import "@/style/mixin";

.adm-panel {
  background: $white;
  border: 1px solid #e3e5e7;
  border-radius: 8px;
  padding: 20px;
}
.adm-panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  h2 {
    margin: 0;
    @include sc(18px, #18191c);
  }
}
.adm-thumb {
  width: 96px;
  height: 54px;
  object-fit: cover;
  border-radius: 4px;
}
.adm-upload__preview {
  margin-bottom: 10px;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid #e3e5e7;
  img {
    display: block;
    width: 100%;
    max-height: 160px;
    object-fit: cover;
  }
}
.adm-upload__placeholder {
  margin-bottom: 10px;
  padding: 24px;
  text-align: center;
  @include sc(12px, #9499a0);
  background: #f6f7f8;
  border-radius: 6px;
  border: 1px dashed #e3e5e7;
}
.adm-upload__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.adm-upload__input {
  display: none;
}
.adm-upload__url {
  margin-top: 4px;
}
</style>
