<template>
  <div class="mb-user-search">
    <p v-if="isUserTab" class="mb-user-search__count">{{ totalLabel }}</p>
    <ul v-if="items.length" class="mb-user-search__list">
      <li
        v-for="user in items"
        :key="`user_${user.mid}`"
        class="mb-user-search__card"
      >
        <router-link
          :to="{ name: 'minibiliUserSpace', params: { userId: user.mid } }"
          class="mb-user-search__face-link"
        >
          <img class="mb-user-search__face" :src="user.face || defaultFace" alt="" />
        </router-link>
        <div class="mb-user-search__body">
          <div class="mb-user-search__head">
            <router-link
              :to="{ name: 'minibiliUserSpace', params: { userId: user.mid } }"
              class="mb-user-search__name"
              custom
              v-slot="{ href, navigate }"
            >
              <a :href="href" class="mb-user-search__name-link" @click.prevent="navigate">
                <span v-html="user.uname"></span>
              </a>
            </router-link>
            <img
              class="mb-user-search__level"
              :src="levelBadgeSrc(user.level)"
              width="36"
              height="36"
              :alt="'LV' + (user.level || 1)"
              :title="'LV' + (user.level || 1)"
            />
            <div v-if="showFollowBtn(user)" class="mb-user-search__follow-wrap">
              <span
                v-show="followHintMid === user.mid && followHint"
                class="mb-user-search__follow-hint"
                role="status"
              >{{ followHint }}</span>
              <button
                type="button"
                class="mb-user-search__follow"
                :class="{ 'mb-user-search__follow--on': user.followed_by_me }"
                :disabled="followBusy === user.mid"
                @click="toggleFollow(user)"
              >
                {{ user.followed_by_me ? "已关注" : "+ 关注" }}
              </button>
            </div>
            <span class="mb-user-search__beta">BETA</span>
          </div>
          <p class="mb-user-search__stats">
            <span>稿件: {{ userCount(user.archives) }}</span>
            <span>粉丝: {{ userCount(user.fans) }}</span>
          </p>
          <p v-if="user.usign" class="mb-user-search__sign" :title="plainSign(user)">
            {{ plainSign(user) }}
          </p>
          <div v-if="user.recent && user.recent.length" class="mb-user-search__archives">
            <ul class="mb-user-search__archive-row">
              <li
                v-for="(arc, idx) in user.recent"
                :key="`arc_${user.mid}_${idx}`"
                class="mb-user-search__archive-item"
              >
                <router-link :to="archiveRoute(arc)" class="mb-user-search__archive-link">
                  <div class="mb-user-search__archive-pic">
                    <img v-lazy="arc.pic || defaultCover" alt="" />
                  </div>
                  <div class="mb-user-search__archive-meta">
                    <p class="mb-user-search__archive-title" :title="arc.title">
                      {{ arc.title }}
                    </p>
                    <p class="mb-user-search__archive-date">{{ formatDate(arc.pubdate) }}</p>
                  </div>
                </router-link>
              </li>
            </ul>
            <router-link
              v-if="contributeRoute(user.mid)"
              :to="contributeRoute(user.mid)"
              class="mb-user-search__all-archives"
            >
              全部{{ user.archives }}个稿件 &gt;
            </router-link>
          </div>
        </div>
      </li>
    </ul>
  </div>
</template>

<script>
import { count2, timeChange } from "../../utils/utils";
import { levelIconUrl as userLevelIconUrl, clampUserLevel } from "../../utils/userLevel";
import { minibiliUserSpaceContributeVideoRoute } from "../../utils/minibiliRoutes";
import { mbToggleUserFollow } from "../../api/minibili";
import { getAccessToken, getUserId } from "../../utils/authTokens";

const defaultFace =
  "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='80' height='80'%3E%3Ccircle fill='%23e3e5e7' cx='40' cy='40' r='40'/%3E%3C/svg%3E";
const defaultCover =
  "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='128' height='72'%3E%3Crect fill='%23e3e5e7' width='128' height='72'/%3E%3C/svg%3E";

