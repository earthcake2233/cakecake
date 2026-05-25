<template>
  <div ref="rootRef" class="vp-root">
    <!-- 顶栏：热点文案（可跳转）+ 切集占位 -->
    <div class="vp-topbar">
      <button type="button" class="vp-pl-arrow vp-pl-arrow--prev" aria-label="上一个">
        ‹
      </button>
      <router-link class="vp-hot-link" :to="hotRoute">{{ hotTitle }}</router-link>
      <button type="button" class="vp-pl-arrow vp-pl-arrow--next" aria-label="下一个">
        ›
      </button>
    </div>

    <div
      ref="videoStageRef"
      class="vp-video-stage notranslate"
      translate="no"
      @dblclick.stop.prevent="onVideoStageDblclick"
    >
      <video
        ref="videoRef"
        class="vp-video"
        translate="no"
        playsinline
        :loop="loopEnabled"
        :muted="muted"
        :src="videoSrc"
        @timeupdate="onTimeUpdate"
        @progress="onProgress"
        @loadedmetadata="onMeta"
        @loadstart="onVideoLoadStart"
        @canplay="onVideoCanPlay"
        @error="onVideoError"
        @play="playing = true"
        @pause="playing = false"
        @click.prevent="togglePlay"
      />
      <canvas
        v-show="minibiliDanmakuOverlayVisible"
        ref="dmCanvasRef"
        class="vp-dm-canvas"
        aria-hidden="true"
      />

      <div
        v-show="videoLoading"
        class="vp-load-mask"
        role="status"
        aria-live="polite"
        :aria-busy="videoLoading"
      >
        <img
          class="vp-load-gif"
          :src="imgLoading"
          width="100"
          height="100"
          alt=""
        />
        <span class="vp-load-status">加载视频地址... 完成</span>
      </div>

      <transition name="vp-hint-fade">
        <div
          v-if="shortcutHint"
          class="vp-shortcut-hint"
          role="status"
          aria-live="polite"
        >
          {{ shortcutHint }}
        </div>
      </transition>

      <button
        v-show="!playing && !videoLoading"
        type="button"
        class="vp-paused-corner-play"
        aria-label="播放"
        @click.stop.prevent="togglePlay"
      >
        <img
          class="vp-paused-corner-play-img"
          :src="imgPausedCornerPlay"
          width="56"
          height="56"
          alt=""
        />
      </button>

      <div class="vp-controls">
        <div class="vp-controls-row">
          <button
            type="button"
            class="vp-btn vp-play"
            :aria-label="playing ? '暂停' : '播放'"
            @click.stop="togglePlay"
          >
            <!-- 资源命名与语义相反：暂停态用 stop.png（实为三角播放）、播放中用 play.png（实为暂停条）；三角图标偏小故放大 -->
            <img
              class="vp-play-img"
              :class="{ 'vp-play-img--triangle': !playing }"
              :src="playing ? imgPlay : imgStop"
              alt=""
            />
          </button>

          <div
            ref="trackRef"
            class="vp-progress"
            role="slider"
            tabindex="0"
            aria-label="进度"
            @click.stop="onSeekClick"
          >
            <div class="vp-progress-bg" />
            <div class="vp-progress-buffer" :style="{ width: bufferPct + '%' }" />
            <div class="vp-progress-played" :style="{ width: playedPct + '%' }" />
            <div class="vp-progress-thumb" :style="{ left: playedPct + '%' }" />
          </div>

          <span class="vp-time">{{ timeLabel }}</span>

          <div
            class="vp-volume-wrap"
            @mouseenter="onVolumeEnter"
            @mouseleave="onVolumeLeave"
          >
            <span class="vp-cluster-inline">
              <button
                type="button"
                class="vp-btn"
                :aria-label="muted ? '取消静音' : '静音'"
                @click.stop="toggleMute"
              >
                <img :src="muted ? imgSilence : imgVoice" alt="" />
              </button>
            </span>
            <div
              v-show="volumePopoverShow"
              class="vp-volume-pop"
              @click.stop
            >
              <div class="vp-volume-num">{{ volumeDisplayPercent }}</div>
              <div
                ref="volumeTrackRef"
                class="vp-vtrack"
                role="slider"
                tabindex="0"
                aria-label="音量"
                aria-valuemin="0"
                aria-valuemax="100"
                :aria-valuenow="volumeDisplayPercent"
                @pointerdown.stop.prevent="onVolumePointerDown"
                @pointermove="onVolumePointerMove"
                @pointerup="onVolumePointerUp"
                @pointercancel="onVolumePointerUp"
                @lostpointercapture="onVolumeLostCapture"
              >
                <div
                  class="vp-vtrack-fill"
                  :style="{ height: volumeDisplayPercent + '%' }"
                />
                <div
                  class="vp-vthumb"
                  :style="{ bottom: `calc(${volumeDisplayPercent}% - 7px)` }"
                />
              </div>
            </div>
          </div>

          <div
            class="vp-quality"
            @mouseenter="onQualityEnter"
            @mouseleave="onQualityLeave"
          >
            <button
              type="button"
              class="vp-quality-trigger"
              @click.stop="qualityOpen = !qualityOpen"
            >
              {{ qualityShort }}
            </button>
            <div v-show="qualityOpen" class="vp-quality-menu" @click.stop>
              <button
                v-for="q in qualities"
                :key="q.key"
                type="button"
                class="vp-quality-item"
                :class="{ on: selectedQuality === q.key }"
                @click.stop="pickQuality(q.key)"
              >
                {{ q.label }}
              </button>
            </div>
          </div>

          <div
            class="vp-danmu-cluster"
            @mouseenter="onDanmuClusterEnter"
            @mouseleave="onDanmuClusterLeave"
          >
            <span class="vp-cluster-inline">
              <button
                type="button"
                class="vp-btn"
                :aria-label="danmuOn ? '关闭弹幕' : '打开弹幕'"
                @click.stop="toggleDanmu"
              >
                <img :src="danmuOn ? imgCommentOn : imgCommentOff" alt="" />
              </button>
              <span class="vp-hover-tip">{{
                danmuOn ? "关闭弹幕" : "打开弹幕"
              }}</span>
            </span>

            <div v-show="danmuPanelShow" class="vp-danmu-panel">
              <div class="vp-danmu-slider-row">
                <span class="vp-danmu-label">不透明度</span>
                <input
                  v-model.number="danmuOpacity"
                  type="range"
                  min="0"
                  max="100"
                  class="vp-range"
                  :style="danmuSliderTrackStyle(danmuOpacity)"
                />
              </div>
              <div class="vp-danmu-slider-row">
                <span class="vp-danmu-label">弹幕密度</span>
                <input
                  v-model.number="danmuDensity"
                  type="range"
                  min="0"
                  max="100"
                  class="vp-range"
                  :style="danmuSliderTrackStyle(danmuDensity)"
                />
              </div>
              <div class="vp-danmu-slider-row">
                <span class="vp-danmu-label">显示区域</span>
                <input
                  v-model.number="danmuArea"
                  type="range"
                  min="0"
                  max="100"
                  class="vp-range"
                  :style="danmuSliderTrackStyle(danmuArea)"
                />
              </div>
              <label class="vp-danmu-check">
                <input v-model="danmuAntiSubtitle" type="checkbox" />
                <span>防挡字幕</span>
              </label>
              <div class="vp-danmu-types">
                <button
                  type="button"
                  class="vp-danmu-type"
                  :class="{ off: !typeTop }"
                  @click.stop="typeTop = !typeTop"
                >
                  <img :src="imgDanmu1" alt="顶端弹幕" />
                  <span>顶端弹幕</span>
                </button>
                <button
                  type="button"
                  class="vp-danmu-type"
                  :class="{ off: !typeBottom }"
                  @click.stop="typeBottom = !typeBottom"
                >
                  <img :src="imgDanmu3" alt="底端弹幕" />
                  <span>底端弹幕</span>
                </button>
                <button
                  type="button"
                  class="vp-danmu-type"
                  :class="{ off: !typeScroll }"
                  @click.stop="typeScroll = !typeScroll"
                >
                  <img :src="imgDanmu2" alt="滚动弹幕" />
                  <span>滚动弹幕</span>
                </button>
              </div>
            </div>
          </div>

          <span class="vp-cluster-inline">
            <button
              type="button"
              class="vp-btn"
              :aria-label="loopEnabled ? '关闭洗脑循环' : '打开洗脑循环'"
              @click.stop="loopEnabled = !loopEnabled"
            >
              <img :src="loopEnabled ? imgLoopOn : imgLoopOff" alt="" />
            </button>
            <span class="vp-hover-tip">{{
              loopEnabled ? "关闭洗脑循环" : "打开洗脑循环"
            }}</span>
          </span>

          <span class="vp-cluster-inline">
            <button
              type="button"
              class="vp-btn"
              :aria-label="wideMode ? '退出宽屏' : '宽屏模式'"
              @click.stop="toggleWide"
            >
              <img :src="imgWide" alt="" class="vp-wide-ico" />
            </button>
            <span class="vp-hover-tip">{{ wideTip }}</span>
          </span>

          <span class="vp-cluster-inline">
            <button
              type="button"
              class="vp-btn vp-btn-fs"
              :aria-label="isFs ? '退出全屏' : '进入全屏'"
              @click.stop="toggleFs"
            >
              <img class="vp-fs-img" :src="imgFull" alt="" />
            </button>
            <span class="vp-hover-tip">{{ isFs ? "退出全屏" : "进入全屏" }}</span>
          </span>
        </div>
      </div>
    </div>

    <!-- 弹幕输入条：Mini-Bili 未登录为游客条；已登录为完整输入 -->
    <div
      class="vp-input-bar"
      :class="{
        'vp-input-bar--guest':
          guestMinibiliDanmakuBar || upClosedMinibiliDanmakuBar
      }"
    >
      <template v-if="upClosedMinibiliDanmakuBar">
        <div class="vp-input-guest-icons">
          <span class="vp-input-guest-ico-wrap">
            <img class="vp-input-bar-ico" :src="imgDanmuSend" alt="" />
          </span>
          <span class="vp-input-guest-divider" />
          <span class="vp-input-guest-ico-wrap">
            <img class="vp-input-bar-ico" :src="imgPlayColor" alt="" />
          </span>
        </div>
        <div class="vp-input-guest-field" aria-live="polite">
          <span class="vp-input-guest-muted">UP主已关闭弹幕</span>
        </div>
        <a href="javascript:;" class="vp-input-etiquette vp-input-etiquette--ghost"
          >弹幕礼仪 &gt;</a
        >
        <button type="button" class="vp-input-send vp-input-send--guest" disabled>
          发送 &gt;
        </button>
      </template>
      <template v-else-if="guestMinibiliDanmakuBar">
        <div class="vp-input-guest-icons">
          <span class="vp-input-guest-ico-wrap">
            <img class="vp-input-bar-ico" :src="imgDanmuSend" alt="" />
          </span>
          <span class="vp-input-guest-divider" />
          <span class="vp-input-guest-ico-wrap">
            <img class="vp-input-bar-ico" :src="imgPlayColor" alt="" />
          </span>
        </div>
        <div class="vp-input-guest-field" aria-live="polite">
          <span class="vp-input-guest-muted">游客不能发送弹幕，请先</span>
          <a
            href="javascript:;"
            class="vp-input-guest-link"
            @click.prevent="openGuestLogin(0)"
            >登录</a
          >
          <span class="vp-input-guest-muted"> 或 </span>
          <a
            href="javascript:;"
            class="vp-input-guest-link"
            @click.prevent="openGuestLogin(1)"
            >注册</a
          >
        </div>
        <a href="javascript:;" class="vp-input-etiquette">弹幕礼仪 &gt;</a>
        <button type="button" class="vp-input-send vp-input-send--guest" disabled>
          发送 &gt;
        </button>
      </template>
      <template v-else>
        <div
          class="vp-input-danmu-select-wrap"
          @mouseenter="onSendSelectEnter"
          @mouseleave="onSendSelectLeave"
        >
          <span class="vp-cluster-inline vp-input-icon-cluster">
            <button
              type="button"
              class="vp-input-bar-btn"
              aria-label="弹幕选择"
            >
              <img class="vp-input-bar-ico" :src="imgDanmuSend" alt="" />
            </button>
            <span class="vp-hover-tip vp-hover-tip--above-input">弹幕选择</span>
          </span>
          <div
            v-show="sendSelectPanelShow"
            class="vp-send-select-panel"
            @click.stop
          >
            <div class="vp-send-row">
              <span class="vp-send-label">字号</span>
              <div class="vp-send-options">
                <button
                  v-for="s in sendFontSizes"
                  :key="s.key"
                  type="button"
                  class="vp-send-pill vp-send-pill--sz"
                  :class="{ on: sendFontSize === s.key }"
                  @click="sendFontSize = s.key"
                >
                  {{ s.label }}
                </button>
              </div>
            </div>
            <div class="vp-send-row vp-send-row--mode">
              <span class="vp-send-label">模式</span>
              <div class="vp-send-mode-grid">
                <button
                  v-for="m in sendModeOptions"
                  :key="m.key"
                  type="button"
                  class="vp-send-mode-cell"
                  :class="{ on: sendMode === m.key }"
                  @click="sendMode = m.key"
                >
                  <span class="vp-send-mode-ico-wrap">
                    <img :src="m.icon" alt="" />
                    <span v-if="sendMode === m.key" class="vp-send-mode-check"
                      >✓</span
                    >
                  </span>
                  <span class="vp-send-mode-lbl">{{ m.label }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>

        <div ref="colorWrapRef" class="vp-input-color-wrap">
          <span class="vp-cluster-inline vp-input-icon-cluster">
            <button
              type="button"
              class="vp-input-bar-btn"
              aria-label="弹幕颜色"
              @click.stop="toggleColorPanel"
            >
              <img class="vp-input-bar-ico" :src="imgPlayColor" alt="" />
            </button>
            <span class="vp-hover-tip vp-hover-tip--above-input">弹幕颜色</span>
          </span>
          <div v-show="colorPanelOpen" class="vp-color-panel" @click.stop>
            <div class="vp-color-title">颜色</div>
            <div class="vp-color-row">
              <input
                v-model="danmuHex"
                type="text"
                class="vp-color-hex"
                maxlength="7"
                spellcheck="false"
                @input="onHexInput"
                @blur="onHexBlur"
              />
              <input
                type="color"
                class="vp-color-native"
                aria-label="颜色取色器"
                :value="danmuColorNativeValue"
                @input="onNativeColorInput"
              />
              <div
                class="vp-color-preview"
                :style="{ backgroundColor: danmuColorCss }"
              />
            </div>
            <div class="vp-color-swatches">
              <button
                v-for="c in presetColors"
                :key="c"
                type="button"
                class="vp-swatch"
                :class="{ on: isPresetActive(c) }"
                :style="{ backgroundColor: c }"
                :title="c"
                @click="pickPresetColor(c)"
              />
            </div>
          </div>
        </div>

        <input
          v-model="danmuInput"
          type="text"
          class="vp-input-field"
          :placeholder="
            minibiliDanmakuClosed
              ? 'UP主已关闭弹幕'
              : '您可以在这里输入弹幕吐槽哦~'
          "
          :disabled="minibiliDanmakuClosed"
          autocomplete="off"
          @keydown.enter.prevent="onDanmuInputEnter"
        />
        <a href="javascript:;" class="vp-input-etiquette">弹幕礼仪 &gt;</a>
        <button
          type="button"
          class="vp-input-send"
          :disabled="danmuSendButtonDisabled"
          @click="sendMinibiliDanmaku"
        >
          {{ danmuSendButtonLabel }}
        </button>
      </template>
    </div>
  </div>
</template>

<script>
import imgPlay from "@/assets/play.png";
import imgStop from "@/assets/stop.png";
import imgVoice from "@/assets/voice.png";
import imgSilence from "@/assets/slience.png";
import imgCommentOn from "@/assets/play_comment.png";
import imgCommentOff from "@/assets/play_comment_stop.png";
import imgDanmu1 from "@/assets/play_danmu_1.png";
import imgDanmu2 from "@/assets/play_danmu_2.png";
import imgDanmu3 from "@/assets/play_danmu_3.png";
import imgLoopOn from "@/assets/play_circulation.png";
import imgLoopOff from "@/assets/play_stop_circulation.png";
import imgWide from "@/assets/contentManagement.png";
import imgFull from "@/assets/play_fullScreen.png";
import imgDanmuSend from "@/assets/danmu_send.png";
import imgPlayColor from "@/assets/play_colar.png";
import imgLoading from "@/assets/loading.gif";
import imgPausedCornerPlay from "@/assets/search/play.png";
import { ElMessage } from "element-plus";
import { mbPostDanmaku, mbPostMeDailyRewardWatch } from "@/api/minibili";
import { extractApiErrorMessage } from "@/utils/apiErrorMessage";
import { getAccessToken } from "@/utils/authTokens";
import {
  dmCanvasFontCss,
  dmFontPxFromKey,
  dmLineHeightForFontPx,
  dmMaxActiveBullets,
  dmNormalizeFontSizeKey,
  dmScrollLanesForDensity,
  dmStackSlotsForDensity,
  dmStrokeWidthForPx,
  loadDmSendFontSizePref,
  saveDmSendFontSizePref
} from "@/utils/danmakuDisplay";

/** Mini-Bili 弹幕可选预设色（SPEC F5；任意合法 #RRGGBB 亦可） */
const MB_DANMAKU_API_COLORS = [
  { hex: "#ffffff", r: 255, g: 255, b: 255 },
  { hex: "#ff0000", r: 255, g: 0, b: 0 },
  { hex: "#00ff00", r: 0, g: 255, b: 0 },
  { hex: "#0000ff", r: 0, g: 0, b: 255 },
  { hex: "#ffff00", r: 255, g: 255, b: 0 }
];

/** 画布与接口：合法 6 位十六进制原样使用，非法回退白 */
function canvasDanmakuColor(raw) {
  const s0 = String(raw || "").trim();
  if (!s0) return "#ffffff";
  const s = s0.startsWith("#") ? s0 : `#${s0}`;
  const m = /^#([0-9a-fA-F]{6})$/.exec(s);
  if (!m) return "#ffffff";
  return `#${m[1]}`.toUpperCase();
}

const PRESET_COLORS = MB_DANMAKU_API_COLORS.map(c => c.hex.toUpperCase());

/** 顶/底弹幕边距（行高随字号动态计算） */
const DM_TOP_PAD = 10;
const DM_BOT_PAD = 14;
const DM_DEFAULT_LINE = dmLineHeightForFontPx(18);

/** 演示用公开短视频（可替换为自有地址） */
const DEMO_MP4 =
  "https://interactive-examples.mdn.mozilla.net/media/cc0-videos/flower.mp4";

export default {
  name: "VideoPlayerBox",
  props: {
    hotTitle: {
      type: String,
      default: "无论如何，总会有一个人相信你"
    },
    /** 热点页路由，默认排行榜总榜 */
    hotRoute: {
      type: Object,
      default: () => ({
        name: "rankingDetail",
        params: { type: "all", rid: "0", rankselect: "0", rankselect2: "0" }
      })
    },
    wideMode: {
      type: Boolean,
      default: false
    },
    /** 非空时优先播放该地址（如 Mini-Bili OSS MP4），否则用内置演示片 */
    mediaSrc: {
      type: String,
      default: ""
    },
    /** Mini-Bili 稿件数字 ID；与 VITE_MINIBILI_API 同时生效时，主栏「发送」走 POST /videos/:id/danmaku（SPEC F5） */
    minibiliVideoId: {
      type: Number,
      default: 0
    },
    /** WebSocket 同步的弹幕目录（SPEC F6：按 video_time 在画面上 Canvas 渲染） */
    danmakuCatalog: {
      type: Array,
      default: () => []
    },
    /** Mini-Bili：UP 关闭弹幕后禁止发送（与接口 40304 一致） */
    minibiliDanmakuClosed: {
      type: Boolean,
      default: false
    }
  },
  emits: ["update:wideMode", "mb-danmaku-committed"],
  data() {
    return {
      demoSrc: DEMO_MP4,
      imgPlay,
      imgStop,
      imgVoice,
      imgSilence,
      imgCommentOn,
      imgCommentOff,
      imgDanmu1,
      imgDanmu2,
      imgDanmu3,
      imgLoopOn,
      imgLoopOff,
      imgWide,
      imgFull,
      imgDanmuSend,
      imgPlayColor,
      imgLoading,
      imgPausedCornerPlay,
      videoLoading: true,
      presetColors: PRESET_COLORS,
      sendSelectPanelShow: false,
      sendSelectLeaveTimer: null,
      volumePercent: 100,
      volumePopoverShow: false,
      volumeLeaveTimer: null,
      volumeDragging: false,
      qualityLeaveTimer: null,
      sendFontSize: loadDmSendFontSizePref(),
      sendFontSizes: [
        { key: "sm", label: "小" },
        { key: "md", label: "中" },
        { key: "lg", label: "大" }
      ],
      sendMode: "scroll",
      colorPanelOpen: false,
      danmuHex: "#FFFFFF",
      playing: false,
      duration: 1,
      currentTime: 0,
      bufferPct: 0,
      muted: false,
      volumeBeforeMute: 1,
      qualityOpen: false,
      qualities: [
        { key: "1080", label: "高清 1080P" },
        { key: "720", label: "准高清 720P" },
        { key: "480", label: "标清 480P" },
        { key: "360", label: "流畅 360P" },
        { key: "auto", label: "自动" }
      ],
      selectedQuality: "1080",
      danmuOn: true,
      danmuPanelShow: false,
      danmuLeaveTimer: null,
      danmuOpacity: 75,
      danmuDensity: 90,
      danmuArea: 25,
      danmuAntiSubtitle: false,
      typeTop: true,
      typeBottom: true,
      typeScroll: true,
      danmuInput: "",
      loopEnabled: true,
      isFs: false,
      fsHandlerBound: false,
      danmuSending: false,
      danmuCooldownUntil: 0,
      danmuCooldownPulse: 0,
      /** Canvas 弹幕（SPEC F6） */
      dmSpawnedMap: {},
      dmActive: [],
      _dmLastVideoClock: 0,
      _dmLastPerf: 0,
      _dmLaneRR: 0,
      _dmRaf: null,
      _dmRafBound: null,
      _dmResizeObs: null,
      mbDailyWatchReported: false,
      shortcutHint: "",
      _shortcutHintTimer: null
    };
  },
  computed: {
    minibiliDanmakuOverlayVisible() {
      return this.minibiliDanmakuMode && this.danmuOn;
    },
    minibiliDanmakuMode() {
      const on =
        import.meta.env.VITE_MINIBILI_API === "true" ||
        import.meta.env.VITE_MINIBILI_API === "1";
      return on && Number(this.minibiliVideoId) > 0;
    },
    mbDanmakuLoggedIn() {
      void this.$store.state.login.signIn;
      void this.$route.fullPath;
      return !!getAccessToken();
    },
    /** Mini-Bili 且未登录：游客提示条（与参考稿一致） */
    guestMinibiliDanmakuBar() {
      return this.minibiliDanmakuMode && !this.mbDanmakuLoggedIn;
    },
    upClosedMinibiliDanmakuBar() {
      return this.minibiliDanmakuMode && this.minibiliDanmakuClosed;
    },
    danmuCooldownLeftSec() {
      void this.danmuCooldownPulse;
      const until = this.danmuCooldownUntil;
      if (!until) return 0;
      const ms = until - Date.now();
      return ms > 0 ? Math.ceil(ms / 1000) : 0;
    },
    danmuSendButtonDisabled() {
      if (!this.minibiliDanmakuMode) return false;
      if (this.minibiliDanmakuClosed) return true;
      if (this.danmuSending) return true;
      if (!this.mbDanmakuLoggedIn) return true;
      const t = String(this.danmuInput || "").trim();
      if (!t) return true;
      const n = Array.from(t).length;
      if (n < 1 || n > 100) return true;
      if (this.danmuCooldownLeftSec > 0) return true;
      if (!this.danmuColorValidForSend) return true;
      return false;
    },
    danmuSendButtonLabel() {
      if (!this.minibiliDanmakuMode) return "发送 >";
      if (this.danmuSending) return "发送中…";
      const left = this.danmuCooldownLeftSec;
      if (left > 0) return `${left}s`;
      return "发送 >";
    },
    videoSrc() {
      const ext = (this.mediaSrc && String(this.mediaSrc).trim()) || "";
      return ext || this.demoSrc;
    },
    playedPct() {
      if (!this.duration) return 0;
      return Math.min(100, (100 * this.currentTime) / this.duration);
    },
    timeLabel() {
      return `${this.fmtTime(this.currentTime)} / ${this.fmtTime(this.duration)}`;
    },
    qualityShort() {
      const q = this.qualities.find((x) => x.key === this.selectedQuality);
      return q ? (q.key === "auto" ? "自动" : q.key + "P") : "1080P";
    },
    wideTip() {
      return this.wideMode ? "退出宽屏" : "宽屏模式";
    },
    sendModeOptions() {
      return [
        { key: "scroll", label: "滚动弹幕", icon: this.imgDanmu2 },
        { key: "top", label: "顶部弹幕", icon: this.imgDanmu1 },
        { key: "bottom", label: "底部弹幕", icon: this.imgDanmu3 }
      ];
    },
    danmuColorCss() {
      const h = this.normalizeHex(this.danmuHex);
      if (h) return h;
      const raw = String(this.danmuHex || "").trim();
      if (!raw) return "#FFFFFF";
      return "#C8CCD0";
    },
    /** `<input type="color">` 绑定值（须为合法 #rrggbb） */
    danmuColorNativeValue() {
      const h = this.normalizeHex(this.danmuHex);
      return (h || "#FFFFFF").toLowerCase();
    },
    /** 空视为发 #FFFFFF；有内容则须为完整六位十六进制（Skill S-014 前端预校验） */
    danmuColorValidForSend() {
      const raw = String(this.danmuHex || "").trim();
      if (!raw) return true;
      return !!this.normalizeHex(this.danmuHex);
    },
    volumeDisplayPercent() {
      if (this.muted) return 0;
      return Math.round(
        Math.min(100, Math.max(0, Number(this.volumePercent) || 0))
      );
    }
  },
  watch: {
    loopEnabled(v) {
      const el = this.$refs.videoRef;
      if (el) el.loop = v;
    },
    videoSrc() {
      this.videoLoading = true;
      this.playing = false;
      this.currentTime = 0;
      this.bufferPct = 0;
      this.dmResetEngine();
      this.$nextTick(() => {
        const v = this.$refs.videoRef;
        if (v) {
          try {
            v.load();
          } catch {
            /* ignore */
          }
          if (v.readyState >= 3) this.videoLoading = false;
        }
      });
    },
    minibiliVideoId() {
      this.dmResetEngine();
      this.mbDailyWatchReported = false;
    },
    minibiliDanmakuMode() {
      if (!this.minibiliDanmakuMode) this.dmResetEngine();
    },
    danmakuCatalog: {
      handler() {
        this.$nextTick(() => this.dmCatchUpCatalog());
      },
      deep: true
    },
    sendFontSize(key) {
      saveDmSendFontSizePref(key);
    },
    danmuDensity() {
      this.$nextTick(() => this.dmTrimToDensityCap());
    }
  },
  mounted() {
    const v = this.$refs.videoRef;
    if (v) {
      v.loop = this.loopEnabled;
      this.volumePercent = Math.round((v.volume || 1) * 100);
      if (v.readyState >= 3) this.videoLoading = false;
    }
    document.addEventListener("fullscreenchange", this.onFsChange);
    document.addEventListener("click", this.onDocClickCloseColor);
    document.addEventListener("keydown", this.onDocKeydown);
    this.fsHandlerBound = true;
    this.dmInitResizeObserver();
    this.dmStartRaf();
  },
  beforeUnmount() {
    if (this.danmuLeaveTimer) clearTimeout(this.danmuLeaveTimer);
    if (this.sendSelectLeaveTimer) clearTimeout(this.sendSelectLeaveTimer);
    if (this.volumeLeaveTimer) clearTimeout(this.volumeLeaveTimer);
    if (this.qualityLeaveTimer) clearTimeout(this.qualityLeaveTimer);
    if (this._danmuCdTimer) {
      clearInterval(this._danmuCdTimer);
      this._danmuCdTimer = null;
    }
    if (this._shortcutHintTimer) {
      clearTimeout(this._shortcutHintTimer);
      this._shortcutHintTimer = null;
    }
    this.dmStopRaf();
    if (this._dmResizeObs) {
      try {
        this._dmResizeObs.disconnect();
      } catch {
        /* noop */
      }
      this._dmResizeObs = null;
    }
    if (this.fsHandlerBound) {
      document.removeEventListener("fullscreenchange", this.onFsChange);
    }
    document.removeEventListener("click", this.onDocClickCloseColor);
    document.removeEventListener("keydown", this.onDocKeydown);
  },
  methods: {
    fmtTime(sec) {
      const s = Math.max(0, Math.floor(sec || 0));
      const m = Math.floor(s / 60);
      const r = s % 60;
      return `${String(m).padStart(2, "0")}:${String(r).padStart(2, "0")}`;
    },
    onTimeUpdate() {
      const v = this.$refs.videoRef;
      if (!v) return;
      const prevClock = this.currentTime;
      this.currentTime = v.currentTime;
      this.duration = v.duration || this.duration;
      this.dmOnPlaybackClock(prevClock, v.currentTime);
      this.maybeReportDailyWatch(v.currentTime);
    },
    maybeReportDailyWatch(currentSec) {
      if (
        this.mbDailyWatchReported ||
        !this.minibiliDanmakuMode ||
        Number(currentSec) < 60
      ) {
        return;
      }
      this.mbDailyWatchReported = true;
      void mbPostMeDailyRewardWatch().catch(() => {
        this.mbDailyWatchReported = false;
      });
    },
    onMeta() {
      const v = this.$refs.videoRef;
      if (v && v.duration) this.duration = v.duration;
    },
    onVideoLoadStart() {
      this.videoLoading = true;
    },
    onVideoCanPlay() {
      this.videoLoading = false;
    },
    onVideoError() {
      this.videoLoading = false;
    },
    onProgress() {
      const v = this.$refs.videoRef;
      if (!v || !v.buffered.length || !v.duration) return;
      try {
        const end = v.buffered.end(v.buffered.length - 1);
        this.bufferPct = Math.min(100, (100 * end) / v.duration);
      } catch {
        this.bufferPct = 0;
      }
    },
    togglePlay() {
      const v = this.$refs.videoRef;
      if (!v) return;
      if (v.paused) v.play().catch(() => {});
      else v.pause();
    },
    isShortcutTargetIgnored(e) {
      const el = e && e.target;
      if (!el || typeof el.closest !== "function") return false;
      return !!el.closest(
        'input, textarea, select, button, a, [contenteditable="true"], [role="textbox"]'
      );
    },
    showShortcutHint(text) {
      const msg = String(text || "").trim();
      if (!msg) return;
      this.shortcutHint = msg;
      if (this._shortcutHintTimer) {
        clearTimeout(this._shortcutHintTimer);
      }
      this._shortcutHintTimer = setTimeout(() => {
        this.shortcutHint = "";
        this._shortcutHintTimer = null;
      }, 1400);
    },
    async toggleFsWithHint() {
      const wasFs = !!document.fullscreenElement;
      await this.toggleFs();
      await this.$nextTick();
      const nowFs = !!document.fullscreenElement;
      if (nowFs) {
        this.showShortcutHint("已进入全屏");
      } else if (wasFs) {
        this.showShortcutHint("已退出全屏");
      }
    },
    onVideoStageDblclick(e) {
      const t = e && e.target;
      if (
        t &&
        typeof t.closest === "function" &&
        t.closest(
          ".vp-controls, .vp-input-bar, .vp-danmu-panel, .vp-send-select-panel, .vp-color-panel, .vp-quality-menu, .vp-volume-popover, .vp-shortcut-hint"
        )
      ) {
        return;
      }
      void this.toggleFsWithHint();
    },
    toggleMuteWithHint() {
      const willMute = !this.muted;
      this.toggleMute();
      this.showShortcutHint(willMute ? "已静音" : "已恢复音量");
    },
    toggleDanmuWithHint() {
      const willOn = !this.danmuOn;
      this.toggleDanmu();
      this.showShortcutHint(willOn ? "弹幕已开启" : "弹幕已关闭");
    },
    /** 空格播放/暂停；M 静音；F 全屏；D 弹幕开关 */
    onDocKeydown(e) {
      if (!e || this.isShortcutTargetIgnored(e)) return;
      if (e.ctrlKey || e.metaKey || e.altKey) return;
      const code = e.code;
      const key = e.key;
      if (code === "Space" || key === " ") {
        e.preventDefault();
        this.togglePlay();
        return;
      }
      if (code === "KeyM" || key === "m" || key === "M") {
        e.preventDefault();
        this.toggleMuteWithHint();
        return;
      }
      if (code === "KeyF" || key === "f" || key === "F") {
        e.preventDefault();
        void this.toggleFsWithHint();
        return;
      }
      if (code === "KeyD" || key === "d" || key === "D") {
        e.preventDefault();
        this.toggleDanmuWithHint();
      }
    },
    /** 弹幕设置横条：与竖条音量一致，左侧已选区间蓝、右侧灰（WebKit 用渐变） */
    danmuSliderTrackStyle(value) {
      const p = Math.min(100, Math.max(0, Number(value) || 0));
      return { "--vp-range-pct": `${p}%` };
    },
    onSeekClick(e) {
      const track = this.$refs.trackRef;
      const v = this.$refs.videoRef;
      if (!track || !v || !v.duration) return;
      const rect = track.getBoundingClientRect();
      const x = Math.min(Math.max(0, e.clientX - rect.left), rect.width);
      v.currentTime = (x / rect.width) * v.duration;
    },
    toggleMute() {
      const v = this.$refs.videoRef;
      if (!v) return;
      if (this.muted) {
        this.muted = false;
        v.muted = false;
        const vol =
          this.volumeBeforeMute > 0 ? this.volumeBeforeMute : this.volumePercent / 100;
        v.volume = Math.min(1, Math.max(0, vol));
        this.volumePercent = Math.round(v.volume * 100);
      } else {
        this.volumeBeforeMute = v.volume;
        this.muted = true;
        v.muted = true;
      }
    },
    onVolumeEnter() {
      if (this.volumeLeaveTimer) {
        clearTimeout(this.volumeLeaveTimer);
        this.volumeLeaveTimer = null;
      }
      this.volumePopoverShow = true;
    },
    onVolumeLeave() {
      if (this.volumeLeaveTimer) clearTimeout(this.volumeLeaveTimer);
      this.volumeLeaveTimer = setTimeout(() => {
        this.volumePopoverShow = false;
        this.volumeLeaveTimer = null;
      }, 220);
    },
    syncVolumeFromSlider() {
      const v = this.$refs.videoRef;
      if (!v) return;
      const p = Math.min(100, Math.max(0, Number(this.volumePercent) || 0));
      this.volumePercent = p;
      v.volume = p / 100;
      if (p > 0) {
        this.muted = false;
        v.muted = false;
        this.volumeBeforeMute = v.volume;
      }
    },
    onVolumePointerDown(e) {
      if (e.button !== undefined && e.button !== 0) return;
      this.volumeDragging = true;
      try {
        e.currentTarget.setPointerCapture(e.pointerId);
      } catch {
        /* ignore */
      }
      this.applyVolumeFromClientY(e.clientY);
    },
    onVolumePointerMove(e) {
      if (!this.volumeDragging) return;
      this.applyVolumeFromClientY(e.clientY);
    },
    onVolumePointerUp(e) {
      if (!this.volumeDragging) return;
      this.volumeDragging = false;
      try {
        e.currentTarget.releasePointerCapture(e.pointerId);
      } catch {
        /* ignore */
      }
    },
    onVolumeLostCapture() {
      this.volumeDragging = false;
    },
    applyVolumeFromClientY(clientY) {
      const el = this.$refs.volumeTrackRef;
      if (!el) return;
      const rect = el.getBoundingClientRect();
      const y = clientY - rect.top;
      const ratio = 1 - y / rect.height;
      this.volumePercent = Math.round(Math.min(100, Math.max(0, ratio * 100)));
      this.syncVolumeFromSlider();
    },
    onQualityEnter() {
      if (this.qualityLeaveTimer) {
        clearTimeout(this.qualityLeaveTimer);
        this.qualityLeaveTimer = null;
      }
      this.qualityOpen = true;
    },
    onQualityLeave() {
      if (this.qualityLeaveTimer) clearTimeout(this.qualityLeaveTimer);
      this.qualityLeaveTimer = setTimeout(() => {
        this.qualityOpen = false;
        this.qualityLeaveTimer = null;
      }, 280);
    },
    pickQuality(key) {
      this.selectedQuality = key;
      this.qualityOpen = false;
      if (this.qualityLeaveTimer) {
        clearTimeout(this.qualityLeaveTimer);
        this.qualityLeaveTimer = null;
      }
    },
    toggleDanmu() {
      this.danmuOn = !this.danmuOn;
    },
    onDanmuClusterEnter() {
      if (this.danmuLeaveTimer) {
        clearTimeout(this.danmuLeaveTimer);
        this.danmuLeaveTimer = null;
      }
      this.danmuPanelShow = true;
    },
    onDanmuClusterLeave() {
      if (this.danmuLeaveTimer) clearTimeout(this.danmuLeaveTimer);
      this.danmuLeaveTimer = setTimeout(() => {
        this.danmuPanelShow = false;
        this.danmuLeaveTimer = null;
      }, 200);
    },
    toggleWide() {
      this.$emit("update:wideMode", !this.wideMode);
    },
    async toggleFs() {
      const root = this.$refs.rootRef;
      if (!root) return;
      try {
        if (!document.fullscreenElement) {
          await root.requestFullscreen();
        } else {
          await document.exitFullscreen();
        }
      } catch {
        /* ignore */
      }
    },
    onFsChange() {
      this.isFs = !!document.fullscreenElement;
    },
    onSendSelectEnter() {
      if (this.sendSelectLeaveTimer) {
        clearTimeout(this.sendSelectLeaveTimer);
        this.sendSelectLeaveTimer = null;
      }
      this.sendSelectPanelShow = true;
    },
    onSendSelectLeave() {
      if (this.sendSelectLeaveTimer) clearTimeout(this.sendSelectLeaveTimer);
      this.sendSelectLeaveTimer = setTimeout(() => {
        this.sendSelectPanelShow = false;
        this.sendSelectLeaveTimer = null;
      }, 200);
    },
    toggleColorPanel() {
      this.colorPanelOpen = !this.colorPanelOpen;
    },
    onDocClickCloseColor(e) {
      if (!this.colorPanelOpen) return;
      const wrap = this.$refs.colorWrapRef;
      if (wrap && !wrap.contains(e.target)) {
        this.colorPanelOpen = false;
      }
    },
    normalizeHex(raw) {
      if (!raw || typeof raw !== "string") return "";
      let s = raw.trim().toUpperCase();
      if (!s.startsWith("#")) s = `#${s}`;
      if (/^#[0-9A-F]{6}$/.test(s)) return s;
      return "";
    },
    onHexInput() {
      let s = String(this.danmuHex || "").trim();
      if (s && !s.startsWith("#")) s = `#${s}`;
      this.danmuHex = s.toUpperCase().slice(0, 7);
    },
    onHexBlur() {
      const n = this.normalizeHex(this.danmuHex);
      if (n) {
        this.danmuHex = n;
      } else {
        this.danmuHex = "#FFFFFF";
      }
    },
    onNativeColorInput(e) {
      const v = e && e.target && e.target.value;
      if (typeof v === "string" && /^#[0-9a-fA-F]{6}$/.test(v)) {
        this.danmuHex = v.toUpperCase();
      }
    },
    pickPresetColor(hex) {
      this.danmuHex = String(hex || "")
        .trim()
        .toUpperCase();
    },
    isPresetActive(c) {
      const cur = (this.normalizeHex(this.danmuHex) || "#FFFFFF").toUpperCase();
      return cur === String(c || "").trim().toUpperCase();
    },
    openGuestLogin(tabIndex) {
      this.$store.commit("login/SET_LOGIN_TAB", tabIndex);
      this.$store.commit("login/OPEN_LOGIN_MODAL");
    },
    armDanmuCooldownTicker() {
      if (this._danmuCdTimer) {
        clearInterval(this._danmuCdTimer);
        this._danmuCdTimer = null;
      }
      this.danmuCooldownUntil = Date.now() + 5000;
      this.danmuCooldownPulse = Date.now();
      this._danmuCdTimer = setInterval(() => {
        this.danmuCooldownPulse = Date.now();
        if (Date.now() >= this.danmuCooldownUntil) {
          clearInterval(this._danmuCdTimer);
          this._danmuCdTimer = null;
          this.danmuCooldownUntil = 0;
        }
      }, 200);
    },
    onDanmuInputEnter() {
      if (!this.danmuSendButtonDisabled) {
        this.sendMinibiliDanmaku();
      }
    },
    dmResetEngine() {
      this.dmSpawnedMap = {};
      this.dmActive = [];
      this._dmLastVideoClock = 0;
      this._dmLaneRR = 0;
    },
    dmOnPlaybackClock(prevT, currT) {
      if (!this.minibiliDanmakuMode || !this.danmuOn) return;
      const cat = this.danmakuCatalog || [];
      if (currT + 0.35 < prevT) {
        this.dmResetEngine();
        this._dmLastVideoClock = currT;
        return;
      }
      for (const d of cat) {
        const id = Number(d.id);
        if (!id || this.dmSpawnedMap[id]) continue;
        const t = Number(d.video_time) || 0;
        if (t > prevT - 0.35 && t <= currT + 0.15) {
          if (this.dmSpawnBullet(d)) {
            this.dmSpawnedMap[id] = 1;
          }
        }
      }
      this._dmLastVideoClock = currT;
    },
    /**
     * 目录更新时补 spawn：解决 WS 晚于播放轴、暂停时发送等导致 timeupdate 窗口跨不过去的问题。
     */
    dmCatchUpCatalog() {
      if (!this.minibiliDanmakuMode || !this.danmuOn) return;
      const curr = Number(this.currentTime) || 0;
      const cat = this.danmakuCatalog || [];
      for (const d of cat) {
        const id = Number(d.id);
        if (!id || this.dmSpawnedMap[id]) continue;
        const vt = Number(d.video_time) || 0;
        if (vt <= curr + 0.45 && vt >= curr - 0.45) {
          if (this.dmSpawnBullet(d)) {
            this.dmSpawnedMap[id] = 1;
          }
        }
      }
    },
    /**
     * 顶/底固定弹幕最大行数。
     * 原先按「显示区域带 × 0.4」算，area 较小时只有约 3 行；改为按画布高度的大比例堆叠，并与显示区域滑条联动。
     */
    dmMaxStackSlots(kind, h, lineH) {
      void kind;
      const areaPct = Math.min(100, Math.max(14, Number(this.danmuArea) || 25)) / 100;
      const bandPx = h * areaPct * 0.92;
      const halfLine = lineH / 2;
      // 堆叠可用高度：取「显示区域」与「半屏」中较大者，再占其中 88%，多行向上/向下
      const stackPx =
        Math.max(bandPx, h * 0.5) * 0.88 - DM_TOP_PAD - halfLine;
      const geo = Math.max(1, Math.floor(stackPx / lineH));
      const scaled = dmStackSlotsForDensity(geo, this.danmuDensity);
      return Math.min(56, Math.max(1, scaled));
    },
    dmScrollLaneCount(h, lineH) {
      const areaPct = Math.min(100, Math.max(14, Number(this.danmuArea) || 25)) / 100;
      const bandH = h * areaPct * 0.92;
      const baseLanes = Math.max(3, Math.floor(bandH / lineH));
      return dmScrollLanesForDensity(baseLanes, this.danmuDensity);
    },
    dmActiveCap(h, scrollLanes) {
      return dmMaxActiveBullets(this.danmuDensity, h, scrollLanes);
    },
    dmAtActiveCap(h, scrollLanes) {
      return this.dmActive.length >= this.dmActiveCap(h, scrollLanes);
    },
    dmTrimToDensityCap() {
      if (!this.minibiliDanmakuMode || !this.danmuOn) return;
      const c = this.$refs.dmCanvasRef;
      if (!c) return;
      const h = c.getBoundingClientRect().height;
      if (h < 8) return;
      const lanes = this.dmScrollLaneCount(h, DM_DEFAULT_LINE);
      const cap = this.dmActiveCap(h, lanes);
      while (this.dmActive.length > cap) {
        const scrollIx = this.dmActive.findIndex(b => b.type === "scroll");
        if (scrollIx >= 0) {
          this.dmActive.splice(scrollIx, 1);
        } else {
          this.dmActive.shift();
        }
      }
    },
    /** 顶部：取最小空行，自上往下排。底部不用此逻辑（新弹幕贴底、旧整体上移）。 */
    dmAllocStackSlot(kind, h, lineH) {
      const maxSlots = this.dmMaxStackSlots(kind, h, lineH);
      const used = new Set();
      for (const b of this.dmActive) {
        if (b.type === kind && typeof b.slot === "number") used.add(b.slot);
      }
      for (let s = 0; s < maxSlots; s++) {
        if (!used.has(s)) return s;
      }
      return maxSlots - 1;
    },
    dmSpawnBullet(d) {
      const typRaw = String(d.type || "scroll")
        .trim()
        .toLowerCase();
      if (typRaw === "scroll" && !this.typeScroll) return false;
      if (typRaw === "top" && !this.typeTop) return false;
      if (typRaw === "bottom" && !this.typeBottom) return false;
      const c = this.$refs.dmCanvasRef;
      if (!c) return false;
      const ctx = c.getContext("2d");
      if (!ctx) return false;
      const rect = c.getBoundingClientRect();
      const w = rect.width;
      const h = rect.height;
      if (w < 8 || h < 8) return false;
      const text = String(d.content || "");
      if (!text) return false;
      const color = String(d.color || "#ffffff");
      const typ = typRaw;
      const fontPx = dmFontPxFromKey(d.font_size);
      const lineH = dmLineHeightForFontPx(fontPx);
      const scrollLanes = this.dmScrollLaneCount(h, lineH);
      if (this.dmAtActiveCap(h, scrollLanes)) {
        return "dropped";
      }
      const fontCss = dmCanvasFontCss(fontPx);
      ctx.font = fontCss;
      const textW = Math.ceil(ctx.measureText(text).width);
      const fillColor = canvasDanmakuColor(color);
      const bulletBase = {
        text,
        color: fillColor,
        fontPx,
        lineH,
        fontCss,
        strokeW: dmStrokeWidthForPx(fontPx)
      };

      if (typ === "top") {
        const slot = this.dmAllocStackSlot("top", h, lineH);
        const x = Math.max(4, (w - textW) / 2);
        const y = DM_TOP_PAD + slot * lineH + lineH / 2;
        this.dmActive.push({
          ...bulletBase,
          type: "top",
          x,
          y,
          w: textW,
          vx: 0,
          life: 5.5,
          slot
        });
        return true;
      }
      if (typ === "bottom") {
        const maxSlots = this.dmMaxStackSlots("bottom", h, lineH);
        for (let i = this.dmActive.length - 1; i >= 0; i--) {
          const b = this.dmActive[i];
          if (b.type !== "bottom") continue;
          const blh = b.lineH || DM_DEFAULT_LINE;
          const ns = (typeof b.slot === "number" ? b.slot : 0) + 1;
          if (ns >= maxSlots) {
            this.dmActive.splice(i, 1);
          } else {
            b.slot = ns;
            b.y = h - DM_BOT_PAD - b.slot * blh - blh / 2;
          }
        }
        const x = Math.max(4, (w - textW) / 2);
        const y = h - DM_BOT_PAD - lineH / 2;
        this.dmActive.push({
          ...bulletBase,
          type: "bottom",
          x,
          y,
          w: textW,
          vx: 0,
          life: 5.5,
          slot: 0
        });
        return true;
      }

      const lanes = scrollLanes;
      const lane = this._dmLaneRR % lanes;
      this._dmLaneRR += 1;
      const y0 = DM_TOP_PAD + lane * lineH;
      const speed = (w + textW + 48) / 9;
      this.dmActive.push({
        ...bulletBase,
        type: "scroll",
        x: w + 10,
        y: Math.min(h - DM_BOT_PAD, y0 + lineH - 6),
        w: textW,
        vx: -speed,
        life: 14
      });
      return true;
    },
    dmRafFrame() {
      const c = this.$refs.dmCanvasRef;
      if (!c || !this.minibiliDanmakuOverlayVisible) return;
      const ctx = c.getContext("2d");
      if (!ctx) return;
      const dpr = window.devicePixelRatio || 1;
      const rect = c.getBoundingClientRect();
      const cssW = Math.max(1, rect.width);
      const cssH = Math.max(1, rect.height);
      const pw = Math.floor(cssW * dpr);
      const ph = Math.floor(cssH * dpr);
      if (c.width !== pw || c.height !== ph) {
        c.width = pw;
        c.height = ph;
      }
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      ctx.clearRect(0, 0, cssW, cssH);
      for (const b of this.dmActive) {
        const lh = b.lineH || DM_DEFAULT_LINE;
        if (b.type === "top" && typeof b.slot === "number") {
          b.x = Math.max(4, (cssW - (b.w || 0)) / 2);
          b.y = DM_TOP_PAD + b.slot * lh + lh / 2;
        } else if (b.type === "bottom" && typeof b.slot === "number") {
          b.x = Math.max(4, (cssW - (b.w || 0)) / 2);
          b.y = cssH - DM_BOT_PAD - b.slot * lh - lh / 2;
        }
      }
      const now = performance.now();
      const dt = this.playing
        ? Math.min(0.055, (now - this._dmLastPerf) / 1000 || 0.016)
        : 0;
      this._dmLastPerf = now;
      const op = Math.min(100, Math.max(15, Number(this.danmuOpacity) || 75)) / 100;
      ctx.globalAlpha = op;
      this.dmActive = this.dmActive.filter(b => {
        if (dt > 0) {
          if (b.type === "scroll" || !b.type) {
            b.x += b.vx * dt;
          }
          b.life -= dt;
        }
        const isStack = b.type === "top" || b.type === "bottom";
        if (b.life <= 0) return false;
        if (!isStack && b.x + (b.w || 0) < -20) return false;
        const prevBase = ctx.textBaseline;
        ctx.textBaseline = isStack ? "middle" : "alphabetic";
        ctx.font = b.fontCss || dmCanvasFontCss(b.fontPx);
        ctx.strokeStyle = "rgba(0,0,0,0.55)";
        ctx.lineWidth = b.strokeW || dmStrokeWidthForPx(b.fontPx);
        ctx.strokeText(b.text, b.x, b.y);
        ctx.fillStyle = canvasDanmakuColor(b.color);
        ctx.fillText(b.text, b.x, b.y);
        ctx.textBaseline = prevBase;
        return true;
      });
      ctx.globalAlpha = 1;
    },
    dmStartRaf() {
      this.dmStopRaf();
      this._dmLastPerf = performance.now();
      this._dmRafBound = () => {
        this.dmRafFrame();
        this._dmRaf = requestAnimationFrame(this._dmRafBound);
      };
      this._dmRaf = requestAnimationFrame(this._dmRafBound);
    },
    dmStopRaf() {
      if (this._dmRaf) {
        cancelAnimationFrame(this._dmRaf);
        this._dmRaf = null;
      }
      this._dmRafBound = null;
    },
    dmInitResizeObserver() {
      this.$nextTick(() => {
        const el = this.$refs.videoStageRef;
        if (!el || typeof ResizeObserver === "undefined") return;
        if (this._dmResizeObs) {
          try {
            this._dmResizeObs.disconnect();
          } catch {
            /* noop */
          }
        }
        this._dmResizeObs = new ResizeObserver(() => {
          this.$nextTick(() => this.dmRafFrame());
        });
        this._dmResizeObs.observe(el);
      });
    },
    async sendMinibiliDanmaku() {
      if (!this.minibiliDanmakuMode) return;
      if (this.danmuSending) return;
      if (this.minibiliDanmakuClosed) {
        ElMessage.error("UP主已关闭弹幕");
        return;
      }
      if (!this.mbDanmakuLoggedIn) {
        this.$store.commit("login/SET_LOGIN_TAB", 0);
        this.$store.commit("login/OPEN_LOGIN_MODAL");
        return;
      }
      if (this.danmuCooldownLeftSec > 0) {
        ElMessage.warning("发送过于频繁，请稍后再试");
        return;
      }
      const text = String(this.danmuInput || "").trim();
      if (!text) return;
      const n = Array.from(text).length;
      if (n < 1 || n > 100) {
        ElMessage.warning("弹幕长度为 1–100 个字符");
        return;
      }
      if (!this.danmuColorValidForSend) {
        ElMessage.warning(
          "弹幕颜色格式无效，请输入有效的十六进制色号（如 #FF0000）"
        );
        return;
      }
      this.danmuSending = true;
      try {
        const color = this.normalizeHex(this.danmuHex) || "#FFFFFF";
        const row = await mbPostDanmaku(Number(this.minibiliVideoId), {
          content: text,
          color,
          type: String(this.sendMode || "scroll"),
          font_size: dmNormalizeFontSizeKey(this.sendFontSize),
          video_time: Math.max(0, Number(this.currentTime) || 0)
        });
        this.$emit("mb-danmaku-committed", row);
        this.$nextTick(() => this.dmCatchUpCatalog());
        this.danmuInput = "";
        this.armDanmuCooldownTicker();
      } catch (e) {
        ElMessage.error(extractApiErrorMessage(e, "发送失败"));
      } finally {
        this.danmuSending = false;
      }
    }
  }
};
</script>

<style lang="scss" scoped>
$b-blue: #00a1d6;
$b-played: #00a1d6;
$b-buffer: #dff6fc;
$b-track: #c9cfd8;

.vp-root {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #000;
  color: #fff;
}

.vp-topbar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  height: 38px;
  padding: 0 8px;
  background: #222;
  box-sizing: border-box;
}

.vp-pl-arrow {
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 2px;
  background: #333;
  color: #aaa;
  font-size: 18px;
  line-height: 1;
  cursor: pointer;
}
.vp-pl-arrow:hover {
  color: #fff;
}

.vp-hot-link {
  flex: 1;
  min-width: 0;
  text-align: center;
  font-size: 13px;
  color: #eee;
  text-decoration: none;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.vp-hot-link:hover {
  color: $b-blue;
}

.vp-video-stage {
  position: relative;
  width: 100%;
  aspect-ratio: 16 / 9;
  flex-shrink: 0;
  background: #000;
}

.vp-video {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
  cursor: pointer;
}

.vp-dm-canvas {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 5;
}

/* 与参考站一致：白底 + 居中 TV loading.gif + 左下角淡灰状态文案 */
.vp-load-mask {
  position: absolute;
  inset: 0;
  z-index: 12;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fff;
  box-sizing: border-box;
  pointer-events: auto;
}

.vp-load-gif {
  display: block;
  width: 100px;
  height: 100px;
  object-fit: contain;
  flex-shrink: 0;
  filter: drop-shadow(2px 4px 3px rgba(0, 0, 0, 0.28));
}

.vp-load-status {
  position: absolute;
  left: 12px;
  bottom: 10px;
  font-size: 12px;
  line-height: 1.4;
  color: #b8bdc2;
  letter-spacing: 0.02em;
  user-select: none;
}

/* 暂停时右下角提示：参考站 — 底栏上沿留小缝、右侧留边（略离开右缘） */
.vp-paused-corner-play {
  position: absolute;
  right: 18px;
  bottom: 58px;
  z-index: 9;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  padding: 0;
  border: none;
  border-radius: 6px;
  background: transparent;
  cursor: pointer;
  line-height: 0;
  box-sizing: border-box;
}

.vp-paused-corner-play:hover .vp-paused-corner-play-img {
  filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.55)) brightness(1.08);
}

