<template>
  <CreatorShell>
    <div class="vp-page">
      <VideoUploadMaintenanceNotice />
      <p v-if="editLoadBlocking" class="vp-edit-loading" role="status">正在加载稿件…</p>
      <template v-else>
      <div class="vp-head">
        <h1 class="vp-title">{{ pageTitle }}</h1>
      </div>

      <div class="vp-part-bar">
        <button type="button" class="vp-btn-add-part">+ 添加分P</button>
      </div>

      <section class="vp-card vp-video-card">
        <div class="vp-upload-panel">
        <div class="vp-file-row">
          <div class="vp-file-main">
            <span class="vp-file-ico" aria-hidden="true">
              <img
                class="vp-file-ico-img"
                :src="icoFileVideo"
                alt=""
              />
            </span>
            <div class="vp-file-col">
              <div class="vp-file-title-line">
                <span class="vp-file-name">{{ videoFileName }}</span>
                <span v-if="show4KBadge" class="vp-tag-4k">4K</span>
              </div>
              <div class="vp-file-status-line">
                <img class="vp-ico-complete" :src="icoComplete" alt="" />
                <span class="vp-file-status-txt">{{ videoFileStatusText }}</span>
              </div>
            </div>
          </div>
          <button
            type="button"
            class="vp-replace"
            :disabled="editReplaceBlocked"
            @click="onPickReplace"
          >
            <img class="vp-ico-update" :src="icoUpdate" alt="" />
            更换视频
          </button>
          <input
            ref="replaceInput"
            type="file"
            accept="video/*"
            class="vp-hidden-input"
            @change="onReplaceChange"
          />
        </div>
        <div
          class="vp-progress-wrap"
          :class="{
            'vp-progress-wrap--mb-upload': mbUploadBarActive,
            'vp-progress-wrap--mb-upload-done': uploadProgressBarCompleteGreen
          }"
        >
          <div class="vp-progress-track">
            <div
              class="vp-progress-fill"
              :class="{
                'vp-progress-fill--indeterminate': progressBarIndeterminate
              }"
              :style="progressBarFillStyle"
            />
          </div>
        </div>
        </div>
      </section>

      <video
        ref="capVideo"
        class="vp-cap-video"
        playsinline
        muted
        preload="auto"
        @progress="onCapMediaProgress"
        @loadedmetadata="onCapMediaMetadata"
        @canplaythrough="onCapMediaCanPlayThrough"
        @loadeddata="onCapLoaded"
        @seeked="onCapSeeked"
      />

      <section class="vp-card vp-form-card">
        <div class="vp-sec-head">
          <h2 class="vp-sec-title">基本设置</h2>
        </div>

        <div class="vp-field vp-field-row vp-field-row--cover">
          <label class="vp-label"><span class="vp-req">*</span> 封面</label>
          <div class="vp-cover-block">
            <div
              class="vp-cover-main"
              role="button"
              tabindex="0"
              @click="openCoverModal"
              @keydown.enter.prevent="openCoverModal"
            >
              <img v-if="coverMainSrc" class="vp-cover-img" :src="coverMainSrc" alt="" />
              <div v-else class="vp-cover-ph">请设置封面</div>
              <div class="vp-cover-mask">
                <span class="vp-cover-btn">封面设置</span>
              </div>
            </div>
          </div>
        </div>

        <Teleport to="body">
          <div v-if="coverModalVisible" class="vp-cmod-root">
            <div class="vp-cmod-backdrop" @click="closeCoverModal" />
            <div class="vp-cmod-dialog" role="dialog" aria-modal="true">
              <div class="vp-cmod-head">
                <div class="vp-cmod-tabs">
                  <button
                    type="button"
                    class="vp-cmod-tab"
                    :class="{ 'vp-cmod-tab--active': coverModalTab === 'capture' }"
                    :disabled="!coverCaptureAvailable"
                    @click="setCoverModalTab('capture')"
                  >
                    截取封面
                  </button>
                  <button
                    type="button"
                    class="vp-cmod-tab"
                    :class="{ 'vp-cmod-tab--active': coverModalTab === 'upload' }"
                    @click="setCoverModalTab('upload')"
                  >
                    上传封面
                  </button>
                </div>
                <button type="button" class="vp-cmod-close" aria-label="关闭" @click="closeCoverModal">
                  ×
                </button>
              </div>

              <div v-if="coverModalTab === 'capture'" class="vp-cmod-body">
                <div v-if="!coverCaptureAvailable" class="vp-cmod-empty">
                  当前无可截取的视频画面，请先在本地选择视频文件，或使用「上传封面」。
                </div>
                <template v-else>
                  <div class="vp-cmod-cap-grid">
                    <div class="vp-cmod-cap-col">
                      <div class="vp-cmod-cap-stage">
                        <video
                          ref="coverModalVideo"
                          class="vp-cmod-cap-video"
                          playsinline
                          muted
                          preload="metadata"
                          @loadedmetadata="onCoverModalMeta"
                          @seeked="onCoverModalSeeked"
                        />
                        <div
                          v-if="cropRect.w"
                          class="vp-cmod-crop"
                          :style="cropOverlayStyle()"
                          @mousedown.prevent="onCropDragStart"
                        />
                      </div>
                      <p class="vp-cmod-res-hint">
                        当前分辨率 {{ coverModalVideoResText }}
                      </p>
                      <p class="vp-cmod-ratio-hint">
                        *部分情况下您的封面将以 4:3 比例展示
                      </p>
                      <p class="vp-cmod-crop-hint">
                        时间轴每秒一格选画面；画面上白框可拖拽，用于框选封面区域。
                      </p>
                      <div class="vp-cmod-strip-wrap">
                        <p v-if="coverModalBusy && !filmstripFrames.length" class="vp-cmod-strip-loading">
                          正在按秒截取预览…
                        </p>
                        <div
                          v-show="filmstripFrames.length"
                          ref="stripTrackRef"
                          class="vp-cmod-strip-track"
                          @mousedown="onStripTrackMouseDown"
                        >
                          <div class="vp-cmod-strip-inner">
                            <div
                              v-for="(fr, i) in filmstripFrames"
                              :key="i"
                              class="vp-cmod-strip-thumb"
                            >
                              <img :src="fr.url" alt="" />
                            </div>
                          </div>
                          <div class="vp-cmod-strip-select-wrap" :style="stripSelectorWrapStyle">
                            <div class="vp-cmod-strip-tooltip">{{ stripTimeLabel }}</div>
                            <div
                              class="vp-cmod-strip-frame"
                              @mousedown.stop.prevent="onStripSelectorMouseDown"
                            />
                          </div>
                        </div>
                      </div>
                    </div>
                    <div class="vp-cmod-prev-col">
                      <div class="vp-cmod-prev-item">
                        <div class="vp-cmod-prev-label">16:9 效果预览</div>
                        <div class="vp-cmod-prev-box vp-cmod-prev-box--169">
                          <div v-if="preview169" class="vp-cmod-prev-stack">
                            <img class="vp-cmod-prev-img" :src="preview169" alt="" />
                            <div
                              class="vp-cmod-prev-title-overlay"
                              :class="{ 'vp-cmod-prev-title-overlay--ph': !coverPreviewHasTitle }"
                            >
                              {{ coverPreviewTitleLine }}
                            </div>
                          </div>
                        </div>
                      </div>
                      <div class="vp-cmod-prev-item">
                        <div class="vp-cmod-prev-label">4:3 效果预览</div>
                        <div class="vp-cmod-prev-box vp-cmod-prev-box--43">
                          <div v-if="preview43" class="vp-cmod-prev-stack">
                            <img class="vp-cmod-prev-img" :src="preview43" alt="" />
                            <div
                              class="vp-cmod-prev-title-overlay vp-cmod-prev-title-overlay--43"
                              :class="{ 'vp-cmod-prev-title-overlay--ph': !coverPreviewHasTitle }"
                            >
                              {{ coverPreviewTitleLine }}
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                  <div class="vp-cmod-actions">
                    <button type="button" class="vp-cmod-btn-done" @click="applyCoverFromCapture">
                      完成
                    </button>
                  </div>
                </template>
              </div>

              <div v-else class="vp-cmod-body vp-cmod-body--upload">
                <input
                  ref="coverUploadInput"
                  type="file"
                  accept="image/jpeg,image/png,image/webp,image/gif"
                  class="vp-hidden-input"
                  @change="onCoverUploadChange"
                />
                <div
                  class="vp-cmod-drop"
                  :class="{ 'vp-cmod-drop--hover': coverDropHover }"
                  @dragenter.prevent="coverDropHover = true"
                  @dragover.prevent="coverDropHover = true"
                  @dragleave.prevent="coverDropHover = false"
                  @drop.prevent="onCoverDrop"
                >
                  <div class="vp-cmod-drop-ico" aria-hidden="true">☁</div>
                  <p class="vp-cmod-drop-txt">拖拽图片到此处或者点击上传</p>
                  <button type="button" class="vp-cmod-btn-pick" @click="pickCoverFile">
                    选择图片
                  </button>
                </div>
              </div>
            </div>
          </div>
        </Teleport>

        <div class="vp-field vp-field-row vp-field-row--middle">
          <label class="vp-label" for="vp-title-inp"><span class="vp-req">*</span> 标题</label>
          <div class="vp-input-wrap">
            <input
              id="vp-title-inp"
              v-model="form.title"
              type="text"
              class="vp-input"
              maxlength="80"
              placeholder="请输入标题"
            />
            <span class="vp-counter">{{ titleRuneCount }}/80</span>
          </div>
        </div>

        <div class="vp-field vp-field-row vp-field-row--middle">
          <span class="vp-label">类型</span>
          <div class="vp-radios">
            <label class="vp-radio">
              <input v-model="form.videoType" type="radio" value="自制" />
              自制
            </label>
            <label class="vp-radio">
              <input v-model="form.videoType" type="radio" value="转载" />
              转载
            </label>
          </div>
        </div>

        <div class="vp-field vp-field-row vp-field-row--middle">
          <label class="vp-label"><span class="vp-req">*</span> 分区</label>
          <VideoZonePicker v-model="form.zone" />
        </div>

        <div class="vp-field vp-field-row vp-field-row--tags">
          <label class="vp-label"><span class="vp-req">*</span> 标签</label>
          <div class="vp-tags-body">
            <div class="vp-tag-box">
              <div class="vp-tag-inner">
                <span v-for="(t, i) in tags" :key="t + i" class="vp-tag">
                  {{ t }}
                  <button type="button" class="vp-tag-x" @click="removeTag(i)">×</button>
                </span>
                <input
                  v-model.trim="tagInput"
                  class="vp-tag-inp"
                  placeholder="按回车键Enter创建标签"
                  @keydown.enter.prevent="addTag"
                />
              </div>
              <span class="vp-tag-cap">还可以添加 {{ tagRemain }} 个标签</span>
            </div>
            <div class="vp-rec-tags">
              <span class="vp-rec-label">推荐标签：</span>
              <button
                v-for="r in recommended"
                :key="r"
                type="button"
                class="vp-rec-chip"
                @click="addRecommended(r)"
              >
                {{ r }}
              </button>
            </div>
          </div>
        </div>

        <div class="vp-field vp-field-row vp-field-row--intro">
          <label class="vp-label" for="vp-intro">简介</label>
          <div class="vp-ta-wrap">
            <textarea
              id="vp-intro"
              v-model="form.intro"
              class="vp-textarea"
              rows="6"
              maxlength="2000"
              placeholder="填写更全面的相关信息，让更多的人能找到你的视频吧：)"
            />
            <span class="vp-counter vp-counter-ta">{{ form.intro.length }}/2000</span>
          </div>
        </div>
      </section>

      <div class="vp-foot">
        <template v-if="isPublishMode || isDraftEditMode">
          <button
            type="button"
            class="vp-btn vp-btn-ghost"
            :disabled="footActionsDisabled"
            @click="onSaveDraft"
          >
            存草稿
          </button>
          <button
            type="button"
            class="vp-btn vp-btn-primary"
            :disabled="footActionsDisabled"
            @click="onSubmit"
          >
            {{ publishPrimaryLabel }}
          </button>
        </template>
        <template v-else>
          <button
            type="button"
            class="vp-btn vp-btn-primary vp-btn-wide"
            :disabled="footActionsDisabled"
            @click="onSubmit"
          >
            {{ publishPrimaryLabel }}
          </button>
        </template>
      </div>
      </template>
    </div>
  </CreatorShell>
