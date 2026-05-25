<template>
  <div class="mb-vh-page">
    <div class="bili-wrapper mb-vh-wrap">
      <ViewHistoryPanel ref="panel" :is-minibili-mode="isMinibiliMode" />
    </div>
  </div>
</template>

<script>
import ViewHistoryPanel from "./ViewHistoryPanel.vue";

const PAGE_TITLE = "历史记录 - cakecake";

export default {
  name: "MinibiliViewHistory",
  components: { ViewHistoryPanel },
  computed: {
    isMinibiliMode() {
      return (
        import.meta.env.VITE_MINIBILI_API === "true" ||
        import.meta.env.VITE_MINIBILI_API === "1"
      );
    }
  },
  mounted() {
    this.onPageEnter();
  },
  activated() {
    this.onPageEnter();
  },
  methods: {
    onPageEnter() {
      document.title = PAGE_TITLE;
      const panel = this.$refs.panel;
      if (panel && typeof panel.refresh === "function") {
        void panel.refresh();
      }
    }
  }
};
</script>

<style lang="scss" scoped>
.mb-vh-page {
  min-height: calc(100vh - 64px);
  padding: 0 0 40px;
  box-sizing: border-box;
  background: #fff;
}

.mb-vh-wrap {
  box-sizing: border-box;
}
</style>