.vp-paused-corner-play-img {
  display: block;
  width: 56px;
  height: 56px;
  object-fit: contain;
  filter: drop-shadow(0 1px 3px rgba(0, 0, 0, 0.5));
  pointer-events: none;
}

.vp-controls {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 10;
  padding: 10px 12px 12px;
  background: linear-gradient(transparent, rgba(0, 0, 0, 0.72));
  box-sizing: border-box;
}

.vp-controls-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: nowrap;
}

.vp-btn {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  padding: 0;
  border: none;
  background: transparent;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.vp-btn img {
  width: 22px;
  height: 22px;
  object-fit: contain;
  display: block;
}

/* .vp-btn img 特异性更高，须挂在 .vp-play 下才能覆盖默认 22px */
.vp-play.vp-btn .vp-play-img {
  width: 24px;
  height: 24px;
  object-fit: contain;
}
/* stop.png 画布留白大，暂停态再放大便于辨认 */
.vp-play.vp-btn .vp-play-img--triangle {
  width: 52px;
  height: 52px;
}

.vp-play.vp-btn {
  width: auto;
  min-width: 40px;
  min-height: 40px;
}

.vp-btn-fs {
  width: 34px;
  height: 34px;
}
.vp-fs-img {
  width: 28px !important;
  height: 28px !important;
  object-fit: contain;
}

.vp-wide-ico {
  width: 20px !important;
  height: 20px !important;
}

.vp-progress {
  flex: 1;
  min-width: 40px;
  height: 26px;
  display: flex;
  align-items: center;
  position: relative;
  cursor: pointer;
}

.vp-progress-bg {
  position: absolute;
  left: 0;
  right: 0;
  top: 50%;
  transform: translateY(-50%);
  height: 6px;
  border-radius: 3px;
  background: $b-track;
}

.vp-progress-buffer {
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  height: 6px;
  border-radius: 3px;
  background: $b-buffer;
  pointer-events: none;
  z-index: 1;
}

.vp-progress-played {
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  height: 6px;
  border-radius: 3px;
  background: $b-played;
  pointer-events: none;
  z-index: 2;
}

.vp-progress-thumb {
  position: absolute;
  top: 50%;
  width: 14px;
  height: 14px;
  margin-left: -7px;
  margin-top: -7px;
  border-radius: 50%;
  background: #fff;
  box-shadow: 0 0 2px rgba(0, 0, 0, 0.35);
  z-index: 3;
  pointer-events: none;
}

.vp-time {
  flex-shrink: 0;
  font-size: 12px;
  color: #fff;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.vp-volume-wrap {
  position: relative;
  flex-shrink: 0;
}

.vp-volume-pop {
  position: absolute;
  bottom: calc(100% + 10px);
  left: 50%;
  transform: translateX(-50%);
  min-width: 56px;
  padding: 12px 10px 14px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.18);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  z-index: 52;
  box-sizing: border-box;
}

