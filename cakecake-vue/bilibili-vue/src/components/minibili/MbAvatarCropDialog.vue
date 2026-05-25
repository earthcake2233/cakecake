<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="mb-avatar-crop-dim"
      role="presentation"
      @click.self="onCancel"
    >
      <div
        class="mb-avatar-crop-modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="mb-avatar-crop-title"
        @click.stop
      >
        <header class="mb-avatar-crop-modal__hd">
          <h2 id="mb-avatar-crop-title" class="mb-avatar-crop-modal__title">
            裁剪头像
          </h2>
          <button
            type="button"
            class="mb-avatar-crop-modal__close"
            aria-label="关闭"
            @click="onCancel"
          >
            ×
          </button>
        </header>
        <div
          ref="viewportRef"
          class="mb-avatar-crop-viewport"
          :class="{ 'is-panning': dragMode === 'pan' }"
          @pointerdown="onViewportPointerDown"
          @pointermove="onPointerMove"
          @pointerup="onPointerUp"
          @pointercancel="onPointerUp"
          @wheel.prevent="onWheel"
        >
          <img
            v-if="src"
            ref="imgRef"
            class="mb-avatar-crop-img"
            :src="src"
            alt=""
            draggable="false"
            :style="imgStyle"
            @load="onImgLoad"
          />
          <div
            class="mb-avatar-crop-frame"
            :class="{ 'mb-avatar-crop-frame--circle': circular }"
            :style="frameStyle"
          >
            <span
              v-for="h in resizeHandles"
              :key="h"
              class="mb-avatar-crop-handle"
              :class="`mb-avatar-crop-handle--${h}`"
              @pointerdown.stop="onResizePointerDown($event)"
            />
          </div>
        </div>
        <p class="mb-avatar-crop-hint">
          {{
            circular
              ? "拖动图片调整位置；拖角或滚轮缩放圆形选框"
              : "拖动图片调整位置；拖角或滚轮缩放选框"
          }}
        </p>
        <div class="mb-avatar-crop-zoom-row">
          <button
            type="button"
            class="mb-avatar-crop-zoom-btn"
            :disabled="!imgReady || cropSize <= minCropSize"
            aria-label="缩小选框"
            @click="nudgeCropSize(-16)"
          >
            −
          </button>
          <input
            v-model.number="cropSizeSlider"
            type="range"
            class="mb-avatar-crop-zoom-range"
            :min="minCropSize"
            :max="maxCropSize"
            :disabled="!imgReady"
          />
          <button
            type="button"
            class="mb-avatar-crop-zoom-btn"
            :disabled="!imgReady || cropSize >= maxCropSize"
            aria-label="放大选框"
            @click="nudgeCropSize(16)"
          >
            +
          </button>
        </div>
        <footer class="mb-avatar-crop-modal__ft">
          <button
            type="button"
            class="mb-avatar-crop-btn mb-avatar-crop-btn--ghost"
            @click="onCancel"
          >
            取消
          </button>
          <button
            type="button"
            class="mb-avatar-crop-btn mb-avatar-crop-btn--primary"
            :disabled="!imgReady || exporting"
            @click="onConfirm"
          >
            {{ exporting ? "处理中…" : "确定" }}
          </button>
        </footer>
      </div>
    </div>
  </Teleport>
</template>

<script>
const VIEWPORT = 320;
const OUTPUT_SIZE = 400;
const MIN_CROP = 96;
const MAX_CROP = 300;
const DEFAULT_CROP = 240;

