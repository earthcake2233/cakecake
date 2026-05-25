<template>
  <Teleport to="body">
    <div
      v-if="modelValue"
      class="video-coin-dialog-overlay"
      role="presentation"
      @click.self="onClose"
    >
      <div
        class="video-coin-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="video-coin-dialog-title"
        @click.stop
      >
        <button
          type="button"
          class="video-coin-dialog__close"
          aria-label="关闭"
          :disabled="loading"
          @click="onClose"
        >
          ×
        </button>
        <h2 id="video-coin-dialog-title" class="video-coin-dialog__title">
          给UP主投上
          <span class="video-coin-dialog__title-num">{{ selectedAmount }}</span>
          枚硬币
        </h2>

        <div
          class="video-coin-dialog__picks"
          :class="{ 'video-coin-dialog__picks--single': singlePickOnly }"
          role="group"
          aria-label="选择投币数量"
        >
          <button
            type="button"
            class="video-coin-dialog__pick"
            :class="pickClass(1)"
            @mouseenter="hoverAmount = 1"
            @mouseleave="hoverAmount = null"
            @click="selectedAmount = 1"
          >
            <span class="video-coin-dialog__pick-label">1硬币</span>
            <img
              class="video-coin-dialog__pick-img"
              :src="selectedAmount === 1 ? img22Gif : img22Gray"
              alt=""
            />
          </button>
          <button
            v-if="!singlePickOnly"
            type="button"
            class="video-coin-dialog__pick"
            :class="pickClass(2)"
            @mouseenter="hoverAmount = 2"
            @mouseleave="hoverAmount = null"
            @click="selectedAmount = 2"
          >
            <span class="video-coin-dialog__pick-label">2硬币</span>
            <img
              class="video-coin-dialog__pick-img"
              :src="selectedAmount === 2 ? img33Gif : img33Gray"
              alt=""
            />
          </button>
        </div>

        <div class="video-coin-dialog__foot">
          <p
            v-if="selfCoinError"
            class="video-coin-dialog__self-tip"
            role="alert"
          >
            up主不能自己投币
          </p>
          <p
            v-else-if="insufficientCoins"
            class="video-coin-dialog__self-tip"
            role="alert"
          >
            硬币不足，当前余额 {{ balanceLabel }}
          </p>
          <button
            type="button"
            class="video-coin-dialog__confirm"
            :disabled="loading || insufficientCoins"
            @click="onConfirm"
          >
            确定
          </button>
          <p class="video-coin-dialog__exp">{{ expHint }}</p>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script>
import img22Gray from "@/assets/video/coin/22_gray.png";
import img33Gray from "@/assets/video/coin/33_gray.png";
import img22Gif from "@/assets/video/coin/22.gif";
import img33Gif from "@/assets/video/coin/33.gif";
import { formatCoinBalance } from "@/utils/coinBalance";

export default {
  name: "VideoCoinDialog",
  props: {
    modelValue: { type: Boolean, default: false },
    loading: { type: Boolean, default: false },
    isOwnVideo: { type: Boolean, default: false },
    coinBalance: { type: Number, default: 0 },
    /** 当前用户已给该视频投过的硬币数（0/1/2） */
    priorCoinAmount: { type: Number, default: 0 },
    dailyCoinExpProgress: { type: Number, default: 0 },
    dailyCoinExpMax: { type: Number, default: 50 }
  },
  emits: ["update:modelValue", "confirm", "cancel"],
  data() {
    return {
      img22Gray,
      img33Gray,
      img22Gif,
      img33Gif,
      selectedAmount: 2,
      hoverAmount: null,
      selfCoinError: false,
      insufficientCoins: false
    };
  },
  computed: {
    balanceLabel() {
      return formatCoinBalance(this.coinBalance);
    },
    singlePickOnly() {
      return Number(this.priorCoinAmount) === 1;
    },
    expHint() {
      const gain = Number(this.selectedAmount) * 10;
      const prog = Math.max(0, Number(this.dailyCoinExpProgress) || 0);
      const max = Number(this.dailyCoinExpMax) || 50;
      return `经验值+${gain} (今日${prog}/${max})`;
    }
  },
  watch: {
    modelValue(open) {
      if (open) {
        this.selectedAmount = this.singlePickOnly ? 1 : 2;
        this.hoverAmount = null;
        this.selfCoinError = false;
        this.insufficientCoins = false;
        this.checkInsufficient();
      }
    },
    priorCoinAmount() {
      if (this.modelValue && this.singlePickOnly) {
        this.selectedAmount = 1;
      }
    },
    selectedAmount() {
      this.selfCoinError = false;
      this.checkInsufficient();
    },
    coinBalance() {
      this.checkInsufficient();
    }
  },
  methods: {
    pickClass(n) {
      const on = this.selectedAmount === n;
      const hover = this.hoverAmount === n && !on;
      return {
        "is-on": on,
        "is-hover": hover
      };
    },
    onClose() {
      if (this.loading) return;
      this.$emit("update:modelValue", false);
      this.$emit("cancel");
    },
    checkInsufficient() {
      const bal = Number(this.coinBalance);
      const need = Number(this.selectedAmount);
      this.insufficientCoins =
        Number.isFinite(bal) && Number.isFinite(need) && bal < need;
    },
    onConfirm() {
      if (this.loading) return;
      if (this.isOwnVideo) {
        this.selfCoinError = true;
        return;
      }
      this.checkInsufficient();
      if (this.insufficientCoins) {
        return;
      }
      this.$emit("confirm", this.singlePickOnly ? 1 : this.selectedAmount);
    }
  }
};
</script>

