<template>
  <div class="recommend-module">
    <div class="recommend-panel">
      <div class="recommend-grid">
        <div
          class="groom-module home-card"
          v-for="(item, index) in displayItems"
          :key="'rec-' + item.aid + '-' + index"
        >
          <div class="groom-cover video-thumb-hover">
            <router-link
              :to="{ name: 'video', params: { aid: 'BV' + item.aid } }"
              :title="item.title"
              class="groom-cover__link"
            >
              <img
                v-lazy="item.pic"
                :alt="item.title"
                width="160"
                height="100"
                class="pic"
              />
              <div class="card-mark">
                <p class="title">{{ item.title }}</p>
                <p class="author">up主：{{ item.author }}</p>
                <p class="play">播放：{{ item.play }}</p>
              </div>
            </router-link>
            <WatchLaterBtn
            :video-id="item.aid"
            :in-watch-later="!!item.in_watch_later"
          />
          </div>
        </div>
      </div>
      <button
        type="button"
        class="rec-btn rec-left"
        aria-label="刷新"
        @click="refresh(-1)"
      >
        <i class="rec-btn-arrow rec-btn-arrow--prev" aria-hidden="true"></i>
        <span class="rec-btn-label"><i>刷</i><i>新</i></span>
      </button>
      <button
        type="button"
        class="rec-btn rec-right"
        aria-label="刷新"
        @click="refresh(1)"
      >
        <span class="rec-btn-label"><i>刷</i><i>新</i></span>
        <i class="rec-btn-arrow rec-btn-arrow--next" aria-hidden="true"></i>
      </button>
    </div>
  </div>
</template>

<script>
import { getHomeRecommendPool } from "../../api";
import {
  fillHomeRecommendSlots,
  HOME_RECOMMEND_PAGE_SIZE,
  nextHomeRecommendBatch
} from "../../utils/videoRecommendFeeds";
import WatchLaterBtn from "../common/WatchLaterBtn.vue";