.vp-volume-num {
  font-size: 14px;
  font-weight: 500;
  color: #9499a0;
  line-height: 1;
}

.vp-vtrack {
  position: relative;
  width: 6px;
  height: 104px;
  border-radius: 3px;
  background: #eceef0;
  cursor: pointer;
  flex-shrink: 0;
  touch-action: none;
}

.vp-vtrack-fill {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  border-radius: 3px;
  background: $b-blue;
  pointer-events: none;
}

.vp-vthumb {
  position: absolute;
  left: 50%;
  width: 14px;
  height: 14px;
  margin-left: -7px;
  border-radius: 50%;
  background: #fff;
  box-shadow:
    0 0 0 2px rgba(0, 161, 214, 0.38),
    0 2px 8px rgba(0, 0, 0, 0.12);
  pointer-events: none;
}

.vp-quality {
  position: relative;
  flex-shrink: 0;
  z-index: 48;
}

.vp-quality-trigger {
  min-width: 52px;
  height: 26px;
  padding: 0 8px;
  border: none;
  border-radius: 2px;
  background: rgba(255, 255, 255, 0.12);
  color: #e8e8e8;
  font-size: 12px;
  cursor: pointer;
}
.vp-quality-trigger:hover {
  background: rgba(255, 255, 255, 0.2);
}

