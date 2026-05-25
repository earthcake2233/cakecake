<template>
  <div class="mb-page bili-wrapper">
    <div v-if="!token" class="mb-card">
      <p>
        请先
        <a href="#" class="mb-card__link" @click.prevent="openLoginModal">登录</a>
        后再上传视频。
      </p>
    </div>
    <div v-else class="mb-up">
      <VideoUploadMaintenanceNotice />
      <h1 class="mb-up__title">上传视频</h1>
      <p class="mb-up__tip">
        将提交至转码队列；需后端 FFmpeg、RabbitMQ、OSS 等已配置。封面为可选项。
      </p>
      <el-form label-width="88px" class="mb-up__form">
        <el-form-item label="标题" required>
          <el-input v-model="title" maxlength="80" show-word-limit placeholder="1–80 字" />
        </el-form-item>
        <el-form-item label="简介">
          <el-input
            v-model="description"
            type="textarea"
            :rows="4"
            maxlength="2000"
            show-word-limit
            placeholder="可选，最多 2000 字"
          />
        </el-form-item>
        <el-form-item label="视频文件" required>
          <input ref="fileRef" type="file" accept="video/*" @change="onVideo" />
        </el-form-item>
        <el-form-item label="封面">
          <input ref="coverRef" type="file" accept="image/*" @change="onCover" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="uploading" :disabled="!canSubmit" @click="submit">
            提交上传
          </el-button>
        </el-form-item>
      </el-form>
      <pre v-if="result" class="mb-up__json">{{ result }}</pre>
    </div>
  </div>
</template>

<script>
import { ElMessage } from "element-plus";
import { mbUploadVideo } from "@/api/minibili";
import { getAccessToken } from "@/utils/authTokens";
import { openMinibiliLoginModal } from "@/utils/minibiliLoginModal";
import VideoUploadMaintenanceNotice from "@/components/creator/VideoUploadMaintenanceNotice.vue";
import {
  guardVideoFileUploadDisabled,
  isVideoUploadDisabled
} from "@/utils/videoUploadPolicy";

export default {
  name: "MinibiliUpload",
  components: { VideoUploadMaintenanceNotice },
  data() {
    return {
      title: "",
      description: "",
      videoFile: null,
      coverFile: null,
      uploading: false,
      result: ""
    };
  },
  computed: {
    token() {
      return getAccessToken();
    },
    canSubmit() {
      return (
        !this.videoUploadDisabled &&
        this.title.trim().length > 0 &&
        this.videoFile != null &&
        !this.uploading
      );
    },
    videoUploadDisabled() {
      return isVideoUploadDisabled();
    }
  },
  methods: {
    openLoginModal() {
      openMinibiliLoginModal({ tab: 0, redirect: "/minibili/upload" });
    },
    onVideo(e) {
      const f = e.target.files && e.target.files[0];
      this.videoFile = f || null;
    },
    onCover(e) {
      const f = e.target.files && e.target.files[0];
      this.coverFile = f || null;
    },
    async submit() {
      if (
        guardVideoFileUploadDisabled(msg => {
          ElMessage.warning({ message: msg, duration: 6000 });
        })
      ) {
        return;
      }
      if (!this.canSubmit) return;
      const fd = new FormData();
      fd.append("title", this.title.trim());
      fd.append("description", (this.description || "").trim());
      fd.append("file", this.videoFile);
      if (this.coverFile) {
        fd.append("cover", this.coverFile);
      }
      this.uploading = true;
      this.result = "";
      try {
        const data = await mbUploadVideo(fd);
        this.result = JSON.stringify(data, null, 2);
      } catch (e) {
        ElMessage.error((e && e.message) || "上传失败");
      } finally {
        this.uploading = false;
      }
    }
  }
};
</script>

<style scoped lang="scss">
.mb-page {
  padding: 32px 16px 64px;
  min-height: 60vh;
}
.mb-card {
  max-width: 560px;
  margin: 0 auto;
  padding: 24px;
  background: #fff;
  border: 1px solid #e3e5e7;
  border-radius: 4px;
}
.mb-up {
  max-width: 640px;
  margin: 0 auto;
}
.mb-up__title {
  font-size: 22px;
  margin: 0 0 8px;
}
.mb-up__tip {
  font-size: 13px;
  color: #61666d;
  margin: 0 0 20px;
}
.mb-up__json {
  margin-top: 20px;
  padding: 12px;
  background: #f6f7f8;
  font-size: 12px;
  overflow: auto;
  border-radius: 4px;
}

.mb-card__link {
  margin-left: 0;
  font-size: 14px;
  color: #00aeec;
  text-decoration: none;
  cursor: pointer;
  &:hover {
    color: #00b5e5;
  }
}
</style>