export default {
  components: { WatchLaterBtn },
  props: {
    recommend: {
      default: () => ({ rec: [], day: 3 })
    }
  },
  data() {
    return {
      pool: [],
      displayItems: [],
      batchOffset: 0,
      poolLoading: false
    };
  },
  watch: {
    "recommend.rec": {
      immediate: true,
      handler(rec) {
        this.mergePool(rec);
        if (this.pool.length) {
          this.applyDisplay(this.batchOffset);
        }
      }
    }
  },
  mounted() {
    this.loadPool();
  },
  methods: {
    mergePool(list) {
      const seen = new Set(
        this.pool
          .map(v => Number(v.aid))
          .filter(id => Number.isFinite(id) && id > 0)
      );
      for (const item of list || []) {
        const id = Number(item.aid);
        if (!Number.isFinite(id) || id <= 0 || seen.has(id)) continue;
        seen.add(id);
        this.pool.push(item);
      }
    },
    async loadPool() {
      if (this.poolLoading) return;
      this.poolLoading = true;
      try {
        const list = await getHomeRecommendPool(48);
        this.mergePool(list);
        if (this.pool.length) {
          this.applyDisplay(this.batchOffset);
        }
      } finally {
        this.poolLoading = false;
      }
    },
    applyDisplay(offset = 0) {
      const slots = fillHomeRecommendSlots(
        this.pool,
        offset,
        HOME_RECOMMEND_PAGE_SIZE
      );
      this.displayItems =
        slots.length >= HOME_RECOMMEND_PAGE_SIZE
          ? slots.slice(0, HOME_RECOMMEND_PAGE_SIZE)
          : slots;
      this.batchOffset = offset;
    },
    refresh(direction) {
      if (!this.pool.length) {
        this.loadPool();
        return;
      }
      const { items, nextOffset } = nextHomeRecommendBatch(
        this.pool,
        this.displayItems,
        this.batchOffset,
        direction
      );
      if (items.length) {
        this.displayItems = items;
        this.batchOffset = nextOffset;
      } else {
        this.applyDisplay(this.batchOffset);
      }
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style lang="scss" scoped>
@import "../../style/mixin";

.recommend-module {
  float: left;
  width: calc(100% - 440px);
  height: 220px;
  position: relative;
  box-sizing: border-box;
}

.recommend-panel {
  position: relative;
  width: 100%;
  height: 220px;
  box-sizing: border-box;

  &:hover .rec-btn {
    opacity: 1;
    visibility: visible;
    pointer-events: auto;
  }
}

/* 4×2 网格：在轮播右侧剩余宽度内始终排满 8 格（避免固定 700px 被裁成 3 列） */
.recommend-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  grid-template-rows: repeat(2, 100px);
  gap: 20px;
  padding-left: 20px;
  padding-right: 0;
  width: 100%;
  height: 220px;
  box-sizing: border-box;
  overflow: visible;
}

.groom-module {
  margin: 0;
  width: 100%;
  height: 100px;
  @include borderRadius(4px);
  position: relative;
  overflow: hidden;
  background: $white;

  .groom-cover {
    position: relative;
    width: 100%;
    height: 100px;
    overflow: hidden;
    @include borderRadius(4px);
  }

  .groom-cover__link {
    display: block;
    position: relative;
    width: 100%;
    height: 100px;
  }

  .pic {
    width: 100%;
    height: 100px;
    display: block;
    object-fit: cover;
  }
  .card-mark {
    position: absolute;
    left: 0;
    top: 68px;
    @include wh(150px, 20px);
    font-size: 12px;
    line-height: 20px;
    padding: 10px 5px;
    overflow: hidden;
    background: linear-gradient(
      transparent,
      rgba(0, 0, 0, 0.1) 20%,
      rgba(0, 0, 0, 0.2) 35%,
      rgba(0, 0, 0, 0.6) 65%,
      rgba(0, 0, 0, 0.9)
    );
    .title {
      height: 40px;
      @include sc(12px, $white);
      overflow: hidden;
      margin-bottom: 5px;
      white-space: nowrap;
      text-overflow: ellipsis;
      line-height: 20px;
    }
  }
  .groom-cover:hover .card-mark,
  &:hover .card-mark {
    top: 0;
    height: 100%;
    background: rgba(0, 0, 0, 0.7);
    .title {
      -webkit-box-orient: vertical;
      -webkit-line-clamp: 2;
      display: -webkit-box;
      white-space: inherit;
    }
  }
}
.groom-module .card-mark .author,
.groom-module .card-mark .play {
  opacity: 0;
  @include sc(12px, $grau);
  height: 20px;
  line-height: 20px;
  overflow: hidden;
  @include transition(0.6s);
}
.groom-module:hover .card-mark .author,
.groom-module:hover .card-mark .play {
  opacity: 1;
}

.rec-btn {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  z-index: 5;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 24px;
  min-height: 76px;
  padding: 12px 5px;
  border: none;
  cursor: pointer;
  background-color: rgba(0, 0, 0, 0.55);
  @include sc(12px, $white);
  @include transition(0.2s);
  opacity: 0;
  visibility: hidden;
  pointer-events: none;
  font-style: normal;

  &:hover {
    background-color: rgba(0, 0, 0, 0.72);
  }
}

.rec-btn-label {
  display: flex;
  flex-direction: column;
  align-items: center;
  line-height: 1.35;
  font-size: 12px;

  i {
    font-style: normal;
  }
}

.rec-btn-arrow {
  display: block;
  flex-shrink: 0;
  @include wh(7px, 12px);
  background-image: url(../../assets/icons2.png);
  background-repeat: no-repeat;
}

.rec-btn-arrow--prev {
  background-position: -478px -218px;
  transform: scaleX(-1);
}

.rec-btn-arrow--next {
  background-position: -478px -218px;
}

.rec-left {
  left: 20px;
  border-radius: 0 4px 4px 0;
  padding: 12px 6px 12px 8px;
}

.rec-right {
  right: 0;
  border-radius: 4px 0 0 4px;
  padding: 12px 8px 12px 6px;
}
</style>