.vp-quality-menu {
  position: absolute;
  bottom: calc(100% + 8px);
  right: 0;
  min-width: 140px;
  padding: 6px 0;
  background: #fff;
  border-radius: 4px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.18);
  z-index: 50;
}

.vp-quality-item {
  display: block;
  width: 100%;
  padding: 8px 14px;
  border: none;
  background: none;
  text-align: left;
  font-size: 13px;
  color: #18191c;
  cursor: pointer;
}
.vp-quality-item:hover {
  background: #f4f5f7;
}
.vp-quality-item.on {
  color: $b-blue;
  font-weight: 600;
}

.vp-danmu-cluster {
  position: relative;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 0;
}

.vp-hover-tip {
  display: none;
  position: absolute;
  bottom: calc(100% + 8px);
  left: 50%;
  transform: translateX(-50%);
  padding: 4px 8px;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.82);
  color: #fff;
  font-size: 12px;
  white-space: nowrap;
  pointer-events: none;
  z-index: 40;
}

.vp-cluster-inline {
  position: relative;
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
}

.vp-cluster-inline:hover .vp-hover-tip {
  display: block;
}

.vp-controls-row {
  position: relative;
}

.vp-danmu-panel {
  position: absolute;
  bottom: calc(100% + 36px);
  right: 0;
  width: 260px;
  padding: 14px 14px 12px;
  background: #fff;
  border-radius: 4px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.18);
  color: #18191c;
  z-index: 45;
  box-sizing: border-box;
}

