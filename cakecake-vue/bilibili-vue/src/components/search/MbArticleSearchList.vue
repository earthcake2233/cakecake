<template>
  <div class="mb-article-search">
    <div class="mb-article-search__bar">
      <span class="mb-article-search__count">{{ totalLabel }}</span>
      <ul class="mb-article-search__sort">
        <li
          v-for="opt in sortOptions"
          :key="opt.id"
          class="mb-article-search__sort-item"
          :class="{ 'mb-article-search__sort-item--on': sort === opt.id }"
        >
          <a href="javascript:;" @click.prevent="$emit('sort-change', opt.id)">{{
            opt.label
          }}</a>
        </li>
      </ul>
    </div>
    <ul v-if="items.length" class="mb-article-search__list">
      <li
        v-for="item in items"
        :key="`article_${item.id}`"
        class="mb-article-search__row"
      >
        <div class="mb-article-search__main">
          <router-link
            :to="{ name: 'minibiliArticleRead', params: { id: item.id } }"
            class="mb-article-search__title"
            custom
            v-slot="{ href, navigate }"
          >
            <a :href="href" @click.prevent="navigate">
              <span v-html="item.title"></span>
            </a>
          </router-link>
          <p v-if="item.desc" class="mb-article-search__desc" :title="plainDesc(item)">
            {{ plainDesc(item) }}
          </p>
          <div class="mb-article-search__meta">
            <router-link
              v-if="item.mid"
              :to="{ name: 'minibiliUserSpace', params: { userId: item.mid } }"
              class="mb-article-search__face-wrap"
            >
              <img
                class="mb-article-search__face"
                :src="item.face || defaultFace"
                alt=""
              />
            </router-link>
            <span class="mb-article-search__cate">{{ item.category_name || "专栏" }}</span>
            <span class="mb-article-search__stat" title="阅读">
              <i class="mb-article-search__ico mb-article-search__ico--view" />
              {{ userCount(item.view) }}
            </span>
            <span class="mb-article-search__stat" title="喜欢">
              <i class="mb-article-search__ico mb-article-search__ico--like" />
              {{ userCount(item.like) }}
            </span>
            <span class="mb-article-search__stat" title="评论">
              <i class="mb-article-search__ico mb-article-search__ico--reply" />
              {{ userCount(item.reply) }}
            </span>
          </div>
        </div>
        <router-link
          :to="{ name: 'minibiliArticleRead', params: { id: item.id } }"
          class="mb-article-search__cover"
        >
          <img v-lazy="item.cover_url || defaultCover" alt="" />
        </router-link>
      </li>
    </ul>
    <mb-search-empty v-else mode="empty-article" />
  </div>
</template>

<script>
import { count2 } from "../../utils/utils";
import MbSearchEmpty from "./MbSearchEmpty.vue";

const defaultFace =
  "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='40' height='40'%3E%3Ccircle fill='%23e3e5e7' cx='20' cy='20' r='20'/%3E%3C/svg%3E";
const defaultCover =
  "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='200' height='120'%3E%3Crect fill='%23e3e5e7' width='200' height='120'/%3E%3C/svg%3E";

export default {
  components: { MbSearchEmpty },
  props: {
    items: {
      type: Array,
      default: () => []
    },
    numResults: {
      type: Number,
      default: 0
    },
    sort: {
      type: String,
      default: "default"
    }
  },
  emits: ["sort-change"],
  data() {
    return {
      defaultFace,
      defaultCover,
      sortOptions: [
        { id: "default", label: "默认排序" },
        { id: "pubdate", label: "最新发布" },
        { id: "click", label: "最多阅读" },
        { id: "like", label: "最多喜欢" },
        { id: "reply", label: "最多评论" }
      ]
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
  methods: {
    userCount(num) {
      return count2(num);
    },
    plainDesc(item) {
      const raw = item.desc || "";
      return raw.replace(/<[^>]+>/g, "");
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../../style/mixin";

.mb-article-search {
  padding-bottom: 24px;
}
.mb-article-search__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 0 12px;
  border-bottom: 1px solid #e5e9ef;
}
.mb-article-search__count {
  @include sc(12px, #6d757a);
}
.mb-article-search__sort {
  display: flex;
  gap: 8px;
  list-style: none;
  margin: 0;
  padding: 0;
}
.mb-article-search__sort-item a {
  display: inline-block;
  padding: 4px 10px;
  @include sc(12px, #18191c);
  @include borderRadius(4px);
  text-decoration: none;
}
.mb-article-search__sort-item--on a,
.mb-article-search__sort-item a:hover {
  background: $blue;
  color: $white;
}
.mb-article-search__list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.mb-article-search__row {
  display: flex;
  gap: 20px;
  padding: 20px 0;
  border-bottom: 1px solid #e5e9ef;
}
.mb-article-search__main {
  flex: 1;
  min-width: 0;
}
.mb-article-search__title {
  display: block;
  margin-bottom: 10px;
  text-decoration: none;
  :deep(.keyword),
  :deep(em.keyword) {
    color: $pink;
    font-style: normal;
  }
  span {
    @include sc(16px, #18191c);
    font-weight: 700;
    line-height: 24px;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  &:hover span {
    color: $blue;
  }
}
.mb-article-search__desc {
  margin: 0 0 12px;
  @include sc(12px, #9499a0);
  line-height: 18px;
  max-height: 36px;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}
.mb-article-search__meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  @include sc(12px, #9499a0);
}
.mb-article-search__face-wrap {
  display: inline-flex;
}
.mb-article-search__face {
  @include wh(20px, 20px);
  @include borderRadius(50%);
  object-fit: cover;
}
.mb-article-search__cate {
  color: #9499a0;
}
.mb-article-search__stat {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.mb-article-search__ico {
  display: inline-block;
  @include wh(14px, 14px);
  background: url(../../assets/search/sprite.png) no-repeat;
  &--view {
    background-position: -485px -543px;
  }
  &--like {
    background-position: -442px -206px;
  }
  &--reply {
    background-position: -442px -124px;
  }
}
.mb-article-search__cover {
  flex-shrink: 0;
  @include wh(160px, 100px);
  @include borderRadius(4px);
  overflow: hidden;
  background: #e3e5e7;
  img {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}
.mb-article-search__empty {
  padding: 48px 0;
  text-align: center;
  @include sc(14px, #9499a0);
}
</style>
