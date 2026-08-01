<template>
  <el-alert
    v-if="visible"
    class="video-upload-notice"
    type="warning"
    :title="title"
    :closable="false"
    show-icon
  >
    <p class="video-upload-notice__body">{{ message }}</p>
  </el-alert>
</template>

<script>
import {
  VIDEO_UPLOAD_DISABLED_MESSAGE,
  VIDEO_UPLOAD_DISABLED_TITLE,
  isVideoUploadDisabled
} from "@/utils/videoUploadPolicy";

export default {
  name: "VideoUploadMaintenanceNotice",
  computed: {
    visible() {
      return isVideoUploadDisabled();
    },
    title() {
      return VIDEO_UPLOAD_DISABLED_TITLE;
    },
    message() {
      return VIDEO_UPLOAD_DISABLED_MESSAGE;
    }
  }
};
</script>

<style scoped lang="scss">
.video-upload-notice {
  margin-bottom: 16px;
  :deep(.el-alert__title) {
    font-size: 15px;
    font-weight: 600;
  }
}
.video-upload-notice__body {
  margin: 6px 0 0;
  font-size: 13px;
  line-height: 1.6;
  color: #61666d;
}
</style>