.vp-danmu-slider-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}

.vp-danmu-label {
  flex-shrink: 0;
  width: 64px;
  font-size: 12px;
  color: #505050;
}

.vp-range {
  flex: 1;
  min-width: 0;
  width: 100%;
  height: 22px;
  margin: 0;
  padding: 0;
  -webkit-appearance: none;
  appearance: none;
  background: transparent;
  cursor: pointer;
  --vp-range-pct: 0%;
}

.vp-range:focus {
  outline: none;
}

.vp-range::-webkit-slider-runnable-track {
  height: 6px;
  border-radius: 3px;
  background: linear-gradient(
    to right,
    $b-blue 0,
    $b-blue var(--vp-range-pct, 0%),
    #eceef0 var(--vp-range-pct, 0%),
    #eceef0 100%
  );
}

.vp-range::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 14px;
  height: 14px;
  margin-top: -4px;
  border-radius: 50%;
  background: #fff;
  box-shadow:
    0 0 0 2px rgba(0, 161, 214, 0.38),
    0 2px 8px rgba(0, 0, 0, 0.12);
  border: none;
}

.vp-range::-moz-range-track {
  height: 6px;
  border-radius: 3px;
  background: #eceef0;
}

.vp-range::-moz-range-progress {
  height: 6px;
  border-radius: 3px;
  background: $b-blue;
}