export default {
  name: "MbAvatarCropDialog",
  props: {
    visible: { type: Boolean, default: false },
    src: { type: String, default: "" },
    fileName: { type: String, default: "avatar.jpg" },
    /** 圆形选框与圆形导出（头像场景默认开启） */
    circular: { type: Boolean, default: true }
  },
  emits: ["cancel", "confirm", "error"],
  data() {
    return {
      imgReady: false,
      exporting: false,
      naturalW: 0,
      naturalH: 0,
      cropSize: DEFAULT_CROP,
      minCropSize: MIN_CROP,
      maxCropSize: MAX_CROP,
      offsetX: 0,
      offsetY: 0,
      dispW: 0,
      dispH: 0,
      scale: 1,
      dragMode: null,
      dragStartX: 0,
      dragStartY: 0,
      dragOrigX: 0,
      dragOrigY: 0,
      dragOrigCropSize: DEFAULT_CROP,
      dragOrigDist: 0,
      resizeHandles: ["nw", "ne", "sw", "se"]
    };
  },
  computed: {
    imgStyle() {
      return {
        left: `${this.offsetX}px`,
        top: `${this.offsetY}px`,
        width: `${this.dispW}px`,
        height: `${this.dispH}px`
      };
    },
    cropLeft() {
      return (VIEWPORT - this.cropSize) / 2;
    },
    cropTop() {
      return (VIEWPORT - this.cropSize) / 2;
    },
    frameStyle() {
      return {
        left: `${this.cropLeft}px`,
        top: `${this.cropTop}px`,
        width: `${this.cropSize}px`,
        height: `${this.cropSize}px`
      };
    },
    cropSizeSlider: {
      get() {
        return this.cropSize;
      },
      set(v) {
        this.setCropSize(Number(v));
      }
    }
  },
  watch: {
    visible(v) {
      if (!v) {
        this.resetTransform();
      }
    },
    src() {
      this.resetTransform();
    }
  },
  methods: {
    resetTransform() {
      this.imgReady = false;
      this.exporting = false;
      this.naturalW = 0;
      this.naturalH = 0;
      this.cropSize = DEFAULT_CROP;
      this.offsetX = 0;
      this.offsetY = 0;
      this.dispW = 0;
      this.dispH = 0;
      this.scale = 1;
      this.dragMode = null;
    },
    onImgLoad() {
      const img = this.$refs.imgRef;
      if (!img || !img.naturalWidth) return;
      this.naturalW = img.naturalWidth;
      this.naturalH = img.naturalHeight;
      this.cropSize = DEFAULT_CROP;
      this.applyImageLayout(true);
      this.imgReady = true;
    },
    minScaleForCrop(size) {
      const s = size || this.cropSize;
      return Math.max(s / this.naturalW, s / this.naturalH);
    },
    applyImageLayout(recenter) {
      if (!this.naturalW || !this.naturalH) return;
      const need = this.minScaleForCrop();
      if (this.scale < need || recenter) {
        this.scale = need;
      }
      this.dispW = this.naturalW * this.scale;
      this.dispH = this.naturalH * this.scale;
      if (recenter) {
        this.offsetX = (VIEWPORT - this.dispW) / 2;
        this.offsetY = (VIEWPORT - this.dispH) / 2;
      }
      this.clampOffset();
    },
    setCropSize(next) {
      const size = Math.round(
        Math.min(this.maxCropSize, Math.max(this.minCropSize, Number(next) || DEFAULT_CROP))
      );
      if (size === this.cropSize) return;
      const cx = this.offsetX + this.dispW / 2;
      const cy = this.offsetY + this.dispH / 2;
      this.cropSize = size;
      this.applyImageLayout(false);
      this.offsetX = cx - this.dispW / 2;
      this.offsetY = cy - this.dispH / 2;
      this.clampOffset();
    },
    nudgeCropSize(delta) {
      this.setCropSize(this.cropSize + delta);
    },
    onWheel(e) {
      if (!this.imgReady) return;
      const step = e.deltaY > 0 ? -10 : 10;
      this.nudgeCropSize(step);
    },
    dragBounds() {
      const cropR = this.cropLeft + this.cropSize;
      const cropB = this.cropTop + this.cropSize;
      return {
        minX: cropR - this.dispW,
        maxX: this.cropLeft,
        minY: cropB - this.dispH,
        maxY: this.cropTop
      };
    },
    clampOffset() {
      const b = this.dragBounds();
      this.offsetX = Math.min(b.maxX, Math.max(b.minX, this.offsetX));
      this.offsetY = Math.min(b.maxY, Math.max(b.minY, this.offsetY));
    },
    viewportCenterClient() {
      const el = this.$refs.viewportRef;
      if (!el) return { x: 0, y: 0 };
      const r = el.getBoundingClientRect();
      return { x: r.left + VIEWPORT / 2, y: r.top + VIEWPORT / 2 };
    },
    pointerDistFromCenter(clientX, clientY) {
      const c = this.viewportCenterClient();
      return Math.hypot(clientX - c.x, clientY - c.y);
    },
    onViewportPointerDown(e) {
      if (!this.imgReady || e.button !== 0) return;
      if (e.target && e.target.classList.contains("mb-avatar-crop-handle")) {
        return;
      }
      this.dragMode = "pan";
      this.dragStartX = e.clientX;
      this.dragStartY = e.clientY;
      this.dragOrigX = this.offsetX;
      this.dragOrigY = this.offsetY;
      this.captureViewport(e);
    },
    onResizePointerDown(e) {
      if (!this.imgReady || e.button !== 0) return;
      this.dragMode = "resize";
      this.dragOrigCropSize = this.cropSize;
      this.dragOrigDist = this.pointerDistFromCenter(e.clientX, e.clientY);
      this.captureViewport(e);
    },
    captureViewport(e) {
      try {
        this.$refs.viewportRef.setPointerCapture(e.pointerId);
      } catch {
        /* noop */
      }
    },
    onPointerMove(e) {
      if (!this.dragMode) return;
      if (this.dragMode === "pan") {
        this.offsetX = this.dragOrigX + (e.clientX - this.dragStartX);
        this.offsetY = this.dragOrigY + (e.clientY - this.dragStartY);
        this.clampOffset();
        return;
      }
      if (this.dragMode === "resize") {
        const dist = this.pointerDistFromCenter(e.clientX, e.clientY);
        const halfDelta = dist - this.dragOrigDist;
        this.setCropSize(this.dragOrigCropSize + halfDelta * 2);
      }
    },
    onPointerUp() {
      this.dragMode = null;
    },
    onCancel() {
      if (this.exporting) return;
      this.$emit("cancel");
    },
    async onConfirm() {
      if (!this.imgReady || this.exporting) return;
      const img = this.$refs.imgRef;
      if (!img) return;
      this.exporting = true;
      try {
        const file = await this.exportCroppedFile(img);
        this.$emit("confirm", file);
      } catch {
        this.exporting = false;
        this.$emit("error");
      }
    },
    exportCroppedFile(img) {
      const nx = (this.cropLeft - this.offsetX) / this.scale;
      const ny = (this.cropTop - this.offsetY) / this.scale;
      const nw = this.cropSize / this.scale;
      const nh = this.cropSize / this.scale;
      const canvas = document.createElement("canvas");
      canvas.width = OUTPUT_SIZE;
      canvas.height = OUTPUT_SIZE;
      const ctx = canvas.getContext("2d");
      if (!ctx) {
        return Promise.reject(new Error("canvas"));
      }
      ctx.fillStyle = "#ffffff";
      ctx.fillRect(0, 0, OUTPUT_SIZE, OUTPUT_SIZE);
      if (this.circular) {
        ctx.save();
        ctx.beginPath();
        ctx.arc(OUTPUT_SIZE / 2, OUTPUT_SIZE / 2, OUTPUT_SIZE / 2, 0, Math.PI * 2);
        ctx.closePath();
        ctx.clip();
      }
      ctx.drawImage(img, nx, ny, nw, nh, 0, 0, OUTPUT_SIZE, OUTPUT_SIZE);
      if (this.circular) {
        ctx.restore();
      }
      const mime = this.pickOutputMime();
      return new Promise((resolve, reject) => {
        canvas.toBlob(
          blob => {
            if (!blob) {
              reject(new Error("empty blob"));
              return;
            }
            const base =
              String(this.fileName || "avatar")
                .replace(/\.[^.]+$/, "")
                .slice(0, 48) || "avatar";
            const ext = mime === "image/png" ? ".png" : ".jpg";
            resolve(new File([blob], `${base}${ext}`, { type: mime }));
          },
          mime,
          mime === "image/jpeg" ? 0.92 : undefined
        );
      });
    },
    pickOutputMime() {
      const n = String(this.fileName || "").toLowerCase();
      if (n.endsWith(".png") || n.endsWith(".webp")) {
        return "image/png";
      }
      return "image/jpeg";
    }
  }
};
</script>

