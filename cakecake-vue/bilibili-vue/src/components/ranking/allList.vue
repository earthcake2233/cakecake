<template>
  <div class="ranking-list-wrap">
    <div v-if="showComingSoon" class="rank-coming-soon">
      <img class="rank-coming-soon__img" :src="emptyImg" alt="" />
      <p class="rank-coming-soon__text">该功能即将开放</p>
    </div>
    <ul
      v-else-if="!loading && rankList.length"
      class="rank-list"
      :class="{ bangumi: type == 2 }"
    >
      <li
        class="rank-item rank-item--nav"
        v-for="(item, index) in rankList"
        :key="`rankall_list_${index}`"
        @click="onRankRowClick(item, $event)"
      >
        <div class="num">
          {{ index + 1 }}
        </div>
        <div class="content">
          <div class="img">
            <div class="cover">
              <img :alt="item.title" v-lazy="thumbSrc(item)" />
            </div>
            <div
              class="watch-later-trigger w-later"
              data-rank-stop
              @click.stop
            />
          </div>
          <div class="info">
            <p class="title rank-item__title">{{ item.title }}</p>
            <div class="bangumi-info" v-if="type == 2">
              连载中，更新至第
              <span class="bangumi-num">{{ item.newest_ep_index }}</span>
              话
            </div>
            <div class="detail">
              <span class="data-box" v-if="type !== 2">
                <i class="b-icon play"></i>
                {{ count2(item.play) }}
              </span>
              <span class="data-box">
                <i class="b-icon view"></i>
                {{
                  type == 2 ? count2(item.dm_count) : count2(item.video_review)
                }}
              </span>
              <span class="data-box" v-if="type == 2">
                <i class="fav"></i>
                {{ count2(item.fav) }}
              </span>
              <a
                v-if="type !== 2 && authorSpaceRoute(item)"
                href="javascript:;"
                class="rank-item__author-hit"
                data-rank-stop
                @click.stop.prevent="goAuthorSpace(item)"
              >
                <span class="data-box">
                  <i class="b-icon author"></i>
                  {{ item.author }}
                </span>
              </a>
              <span v-else-if="type !== 2" class="data-box">
                <i class="b-icon author"></i>
                {{ item.author }}
              </span>
            </div>
            <div class="pts">
              <div>
                {{ item.pts }}
              </div>
              综合得分
            </div>
          </div>
        </div>
      </li>
    </ul>
    <bili-loading v-if="loading" style="top:20%;"></bili-loading>
  </div>
</template>

<script>
import biliLoading from "../loading/loading";
import { count2 } from "../../utils/utils";
import akariCover from "@/assets/akari.jpg";
import emptyImg from "@/assets/empty.png";
import { formatVideoBvid } from "@/utils/videoBvid";
import { minibiliUserSpaceRoute } from "@/utils/minibiliRoutes";

export default {
  props: {
    rankAll: {
      type: [Object, Array],
      default: () => []
    },
    type: {
      type: [Number, String]
    },
    loading: {
      type: Boolean,
      default: true
    }
  },
  components: {
    biliLoading
  },
  data() {
    return {
      emptyImg
    };
  },
  computed: {
    isMb() {
      return (
        import.meta.env.VITE_MINIBILI_API === "true" ||
        import.meta.env.VITE_MINIBILI_API === "1"
      );
    },
    rankList() {
      const raw = this.rankAll && this.rankAll.list;
      return Array.isArray(raw) ? raw : [];
    },
    /** Mini-Bili：原创/新番/影视/新人榜暂未接入 */
    showComingSoon() {
      if (!this.isMb) {
        return false;
      }
      const t = Number(this.type);
      return t >= 1 && t <= 4;
    }
  },
  methods: {
    count2(num) {
      return count2(num);
    },
    thumbSrc(item) {
      const src = this.type == 2 ? item.cover : item.pic;
      const s = src != null ? String(src).trim() : "";
      return s || akariCover;
    },
    videoPlayRoute(item) {
      const id = Number(item && item.aid);
      const aid =
        Number.isFinite(id) && id > 0 ? formatVideoBvid(id) : String(item.aid || "0");
      return { name: "video", params: { aid } };
    },
    authorSpaceRoute(item) {
      if (!this.isMb) {
        return null;
      }
      const uid = Number(item && (item.userId ?? item.user_id));
      return minibiliUserSpaceRoute(uid);
    },
    onRankRowClick(item, e) {
      if (e && e.target && e.target.closest("[data-rank-stop]")) {
        return;
      }
      const route = this.videoPlayRoute(item);
      const resolved = this.$router.resolve(route);
      const href = resolved.href;
      if (!href) return;
      window.open(href, "_blank", "noopener,noreferrer");
    },
    goAuthorSpace(item) {
      const route = this.authorSpaceRoute(item);
      if (!route) return;
      this.$router.push(route);
    }
  }
};
</script>

<style lang="scss" scoped>
.rank-item--nav {
  cursor: pointer;
}

.rank-item__title {
  margin: 0;
  height: 20px;
  line-height: 20px;
  font-weight: 700;
  font-size: 14px;
  color: #212121;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rank-item--nav:hover .rank-item__title {
  color: #00a1d6;
}

.rank-item__author-hit {
  color: inherit;
  text-decoration: none;

  &:hover .data-box {
    color: #00a1d6;
  }
}

.rank-coming-soon {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 420px;
  padding: 48px 24px 64px;
  box-sizing: border-box;
}

.rank-coming-soon__img {
  display: block;
  width: 280px;
  max-width: 100%;
  height: auto;
  object-fit: contain;
}

.rank-coming-soon__text {
  margin: 20px 0 0;
  font-size: 14px;
  line-height: 22px;
  color: #9499a0;
}
</style>