.vp-range::-moz-range-thumb {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: #fff;
  border: none;
  box-shadow:
    0 0 0 2px rgba(0, 161, 214, 0.38),
    0 2px 8px rgba(0, 0, 0, 0.12);
}

.vp-danmu-check {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  font-size: 12px;
  color: #505050;
  cursor: pointer;
}

.vp-danmu-types {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 12px;
}

.vp-danmu-type {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 6px 4px;
  border: 1px solid #e3e5e7;
  border-radius: 4px;
  background: #fafafa;
  cursor: pointer;
  font-size: 11px;
  color: #61666d;
  position: relative;
}
.vp-danmu-type img {
  width: 28px;
  height: 28px;
  object-fit: contain;
}
.vp-danmu-type.off::after {
  content: "";
  position: absolute;
  right: 4px;
  bottom: 22px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='%23e53935' stroke-width='2.5'%3E%3Ccircle cx='12' cy='12' r='9'/%3E%3Cpath d='M7 7l10 10'/%3E%3C/svg%3E")
    center / contain no-repeat;
}

.vp-input-bar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  background: #fff;
  border-top: 1px solid #e5e9ef;
  box-sizing: border-box;
  position: relative;
  z-index: 20;
}
.vp-input-bar--guest {
  background: #f4f4f4;
  gap: 8px;
}
.vp-input-guest-icons {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 0;
  padding: 0 6px;
  height: 34px;
  background: #fff;
  border: 1px solid #e5e9ef;
  border-radius: 4px;
  box-sizing: border-box;
}
.vp-input-guest-ico-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  opacity: 0.55;
}
.vp-input-guest-divider {
  width: 1px;
  height: 20px;
  background: #e5e9ef;
  margin: 0 2px;
}
.vp-input-guest-field {
  flex: 1;
  min-width: 0;
  min-height: 34px;
  padding: 0 12px;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  font-size: 13px;
  line-height: 1.5;
  background: #f1f2f3;
  border: 1px solid #e5e9ef;
  border-radius: 4px;
  box-sizing: border-box;
}
.vp-input-guest-muted {
  color: #9499a0;
}
.vp-input-guest-link {
  color: #00a1d6;
  text-decoration: none;
  cursor: pointer;
}
.vp-input-guest-link:hover {
  color: #00aeec;
}
.vp-input-send--guest {
  background: #e3e5e7 !important;
  color: #fff !important;
  cursor: not-allowed;
  filter: none !important;
}
.vp-input-send--guest:hover {
  filter: none !important;
}