<style lang="scss" scoped>
$b-blue: #00a1d6;

.mb-avatar-crop-dim {
  position: fixed;
  inset: 0;
  z-index: 5200;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: rgba(0, 0, 0, 0.55);
}

.mb-avatar-crop-modal {
  width: 100%;
  max-width: 420px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.2);
  overflow: hidden;
}

.mb-avatar-crop-modal__hd {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  border-bottom: 1px solid #e3e5e7;
}

.mb-avatar-crop-modal__title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #18191c;
}

.mb-avatar-crop-modal__close {
  border: none;
  background: transparent;
  font-size: 22px;
  line-height: 1;
  color: #9499a0;
  cursor: pointer;
  padding: 0 4px;
  &:hover {
    color: #18191c;
  }
}

.mb-avatar-crop-viewport {
  position: relative;
  width: 320px;
  height: 320px;
  margin: 20px auto 8px;
  overflow: hidden;
  background: #1a1a1a;
  cursor: grab;
  touch-action: none;
  user-select: none;
  &.is-panning:active {
    cursor: grabbing;
  }
}

.mb-avatar-crop-img {
  position: absolute;
  max-width: none;
  pointer-events: none;
}

.mb-avatar-crop-frame {
  position: absolute;
  border: 2px solid #fff;
  border-radius: 4px;
  box-shadow: 0 0 0 9999px rgba(0, 0, 0, 0.55);
  pointer-events: none;
  z-index: 2;
  &--circle {
    border-radius: 50%;
  }
}