</template>

<script>
import CreatorShell from "@/components/creator/CreatorShell.vue";
import VideoUploadMaintenanceNotice from "@/components/creator/VideoUploadMaintenanceNotice.vue";
import VideoZonePicker from "@/components/creator/VideoZonePicker.vue";
import { findCreatorVideo } from "./creatorVideoMock.js";
import { ElMessage, ElMessageBox } from "element-plus";
import "element-plus/es/components/message-box/style/css";
import "element-plus/es/components/message/style/css";
import {
  mbFetchDraftVideoObjectUrl,
  mbGetVideo,
  mbPublishVideoDraft,
  mbReplaceVideoMedia,
  mbSaveVideoDraft,
  mbUpdateMyVideo,
  mbUpdateVideoCover,
  mbUpdateVideoDraft,
  mbUploadVideo
} from "@/api/minibili";
import icoComplete from "@/assets/upload_manager/article/complete.png";
import icoUpdate from "@/assets/upload_manager/article/update.png";
import icoFileVideo from "@/assets/upload_manager/article/file-video-line.png";
import defaultUploadCoverPlaceholder from "@/assets/85251fe9cc54ac2b826a965a90f8dba811edbc7a.gif@920w_518h.webp";
import {
  clearPendingVideoFile,
  takePendingVideoFile
} from "@/utils/creatorPendingVideo";
import {
  isKnownVideoZone,
  normalizeVideoZoneValue
} from "@/constants/videoZones";
import {
  guardVideoFileUploadDisabled,
  isVideoUploadDisabled,
  STORAGE_METADATA_ONLY
} from "@/utils/videoUploadPolicy";

const STORAGE_PENDING = "creator_center_pending_video";
const LEAVE_MSG =
  "系统可能不会保存填写的稿件信息噢...(´；ω；`)";

