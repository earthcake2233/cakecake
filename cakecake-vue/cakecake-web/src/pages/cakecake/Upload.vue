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
        提交后将自动处理并发布，请耐心等待；封面为可选项。
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
import {
  mbCreateUploadTicket,
  mbCreateVideoDirect
} from "@/api/cakecake";
import { getAccessToken } from "@/utils/authTokens";
import { openCakecakeLoginModal } from "@/utils/cakecakeLoginModal";
import { uploadToPresignedURL } from "@/utils/directVideoUpload";
import VideoUploadMaintenanceNotice from "@/components/creator/VideoUploadMaintenanceNotice.vue";
import {
  guardVideoFileUploadDisabled,
  isVideoUploadDisabled
} from "@/utils/videoUploadPolicy";

export default {
  name: "CakecakeUpload",
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
      openCakecakeLoginModal({ tab: 0, redirect: "/cakecake/upload" });
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
      const title = this.title.trim();
      const description = (this.description || "").trim();
      this.uploading = true;
      this.result = "";
      try {
        let data;
        try {
          // 首选客户端直传：浏览器把文件 PUT 到 OSS，再提交元数据。
          const ticket = await mbCreateUploadTicket(
            this.videoFile.name,
            this.coverFile ? this.coverFile.name : "",
            this.videoFile.type || "",
            this.coverFile ? this.coverFile.type || "" : ""
          );
          // 视频与封面并行直传，缩短整体上传耗时。
          const jobs = [
            uploadToPresignedURL(
              ticket.raw_upload_url,
              this.videoFile,
              undefined,
              this.videoFile.type || ""
            ).then(() => ticket.raw_key)
          ];
          if (this.coverFile && ticket.cover_upload_url) {
            jobs.push(
              uploadToPresignedURL(
                ticket.cover_upload_url,
                this.coverFile,
                undefined,
                this.coverFile.type || ""
              ).then(() => ticket.cover_key || "")
            );
          }
          const [rawKey, coverKey] = await Promise.all(jobs);
          data = await mbCreateVideoDirect({
            title,
            description,
            tags: [],
            zone: "",
            raw_key: rawKey,
            cover_key: coverKey || undefined
          });
        } catch (directErr) {
          // OSS 是转码硬依赖：直传不可用时不再回退 multipart（那样只会
          // 产生一个必然失败的转码任务），而是明确提示线下处理。
          ElMessage.error(
            (directErr && directErr.message) ||
              "上传服务暂不可用，视频文件需线下处理",
            { duration: 8000 }
          );
          return;
        }
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