.mb-avatar-crop-handle {
  position: absolute;
  width: 14px;
  height: 14px;
  background: #fff;
  border: 2px solid $b-blue;
  border-radius: 2px;
  pointer-events: auto;
  z-index: 3;
  box-sizing: border-box;
  &--nw {
    left: -8px;
    top: -8px;
    cursor: nwse-resize;
  }
  &--ne {
    right: -8px;
    top: -8px;
    cursor: nesw-resize;
  }
  &--sw {
    left: -8px;
    bottom: -8px;
    cursor: nesw-resize;
  }
  &--se {
    right: -8px;
    bottom: -8px;
    cursor: nwse-resize;
  }
}

.mb-avatar-crop-hint {
  margin: 0 0 10px;
  text-align: center;
  font-size: 12px;
  color: #9499a0;
}

.mb-avatar-crop-zoom-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 20px 12px;
}

.mb-avatar-crop-zoom-btn {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border: 1px solid #e3e5e7;
  border-radius: 4px;
  background: #fff;
  font-size: 18px;
  line-height: 1;
  color: #61666d;
  cursor: pointer;
  &:hover:not(:disabled) {
    border-color: $b-blue;
    color: $b-blue;
  }
  &:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }
}

.mb-avatar-crop-zoom-range {
  flex: 1;
  height: 4px;
  accent-color: $b-blue;
}

.mb-avatar-crop-modal__ft {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 0 16px 16px;
}

.mb-avatar-crop-btn {
  min-width: 88px;
  height: 36px;
  padding: 0 16px;
  border-radius: 4px;
  font-size: 14px;
  cursor: pointer;
  border: 1px solid transparent;
  &--ghost {
    background: #fff;
    border-color: #e3e5e7;
    color: #61666d;
    &:hover {
      border-color: $b-blue;
      color: $b-blue;
    }
  }
  &--primary {
    background: $b-blue;
    color: #fff;
    &:hover:not(:disabled) {
      filter: brightness(1.05);
    }
    &:disabled {
      opacity: 0.55;
      cursor: not-allowed;
    }
  }
}
</style>