.vp-input-danmu-select-wrap,
.vp-input-color-wrap {
  position: relative;
  flex-shrink: 0;
}

.vp-input-bar-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  padding: 0;
  border: none;
  border-radius: 4px;
  background: transparent;
  cursor: pointer;
}
.vp-input-bar-btn:hover {
  background: #f4f5f7;
}

.vp-input-bar-ico {
  width: 22px;
  height: 22px;
  object-fit: contain;
  display: block;
}

.vp-input-icon-cluster:hover .vp-hover-tip--above-input {
  display: block;
}

.vp-hover-tip--above-input {
  bottom: calc(100% + 8px);
  z-index: 60;
}

/* —— 弹幕选择面板（白底，与参考图一致） —— */
.vp-send-select-panel {
  position: absolute;
  left: 0;
  bottom: calc(100% + 10px);
  min-width: 300px;
  padding: 14px 16px 16px;
  background: #fff;
  border-radius: 4px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.14);
  border: 1px solid #e5e9ef;
  box-sizing: border-box;
  z-index: 50;
}

.vp-send-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 14px;
}
.vp-send-row:last-child {
  margin-bottom: 0;
}

.vp-send-label {
  flex-shrink: 0;
  width: 36px;
  padding-top: 6px;
  font-size: 13px;
  color: #18191c;
}

