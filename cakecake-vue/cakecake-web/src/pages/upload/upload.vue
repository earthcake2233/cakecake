<template>
  <CreatorShell>
    <VideoUploadMaintenanceNotice />
    <div class="creator-tabs">
      <button
        type="button"
        class="creator-tab"
        :class="{ on: uploadTab === 'video' }"
        @click="uploadTab = 'video'"
      >
        视频投稿
      </button>
      <button
        type="button"
        class="creator-tab"
        :class="{ on: uploadTab === 'article' }"
        @click="uploadTab = 'article'"
      >
        专栏投稿
      </button>
    </div>

    <div v-if="uploadTab === 'video'" class="creator-panel creator-panel--video">
      <template v-if="showVideoSubmitDone">
        <div class="us-success-panel">
          <div class="us-illus-wrap" aria-hidden="true">
            <img class="us-illus" :src="successIllus" alt="" />
          </div>
          <h1 class="us-title">{{ successHeadline }}</h1>
          <p class="us-sub">{{ successSubline }}</p>
          <div class="us-actions">
            <button
              type="button"
              class="us-btn us-btn--ghost"
              @click="onViewProgress"
            >
              查看进度
            </button>
            <button
              type="button"
              class="us-btn us-btn--primary"
              @click="onPostAnother"
            >
              再投一个
            </button>
          </div>
        </div>
      </template>
      <template v-else>
        <div
          class="creator-dropzone"
          :class="{
            'creator-dropzone--hover': videoDropHover && !videoUploadDisabled,
            'creator-dropzone--disabled': videoUploadDisabled
          }"
          role="button"
          tabindex="0"
          @click="onPickVideo"
          @keydown.enter.prevent="onPickVideo"
          @dragenter.prevent="onVideoDragEnter"
          @dragover.prevent="onVideoDragOver"
          @dragleave.prevent="onVideoDragLeave"
          @drop.prevent="onVideoDrop"
        >
          <input
            ref="videoInput"
            type="file"
            accept="video/*"
            class="creator-file-input"
            @change="onVideoChange"
          />
          <div class="creator-upload-illus-wrap" aria-hidden="true">
            <img class="creator-upload-illus" src="@/assets/upload.png" alt="" />
          </div>
          <p class="creator-dropzone-tip">
            {{
              videoUploadDisabled
                ? "网页端暂不支持上传视频文件，可先填写稿件信息保存到服务器"
                : "点击上传或将视频拖拽到此区域"
            }}
          </p>
          <button
            type="button"
            class="creator-upload-btn"
            @click.stop="onPickVideo"
          >
            {{ videoUploadDisabled ? "填写稿件信息" : "上传视频" }}
          </button>
        </div>
        <p class="creator-upload-agreement">
          上传视频，即表示您已同意
          <a href="javascript:;">cakecake 使用协议</a>
          与
          <a href="javascript:;">cakecake 社区公约</a>
          ，请勿上传色情，反动等违法视频，
          <a href="javascript:;">查看社区规则</a>
        </p>
      </template>
    </div>

    <div v-else class="creator-panel creator-panel--article">
      <div class="creator-dropzone creator-dropzone--article">
        <div class="creator-upload-illus-wrap" aria-hidden="true">
          <img class="creator-upload-illus" src="@/assets/upload.png" alt="" />
        </div>
        <p class="creator-dropzone-tip">撰写专栏内容</p>
        <button
          type="button"
          class="creator-upload-btn"
          @click="$router.push({ name: 'articlePublish' })"
        >
          新建专栏
        </button>
      </div>
    </div>
  </CreatorShell>
</template>

<script>
import { ElMessage } from "element-plus";
import CreatorShell from "@/components/creator/CreatorShell.vue";
import VideoUploadMaintenanceNotice from "@/components/creator/VideoUploadMaintenanceNotice.vue";
import { stashPendingVideoFile } from "@/utils/creatorPendingVideo";
import { showCreatorVideoReviewNotice } from "@/utils/creatorVideoReviewNotice";
import {
  guardVideoFileUploadDisabled,
  isVideoUploadDisabled,
  STORAGE_METADATA_ONLY
} from "@/utils/videoUploadPolicy";
import successIllus from "@/assets/video_complete.565c959a.png";

const STORAGE_PENDING = "creator_center_pending_video";
const MAX_VIDEO_BYTES = 500 * 1024 * 1024;

