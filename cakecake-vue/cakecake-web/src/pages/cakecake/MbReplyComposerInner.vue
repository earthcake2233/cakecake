<template>
  <div class="mb-r-compose">
    <img
      class="mb-r-compose__av"
      :src="avatarSrc"
      width="48"
      height="48"
      alt=""
    />
    <div class="mb-r-compose__main">
      <div class="mb-r-compose__grid">
        <div class="mb-uni-inbox">
          <textarea
            class="mb-uni-inbox__field mb-inbox__field"
            :value="draft"
            :placeholder="inputPlaceholder"
            rows="3"
            maxlength="1000"
            @input="$emit('update:draft', $event.target.value)"
          />
          <div class="mb-uni-inbox__bar">
            <button
              type="button"
              class="mb-uni-inbox__emoji"
              title="表情（演示）"
              @click.prevent
            >
              <span class="mb-uni-inbox__emoji-ico" aria-hidden="true" />
              表情
            </button>
          </div>
        </div>
        <button
          type="button"
          class="mb-uni-send"
          :disabled="!canSend || posting"
          @click="$emit('submit')"
        >
          <template v-if="posting">发送中…</template>
          <span v-else class="mb-uni-send__lines">发表<br />评论</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: "MbReplyComposerInner",
  props: {
    inputPlaceholder: { type: String, default: "回复 @用户 :" },
    draft: { type: String, default: "" },
    posting: { type: Boolean, default: false },
    avatarSrc: { type: String, required: true }
  },
  emits: ["update:draft", "submit"],
  computed: {
    canSend() {
      return String(this.draft || "").trim().length > 0;
    }
  }
};
</script>

<style scoped lang="scss">
$accent: #00a1d6;
$accent-hover: #0097cc;
$ico-w: 1000px;
$ico-h: 1000px;

.mb-r-compose {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.mb-r-compose__av {
  flex-shrink: 0;
  border-radius: 50%;
  object-fit: cover;
}

.mb-r-compose__main {
  flex: 1;
  min-width: 0;
}

.mb-r-compose__grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 76px;
  gap: 10px;
  align-items: stretch;
}

.mb-uni-inbox {
  border: 1px solid rgba(22, 24, 35, 0.1);
  border-radius: 12px;
  background: #fff;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
  overflow: hidden;
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease;
}

.mb-uni-inbox:focus-within {
  border-color: $accent;
  box-shadow: 0 0 0 3px rgba(0, 161, 214, 0.14);
}

.mb-uni-inbox__field {
  display: block;
  width: 100%;
  box-sizing: border-box;
  margin: 0;
  padding: 12px 14px;
  min-height: 72px;
  border: none;
  resize: none;
  font-size: 15px;
  line-height: 1.55;
  color: #18191c;
  background: transparent;
  outline: none;
}

.mb-uni-inbox__field::placeholder {
  color: #9499a0;
}

.mb-uni-inbox__bar {
  display: flex;
  align-items: center;
  padding: 6px 10px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  background: #f8f9fb;
}

.mb-uni-inbox__emoji {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  padding: 4px 8px;
  border: none;
  border-radius: 8px;
  background: transparent;
  font-size: 13px;
  color: #64748b;
  cursor: default;
  transition: color 0.15s ease, background 0.15s ease;
}

.mb-uni-inbox__emoji:hover {
  color: $accent;
  background: rgba(0, 161, 214, 0.08);
}

.mb-uni-inbox__emoji-ico {
  display: inline-block;
  width: 18px;
  height: 18px;
  background-image: url("@/assets/icons-comment.2f36fc5.png");
  background-repeat: no-repeat;
  background-size: $ico-w $ico-h;
  background-position: -408px -24px;
  opacity: 0.85;
}

.mb-uni-send {
  margin: 0;
  padding: 8px 10px;
  border: none;
  border-radius: 10px;
  background: $accent;
  color: #fff;
  font-size: 13px;
  font-weight: 500;
  line-height: 1.35;
  cursor: pointer;
  white-space: normal;
  transition: background 0.15s ease, transform 0.1s ease;
}

.mb-uni-send:hover:not(:disabled) {
  background: $accent-hover;
}

.mb-uni-send:active:not(:disabled) {
  transform: scale(0.98);
}

.mb-uni-send:disabled {
  background: #e3e5e7;
  color: #fff;
  cursor: not-allowed;
}

.mb-uni-send__lines {
  display: inline-block;
  text-align: center;
}
</style>
