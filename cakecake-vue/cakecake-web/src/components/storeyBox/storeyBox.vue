<template>
  <div class="new-comers-module l-con">
    <div class="zone-title">
      <div
        class="headline"
        :class="{ fj: storeydata.rid == 13 || storeydata.rid == 168 }"
      >
        <i class="icon icon_t" :class="storeydata.icon"></i>
        <a :href="storeydata.moreUrl" class="name">{{ title }}</a>
        <div class="bili-tab">
          <div
            class="bili-tab-item"
            v-for="(item, index) in storeydata.tab"
            :key="`storey_data_${index}`"
            :class="{ on: index === nowtab }"
            @click="nowtabclick(index)"
          >
            {{ item.name }}
          </div>
        </div>
        <a :href="storeydata.moreUrl" target="_blank" class="link-more">
          更多
          <i class="icon"></i>
        </a>
        <div
          class="read-push"
          :class="{ refreshing }"
          role="button"
          tabindex="0"
          title="刷新"
          @click="refreshZone"
          @keydown.enter.prevent="refreshZone"
          @keydown.space.prevent="refreshZone"
        >
          <i class="icon icon_read"></i>
          <span class="info">
            <b>{{ storeydata.dynamic }}</b
            >条新动态
          </span>
        </div>
      </div>
    </div>
    <div class="storey-box" v-if="storeydata.data">
      <template
        v-for="(item, index) in storeydata.data.archives"
        :key="`storeydata_data_archives_${index}`"
      >
        <div v-if="index < 10" class="spread-module">
          <div class="spread-thumb video-thumb-hover">
            <div class="pic">
              <router-link
                :to="{ name: 'video', params: { aid: 'BV' + item.aid } }"
                :title="item.title"
                class="spread-thumb__link"
              >
                <div class="lazy-img">
                  <img :alt="item.title" v-lazy="item.pic" />
                </div>
                <i class="icon medal "></i>
                <div class="mask-video"></div>
                <span class="dur">{{ formatDuration(item.duration) }}</span>
              </router-link>
              <WatchLaterBtn
                :video-id="item.aid"
                :in-watch-later="!!item.in_watch_later"
              />
            </div>
          </div>
          <router-link
            :to="{ name: 'video', params: { aid: 'BV' + item.aid } }"
            :title="item.title"
            class="spread-meta__link"
          >
            <p :title="item.title" class="t">{{ item.title }}</p>
            <p class="num">
              <span class="play">
                <i class="icon"></i>{{ count2(item.stat.view) }}
              </span>
              <span class="danmu">
                <i class="icon"></i>{{ count2(item.stat.danmaku) }}
              </span>
            </p>
          </router-link>
        </div>
      </template>
    </div>
  </div>
</template>

<script>
import { count2 } from "../../utils/utils";
import { formatDuration as fmtVideoDuration } from "../../utils/formatDuration";
import WatchLaterBtn from "../common/WatchLaterBtn.vue";

export default {
  components: { WatchLaterBtn },
  mounted() {},
  watch: {
    offsetTop() {
      this.getData();
    },
    scrollTop() {
      this.getData();
    }
  },
  props: {
    scrollTop: {
      type: Number,
      default: 0
    },
    storeydata: {
      type: [Object, Array],
      default: () => []
    }
  },
  computed: {
    offsetTop() {
      return this.storeydata.offsetTop;
    },
    title() {
      return this.storeydata.rid == 13 || this.storeydata.rid == 168
        ? this.storeydata.title2
        : this.storeydata.title;
    }
  },
  data() {
    return {
      loading: true,
      refreshing: false,
      nowtab: 0
    };
  },
  methods: {
    getData() {
      if (
        this.scrollTop + 900 > this.storeydata.offsetTop &&
        this.loading == true
      ) {
        this.loading = false;
        // 默认显示新动态
        this.$emit("setDynamicRegion", {
          id: this.storeydata.id,
          ps: 10,
          rid: this.storeydata.rid
        });
      }
    },
    zonePayload() {
      return {
        id: this.storeydata.id,
        ps: 10,
        rid: this.storeydata.rid
      };
    },
    loadZoneTab(index) {
      const payload = this.zonePayload();
      if (index === 0) {
        this.$emit("setDynamicRegion", payload);
      } else if (index === 1) {
        this.$emit("setNewlist", payload);
      }
    },
    nowtabclick(index) {
      this.nowtab = index;
      this.loadZoneTab(index);
    },
    async refreshZone() {
      if (this.refreshing) {
        return;
      }
      this.refreshing = true;
      const action = this.nowtab === 0 ? "setDynamicRegion" : "setNewlist";
      try {
        await this.$store.dispatch(action, this.zonePayload());
      } catch (err) {
        console.warn("分区刷新失败", err);
      } finally {
        this.refreshing = false;
      }
    },
    formatDuration(sec) {
      return fmtVideoDuration(sec);
    },
    count2(num) {
      return count2(num);
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../../style/mixin";

.spread-thumb {
  position: relative;
  width: 160px;
  height: 100px;
  overflow: hidden;
  border-radius: 4px;

  .pic {
    position: relative;
    width: 160px;
    height: 100px;
  }

  &:hover :deep(.home-wl-btn.watch-later-trigger) {
    opacity: 1 !important;
    visibility: visible !important;
    pointer-events: auto !important;
  }
}

.spread-thumb__link {
  display: block;
  width: 160px;
  height: 100px;
}

.spread-meta__link {
  display: block;
  color: inherit;
  text-decoration: none;
}

.read-push {
  &.refreshing {
    pointer-events: none;
    opacity: 0.85;

    .icon_read {
      animation: storey-read-spin 0.6s linear infinite;
    }
  }
}

@keyframes storey-read-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