export default {
  name: "UploadPage",
  components: { CreatorShell, VideoUploadMaintenanceNotice },
  data() {
    return {
      uploadTab: "video",
      successIllus,
      videoDropHover: false,
      publishReviewDialogShown: false
    };
  },
  computed: {
    videoUploadDisabled() {
      return isVideoUploadDisabled();
    },
    successMode() {
      const s = String(this.$route.query.success || "").toLowerCase();
      if (s === "edit" || s === "publish") return s;
      return "";
    },
    showVideoSubmitDone() {
      return this.successMode !== "";
    },
    successHeadline() {
      return "稿件投递成功";
    },
    successSubline() {
      return this.successMode === "edit"
        ? "修改已同步至服务器，可在稿件管理中继续管理该稿件。"
        : "视频已提交，转码完成后将进入审核，马上就能和大家见面啦~";
    }
  },
  watch: {
    "$route.fullPath"() {
      this.syncTabForSuccess();
      this.tryShowPublishReviewDialog();
    },
    successMode(mode) {
      if (mode !== "publish") {
        this.publishReviewDialogShown = false;
      } else {
        this.tryShowPublishReviewDialog();
      }
    }
  },
  mounted() {
    this.syncTabForSuccess();
    this.tryShowPublishReviewDialog();
    this._onWindowDragOver = (e) => {
      if (this.isVideoDragEvent(e)) e.preventDefault();
    };
    window.addEventListener("dragover", this._onWindowDragOver);
  },
  activated() {
    this.syncTabForSuccess();
    this.tryShowPublishReviewDialog();
  },
  beforeUnmount() {
    window.removeEventListener("dragover", this._onWindowDragOver);
    this._onWindowDragOver = null;
  },
  methods: {
    syncTabForSuccess() {
      if (this.showVideoSubmitDone) {
        this.uploadTab = "video";
      }
    },
    tryShowPublishReviewDialog() {
      if (this.successMode !== "publish" || this.publishReviewDialogShown) {
        return;
      }
      this.publishReviewDialogShown = true;
      void showCreatorVideoReviewNotice();
    },
    onPickVideo() {
      if (this.videoUploadDisabled) {
        this.goToMetadataPublish();
        return;
      }
      this.$refs.videoInput?.click();
    },
    isVideoDragEvent(e) {
      const dt = e && e.dataTransfer;
      if (!dt) return false;
      const types = dt.types;
      if (!types) return false;
      if (typeof types.includes === "function") {
        return types.includes("Files");
      }
      return Array.from(types).indexOf("Files") >= 0;
    },
    onVideoDragEnter(e) {
      if (!this.isVideoDragEvent(e)) return;
      this.videoDropHover = true;
    },
    onVideoDragOver(e) {
      if (!this.isVideoDragEvent(e)) return;
      e.dataTransfer.dropEffect = "copy";
      this.videoDropHover = true;
    },
    onVideoDragLeave(e) {
      const related = e.relatedTarget;
      if (related && e.currentTarget.contains(related)) return;
      this.videoDropHover = false;
    },
    onVideoDrop(e) {
      this.videoDropHover = false;
      if (this.videoUploadDisabled) {
        ElMessage.info("当前不支持上传视频文件，请先填写稿件信息");
        this.goToMetadataPublish();
        return;
      }
      if (
        guardVideoFileUploadDisabled(msg => {
          ElMessage.warning({ message: msg, duration: 6000 });
        })
      ) {
        return;
      }
      const file = this.pickVideoFromFileList(e.dataTransfer?.files);
      if (!file) {
        ElMessage.warning("请拖入视频文件（如 MP4、WebM、MOV）");
        return;
      }
      this.applyPendingVideoFile(file);
    },
    isVideoFile(file) {
      if (!file) return false;
      const type = String(file.type || "").toLowerCase();
      if (type.startsWith("video/")) return true;
      return /\.(mp4|webm|mkv|mov|avi|flv|wmv|m4v|mpeg|mpg|3gp)$/i.test(
        String(file.name || "")
      );
    },
    pickVideoFromFileList(fileList) {
      if (!fileList || !fileList.length) return null;
      for (let i = 0; i < fileList.length; i += 1) {
        const f = fileList[i];
        if (this.isVideoFile(f)) return f;
      }
      return null;
    },
    applyPendingVideoFile(file) {
      if (
        guardVideoFileUploadDisabled(msg => {
          ElMessage.warning({ message: msg, duration: 6000 });
        })
      ) {
        return;
      }
      if (!file || file.size === 0) {
        ElMessage.warning("视频文件为空");
        return;
      }
      if (file.size > MAX_VIDEO_BYTES) {
        ElMessage.warning("视频文件须不超过 500 MB");
        return;
      }
      stashPendingVideoFile(file);
      const objectUrl = URL.createObjectURL(file);
      sessionStorage.setItem(
        STORAGE_PENDING,
        JSON.stringify({
          objectUrl,
          fileName: file.name,
          fileSize: file.size
        })
      );
      this.$router.push({ name: "videoPublish" });
    },
    goToMetadataPublish() {
      sessionStorage.removeItem(STORAGE_PENDING);
      sessionStorage.setItem(STORAGE_METADATA_ONLY, "1");
      this.$router.push({ name: "videoPublish" });
    },
    onVideoChange(e) {
      const file = e.target.files?.[0];
      e.target.value = "";
      if (!file) return;
      if (!this.isVideoFile(file)) {
        ElMessage.warning("请选择视频文件");
        return;
      }
      this.applyPendingVideoFile(file);
    },
    onViewProgress() {
      this.$router.push({
        name: "manuscript",
        query: { tab: "video", status: "processing" }
      });
    },
    onPostAnother() {
      this.$router.replace({ name: "upload" });
    }
  }
};
</script>