export default {
  props: {
    items: {
      type: Array,
      default: () => []
    },
    numResults: {
      type: Number,
      default: 0
    },
    isUserTab: {
      type: Boolean,
      default: false
    }
  },
  data() {
    return {
      defaultFace,
      defaultCover,
      followBusy: 0,
      followHint: "",
      followHintMid: 0,
      _followHintTimer: null
    };
  },
  computed: {
    totalLabel() {
      const n = this.numResults;
      if (n > 1000) {
        return "共1000+条数据";
      }
      return `共${n}条数据`;
    }
  },
  beforeUnmount() {
    this.clearFollowHintTimer();
  },
  methods: {
    levelBadgeSrc(lv) {
      return userLevelIconUrl(clampUserLevel(lv));
    },
    userCount(n) {
      return count2(n);
    },
    formatDate(pubdate) {
      if (!pubdate) {
        return "";
      }
      return timeChange(pubdate);
    },
    plainSign(user) {
      return (user.usign || "").replace(/<[^>]+>/g, "");
    },
    archiveRoute(arc) {
      if (arc.rtype === "article") {
        return { name: "minibiliArticleRead", params: { id: arc.aid } };
      }
      return { name: "video", params: { aid: "BV" + arc.aid } };
    },
    contributeRoute(userId) {
      return minibiliUserSpaceContributeVideoRoute(userId);
    },
    showFollowBtn(user) {
      return !!getAccessToken();
    },
    isSelfUser(user) {
      const me = getUserId();
      const mid = Number(user.mid);
      return me != null && Number.isFinite(mid) && me === mid;
    },
    clearFollowHintTimer() {
      if (this._followHintTimer) {
        clearTimeout(this._followHintTimer);
        this._followHintTimer = null;
      }
    },
    showFollowHint(mid, message) {
      this.followHintMid = mid;
      this.followHint = String(message || "");
      this.clearFollowHintTimer();
      if (!this.followHint) {
        return;
      }
      this._followHintTimer = setTimeout(() => {
        this.followHint = "";
        this.followHintMid = 0;
        this._followHintTimer = null;
      }, 2000);
    },
    async toggleFollow(user) {
      if (!getAccessToken()) {
        this.$router.push({ name: "minibiliLogin" });
        return;
      }
      if (this.isSelfUser(user)) {
        this.showFollowHint(user.mid, "不能关注自己");
        return;
      }
      this.followBusy = user.mid;
      try {
        const res = await mbToggleUserFollow(user.mid);
        user.followed_by_me = !!res.followed;
        if (typeof res.follower_count === "number") {
          user.fans = res.follower_count;
        }
      } catch (e) {
        const msg = (e && e.message) || "操作失败";
        this.showFollowHint(user.mid, msg);
      } finally {
        this.followBusy = 0;
      }
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../../style/mixin";

.mb-user-search {
  padding-bottom: 16px;
}
.mb-user-search__count {
  padding: 16px 0 8px;
  @include sc(12px, #6d757a);
}
.mb-user-search__list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.mb-user-search__card {
  display: flex;
  gap: 16px;
  padding: 20px 0;
  border-bottom: 1px solid #e5e9ef;
  position: relative;
}
.mb-user-search__face-link {
  flex-shrink: 0;
}
.mb-user-search__face {
  @include wh(80px, 80px);
  @include borderRadius(50%);
  object-fit: cover;
  display: block;
}
.mb-user-search__body {
  flex: 1;
  min-width: 0;
}
.mb-user-search__head {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 8px;
}
.mb-user-search__name-link {
  text-decoration: none;
  :deep(.keyword),
  :deep(em.keyword) {
    color: $pink;
    font-style: normal;
  }
  span {
    @include sc(18px, #18191c);
    font-weight: 700;
    line-height: 24px;
  }
  &:hover span {
    color: $blue;
  }
}
.mb-user-search__level {
  display: inline-block;
  width: 36px;
  height: 36px;
  vertical-align: middle;
  flex-shrink: 0;
}
.mb-user-search__follow-wrap {
  position: relative;
}
.mb-user-search__follow-hint {
  position: absolute;
  left: 0;
  bottom: calc(100% + 6px);
  z-index: 5;
  padding: 6px 10px;
  border-radius: 4px;
  background: #f85a54;
  color: #fff;
  font-size: 12px;
  line-height: 1.2;
  white-space: nowrap;
  pointer-events: none;
  box-shadow: 0 2px 8px rgba(248, 90, 84, 0.35);
}
.mb-user-search__follow {
  padding: 4px 14px;
  line-height: 20px;
  @include sc(12px, #6d757a);
  background: #fff;
  border: 1px solid #ccd0d7;
  @include borderRadius(4px);
  cursor: pointer;
  &--on {
    color: #9499a0;
    background: #f1f2f3;
  }
  &:hover:not(:disabled) {
    color: $blue;
    border-color: $blue;
  }
  &:disabled {
    opacity: 0.6;
    cursor: wait;
  }
}
.mb-user-search__beta {
  position: absolute;
  top: 20px;
  right: 0;
  @include sc(11px, #c9ccd0);
  letter-spacing: 0.5px;
}
.mb-user-search__stats {
  margin: 0 0 8px;
  @include sc(12px, #9499a0);
  span + span {
    margin-left: 16px;
  }
}
.mb-user-search__sign {
  margin: 0 0 14px;
  @include sc(12px, #61666d);
  line-height: 18px;
  max-height: 36px;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}
.mb-user-search__archives {
  position: relative;
  padding-right: 130px;
}
.mb-user-search__archive-row {
  display: flex;
  gap: 16px;
  list-style: none;
  margin: 0;
  padding: 0;
}
.mb-user-search__archive-item {
  flex: 1;
  min-width: 0;
  max-width: 240px;
}
.mb-user-search__archive-link {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  text-decoration: none;
  color: inherit;
}
.mb-user-search__archive-pic {
  flex-shrink: 0;
  @include wh(128px, 72px);
  @include borderRadius(4px);
  overflow: hidden;
  background: #e3e5e7;
  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }
}
.mb-user-search__archive-meta {
  flex: 1;
  min-width: 0;
  padding-top: 2px;
}
.mb-user-search__archive-title {
  margin: 0 0 6px;
  @include sc(12px, #18191c);
  line-height: 16px;
  max-height: 32px;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}
.mb-user-search__archive-date {
  margin: 0;
  @include sc(12px, #9499a0);
  line-height: 16px;
}
.mb-user-search__all-archives {
  position: absolute;
  right: 0;
  bottom: 4px;
  @include sc(12px, $blue);
  text-decoration: none;
  white-space: nowrap;
  &:hover {
    text-decoration: underline;
  }
}
.mb-user-search__empty {
  padding: 48px 0;
  text-align: center;
  @include sc(14px, #9499a0);
}
</style>