.vp-send-options {
  flex: 1;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.vp-send-pill {
  padding: 6px 14px;
  border: 1px solid #e3e5e7;
  border-radius: 4px;
  background: #fff;
  font-size: 13px;
  color: #18191c;
  cursor: pointer;
}
.vp-send-pill:hover {
  border-color: rgba(0, 161, 214, 0.45);
  color: $b-blue;
}
.vp-send-pill.on {
  border-color: $b-blue;
  color: $b-blue;
  font-weight: 600;
}

.vp-send-pill--sz {
  padding: 5px 16px;
  border-radius: 16px;
}

.vp-send-row--mode .vp-send-label {
  padding-top: 18px;
}

.vp-send-mode-grid {
  flex: 1;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}

.vp-send-mode-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 4px;
  border: none;
  background: transparent;
  cursor: pointer;
}

.vp-send-mode-ico-wrap {
  position: relative;
  width: 56px;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f4f5f7;
  border-radius: 8px;
  border: 2px solid transparent;
  box-sizing: border-box;
}
.vp-send-mode-ico-wrap img {
  width: 32px;
  height: 32px;
  object-fit: contain;
}

.vp-send-mode-cell.on .vp-send-mode-ico-wrap {
  border-color: $b-blue;
  background: #f0fafc;
}

.vp-send-mode-check {
  position: absolute;
  right: -4px;
  bottom: -4px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: $b-blue;
  color: #fff;
  font-size: 11px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}

.vp-send-mode-lbl {
  font-size: 12px;
  color: #18191c;
}
.vp-send-mode-cell.on .vp-send-mode-lbl {
  color: $b-blue;
  font-weight: 600;
}

/* —— 弹幕颜色面板（深色） —— */
.vp-color-panel {
  position: absolute;
  left: 0;
  bottom: calc(100% + 10px);
  width: 268px;
  padding: 12px 14px 14px;
  background: rgba(28, 28, 30, 0.94);
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: 0 8px 28px rgba(0, 0, 0, 0.35);
  box-sizing: border-box;
  z-index: 55;
}

.vp-color-title {
  margin-bottom: 10px;
  font-size: 13px;
  font-weight: 600;
  color: #fff;
}

.vp-color-row {
  display: flex;
  align-items: stretch;
  gap: 10px;
  margin-bottom: 12px;
}

.vp-color-hex {
  flex: 1;
  min-width: 0;
  height: 34px;
  padding: 0 10px;
  border: 1px solid rgba(255, 255, 255, 0.22);
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.35);
  color: #fff;
  font-size: 13px;
  font-family: ui-monospace, monospace;
  outline: none;
  box-sizing: border-box;
}
.vp-color-hex:focus {
  border-color: $b-blue;
}

.vp-color-native {
  width: 44px;
  height: 34px;
  flex-shrink: 0;
  padding: 0;
  border: 1px solid rgba(255, 255, 255, 0.22);
  border-radius: 4px;
  background: transparent;
  cursor: pointer;
  box-sizing: border-box;
}

.vp-color-preview {
  width: 44px;
  height: 34px;
  flex-shrink: 0;
  border-radius: 6px;
  border: 1px solid rgba(255, 255, 255, 0.25);
  box-sizing: border-box;
}

.vp-color-swatches {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 8px;
}

.vp-swatch {
  width: 100%;
  aspect-ratio: 1;
  max-height: 28px;
  padding: 0;
  border: 2px solid transparent;
  border-radius: 6px;
  cursor: pointer;
  box-sizing: border-box;
}
.vp-swatch.on {
  box-shadow:
    inset 0 0 0 2px #000,
    0 0 0 2px rgba(255, 255, 255, 0.55);
}
.vp-swatch:focus-visible {
  outline: 2px solid $b-blue;
  outline-offset: 2px;
}

.vp-input-field {
  position: relative;
  z-index: 30;
  flex: 1;
  min-width: 0;
  color: #18191c;
  height: 34px;
  padding: 0 12px;
  border: 1px solid #e3e5e7;
  border-radius: 4px;
  font-size: 13px;
  outline: none;
}
.vp-input-field::placeholder {
  color: #c0c4cc;
}

.vp-input-etiquette {
  flex-shrink: 0;
  font-size: 12px;
  color: #9499a0;
  text-decoration: none;
}
.vp-input-etiquette:hover {
  color: $b-blue;
}

.vp-input-etiquette--ghost {
  pointer-events: none;
  cursor: default;
}

.vp-input-send {
  flex-shrink: 0;
  height: 32px;
  padding: 0 18px;
  border: none;
  border-radius: 4px;
  background: $b-blue;
  color: #fff;
  font-size: 13px;
  cursor: pointer;
}
.vp-input-send:hover {
  filter: brightness(1.05);
}
.vp-input-send:disabled {
  opacity: 0.55;
  cursor: not-allowed;
  filter: none;
}

/*
 * 全屏时若仍用 aspect-ratio:16/9，舞台高度会按「整行宽度」计算，与顶栏、弹幕条叠加后
 * 总高度超过视口，flex 把 .vp-video-stage 压扁，绝对定位在舞台底部的控制条被裁到视口外。
 */
.vp-root:fullscreen,
.vp-root:-webkit-full-screen,
.vp-root:-moz-full-screen {
  width: 100%;
  height: 100%;
  min-height: 100vh;
  min-height: 100dvh;
  max-height: 100vh;
  max-height: 100dvh;
  box-sizing: border-box;
}

.vp-root:fullscreen .vp-video-stage,
.vp-root:-webkit-full-screen .vp-video-stage,
.vp-root:-moz-full-screen .vp-video-stage {
  flex: 1 1 auto;
  min-height: 0;
  aspect-ratio: unset;
  width: 100%;
}

.vp-shortcut-hint {
  position: absolute;
  left: 50%;
  top: 50%;
  z-index: 28;
  transform: translate(-50%, -50%);
  padding: 12px 28px;
  border-radius: 6px;
  background: rgba(0, 0, 0, 0.72);
  color: #fff;
  font-size: 16px;
  font-weight: 500;
  line-height: 1.4;
  pointer-events: none;
  white-space: nowrap;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.35);
}

.vp-hint-fade-enter-active,
.vp-hint-fade-leave-active {
  transition: opacity 0.22s ease;
}

.vp-hint-fade-enter-from,
.vp-hint-fade-leave-to {
  opacity: 0;
}
</style>