<style lang="scss" scoped>
$coin-blue: #00aeec;
$coin-gray-border: #c9ccd0;

.video-coin-dialog-overlay {
  position: fixed;
  inset: 0;
  z-index: 10060;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  box-sizing: border-box;
  background: rgba(0, 0, 0, 0.45);
}

.video-coin-dialog {
  position: relative;
  width: 100%;
  max-width: 520px;
  padding: 28px 32px 24px;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.18);
  box-sizing: border-box;
}

.video-coin-dialog__close {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 32px;
  height: 32px;
  border: none;
  padding: 0;
  border-radius: 6px;
  background: transparent;
  color: #9499a0;
  font-size: 22px;
  line-height: 1;
  cursor: pointer;
  &:hover:not(:disabled) {
    color: #18191c;
    background: rgba(0, 0, 0, 0.05);
  }
  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
}

.video-coin-dialog__title {
  margin: 0 28px 20px;
  text-align: center;
  font-size: 16px;
  font-weight: 600;
  color: #18191c;
  line-height: 1.4;
}

.video-coin-dialog__title-num {
  margin: 0 2px;
  font-size: 22px;
  font-weight: 700;
  color: $coin-blue;
  vertical-align: -1px;
}

.video-coin-dialog__picks {
  display: flex;
  align-items: stretch;
  justify-content: center;
  gap: 20px;
  margin-bottom: 22px;

  &--single {
    .video-coin-dialog__pick {
      flex: 0 0 auto;
      width: 210px;
      max-width: 100%;
    }
  }
}

.video-coin-dialog__pick {
  position: relative;
  flex: 1;
  max-width: 210px;
  min-height: 200px;
  padding: 12px 10px 10px;
  border: 1px dashed $coin-gray-border;
  border-radius: 4px;
  background: #fff;
  cursor: pointer;
  box-sizing: border-box;
  transition: border-color 0.15s ease;
  &:hover,
  &.is-hover {
    border-color: $coin-blue;
  }
  &.is-on {
    border-style: solid;
    border-color: $coin-blue;
  }
}

.video-coin-dialog__pick-label {
  position: absolute;
  top: 10px;
  left: 12px;
  font-size: 13px;
  color: #9499a0;
  line-height: 1;
  pointer-events: none;
  .video-coin-dialog__pick.is-on & {
    color: $coin-blue;
    font-weight: 600;
  }
}

.video-coin-dialog__pick-img {
  display: block;
  width: 100%;
  max-width: 140px;
  height: auto;
  margin: 28px auto 0;
  object-fit: contain;
  pointer-events: none;
  user-select: none;
}

.video-coin-dialog__foot {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
}

.video-coin-dialog__self-tip {
  margin: 0;
  padding: 6px 14px;
  border-radius: 4px;
  background: #ff6699;
  color: #fff;
  font-size: 13px;
  line-height: 1.3;
  white-space: nowrap;
}

.video-coin-dialog__confirm {
  min-width: 132px;
  height: 40px;
  padding: 0 28px;
  border: none;
  border-radius: 6px;
  background: $coin-blue;
  color: #fff;
  font-size: 15px;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.15s ease;
  &:hover:not(:disabled) {
    opacity: 0.92;
  }
  &:disabled {
    opacity: 0.65;
    cursor: not-allowed;
  }
}

.video-coin-dialog__exp {
  margin: 0;
  font-size: 12px;
  color: #9499a0;
  line-height: 1.3;
}
</style>
