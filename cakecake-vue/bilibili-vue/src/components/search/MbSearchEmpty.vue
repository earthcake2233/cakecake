<template>
  <div class="mb-search-empty">
    <img class="mb-search-empty__img" :src="emptyImg" alt="" />
    <p class="mb-search-empty__text">{{ message }}</p>
    <p v-if="hint" class="mb-search-empty__hint">{{ hint }}</p>
  </div>
</template>

<script>
import emptyImg from "@/assets/empty.png";

const COPY = {
  empty: {
    message: "没有找到相关结果",
    hint: "换个关键词试试吧"
  },
  "empty-user": {
    message: "没有找到相关用户",
    hint: "换个关键词试试吧"
  },
  "empty-article": {
    message: "没有找到相关专栏",
    hint: "换个关键词试试吧"
  },
  unavailable: {
    message: "搜索服务暂未就绪",
    hint: "请配置 Elasticsearch（ELASTICSEARCH_URL）并确保已建立视频索引"
  }
};

export default {
  props: {
    mode: {
      type: String,
      default: "empty"
    }
  },
  data() {
    return { emptyImg };
  },
  computed: {
    message() {
      return (COPY[this.mode] || COPY.empty).message;
    },
    hint() {
      return (COPY[this.mode] || COPY.empty).hint;
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../../style/mixin";

.mb-search-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 360px;
  padding: 48px 16px 64px;
  box-sizing: border-box;
}
.mb-search-empty__img {
  width: min(380px, 88vw);
  height: auto;
  display: block;
  user-select: none;
  pointer-events: none;
}
.mb-search-empty__text {
  margin: 16px 0 0;
  @include sc(14px, #18191c);
  line-height: 22px;
}
.mb-search-empty__hint {
  margin: 8px 0 0;
  max-width: 420px;
  text-align: center;
  @include sc(13px, #9499a0);
  line-height: 20px;
}
</style>