<style lang="scss" scoped>
$c-blue: #00a1d6;
$c-text: #18191c;
$c-sub: #9499a0;
$c-line: #e3e5e7;
$c-drop: #f5f6f8;

.creator-tabs {
  display: flex;
  gap: 28px;
  border-bottom: 1px solid $c-line;
  margin-bottom: 28px;
}

.creator-tab {
  padding: 0 2px 12px;
  margin-bottom: -1px;
  border: none;
  background: none;
  font-size: 15px;
  color: $c-text;
  cursor: pointer;
  border-bottom: 3px solid transparent;
  &.on {
    color: $c-blue;
    font-weight: 600;
    border-bottom-color: $c-blue;
  }
  &:hover:not(.on) {
    color: $c-blue;
  }
}

.creator-panel {
  width: 100%;
  max-width: 920px;
}

.creator-panel--video,
.creator-panel--article {
  margin-left: auto;
  margin-right: auto;
}

/* —— 投稿成功态（对齐主站创作中心：Tab 下图 80–100px、标题距图 20–30px、按钮距文案 40–50px、双钮间距约 20px）—— */
.us-success-panel {
  text-align: center;
  padding: 0 40px 80px;
  background: #fff;
  border-radius: 8px;
  box-sizing: border-box;
}

.us-illus-wrap {
  display: flex;
  justify-content: center;
  align-items: center;
  margin-top: 88px;
  margin-bottom: 0;
}

/* 主站约 16:9 横版插图区：固定展示宽度，与截图比例一致 */
.us-illus {
  display: block;
  width: 380px;
  max-width: 100%;
  height: auto;
  object-fit: contain;
}

.us-title {
  margin: 24px 0 0;
  font-size: 20px;
  font-weight: 600;
  line-height: 28px;
  color: $c-text;
}

.us-sub {
  margin: 10px auto 0;
  max-width: 480px;
  font-size: 13px;
  line-height: 22px;
  color: #9499a0;
}

.us-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 20px;
  margin-top: 44px;
}

.us-btn {
  min-width: 132px;
  height: 40px;
  padding: 0 20px;
  border-radius: 4px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  border: 1px solid transparent;
  transition:
    background 0.15s ease,
    border-color 0.15s ease;
}

.us-btn--ghost {
  background: #fff;
  border-color: $c-line;
  color: $c-text;
  &:hover {
    border-color: #d7d9dc;
    background: $c-drop;
  }
}

.us-btn--primary {
  background: $c-blue;
  border-color: $c-blue;
  color: #fff;
  &:hover {
    background: #0091c8;
    border-color: #0091c8;
  }
}

.creator-dropzone {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 320px;
  padding: 40px 24px;
  background: $c-drop;
  border-radius: 8px;
  border: 1px dashed #dcdfe0;
  box-sizing: border-box;
  cursor: pointer;
  transition:
    border-color 0.15s ease,
    background 0.15s ease;
  &:hover,
  &.creator-dropzone--hover {
    border-color: rgba(0, 161, 214, 0.55);
    background: #f2f9fc;
  }
  &.creator-dropzone--disabled {
    cursor: not-allowed;
    opacity: 0.72;
    background: #f6f7f8;
    border-color: #e3e5e7;
    &:hover {
      border-color: #e3e5e7;
      background: #f6f7f8;
    }
  }
}

.creator-file-input {
  position: absolute;
  width: 0;
  height: 0;
  opacity: 0;
  pointer-events: none;
}

.creator-upload-illus-wrap {
  margin-bottom: 16px;
}

.creator-upload-illus {
  width: 88px;
  height: auto;
  display: block;
  margin: 0 auto;
  object-fit: contain;
}

.creator-dropzone-tip {
  margin: 0 0 20px;
  font-size: 14px;
  color: $c-sub;
}

.creator-upload-agreement {
  margin: 16px 0 0;
  padding: 0 12px;
  font-size: 12px;
  line-height: 1.85;
  color: #99a2aa;
  text-align: center;
  box-sizing: border-box;
}

.creator-upload-agreement a {
  color: $c-blue;
  text-decoration: none;
}

.creator-upload-agreement a:hover {
  text-decoration: underline;
}

.creator-upload-btn {
  min-width: 120px;
  height: 40px;
  padding: 0 24px;
  border: none;
  border-radius: 4px;
  background: $c-blue;
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  &:hover {
    background: #008ebd;
  }
}

.creator-panel--article .creator-dropzone--article {
  cursor: default;
  &:hover {
    border-color: #dcdfe0;
    background: $c-drop;
  }
}
</style>