export default {
  name: "VideoPublishPage",
  components: { CreatorShell, VideoUploadMaintenanceNotice, VideoZonePicker },
  data() {
    return {
      leaveGuardEnabled: false,
      publishCommitted: false,
      editDirty: false,
      skipEditDirty: true,
      /** 多压一条相同 hash，第一次点浏览器后退就会触发 popstate，而不会先悄悄离开路由 */
      leaveBackTrapInstalled: false,
      suppressLeavePopstateOnce: false,
      leavePromptOpen: false,
      leaveWindowListenersBound: false,
      videoObjectUrl: "",
      videoFileName: "",
      show4KBadge: false,
      coverDisplay: "",
      coverModalVisible: false,
      coverModalTab: "capture",
      filmstripFrames: [],
      /** 时间轴选框当前对齐的秒数（整数 0…floor(duration)） */
      coverStripSecond: 0,
      /** 视频总时长对应的末秒索引，用于比例映射（与胶片格数无关） */
      coverModalMaxSec: 0,
      stripDragActive: false,
      stripSeekToken: 0,
      cropRect: { x: 0, y: 0, w: 0, h: 0 },
      preview169: "",
      preview43: "",
      coverDropHover: false,
      coverModalBusy: false,
      cropDragSession: null,
      /** 发布/更换视频后自动截取首帧作为默认封面 */
      needCapture: false,
      captureScheduled: false,
      form: {
        title: "",
        videoType: "自制",
        zone: "动画",
        intro: ""
      },
      tags: [],
      tagInput: "",
      maxTags: 10,
      recommended: ["录屏", "直播", "记录", "Vlog", "学习", "手游", "日常", "动画"],
      icoComplete,
      icoUpdate,
      icoFileVideo,
      /** 从「更换视频」直接拿到的 File，优先于 sessionStorage 中的 blob URL */
      localVideoFile: null,
      pendingFileSize: null,
      uploadSubmitting: false,
      uploadProgress: 0,
      /** 上传已成功：用于在 finally 清空 progress 后仍显示满格绿色条，直至路由离开 */
      mbUploadFinishedOk: false,
      /** 0–100：本地隐藏 video 的缓冲/解码进度（发布页未上传前） */
      localMediaProgress: 0,
      /** bump 后让「可截取封面」等计算属性随 capVideo 赋值刷新 */
      capVideoMediaVersion: 0,
      editVideoLoading: false,
      editVideoStatus: "",
      editSourceCoverUrl: "",
      editVideoNumericId: 0,
      /** 编辑草稿时从服务端拉取的 blob 预览地址 */
      draftBlobUrl: "",
      /** 用户主动改过封面（上传/截取），与 OSS 旧地址区分 */
      coverDirty: false,
      /** 仅保存稿件元数据，不上传视频文件 */
      metadataOnlyMode: false
    };
  },
  computed: {
    /** SPEC F2：与全局 Mini-Bili API 开关一致 */
    isMinibiliApiEnv() {
      return (
        import.meta.env.VITE_MINIBILI_API === "true" ||
        import.meta.env.VITE_MINIBILI_API === "1"
      );
    },
    isPublishMode() {
      return this.$route.name === "videoPublish";
    },
    isEditMode() {
      return this.$route.name === "videoEdit";
    },
    editLoadBlocking() {
      return (
        this.isMinibiliApiEnv &&
        this.isEditMode &&
        this.editVideoLoading
      );
    },
    isDraftEditMode() {
      return (
        this.isMinibiliApiEnv &&
        this.isEditMode &&
        this.editVideoStatus === "draft"
      );
    },
    isRejectedOrFailedEditMode() {
      const st = String(this.editVideoStatus || "");
      return (
        this.isMinibiliApiEnv &&
        this.isEditMode &&
        (st === "failed" || st === "rejected")
      );
    },
    canReplaceVideoInEdit() {
      return this.isDraftEditMode || this.isRejectedOrFailedEditMode;
    },
    editReplaceBlocked() {
      if (this.videoUploadDisabled) return true;
      return this.isMinibiliApiEnv && this.isEditMode && !this.canReplaceVideoInEdit;
    },
    isMetadataOnlyPublish() {
      return (
        this.isPublishMode &&
        this.isMinibiliApiEnv &&
        (this.metadataOnlyMode ||
          (this.videoUploadDisabled &&
            !this.videoObjectUrl &&
            !this.localVideoFile))
      );
    },
    pageTitle() {
      return this.isEditMode ? "编辑视频" : "发布视频";
    },
    tagRemain() {
      return Math.max(0, this.maxTags - this.tags.length);
    },
    /** 当前可用于截取封面的视频地址（不依赖 capVideo.src 的响应式） */
    videoPreviewSrc() {
      void this.capVideoMediaVersion;
      if (this.localVideoFile && this.videoObjectUrl) {
        return String(this.videoObjectUrl).trim();
      }
      if (this.draftBlobUrl) return this.draftBlobUrl;
      const u = String(this.videoObjectUrl || "").trim();
      if (u) return u;
      const el = this.$refs.capVideo;
      return el && el.src ? String(el.src).trim() : "";
    },
    /** 发布/草稿编辑：是否有可截取封面的视频源 */
    coverCaptureAvailable() {
      return !!this.videoPreviewSrc;
    },
    /** 上传未完成或封面尚未截取时，先展示默认占位图 */
    showUploadPendingCoverPlaceholder() {
      if ((this.coverDisplay || "").trim()) return false;
      if (this.uploadSubmitting) return true;
      if (this.needCapture) return true;
      if (
        this.coverCaptureAvailable &&
        (this.isPublishMode || this.isDraftEditMode)
      ) {
        return true;
      }
      return false;
    },
    coverMainSrc() {
      const c = (this.coverDisplay || "").trim();
      if (c) return c;
      if (this.showUploadPendingCoverPlaceholder) {
        return defaultUploadCoverPlaceholder;
      }
      return "";
    },
    /** Mini-Bili 发布：multipart 正在上传（用于进度条配色） */
    mbUploadBarActive() {
      return (
        this.isMinibiliApiEnv &&
        this.isPublishMode &&
        this.uploadSubmitting
      );
    },
    mbUploadBarComplete() {
      if (!this.isMinibiliApiEnv || !this.isPublishMode) return false;
      return this.uploadProgress >= 100 || this.mbUploadFinishedOk;
    },
    /** 整条进度条用成功色 #43ce5b：含服务端上传完成，或本地文件已缓冲至 100%（见文案「本地已载入」） */
    uploadProgressBarCompleteGreen() {
      if (this.mbUploadBarComplete) return true;
      if (
        this.isPublishMode &&
        !this.uploadSubmitting &&
        (this.videoObjectUrl || this.localVideoFile) &&
        this.localMediaProgress >= 100
      ) {
        return true;
      }
      return false;
    },
    /** Mini-Bili 发布且正在提交：用 XHR 上传进度；不确定态仅在上传阶段 */
    progressBarIndeterminate() {
      return (
        this.isMinibiliApiEnv &&
        this.isPublishMode &&
        this.uploadSubmitting &&
        this.uploadProgress < 1
      );
    },
    /** 进度条宽度：默认本地载入；Mini-Bili 上传中改为上传进度 */
    progressBarFillStyle() {
      if (
        this.isMinibiliApiEnv &&
        this.isPublishMode &&
        (this.uploadSubmitting || this.mbUploadFinishedOk)
      ) {
        if (this.mbUploadFinishedOk) {
          return { width: "100%" };
        }
        if (this.uploadProgress < 1) return {};
        const p = Math.max(2, Math.min(100, this.uploadProgress));
        return { width: `${p}%` };
      }
      const hasLocalPublish =
        this.isPublishMode && (this.videoObjectUrl || this.localVideoFile);
      const hasEditRemote =
        this.isEditMode &&
        this.isMinibiliApiEnv &&
        this.editVideoNumericId > 0 &&
        !this.editVideoLoading;
      if (hasLocalPublish || hasEditRemote) {
        const p = Math.max(0, Math.min(100, this.localMediaProgress));
        return { width: `${p}%` };
      }
      return { width: "0%" };
    },
    coverModalVideoResText() {
      const el = this.$refs.coverModalVideo;
      if (!el?.videoWidth) return "—";
      return `${el.videoWidth}×${el.videoHeight}`;
    },
    /** 封面右侧「阅览」预览：封面图 + 底部标题条 */
    coverPreviewHasTitle() {
      return !!(this.form.title && String(this.form.title).trim());
    },
    coverPreviewTitleLine() {
      const t = (this.form.title || "").trim();
      return t || "请输入标题";
    },
    /** 与后端 utf8.RuneCountInString 对齐（按 Unicode 码位计数） */
    titleRuneCount() {
      return Array.from(this.form.title || "").length;
    },
    /** 白框宽度 = 一格 = 1 秒；总长固定，秒数越多格越窄 */
    stripSelectorWrapStyle() {
      const maxSec = this.coverModalMaxSec;
      const cells = maxSec + 1;
      const tilePct = 100 / cells;
      if (maxSec <= 0) {
        return { width: "100%", left: "0%" };
      }
      const ratio = this.coverStripSecond / maxSec;
      const leftPct = ratio * (100 - tilePct);
      return {
        width: `${tilePct}%`,
        left: `${leftPct}%`
      };
    },
    stripTimeLabel() {
      const sec = this.coverStripSecond;
      const m = Math.floor(sec / 60);
      const s = sec % 60;
      return `${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
    },
    /** 发布页：Mini-Bili 走真实上传时的主按钮文案（R-FE-6 loading） */
    publishPrimaryLabel() {
      if (this.isMetadataOnlyPublish) {
        return this.uploadSubmitting ? "保存中…" : "保存稿件信息";
      }
      if (this.isDraftEditMode) {
        return this.uploadSubmitting ? "投稿中…" : "立即投稿";
      }
      if (this.isEditMode && this.isMinibiliApiEnv) {
        if (this.isRejectedOrFailedEditMode && this.localVideoFile) {
          return this.uploadSubmitting ? "投稿中…" : "重新投稿";
        }
        return this.uploadSubmitting ? "保存中…" : "保存修改";
      }
      if (!this.uploadSubmitting) {
        return "立即投稿";
      }
      if (this.isMinibiliApiEnv && this.isPublishMode && this.uploadProgress > 0) {
        return `提交中 ${this.uploadProgress}%`;
      }
      return "提交中…";
    },
    /** 视频文件行状态（SPEC：上传为服务端异步；本地仅表示已载入） */
    videoFileStatusText() {
      if (this.isMinibiliApiEnv && this.isEditMode) {
        const st = String(this.editVideoStatus || "");
        if (st === "draft") return "草稿已保存，可继续编辑或立即投稿";
        if (st === "published") return "视频已发布，可修改标题、简介与封面";
        if (st === "pending_review") return "视频审核中，马上就能和大家见面啦~，可修改标题与简介";
        if (st === "processing") return "视频转码处理中，可修改标题与简介";
        if (st === "rejected") {
          return "审核未通过，可更换视频或修改标题与简介后重新投稿";
        }
        if (st === "failed") {
          return "转码未通过，可更换视频或修改标题与简介后重新投稿";
        }
        return "稿件已从服务器加载";
      }
      if (this.isMinibiliApiEnv && this.isPublishMode) {
        if (this.isMetadataOnlyPublish) {
          return "视频文件将由管理员线下处理，可先保存标题、简介等信息";
        }
        if (this.uploadSubmitting) {
          return this.uploadProgress > 0
            ? `正在上传 ${this.uploadProgress}%`
            : "正在提交…";
        }
        if (this.videoObjectUrl || this.localVideoFile) {
          if (this.localMediaProgress < 100) {
            return `本地载入 ${this.localMediaProgress}%`;
          }
          return "本地已载入，提交后将上传至服务器";
        }
        return "请选择视频文件";
      }
      return "上传完成";
    },
    shouldConfirmLeave() {
      if (!this.leaveGuardEnabled) return false;
      // 发布页 / 草稿编辑 / 已发布编辑：未成功存草稿或投稿前均提示（含浏览器后退）
      if (
        this.isPublishMode ||
        this.isDraftEditMode ||
        (this.isEditMode && this.isMinibiliApiEnv)
      ) {
        return !this.publishCommitted;
      }
      return !this.publishCommitted;
    },
    footActionsDisabled() {
      return (
        this.uploadSubmitting ||
        (this.isMinibiliApiEnv && this.isEditMode && this.editVideoLoading)
      );
    },
    videoUploadDisabled() {
      return isVideoUploadDisabled();
    }
  },
  watch: {
    "$route.fullPath"() {
      this.bootstrap();
    },
    form: {
      deep: true,
      handler() {
        if (this.skipEditDirty || !this.isEditMode) return;
        this.editDirty = true;
      }
    },
    tags: {
      deep: true,
      handler() {
        if (this.skipEditDirty || !this.isEditMode) return;
        this.editDirty = true;
      }
    },
    coverDisplay() {
      if (this.skipEditDirty || !this.isEditMode) return;
      this.editDirty = true;
    }
  },
  mounted() {
    this.bootstrap();
    this.bindLeaveWindowListeners();
    window.addEventListener("keydown", this.onCoverModalKeydown);
  },
  activated() {
    this.bindLeaveWindowListeners();
  },
  deactivated() {
    this.unbindLeaveWindowListeners();
    ElMessageBox.close();
    this.leavePromptOpen = false;
    this.mbUploadFinishedOk = false;
  },
  beforeUnmount() {
    this.mbUploadFinishedOk = false;
    this.unbindLeaveWindowListeners();
    window.removeEventListener("keydown", this.onCoverModalKeydown);
    this.teardownCropDrag();
    this.teardownStripDrag();
    ElMessageBox.close();
    this.leavePromptOpen = false;
    this.revokeBlob();
    this.revokeDraftBlob();
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
        /* 勿在此处 history.back()：会与 Vue Router 的 next() 打架，导致无法跳转 */
        this.clearLeaveBackTrapOnly();
        this.publishCommitted = true;
        this.editDirty = false;
        next();
      })
      .catch(() => next(false));
  },
  methods: {
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
        this.removeLeaveBackTrapSilent();
        return;
      }
      if (this.leaveBackTrapInstalled) return;
      window.history.pushState({ __vpLeaveTrap: true }, "", window.location.href);
      this.leaveBackTrapInstalled = true;
    },
    /** 仅清除标记；用于路由 next() 离开时，由 Router 改 URL，不能再 history.back() */
    clearLeaveBackTrapOnly() {
      if (this.leaveBackTrapInstalled) this.leaveBackTrapInstalled = false;
    },
    /** 存草稿成功后离开编辑页：勿 history.back()，否则会留在当前路由仅消掉陷阱 */
    async navigateToManuscriptAfterSave(query) {
      this.clearLeaveBackTrapOnly();
      await this.$router.replace({
        name: "manuscript",
        query
      });
    },
    /** 消费多 push 的那条历史（存草稿/投稿等仍留在当前页时用） */
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
          this.publishCommitted = true;
          this.editDirty = false;
          this.leaveBackTrapInstalled = false;
          window.history.back();
        })
        .catch(() => {
          window.history.pushState({ __vpLeaveTrap: true }, "", window.location.href);
          this.leaveBackTrapInstalled = true;
        });
    },
    onWindowBeforeUnload(e) {
      if (!this.shouldConfirmLeave) return;
      e.preventDefault();
      e.returnValue = "";
    },
    revokeBlob() {
      if (this.videoObjectUrl && this.videoObjectUrl.startsWith("blob:")) {
        URL.revokeObjectURL(this.videoObjectUrl);
      }
      this.videoObjectUrl = "";
      this.localMediaProgress = 0;
      this.capVideoMediaVersion += 1;
    },
    revokeDraftBlob() {
      if (this.draftBlobUrl) {
        URL.revokeObjectURL(this.draftBlobUrl);
        this.draftBlobUrl = "";
      }
    },
    bootstrap() {
      this.leaveGuardEnabled = false;
      this.localVideoFile = null;
      this.pendingFileSize = null;
      this.revokeBlob();
      this.revokeDraftBlob();
      this.resetFormPartial();
      this.closeCoverModal();
      this.needCapture = false;
      this.captureScheduled = false;
      this.editVideoLoading = false;
      this.editVideoStatus = "";
      this.editSourceCoverUrl = "";
      this.editVideoNumericId = 0;
      this.localMediaProgress = 0;
      this.coverDirty = false;
      this.metadataOnlyMode = false;

      if (this.isEditMode) {
        if (this.isMinibiliApiEnv) {
          void this.bootstrapEditMinibili();
          return;
        }
        const row = findCreatorVideo(this.$route.params.id);
        if (row) {
          this.videoFileName = row.fileName;
          this.show4KBadge = row.fileName.includes("4K");
          this.form.title = row.title;
          this.form.videoType = row.videoType;
          this.form.zone = row.zone;
          this.form.intro = row.intro || "";
          this.tags = [...row.tags];
          const c = row.cover;
          this.coverDisplay = c;
        } else {
          this.videoFileName = "未找到稿件.mp4";
          this.coverDisplay = "";
        }
        this.publishCommitted = false;
        this.editDirty = false;
        this.skipEditDirty = true;
        this.leaveGuardEnabled = true;
        this.$nextTick(() => {
          if (this.$refs.capVideo) this.$refs.capVideo.removeAttribute("src");
          this.skipEditDirty = false;
          this.syncLeaveBackTrap();
        });
        return;
      }

      const raw = sessionStorage.getItem(STORAGE_PENDING);
      const metadataFlag =
        sessionStorage.getItem(STORAGE_METADATA_ONLY) === "1";
      if (!raw) {
        if (metadataFlag && this.videoUploadDisabled) {
          this.bootstrapMetadataOnlyPublish();
          return;
        }
        this.$router.replace({ name: "upload" });
        return;
      }
      try {
        const parsed = JSON.parse(raw);
        const { objectUrl, fileName, fileSize } = parsed;
        if (!objectUrl || !fileName) {
          this.$router.replace({ name: "upload" });
          return;
        }
        const picked = takePendingVideoFile();
        this.localVideoFile = picked || null;
        this.pendingFileSize =
          picked?.size ??
          (typeof fileSize === "number" && fileSize >= 0 ? fileSize : null);
        this.videoObjectUrl = objectUrl;
        this.videoFileName = fileName;
        this.form.videoType = "自制";
        this.form.zone = "动画";
        this.form.intro = "";
        this.tags = [];
        this.show4KBadge =
          /4k|2160|uhd/i.test(fileName) || fileName.includes("4K");
        const base = fileName.replace(/\.[^.]+$/, "");
        this.form.title = base.slice(0, 80);
        this.coverDisplay = "";
        this.needCapture = true;
        this.captureScheduled = false;
        this.publishCommitted = false;
        this.leaveGuardEnabled = true;
        this.$nextTick(() => {
          this.attachVideoSrc(objectUrl);
          this.syncLeaveBackTrap();
        });
        window.setTimeout(() => {
          if (this.needCapture) this.tryCaptureFallback();
        }, 600);
      } catch {
        this.$router.replace({ name: "upload" });
      }
    },
    bootstrapMetadataOnlyPublish() {
      sessionStorage.removeItem(STORAGE_METADATA_ONLY);
      this.metadataOnlyMode = true;
      this.videoFileName = "待关联视频（管理员处理）";
      this.show4KBadge = false;
      this.form.videoType = "自制";
      this.form.zone = "动画";
      this.form.title = "";
      this.form.intro = "";
      this.tags = [];
      this.coverDisplay = "";
      this.needCapture = false;
      this.captureScheduled = false;
      this.publishCommitted = false;
      this.leaveGuardEnabled = true;
      this.$nextTick(() => {
        if (this.$refs.capVideo) this.$refs.capVideo.removeAttribute("src");
        this.syncLeaveBackTrap();
      });
    },
    async bootstrapEditMinibili() {
      this.editVideoLoading = true;
      this.editVideoStatus = "";
      this.editSourceCoverUrl = "";
      this.editVideoNumericId = 0;
      this.localMediaProgress = 0;
      const id = Number(this.$route.params.id);
      if (!Number.isFinite(id) || id <= 0) {
        this.editVideoLoading = false;
        this.$router.replace({ name: "manuscript" });
        return;
      }
      this.editVideoNumericId = id;
      try {
        const d = await mbGetVideo(id, { skipGlobalErrorToast: true });
        this.form.title = String(d.title || "");
        this.form.intro = String(d.description || "");
        this.form.videoType = "自制";
        const zoneRaw =
          d.zone ||
          (d.zone_parent && d.zone_child
            ? `${d.zone_parent}-${d.zone_child}`
            : d.zone_parent || "");
        this.form.zone = this.resolveStoredVideoZone(zoneRaw);
        this.tags = Array.isArray(d.tags) ? d.tags.map(t => String(t)) : [];
        this.coverDisplay = String(d.cover_url || "").trim();
        this.editSourceCoverUrl = this.coverDisplay;
        this.editVideoStatus = String(d.status || "").trim();
        this.coverDirty = false;
        const base = (d.title || `video_${id}`).slice(0, 60);
        this.videoFileName = `${base}.mp4`;
        this.show4KBadge = /4k|2160|uhd/i.test(this.videoFileName);
        this.localVideoFile = null;
        this.pendingFileSize = null;
        this.videoObjectUrl = "";
        const vurl = String(d.video_url || "").trim();
        const isDraft = this.editVideoStatus === "draft";
        const hasCover = !!this.coverDisplay;
        const willPreview = isDraft
          ? Boolean(d.draft_has_source)
          : Boolean(vurl);
        this.needCapture = willPreview && !hasCover;
        this.captureScheduled = false;
        this.publishCommitted = false;
        this.editDirty = false;
        this.skipEditDirty = true;
        this.leaveGuardEnabled = true;
        let previewUrl = vurl;
        if (isDraft && d.draft_has_source) {
          try {
            this.revokeDraftBlob();
            previewUrl = await mbFetchDraftVideoObjectUrl(id);
            this.draftBlobUrl = previewUrl;
          } catch {
            previewUrl = "";
          }
        }
        this.$nextTick(() => {
          if (previewUrl) {
            this.attachVideoSrc(previewUrl);
          } else if (this.$refs.capVideo) {
            this.$refs.capVideo.removeAttribute("src");
          }
          this.skipEditDirty = false;
          this.syncLeaveBackTrap();
        });
        if (this.needCapture && previewUrl) {
          window.setTimeout(() => {
            if (this.needCapture) this.tryCaptureFallback();
          }, 600);
        }
      } catch (e) {
        const msg = e instanceof Error ? e.message : "加载失败";
        ElMessage.error(msg);
        this.$router.replace({ name: "manuscript" });
      } finally {
        this.editVideoLoading = false;
      }
    },
    resetFormPartial() {
      this.tagInput = "";
    },
    attachVideoSrc(url) {
      const el = this.$refs.capVideo;
      if (!el) return;
      const u = String(url || "");
      if (/^https?:\/\//i.test(u)) {
        el.crossOrigin = "anonymous";
      } else {
        el.removeAttribute("crossorigin");
      }
      this.localMediaProgress = 0;
      el.src = u;
      el.load();
      this.capVideoMediaVersion += 1;
    },
    updateLocalMediaProgressFromVideo() {
      if (
        this.isMinibiliApiEnv &&
        this.isPublishMode &&
        this.uploadSubmitting
      ) {
        return;
      }
      const v = this.$refs.capVideo;
      if (!v) return;
      if (!v.duration || !Number.isFinite(v.duration) || v.duration <= 0) {
        this.localMediaProgress = Math.max(this.localMediaProgress, 3);
        return;
      }
      if (!v.buffered || v.buffered.length === 0) {
        this.localMediaProgress = Math.max(this.localMediaProgress, 8);
        return;
      }
      const end = v.buffered.end(v.buffered.length - 1);
      const pct = Math.round((100 * end) / v.duration);
      this.localMediaProgress = Math.min(100, Math.max(this.localMediaProgress, pct));
    },
    onCapMediaProgress() {
      this.updateLocalMediaProgressFromVideo();
    },
    onCapMediaMetadata() {
      this.updateLocalMediaProgressFromVideo();
    },
    onCapMediaCanPlayThrough() {
      if (
        this.isMinibiliApiEnv &&
        this.isPublishMode &&
        this.uploadSubmitting
      ) {
        return;
      }
      this.localMediaProgress = 100;
    },
    /** 靠近视频起点的一帧（避免 currentTime=0 时不触发 seeked） */
    firstFrameSeekTime(dur) {
      if (!dur || !Number.isFinite(dur)) return 0;
      return dur > 0.08 ? 0.001 : dur * 0.25;
    },
    onCapLoaded() {
      this.updateLocalMediaProgressFromVideo();
      if (!this.needCapture || !this.$refs.capVideo) return;
      const v = this.$refs.capVideo;
      const dur = v.duration;
      if (!dur || !Number.isFinite(dur)) return;
      this.captureScheduled = true;
      v.currentTime = this.firstFrameSeekTime(dur);
    },
    onCapSeeked() {
      if (!this.captureScheduled || !this.needCapture) return;
      this.captureScheduled = false;
      this.captureDefaultCoverFromVideo();
    },
    tryCaptureFallback() {
      if (!this.needCapture || !this.$refs.capVideo) return;
      const v = this.$refs.capVideo;
      const dur = v.duration;
      if (!dur || !Number.isFinite(dur)) return;
      this.captureScheduled = true;
      try {
        v.currentTime = this.firstFrameSeekTime(dur);
      } catch {
        this.captureScheduled = false;
        this.captureDefaultCoverFromVideo();
      }
    },
    captureDefaultCoverFromVideo() {
      const v = this.$refs.capVideo;
      if (!v) {
        this.needCapture = false;
        return;
      }
      const w = v.videoWidth;
      const h = v.videoHeight;
      if (!w || !h) {
        this.needCapture = false;
        return;
      }
      const canvas = document.createElement("canvas");
      canvas.width = w;
      canvas.height = h;
      const ctx = canvas.getContext("2d");
      if (!ctx) {
        this.needCapture = false;
        return;
      }
      try {
        ctx.drawImage(v, 0, 0, w, h);
        this.coverDisplay = canvas.toDataURL("image/jpeg", 0.85);
        this.coverDirty = true;
        if (this.isEditMode) this.editDirty = true;
      } catch {
        /* 解码失败等 */
      }
      this.needCapture = false;
    },
    onCoverModalKeydown(e) {
      if (e.key === "Escape" && this.coverModalVisible) this.closeCoverModal();
    },
    openCoverModal() {
      this.coverModalVisible = true;
      this.coverDropHover = false;
      if (!this.coverCaptureAvailable) {
        this.coverModalTab = "upload";
      } else {
        this.coverModalTab = "capture";
      }
      this.$nextTick(() => this.prepareCoverModalVideo());
    },
    closeCoverModal() {
      this.teardownCropDrag();
      const mv = this.$refs.coverModalVideo;
      if (mv) {
        mv.removeAttribute("src");
        mv.load();
      }
      this.coverModalVisible = false;
      this.coverDropHover = false;
      this.teardownStripDrag();
      this.filmstripFrames = [];
      this.coverModalMaxSec = 0;
      this.coverStripSecond = 0;
      this.stripSeekToken = 0;
      this.preview169 = "";
      this.preview43 = "";
    },
    pickCoverFile() {
      this.$refs.coverUploadInput?.click();
    },
    setCoverModalTab(tab) {
      if (tab === "capture" && !this.coverCaptureAvailable) return;
      this.coverModalTab = tab;
      if (tab === "capture") {
        this.$nextTick(() => this.prepareCoverModalVideo());
      }
    },
    async prepareCoverModalVideo() {
      if (!this.coverModalVisible || this.coverModalTab !== "capture") return;
      if (!this.coverCaptureAvailable) return;
      const mv = this.$refs.coverModalVideo;
      if (!mv) return;
      const src = this.videoPreviewSrc;
      if (!src) return;
      if (this.coverModalBusy) return;
      this.coverModalBusy = true;
      try {
        if (/^https?:\/\//i.test(src)) {
          mv.crossOrigin = "anonymous";
        } else {
          mv.removeAttribute("crossorigin");
        }
        mv.src = src;
        mv.load();
        await new Promise((resolve, reject) => {
          const to = window.setTimeout(() => {
            cleanup();
            reject(new Error("timeout"));
          }, 12000);
          const cleanup = () => {
            window.clearTimeout(to);
            mv.removeEventListener("loadedmetadata", onMeta);
            mv.removeEventListener("error", onErr);
          };
          const onMeta = () => {
            cleanup();
            resolve();
          };
          const onErr = () => {
            cleanup();
            reject(new Error("video"));
          };
          mv.addEventListener("loadedmetadata", onMeta);
          mv.addEventListener("error", onErr);
          if (mv.readyState >= 1 && Number.isFinite(mv.duration) && mv.duration > 0) {
            cleanup();
            resolve();
          }
        });
        this.initCropRect169();
        await this.buildFilmstrip();
        if (this.filmstripFrames.length) {
          this.coverStripSecond = 0;
          await this.seekCoverModal(0);
          this.initCropRect169();
          this.flushCoverPreviews();
        }
      } catch {
        /* 解码失败时仍可上传封面 */
      } finally {
        this.coverModalBusy = false;
      }
    },
    onCoverModalMeta() {
      if (!this.cropRect.w) this.initCropRect169();
      this.flushCoverPreviews();
    },
    onCoverModalSeeked() {
      this.flushCoverPreviews();
    },
    initCropRect169() {
      const v = this.$refs.coverModalVideo;
      if (!v?.videoWidth) return;
      const vw = v.videoWidth;
      const vh = v.videoHeight;
      const r = 16 / 9;
      let w = vw;
      let h = vw / r;
      if (h > vh) {
        h = vh;
        w = h * r;
      }
      const x = (vw - w) / 2;
      const y = (vh - h) / 2;
      this.cropRect = { x, y, w, h };
    },
    getVideoDisplayMetrics() {
      const v = this.$refs.coverModalVideo;
      if (!v?.videoWidth) return null;
      const vw = v.videoWidth;
      const vh = v.videoHeight;
      const cw = v.clientWidth;
      const ch = v.clientHeight;
      if (!cw || !ch) return null;
      const scale = Math.min(cw / vw, ch / vh);
      const dw = vw * scale;
      const dh = vh * scale;
      const ox = (cw - dw) / 2;
      const oy = (ch - dh) / 2;
      return { vw, vh, scale, ox, oy, dw, dh, cw, ch };
    },
    cropOverlayStyle() {
      const m = this.getVideoDisplayMetrics();
      if (!m || !this.cropRect.w) return { display: "none" };
      const { scale, ox, oy } = m;
      const { x, y, w, h } = this.cropRect;
      return {
        left: `${ox + x * scale}px`,
        top: `${oy + y * scale}px`,
        width: `${w * scale}px`,
        height: `${h * scale}px`
      };
    },
    onCropDragStart(e) {
      const m = this.getVideoDisplayMetrics();
      if (!m) return;
      this.cropDragSession = {
        startX: e.clientX,
        startY: e.clientY,
        origX: this.cropRect.x,
        origY: this.cropRect.y
      };
      window.addEventListener("mousemove", this.onCropDragMove);
      window.addEventListener("mouseup", this.onCropDragEnd);
    },
    onCropDragMove(e) {
      if (!this.cropDragSession) return;
      const m = this.getVideoDisplayMetrics();
      if (!m) return;
      const dx = (e.clientX - this.cropDragSession.startX) / m.scale;
      const dy = (e.clientY - this.cropDragSession.startY) / m.scale;
      let nx = this.cropDragSession.origX + dx;
      let ny = this.cropDragSession.origY + dy;
      nx = Math.max(0, Math.min(nx, m.vw - this.cropRect.w));
      ny = Math.max(0, Math.min(ny, m.vh - this.cropRect.h));
      this.cropRect = { ...this.cropRect, x: nx, y: ny };
      this.flushCoverPreviews();
    },
    onCropDragEnd() {
      this.teardownCropDrag();
    },
    teardownCropDrag() {
      if (!this.cropDragSession) return;
      this.cropDragSession = null;
      window.removeEventListener("mousemove", this.onCropDragMove);
      window.removeEventListener("mouseup", this.onCropDragEnd);
    },
    seekCoverModal(t) {
      const mv = this.$refs.coverModalVideo;
      if (!mv || !Number.isFinite(mv.duration) || mv.duration <= 0) {
        return Promise.resolve();
      }
      const tt = Math.min(Math.max(0, t), Math.max(0, mv.duration - 0.001));
      return new Promise(resolve => {
        const done = () => {
          mv.removeEventListener("seeked", done);
          resolve();
        };
        mv.addEventListener("seeked", done);
        mv.currentTime = tt;
      });
    },
    grabFrameDataUrl(video) {
      const canvas = document.createElement("canvas");
      canvas.width = video.videoWidth;
      canvas.height = video.videoHeight;
      const ctx = canvas.getContext("2d");
      if (!ctx) return "";
      ctx.drawImage(video, 0, 0);
      return canvas.toDataURL("image/jpeg", 0.72);
    },
    async buildFilmstrip() {
      const mv = this.$refs.coverModalVideo;
      if (!mv?.duration || !Number.isFinite(mv.duration)) {
        this.filmstripFrames = [];
        this.coverModalMaxSec = 0;
        return;
      }
      const dur = mv.duration;
      const maxSec = Math.max(0, Math.floor(dur));
      this.coverModalMaxSec = maxSec;
      const frames = [];
      for (let s = 0; s <= maxSec; s++) {
        await this.seekCoverModal(s);
        frames.push({ t: s, url: this.grabFrameDataUrl(mv) });
        if (s % 12 === 0) {
          await new Promise(r => requestAnimationFrame(r));
        }
      }
      this.filmstripFrames = frames;
    },
    secondFromStripClientX(clientX) {
      const track = this.$refs.stripTrackRef;
      const inner = track?.querySelector(".vp-cmod-strip-inner");
      if (!inner?.offsetWidth || !this.filmstripFrames.length) return 0;
      const rect = inner.getBoundingClientRect();
      const x = clientX - rect.left;
      const p = Math.max(0, Math.min(1, x / inner.offsetWidth));
      const maxSec = this.coverModalMaxSec;
      if (maxSec <= 0) return 0;
      const cells = maxSec + 1;
      return Math.min(maxSec, Math.floor(p * cells));
    },
    onStripTrackMouseDown(e) {
      if (e.button !== 0) return;
      if (e.target.closest(".vp-cmod-strip-frame")) return;
      const sec = this.secondFromStripClientX(e.clientX);
      this.applyStripSecond(sec);
    },
    onStripSelectorMouseDown(e) {
      e.stopPropagation();
      this.stripDragActive = true;
      window.addEventListener("mousemove", this.onStripDragMove);
      window.addEventListener("mouseup", this.onStripDragEnd);
    },
    onStripDragMove(e) {
      if (!this.stripDragActive) return;
      this.applyStripSecond(this.secondFromStripClientX(e.clientX));
    },
    onStripDragEnd() {
      this.teardownStripDrag();
    },
    teardownStripDrag() {
      if (!this.stripDragActive) return;
      this.stripDragActive = false;
      window.removeEventListener("mousemove", this.onStripDragMove);
      window.removeEventListener("mouseup", this.onStripDragEnd);
    },
    async applyStripSecond(sec) {
      const maxSec = this.coverModalMaxSec;
      const s = Math.max(0, Math.min(maxSec, Math.floor(Number(sec))));
      if (s === this.coverStripSecond) return;
      const token = ++this.stripSeekToken;
      this.coverStripSecond = s;
      await this.seekCoverModal(s);
      if (token !== this.stripSeekToken) return;
      this.initCropRect169();
      this.flushCoverPreviews();
    },
    rect43Inside169(x, y, w, h) {
      const innerR = 4 / 3;
      let sw = w;
      let sh = w / innerR;
      if (sh > h) {
        sh = h;
        sw = sh * innerR;
      }
      const sx = x + (w - sw) / 2;
      const sy = y + (h - sh) / 2;
      return { sx, sy, sw, sh };
    },
    flushCoverPreviews() {
      const v = this.$refs.coverModalVideo;
      if (!v?.videoWidth || !this.cropRect.w) {
        this.preview169 = "";
        this.preview43 = "";
        return;
      }
      const { x, y, w, h } = this.cropRect;
      const c169 = document.createElement("canvas");
      c169.width = 320;
      c169.height = 180;
      const ctx169 = c169.getContext("2d");
      if (!ctx169) return;
      ctx169.drawImage(v, x, y, w, h, 0, 0, 320, 180);
      this.preview169 = c169.toDataURL("image/jpeg", 0.88);

      const r43 = this.rect43Inside169(x, y, w, h);
      const c43 = document.createElement("canvas");
      c43.width = 240;
      c43.height = 180;
      const ctx43 = c43.getContext("2d");
      if (!ctx43) return;
      ctx43.drawImage(v, r43.sx, r43.sy, r43.sw, r43.sh, 0, 0, 240, 180);
      this.preview43 = c43.toDataURL("image/jpeg", 0.88);
    },
    applyCoverFromCapture() {
      const v = this.$refs.coverModalVideo;
      if (!v?.videoWidth || !this.cropRect.w) return;
      const { x, y, w, h } = this.cropRect;
      const canvas = document.createElement("canvas");
      canvas.width = Math.round(w);
      canvas.height = Math.round(h);
      const ctx = canvas.getContext("2d");
      if (!ctx) return;
      try {
        ctx.drawImage(v, x, y, w, h, 0, 0, canvas.width, canvas.height);
        this.coverDisplay = canvas.toDataURL("image/jpeg", 0.92);
        this.needCapture = false;
        this.coverDirty = true;
        if (this.isEditMode) this.editDirty = true;
        this.closeCoverModal();
      } catch {
        window.alert("截取失败，请尝试更换时间点或上传封面");
      }
    },
    onCoverUploadChange(e) {
      const f = e.target.files?.[0];
      e.target.value = "";
      this.readCoverImageFile(f);
    },
    onCoverDrop(e) {
      this.coverDropHover = false;
      const f = e.dataTransfer?.files?.[0];
      this.readCoverImageFile(f);
    },
    readCoverImageFile(file) {
      if (!file || !file.type.startsWith("image/")) {
        window.alert("请选择图片文件");
        return;
      }
      const reader = new FileReader();
      reader.onload = () => {
        const url = reader.result;
        if (typeof url === "string") {
          this.coverDisplay = url;
          this.needCapture = false;
          this.coverDirty = true;
          this.closeCoverModal();
          if (this.isEditMode) this.editDirty = true;
        }
      };
      reader.readAsDataURL(file);
    },
    onPickReplace() {
      if (this.warnVideoFileUploadDisabled()) return;
      this.$refs.replaceInput?.click();
    },
    onReplaceChange(e) {
      const f = e.target.files?.[0];
      e.target.value = "";
      if (!f) return;
      if (this.warnVideoFileUploadDisabled()) return;
      this.revokeBlob();
      this.localVideoFile = f;
      this.pendingFileSize = f.size;
      const url = URL.createObjectURL(f);
      this.videoObjectUrl = url;
      this.videoFileName = f.name;
      this.show4KBadge = /4k|2160|uhd/i.test(f.name);
      this.coverDisplay = "";
      this.coverDirty = true;
      this.needCapture = true;
      this.captureScheduled = false;
      this.$nextTick(() => this.attachVideoSrc(url));
      window.setTimeout(() => {
        if (this.needCapture) this.tryCaptureFallback();
      }, 600);
      if (this.isEditMode) this.editDirty = true;
    },
    addTag() {
      const t = this.tagInput.trim();
      if (!t || this.tags.includes(t)) return;
      if (this.tags.length >= this.maxTags) return;
      this.tags.push(t);
      this.tagInput = "";
    },
    removeTag(i) {
      this.tags.splice(i, 1);
    },
    addRecommended(r) {
      if (this.tags.includes(r) || this.tags.length >= this.maxTags) return;
      this.tags.push(r);
    },
    resolveStoredVideoZone(raw) {
      const normalized = normalizeVideoZoneValue(raw);
      if (normalized) {
        return normalized;
      }
      const z = String(raw || "").trim();
      if (isKnownVideoZone(z)) {
        return z;
      }
      return "动画";
    },
    appendVideoZone(fd) {
      const z = this.resolveStoredVideoZone(this.form.zone);
      if (z) {
        this.form.zone = z;
        fd.append("zone", z);
      }
    },
    warnVideoFileUploadDisabled() {
      return guardVideoFileUploadDisabled(msg => {
        ElMessage.warning({ message: msg, duration: 6000 });
      });
    },
    onSaveDraft() {
      if (this.footActionsDisabled) return;
      if (this.isMinibiliApiEnv && (this.isPublishMode || this.isDraftEditMode)) {
        if (this.localVideoFile && this.warnVideoFileUploadDisabled()) return;
        void this.submitMinibiliDraftSave();
        return;
      }
      window.alert("已保存草稿（演示）");
      this.publishCommitted = true;
      this.removeLeaveBackTrapSilent();
    },
    onSubmit() {
      if (this.isMinibiliApiEnv && this.isDraftEditMode) {
        void this.submitMinibiliPublishDraft();
        return;
      }
      if (this.isMinibiliApiEnv && this.isPublishMode) {
        if (this.isMetadataOnlyPublish) {
          void this.submitMinibiliDraftSave();
          return;
        }
        if (this.warnVideoFileUploadDisabled()) return;
        void this.submitMinibiliUpload();
        return;
      }
      if (this.isMinibiliApiEnv && this.isEditMode) {
        if (this.localVideoFile && this.warnVideoFileUploadDisabled()) return;
        void this.submitMinibiliEdit();
        return;
      }
      if (!this.coverDisplay) {
        window.alert("请先设置封面");
        return;
      }
      window.alert("投稿成功（演示）");
      sessionStorage.removeItem(STORAGE_PENDING);
      clearPendingVideoFile();
      this.publishCommitted = true;
      this.editDirty = false;
      this.removeLeaveBackTrapSilent();
      this.$router.push({ name: "manuscript" });
    },
    async submitMinibiliDraftSave(
      opts = { redirect: true, allowWhileSubmitting: false }
    ) {
      if (this.uploadSubmitting && !opts.allowWhileSubmitting) {
        ElMessage.warning("正在保存，请稍候");
        return;
      }
      if (
        this.isMinibiliApiEnv &&
        this.isEditMode &&
        !this.isDraftEditMode &&
        this.editVideoLoading
      ) {
        return;
      }
      const title = (this.form.title || "").trim();
      const desc = (this.form.intro || "").trim();
      const titleLen = Array.from(title).length;
      if (titleLen > 80) {
        ElMessage.warning("标题不能超过 80 个字");
        return;
      }
      if (Array.from(desc).length > 2000) {
        ElMessage.warning("简介不能超过 2000 个字");
        return;
      }
      this.uploadSubmitting = true;
      this.uploadProgress = 0;
      try {
        if (this.isDraftEditMode) {
          const id = this.editVideoNumericId;
          if (!id) {
            this.$router.replace({ name: "manuscript" });
            return;
          }
          if (!this.coverDirty) {
            await this.ensureCoverCapturedIfNeeded();
          }
          let coverFile = null;
          if (this.coverNeedsUpload()) {
            try {
              coverFile = await this.coverDisplayToOptionalCoverFile();
            } catch (err) {
              ElMessage.error((err && err.message) || "封面不符合要求");
              return;
            }
          }
          const needMultipart = !!(this.localVideoFile || coverFile);
          if (!title && !desc && !needMultipart) {
            ElMessage.warning("请至少填写标题或简介");
            return;
          }
          if (needMultipart) {
            const fd = new FormData();
            fd.append("title", title);
            fd.append("description", desc);
            fd.append("tags", JSON.stringify(this.tags || []));
            if (this.localVideoFile) {
              fd.append("file", this.localVideoFile);
            }
            if (coverFile) {
              fd.append("cover", coverFile);
            }
            this.appendVideoZone(fd);
            const res = await mbUpdateVideoDraft(id, fd, {
              onUploadProgress: e => {
                const t = e.total;
                const l = e.loaded;
                if (t && t > 0) {
                  this.uploadProgress = Math.min(99, Math.round((100 * l) / t));
                }
              }
            });
            this.applyServerCoverAfterSave(res.cover_url);
            this.localVideoFile = null;
          } else {
            await mbUpdateVideoDraft(id, {
              title,
              description: desc,
              tags: [...this.tags],
              zone: this.resolveStoredVideoZone(this.form.zone)
            });
          }
          this.markDraftEditSaved();
          if (opts.redirect) {
            ElMessage.success("草稿已保存");
            await this.navigateToManuscriptAfterSave({
              tab: "video",
              status: "draft"
            });
          } else {
            this.clearLeaveBackTrapOnly();
          }
          return;
        }
        if (this.isMetadataOnlyPublish) {
          const titleLen = Array.from(title).length;
          if (titleLen < 1 || titleLen > 80) {
            ElMessage.warning("请先填写标题（1–80 个字）");
            return;
          }
          let coverFile = null;
          if (this.coverNeedsUpload()) {
            try {
              coverFile = await this.coverDisplayToOptionalCoverFile();
            } catch (err) {
              ElMessage.error((err && err.message) || "封面不符合要求");
              return;
            }
          }
          const fd = new FormData();
          fd.append("title", title);
          fd.append("description", desc);
          fd.append("tags", JSON.stringify(this.tags || []));
          if (coverFile) {
            fd.append("cover", coverFile);
          }
          this.appendVideoZone(fd);
          await mbSaveVideoDraft(fd, {
            onUploadProgress: e => {
              const t = e.total;
              const l = e.loaded;
              if (t && t > 0) {
                this.uploadProgress = Math.min(
                  99,
                  Math.round((100 * l) / t)
                );
              }
            }
          });
          sessionStorage.removeItem(STORAGE_METADATA_ONLY);
          this.metadataOnlyMode = false;
          this.markDraftEditSaved();
          if (opts.redirect) {
            ElMessage.success("稿件信息已保存");
            await this.navigateToManuscriptAfterSave({
              tab: "video",
              status: "draft"
            });
          } else {
            this.clearLeaveBackTrapOnly();
          }
          return;
        }
        let videoFile;
        try {
          videoFile = await this.resolveVideoFileForUpload();
        } catch (err) {
          ElMessage.error((err && err.message) || "无法读取视频文件");
          return;
        }
        if (!title && !desc) {
          ElMessage.warning("请至少填写标题或简介");
          return;
        }
        const maxVideo = 500 * 1024 * 1024;
        if (videoFile.size > maxVideo) {
          ElMessage.warning("视频文件须不超过 500 MB");
          return;
        }
        const cap = this.$refs.capVideo;
        if (
          cap &&
          Number.isFinite(cap.duration) &&
          cap.duration > 0 &&
          cap.duration > 30 * 60
        ) {
          ElMessage.warning("视频时长不能超过 30 分钟");
          return;
        }
        await this.ensureCoverCapturedIfNeeded();
        let coverFile = null;
        if (this.coverNeedsUpload()) {
          try {
            coverFile = await this.coverDisplayToOptionalCoverFile();
          } catch (err) {
            ElMessage.error((err && err.message) || "封面不符合要求");
            return;
          }
        }
        const fd = new FormData();
        fd.append("title", title);
        fd.append("description", desc);
        fd.append("tags", JSON.stringify(this.tags || []));
        fd.append("file", videoFile);
        if (coverFile) {
          fd.append("cover", coverFile);
        }
        this.appendVideoZone(fd);
        await mbSaveVideoDraft(fd, {
          onUploadProgress: e => {
            const t = e.total;
            const l = e.loaded;
            if (t && t > 0) {
              this.uploadProgress = Math.min(99, Math.round((100 * l) / t));
            }
          }
        });
        sessionStorage.removeItem(STORAGE_PENDING);
        clearPendingVideoFile();
        this.markDraftEditSaved();
        this.localVideoFile = null;
        this.revokeBlob();
        if (opts.redirect) {
          ElMessage.success("草稿已保存");
          await this.navigateToManuscriptAfterSave({
            tab: "video",
            status: "draft"
          });
        } else {
          this.clearLeaveBackTrapOnly();
        }
      } catch (err) {
        const msg =
          (err &&
            err.response &&
            err.response.data &&
            err.response.data.msg) ||
          (err && err.message) ||
          "保存草稿失败";
        ElMessage.error(String(msg));
        throw err;
      } finally {
        if (!opts.allowWhileSubmitting) {
          this.uploadSubmitting = false;
        }
        this.uploadProgress = 0;
      }
    },
    async submitMinibiliPublishDraft() {
      if (this.uploadSubmitting) return;
      if (this.videoUploadDisabled) {
        ElMessage.warning(
          "当前不支持网页端上传视频，稿件信息已可保存为草稿；发布需管理员线下关联视频后再处理"
        );
        return;
      }
      const id = this.editVideoNumericId;
      if (!id) {
        this.$router.replace({ name: "manuscript" });
        return;
      }
      const title = (this.form.title || "").trim();
      const titleLen = Array.from(title).length;
      if (titleLen < 1 || titleLen > 80) {
        ElMessage.warning("请先填写标题（1–80 个字）");
        return;
      }
      const desc = (this.form.intro || "").trim();
      if (Array.from(desc).length > 2000) {
        ElMessage.warning("简介不能超过 2000 个字");
        return;
      }
      this.uploadSubmitting = true;
      try {
        if (this.editDirty || this.localVideoFile) {
          await this.submitMinibiliDraftSave({
            redirect: false,
            allowWhileSubmitting: true
          });
        }
        await mbPublishVideoDraft(id);
        this.markDraftEditSaved();
        await this.navigateToManuscriptAfterSave({
          tab: "video",
          status: "processing",
          reviewNotice: "1"
        });
      } catch (err) {
        const msg =
          (err &&
            err.response &&
            err.response.data &&
            err.response.data.msg) ||
          (err && err.message) ||
          "投稿失败";
        ElMessage.error(String(msg));
      } finally {
        this.uploadSubmitting = false;
      }
    },
    /**
     * SPEC F2：multipart file + title + description，可选 cover；
     * 与后端 internal/handler/video.go UploadVideo 字段一致。
     */
    async submitMinibiliUpload() {
      if (this.uploadSubmitting) return;
      const title = (this.form.title || "").trim();
      const titleLen = Array.from(title).length;
      if (titleLen < 1 || titleLen > 80) {
        ElMessage.warning("请先填写标题（1–80 个字，与后端校验一致）");
        return;
      }
      const desc = (this.form.intro || "").trim();
      if (Array.from(desc).length > 2000) {
        ElMessage.warning("简介不能超过 2000 个字");
        return;
      }
      const cap = this.$refs.capVideo;
      if (
        cap &&
        Number.isFinite(cap.duration) &&
        cap.duration > 0 &&
        cap.duration > 30 * 60
      ) {
        ElMessage.warning("视频时长不能超过 30 分钟");
        return;
      }
      const maxVideo = 500 * 1024 * 1024;
      if (this.pendingFileSize != null && this.pendingFileSize > maxVideo) {
        ElMessage.warning("视频文件须不超过 500 MB");
        return;
      }
      let videoFile;
      try {
        videoFile = await this.resolveVideoFileForUpload();
      } catch (err) {
        ElMessage.error((err && err.message) || "无法读取视频文件");
        return;
      }
      if (videoFile.size === 0) {
        ElMessage.error("视频文件为空，请从创作中心重新选择或更换视频");
        return;
      }
      if (videoFile.size > maxVideo) {
        ElMessage.warning("视频文件须不超过 500 MB");
        return;
      }
      await this.ensureCoverCapturedIfNeeded();
      let coverFile = null;
      if (this.coverNeedsUpload()) {
        try {
          coverFile = await this.coverDisplayToOptionalCoverFile();
        } catch (err) {
          ElMessage.error((err && err.message) || "封面不符合要求");
          return;
        }
      }
      const fd = new FormData();
      fd.append("title", title);
      fd.append("description", desc);
      fd.append("tags", JSON.stringify(this.tags || []));
      fd.append("file", videoFile);
      if (coverFile) {
        fd.append("cover", coverFile);
      }
      this.appendVideoZone(fd);
      this.uploadSubmitting = true;
      this.uploadProgress = 0;
      this.mbUploadFinishedOk = false;
      try {
        await mbUploadVideo(fd, {
          onUploadProgress: e => {
            const t = e.total;
            const l = e.loaded;
            if (t && t > 0) {
              this.uploadProgress = Math.min(99, Math.round((100 * l) / t));
            } else if (l > 0) {
              this.uploadProgress = Math.max(this.uploadProgress, 1);
            }
          }
        });
        this.uploadProgress = 100;
        this.mbUploadFinishedOk = true;
        await this.$nextTick();
        await new Promise(resolve => {
          requestAnimationFrame(() => {
            requestAnimationFrame(resolve);
          });
        });
        await new Promise(resolve => setTimeout(resolve, 200));
        sessionStorage.removeItem(STORAGE_PENDING);
        clearPendingVideoFile();
        this.publishCommitted = true;
        this.editDirty = false;
        this.localVideoFile = null;
        this.removeLeaveBackTrapSilent();
        this.revokeBlob();
        this.$router.push({
          name: "upload",
          query: { success: "publish" }
        });
      } catch (err) {
        const msg =
          (err &&
            err.response &&
            err.response.data &&
            err.response.data.msg) ||
          (err && err.message) ||
          "上传失败";
        ElMessage.error(String(msg));
      } finally {
        this.uploadSubmitting = false;
        this.uploadProgress = 0;
      }
    },
    async submitMinibiliReplaceMedia() {
      if (this.uploadSubmitting) return;
      const id = this.editVideoNumericId;
      if (!id || !this.localVideoFile) {
        return;
      }
      const title = (this.form.title || "").trim();
      const titleLen = Array.from(title).length;
      if (titleLen < 1 || titleLen > 80) {
        ElMessage.warning("请先填写标题（1–80 个字，与后端校验一致）");
        return;
      }
      const desc = (this.form.intro || "").trim();
      if (Array.from(desc).length > 2000) {
        ElMessage.warning("简介不能超过 2000 个字");
        return;
      }
      const maxVideo = 500 * 1024 * 1024;
      if (this.localVideoFile.size > maxVideo) {
        ElMessage.warning("视频文件须不超过 500 MB");
        return;
      }
      const cap = this.$refs.capVideo;
      if (
        cap &&
        Number.isFinite(cap.duration) &&
        cap.duration > 0 &&
        cap.duration > 30 * 60
      ) {
        ElMessage.warning("视频时长不能超过 30 分钟");
        return;
      }
      await this.ensureCoverCapturedIfNeeded();
      let coverFile = null;
      if (this.coverNeedsUpload()) {
        try {
          coverFile = await this.coverDisplayToOptionalCoverFile();
        } catch (err) {
          ElMessage.error((err && err.message) || "封面不符合要求");
          return;
        }
      }
      const fd = new FormData();
      fd.append("title", title);
      fd.append("description", desc);
      fd.append("tags", JSON.stringify(this.tags || []));
      fd.append("file", this.localVideoFile);
      if (coverFile) {
        fd.append("cover", coverFile);
      }
      this.appendVideoZone(fd);
      this.uploadSubmitting = true;
      this.uploadProgress = 0;
      try {
        await mbReplaceVideoMedia(id, fd, {
          onUploadProgress: e => {
            const t = e.total;
            const l = e.loaded;
            if (t && t > 0) {
              this.uploadProgress = Math.min(99, Math.round((100 * l) / t));
            }
          }
        });
        this.publishCommitted = true;
        this.editDirty = false;
        this.localVideoFile = null;
        this.revokeBlob();
        ElMessage.success("已提交，视频转码处理中");
        await this.navigateToManuscriptAfterSave({
          tab: "video",
          status: "processing",
          reviewNotice: "1"
        });
      } catch (err) {
        const msg =
          (err &&
            err.response &&
            err.response.data &&
            err.response.data.msg) ||
          (err && err.message) ||
          "更换视频失败";
        ElMessage.error(String(msg));
      } finally {
        this.uploadSubmitting = false;
        this.uploadProgress = 0;
      }
    },
    async submitMinibiliEdit() {
      if (this.uploadSubmitting) return;
      const id = this.editVideoNumericId;
      if (!id) {
        this.$router.replace({ name: "manuscript" });
        return;
      }
      if (this.isRejectedOrFailedEditMode && this.localVideoFile) {
        void this.submitMinibiliReplaceMedia();
        return;
      }
      const title = (this.form.title || "").trim();
      const titleLen = Array.from(title).length;
      if (titleLen < 1 || titleLen > 80) {
        ElMessage.warning("请先填写标题（1–80 个字，与后端校验一致）");
        return;
      }
      const desc = (this.form.intro || "").trim();
      if (Array.from(desc).length > 2000) {
        ElMessage.warning("简介不能超过 2000 个字");
        return;
      }
      this.uploadSubmitting = true;
      try {
        await mbUpdateMyVideo(id, {
          title,
          description: desc,
          tags: [...this.tags],
          zone: this.resolveStoredVideoZone(this.form.zone)
        });
        const published = this.editVideoStatus === "published";
        const curCover = (this.coverDisplay || "").trim();
        const srcCover = (this.editSourceCoverUrl || "").trim();
        if (published && curCover && curCover !== srcCover) {
          try {
            const coverFile = await this.coverDisplayToOptionalCoverFile();
            if (coverFile) {
              await mbUpdateVideoCover(id, coverFile);
              this.editSourceCoverUrl = curCover;
            }
          } catch (coverErr) {
            const cmsg =
              (coverErr && coverErr.message) ||
              "封面未同步（常为跨域或网络问题），标题与简介已保存";
            ElMessage.warning(String(cmsg));
          }
        }
        this.publishCommitted = true;
        this.editDirty = false;
        ElMessage({
          message: "保存成功",
          type: "success",
          duration: 2200,
          showClose: true
        });
        const apiSt = String(this.editVideoStatus || "").trim();
        const query = { tab: "video" };
        if (apiSt === "published") {
          query.status = "passed";
        } else if (apiSt === "failed" || apiSt === "rejected") {
          query.status = "rejected";
        } else if (
          apiSt === "draft" ||
          apiSt === "processing" ||
          apiSt === "pending_review"
        ) {
          query.status = "processing";
        }
        await this.navigateToManuscriptAfterSave(query);
      } catch (err) {
        const msg =
          (err &&
            err.response &&
            err.response.data &&
            err.response.data.msg) ||
          (err && err.message) ||
          "保存失败";
        ElMessage.error(String(msg));
      } finally {
        this.uploadSubmitting = false;
      }
    },
    async resolveVideoFileForUpload() {
      if (this.localVideoFile) {
        return this.localVideoFile;
      }
      if (!this.videoObjectUrl) {
        throw new Error("未找到视频文件，请从创作中心重新选择");
      }
      const res = await fetch(this.videoObjectUrl);
      if (!res.ok) {
        throw new Error("无法读取本地视频");
      }
      const blob = await res.blob();
      const name = this.videoFileName || "video";
      return new File([blob], name, {
        type: blob.type || "video/mp4"
      });
    },
    markDraftEditSaved() {
      this.publishCommitted = true;
      this.editDirty = false;
      this.coverDirty = false;
    },
    applyServerCoverAfterSave(coverUrl) {
      const url = String(coverUrl || "").trim();
      if (!url) return;
      this.skipEditDirty = true;
      this.coverDisplay = url;
      this.editSourceCoverUrl = url;
      this.coverDirty = false;
      this.$nextTick(() => {
        this.skipEditDirty = false;
      });
    },
    /** 本地新封面（data/blob）或用户主动修改过封面时才需上传 */
    coverNeedsUpload() {
      if (this.coverDirty) return true;
      const cur = (this.coverDisplay || "").trim();
      if (!cur) return false;
      if (cur.startsWith("data:") || cur.startsWith("blob:")) return true;
      const orig = (this.editSourceCoverUrl || "").trim();
      return cur !== orig;
    },
    async ensureCoverCapturedIfNeeded() {
      if ((this.coverDisplay || "").trim()) return;
      if (!this.coverCaptureAvailable) return;
      const v = this.$refs.capVideo;
      if (!v) return;
      if (v.readyState < 2) {
        await new Promise(resolve => {
          const onReady = () => {
            v.removeEventListener("loadeddata", onReady);
            resolve();
          };
          v.addEventListener("loadeddata", onReady, { once: true });
          window.setTimeout(resolve, 2500);
        });
      }
      this.needCapture = true;
      this.tryCaptureFallback();
      if (!(this.coverDisplay || "").trim()) {
        this.captureDefaultCoverFromVideo();
      }
      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));
    },
    blobToCoverFile(blob) {
      const maxCover = 10 * 1024 * 1024;
      if (blob.size > maxCover) {
        throw new Error("封面大小超过 10MB");
      }
      const mime = blob.type || "";
      const byMime = {
        "image/jpeg": "cover.jpg",
        "image/jpg": "cover.jpg",
        "image/png": "cover.png",
        "image/gif": "cover.gif",
        "image/bmp": "cover.bmp",
        "image/x-ms-bmp": "cover.bmp",
        "image/webp": "cover.webp"
      };
      const fname = byMime[mime];
      if (!fname) {
        throw new Error("封面格式不支持（请使用 JPEG、PNG、GIF、BMP、WEBP）");
      }
      return new File([blob], fname, { type: mime || "image/jpeg" });
    },
    dataUrlToCoverFile(dataUrl) {
      const m = dataUrl.match(/^data:([^;,]+)(?:;[^,]*)?;base64,(.+)$/i);
      if (!m) {
        throw new Error("无法解析封面图片数据");
      }
      const mime = m[1].trim().toLowerCase();
      const b64 = m[2];
      const raw = atob(b64);
      const u8 = new Uint8Array(raw.length);
      for (let i = 0; i < raw.length; i++) u8[i] = raw.charCodeAt(i);
      return this.blobToCoverFile(new Blob([u8], { type: mime }));
    },
    /**
     * SPEC F2 可选 cover；与 internal/pkg/coverval 扩展名校验一致。
     */
    async coverDisplayToOptionalCoverFile() {
      const src = (this.coverDisplay || "").trim();
      if (!src) {
        return null;
      }
      if (src.startsWith("data:")) {
        return this.dataUrlToCoverFile(src);
      }
      if (src.startsWith("blob:")) {
        const res = await fetch(src);
        if (!res.ok) throw new Error("无法读取封面");
        return this.blobToCoverFile(await res.blob());
      }
      if (/^https?:\/\//i.test(src)) {
        const orig = (this.editSourceCoverUrl || "").trim();
        if (orig && src === orig) {
          return null;
        }
        const ctrl = new AbortController();
        const tid = setTimeout(() => ctrl.abort(), 30000);
        try {
          const res = await fetch(src, { signal: ctrl.signal, mode: "cors" });
          if (!res.ok) throw new Error("无法读取封面");
          return this.blobToCoverFile(await res.blob());
        } catch (e) {
          if (e && e.name === "AbortError") {
            throw new Error("读取封面超时，请检查网络或更换封面文件");
          }
          throw new Error(
            "无法读取网络封面，请重新上传封面或使用截取封面"
          );
        } finally {
          clearTimeout(tid);
        }
      }
      throw new Error("封面地址无效，请重新设置封面");
    }
  }
};
</script>

<style lang="scss" scoped>
$c-blue: #00a1d6;
$c-text: #18191c;
$c-sub: #9499a0;
$c-line: #e3e5e7;

.vp-page {
  max-width: 1040px;
  margin: 0 auto;
}

.vp-edit-loading {
  margin: 48px 0;
  text-align: center;
  font-size: 15px;
  color: $c-sub;
}

.vp-replace:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.vp-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.vp-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: $c-text;
}

.vp-part-bar {
  margin-bottom: 16px;
}

.vp-btn-add-part {
  height: 30px;
  padding: 0 14px;
  border: none;
  border-radius: 4px;
  background: $c-blue;
  color: #fff;
  font-size: 13px;
  cursor: pointer;
}
.vp-btn-add-part:hover {
  background: #008ebd;
}

.vp-card {
  background: #fff;
  border: 1px solid $c-line;
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 20px;
  box-sizing: border-box;
}

.vp-upload-panel {
  background: #fafbfc;
  border-radius: 6px;
  padding: 14px 16px 12px;
  box-sizing: border-box;
}

.vp-ico-complete {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  object-fit: contain;
  vertical-align: middle;
}

.vp-file-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  font-size: 13px;
  color: #505050;
}

.vp-file-main {
  display: flex;
  align-items: stretch;
  gap: 10px;
  flex: 1;
  min-width: 0;
}

.vp-file-ico {
  display: flex;
  align-items: center;
  align-self: stretch;
  flex-shrink: 0;
}

.vp-file-ico-img {
  height: 100%;
  width: auto;
  max-width: 42px;
  object-fit: contain;
  display: block;
}

.vp-file-col {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  flex: 1;
}

.vp-file-title-line {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.vp-file-name {
  word-break: break-all;
  line-height: 1.45;
}

.vp-file-status-line {
  display: flex;
  align-items: center;
  gap: 6px;
}

.vp-file-status-txt {
  font-size: 13px;
  color: #99a2aa;
  line-height: 16px;
}

.vp-tag-4k {
  padding: 1px 6px;
  font-size: 11px;
  font-weight: 600;
  color: #fff;
  background: #00a1d6;
  border-radius: 2px;
}

.vp-replace {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  align-self: center;
  border: none;
  background: none;
  padding: 0;
  font-size: 13px;
  color: $c-blue;
  cursor: pointer;
}

.vp-ico-update {
  width: 22px;
  height: 22px;
  flex-shrink: 0;
  object-fit: contain;
}
.vp-replace:hover {
  text-decoration: underline;
}
.vp-replace:hover .vp-ico-update {
  opacity: 0.85;
}

.vp-progress-wrap {
  margin-top: 12px;
}
.vp-progress-track {
  height: 4px;
  border-radius: 2px;
  background: #e3e5e7;
  overflow: hidden;
}
.vp-progress-fill {
  height: 100%;
  width: 0;
  border-radius: 2px;
  background: #00aeec;
  transition:
    width 0.18s ease-out,
    background 0.2s ease;
}
.vp-progress-wrap--mb-upload-done .vp-progress-track {
  background: #43ce5b;
}
.vp-progress-wrap--mb-upload-done .vp-progress-fill {
  width: 100% !important;
  background: #43ce5b;
}
.vp-progress-fill--indeterminate {
  width: 35% !important;
  background: #00aeec;
  animation: vp-progress-indet 1.1s ease-in-out infinite;
}

@keyframes vp-progress-indet {
  0% {
    transform: translateX(-100%);
    opacity: 0.85;
  }
  50% {
    opacity: 1;
  }
  100% {
    transform: translateX(320%);
    opacity: 0.85;
  }
}

.vp-hidden-input {
  position: absolute;
  width: 0;
  height: 0;
  opacity: 0;
  pointer-events: none;
}

.vp-cap-video {
  position: fixed;
  left: -9999px;
  top: 0;
  width: 640px;
  height: 360px;
  opacity: 0;
  pointer-events: none;
}

.vp-sec-head {
  margin-bottom: 24px;
}

.vp-sec-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: $c-text;
  padding-left: 10px;
  border-left: 4px solid $c-blue;
}

.vp-field {
  margin-bottom: 22px;
}

.vp-field-row {
  display: grid;
  grid-template-columns: 140px minmax(0, 1fr);
  column-gap: 16px;
  align-items: start;
}

/* 标题 / 类型 / 分区：标签与控件垂直居中 */
.vp-field-row--middle {
  align-items: center;
}

.vp-field-row--middle > :nth-child(2) {
  align-self: center;
}

/* 封面：标签与封面区域垂直居中 */
.vp-field-row--cover {
  align-items: center;
}

.vp-field-row--cover > .vp-label {
  align-self: center;
}

/* 标签、简介：标签顶对齐，并与首行内容对齐 */
.vp-field-row--tags {
  align-items: start;
}

.vp-field-row--tags > .vp-label {
  padding-top: 8px;
}

.vp-field-row--intro {
  align-items: start;
}

.vp-field-row--intro > .vp-label {
  padding-top: 10px;
}

.vp-label {
  margin: 0;
  box-sizing: border-box;
  font-size: 14px;
  color: $c-text;
  line-height: 20px;
  text-align: right;
  white-space: nowrap;
  justify-self: end;
  width: fit-content;
  max-width: 140px;
}

.vp-field-row--middle .vp-label {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0;
  min-height: 36px;
}

.vp-tags-body {
  min-width: 0;
  width: 100%;
}

.vp-req {
  color: #f5222d;
  margin-right: 0;
}

.vp-cover-block {
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
  align-items: flex-start;
  min-width: 0;
}

.vp-cover-main {
  position: relative;
  width: 280px;
  height: 158px;
  border-radius: 6px;
  overflow: hidden;
  background: #f4f5f7;
  cursor: pointer;
}

.vp-cover-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.vp-cover-ph {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  color: $c-sub;
}

.vp-cover-mask {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 24px 8px 10px;
  background: linear-gradient(transparent, rgba(0, 0, 0, 0.55));
  display: flex;
  justify-content: center;
  pointer-events: none;
}

.vp-cover-btn {
  font-size: 13px;
  color: #fff;
  padding: 4px 14px;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.35);
}

/* —— 封面设置弹窗（主站蓝） —— */

.vp-cmod-root {
  position: fixed;
  inset: 0;
  z-index: 5200;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px 16px;
  box-sizing: border-box;
}

.vp-cmod-backdrop {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
}

.vp-cmod-dialog {
  position: relative;
  width: min(920px, 96vw);
  max-height: min(640px, 92vh);
  overflow: auto;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 12px 48px rgba(0, 0, 0, 0.18);
  box-sizing: border-box;
}

.vp-cmod-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px 0;
  border-bottom: 1px solid #e3e5e7;
}

.vp-cmod-tabs {
  display: flex;
  gap: 24px;
}

.vp-cmod-tab {
  border: none;
  background: none;
  padding: 10px 2px 12px;
  font-size: 15px;
  color: #9499a0;
  cursor: pointer;
  position: relative;
}
.vp-cmod-tab:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.vp-cmod-tab--active {
  color: $c-blue;
  font-weight: 600;
}
.vp-cmod-tab--active::after {
  content: "";
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 2px;
  background: $c-blue;
  border-radius: 1px;
}

.vp-cmod-close {
  border: none;
  background: none;
  font-size: 22px;
  line-height: 1;
  color: #9499a0;
  cursor: pointer;
  padding: 4px 8px;
}
.vp-cmod-close:hover {
  color: #18191c;
}

.vp-cmod-body {
  padding: 16px 20px 20px;
}

.vp-cmod-empty {
  font-size: 14px;
  color: $c-sub;
  line-height: 1.6;
  padding: 12px 0;
}

.vp-cmod-cap-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 200px;
  gap: 20px;
  align-items: start;
}

@media (max-width: 720px) {
  .vp-cmod-cap-grid {
    grid-template-columns: 1fr;
  }
}

.vp-cmod-cap-stage {
  position: relative;
  width: 100%;
  max-width: 520px;
  aspect-ratio: 16 / 9;
  margin: 0 auto;
  background: #000;
  border-radius: 4px;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
}

.vp-cmod-cap-video {
  width: 100%;
  height: 100%;
  object-fit: contain;
  vertical-align: middle;
}

.vp-cmod-crop {
  position: absolute;
  box-sizing: border-box;
  border: 2px solid #fff;
  border-radius: 2px;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.35) inset, 0 0 0 9999px rgba(0, 0, 0, 0.25);
  cursor: move;
  pointer-events: auto;
}

.vp-cmod-res-hint {
  margin: 10px 0 4px;
  font-size: 13px;
  color: $c-text;
}

.vp-cmod-ratio-hint {
  margin: 0;
  font-size: 12px;
  color: $c-sub;
}

.vp-cmod-crop-hint {
  margin: 8px 0 0;
  font-size: 12px;
  color: $c-sub;
  line-height: 1.45;
}

.vp-cmod-prev-col {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.vp-cmod-prev-label {
  font-size: 12px;
  color: $c-sub;
  margin-bottom: 6px;
}

.vp-cmod-prev-box {
  background: #f4f5f7;
  border-radius: 4px;
  overflow: hidden;
  border: 1px solid #e3e5e7;
}
.vp-cmod-prev-box--169 {
  aspect-ratio: 16 / 9;
}
.vp-cmod-prev-box--43 {
  aspect-ratio: 4 / 3;
}

.vp-cmod-prev-stack {
  position: relative;
  width: 100%;
  height: 100%;
  min-height: 0;
}

.vp-cmod-prev-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  vertical-align: top;
}

.vp-cmod-prev-title-overlay {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 14px 8px 7px;
  font-size: 11px;
  line-height: 1.35;
  font-weight: 500;
  color: #fff;
  background: linear-gradient(transparent, rgba(0, 0, 0, 0.78));
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.45);
  pointer-events: none;
  box-sizing: border-box;
}

.vp-cmod-prev-title-overlay--43 {
  padding: 10px 6px 5px;
  font-size: 10px;
  -webkit-line-clamp: 2;
}

.vp-cmod-prev-title-overlay--ph {
  color: rgba(255, 255, 255, 0.72);
  font-weight: 400;
}

.vp-cmod-strip-wrap {
  margin-top: 12px;
  width: 100%;
  max-width: 520px;
  box-sizing: border-box;
}

.vp-cmod-strip-loading {
  margin: 0;
  padding: 10px 4px;
  font-size: 12px;
  color: $c-sub;
}

.vp-cmod-strip-track {
  position: relative;
  overflow: hidden;
  padding-top: 30px;
  border: 1px solid #e3e5e7;
  border-radius: 6px;
  background: #fff;
  box-sizing: border-box;
}

.vp-cmod-strip-inner {
  display: flex;
  flex-direction: row;
  flex-wrap: nowrap;
  width: 100%;
  height: 52px;
  border-radius: 0 0 5px 5px;
  overflow: hidden;
  background: #0d0d0d;
}

.vp-cmod-strip-thumb {
  flex: 1 1 0;
  min-width: 0;
  height: 52px;
  margin: 0;
  padding: 0;
  overflow: hidden;
}

.vp-cmod-strip-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  vertical-align: top;
}

.vp-cmod-strip-select-wrap {
  position: absolute;
  top: 30px;
  bottom: auto;
  height: 52px;
  box-sizing: border-box;
  pointer-events: none;
  z-index: 2;
}

.vp-cmod-strip-tooltip {
  position: absolute;
  bottom: calc(100% + 6px);
  left: 50%;
  transform: translateX(-50%);
  padding: 4px 10px;
  background: #4e5969;
  color: #fff;
  font-size: 12px;
  font-weight: 500;
  line-height: 1.2;
  border-radius: 4px;
  white-space: nowrap;
  box-shadow: 0 1px 6px rgba(0, 0, 0, 0.18);
  pointer-events: none;
}

.vp-cmod-strip-tooltip::after {
  content: "";
  position: absolute;
  top: 100%;
  left: 50%;
  transform: translateX(-50%);
  border: 5px solid transparent;
  border-top-color: #4e5969;
}

.vp-cmod-strip-frame {
  position: absolute;
  inset: 0;
  box-sizing: border-box;
  border: 3px solid #fff;
  border-radius: 4px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.28);
  cursor: grab;
  pointer-events: auto;
}

.vp-cmod-strip-frame:active {
  cursor: grabbing;
}

.vp-cmod-actions {
  display: flex;
  justify-content: center;
  margin-top: 18px;
}

.vp-cmod-btn-done {
  min-width: 160px;
  height: 40px;
  padding: 0 24px;
  border: none;
  border-radius: 4px;
  background: $c-blue;
  color: #fff;
  font-size: 15px;
  font-weight: 500;
  cursor: pointer;
}
.vp-cmod-btn-done:hover {
  background: #008ebd;
}

.vp-cmod-body--upload {
  padding: 24px 28px 32px;
}

.vp-cmod-drop {
  border: 1px dashed #ccd0d7;
  border-radius: 8px;
  padding: 48px 24px 36px;
  text-align: center;
  background: #fafbfc;
  transition: border-color 0.15s, background 0.15s;
}
.vp-cmod-drop--hover {
  border-color: rgba(0, 161, 214, 0.55);
  background: #f5fbfd;
}

.vp-cmod-drop-ico {
  font-size: 40px;
  line-height: 1;
  opacity: 0.35;
  margin-bottom: 12px;
}

.vp-cmod-drop-txt {
  margin: 0 0 20px;
  font-size: 14px;
  color: $c-sub;
}

.vp-cmod-btn-pick {
  min-width: 140px;
  height: 38px;
  padding: 0 20px;
  border: none;
  border-radius: 4px;
  background: $c-blue;
  color: #fff;
  font-size: 14px;
  cursor: pointer;
}
.vp-cmod-btn-pick:hover {
  background: #008ebd;
}

.vp-input-wrap {
  position: relative;
  min-width: 0;
  width: 100%;
}

.vp-input {
  width: 100%;
  height: 36px;
  padding: 0 52px 0 12px;
  border: 1px solid #ccd0d7;
  border-radius: 4px;
  font-size: 14px;
  box-sizing: border-box;
  outline: none;
}
.vp-input:focus {
  border-color: rgba(0, 161, 214, 0.55);
}

.vp-counter {
  position: absolute;
  right: 10px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 12px;
  color: $c-sub;
}

.vp-radios {
  display: flex;
  gap: 24px;
  align-items: center;
  height: 36px;
}

.vp-radio {
  font-size: 14px;
  color: $c-text;
  cursor: pointer;
}
.vp-radio input {
  margin-right: 6px;
  vertical-align: middle;
}

.vp-select {
  width: 100%;
  max-width: 360px;
  height: 36px;
  padding: 0 12px;
  border: 1px solid #ccd0d7;
  border-radius: 4px;
  font-size: 14px;
  background: #fff;
  outline: none;
}

.vp-tag-box {
  border: 1px solid #ccd0d7;
  border-radius: 4px;
  padding: 8px 10px;
  min-height: 72px;
  box-sizing: border-box;
}

.vp-tag-inner {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.vp-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  background: rgba(0, 161, 214, 0.12);
  color: $c-blue;
  border-radius: 4px;
  font-size: 13px;
}

.vp-tag-x {
  border: none;
  background: none;
  padding: 0;
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  color: $c-blue;
}

.vp-tag-inp {
  flex: 1;
  min-width: 160px;
  border: none;
  outline: none;
  font-size: 14px;
  height: 28px;
}

.vp-tag-cap {
  display: block;
  text-align: right;
  font-size: 12px;
  color: $c-sub;
  margin-top: 8px;
}

.vp-rec-tags {
  margin-top: 12px;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.vp-rec-label {
  font-size: 13px;
  color: $c-sub;
}

.vp-rec-chip {
  padding: 4px 10px;
  border: 1px solid $c-line;
  border-radius: 4px;
  background: #fafafa;
  font-size: 12px;
  color: #505050;
  cursor: pointer;
}
.vp-rec-chip:hover {
  border-color: $c-blue;
  color: $c-blue;
}

.vp-ta-wrap {
  position: relative;
  min-width: 0;
  width: 100%;
}

.vp-textarea {
  width: 100%;
  padding: 10px 12px 28px;
  border: 1px solid #ccd0d7;
  border-radius: 4px;
  font-size: 14px;
  line-height: 1.6;
  resize: vertical;
  box-sizing: border-box;
  outline: none;
  font-family: inherit;
}
.vp-textarea:focus {
  border-color: rgba(0, 161, 214, 0.55);
}

.vp-counter-ta {
  position: absolute;
  right: 10px;
  bottom: 8px;
  transform: none;
}

.vp-foot {
  display: flex;
  justify-content: center;
  gap: 16px;
  padding: 32px 0 16px;
}

.vp-btn {
  min-width: 140px;
  height: 42px;
  padding: 0 28px;
  border-radius: 4px;
  font-size: 15px;
  font-weight: 500;
  cursor: pointer;
  border: none;
}

.vp-btn-ghost {
  background: #fff;
  border: 1px solid #ccd0d7;
  color: #505050;
}
.vp-btn-ghost:hover {
  border-color: $c-blue;
  color: $c-blue;
}

.vp-btn-primary {
  background: $c-blue;
  color: #fff;
}
.vp-btn-primary:hover {
  background: #008ebd;
}

.vp-btn-wide {
  min-width: 200px;
}
</style>

<style lang="scss">
/* MessageBox is teleported — not scoped */
.vp-leave-msgbox.el-message-box {
  border-radius: 6px;
  padding-bottom: 14px;
  width: auto;
  min-width: 260px;
  max-width: min(360px, 92vw);

  .el-message-box__header {
    padding: 12px 16px 0;
  }

  .el-message-box__headerbtn {
    display: none;
  }

  .el-message-box__title {
    justify-content: center;
    font-size: 13px;
    font-weight: 600;
    color: #18191c;
    line-height: 1.4;
  }

  .el-message-box__content {
    padding: 8px 16px 4px;
  }

  .el-message-box__message {
    text-align: center;
    font-size: 12px;
    line-height: 1.5;
    color: #9499a0;
  }

  .el-message-box__message p {
    margin: 0;
  }

  .el-message-box__btns {
    justify-content: center;
    gap: 8px;
    padding: 8px 12px 0;
    flex-direction: row-reverse;
  }

  .el-message-box__btns .el-button {
    font-size: 12px;
    padding: 5px 14px;
    min-height: 30px;
    border-radius: 4px;
  }

  .el-button--primary {
    --el-button-bg-color: #00a1d6;
    --el-button-border-color: #00a1d6;
    --el-button-hover-bg-color: #008ebd;
    --el-button-hover-border-color: #008ebd;
    min-width: 72px;
  }

  .el-button:not(.el-button--primary) {
    min-width: 72px;
    background: #fff;
    border-color: #e3e5e7;
    color: #18191c;
  }

  .el-button:not(.el-button--primary):hover {
    border-color: #00a1d6;
    color: #00a1d6;
    background: #fff;
  }
}
</style>
