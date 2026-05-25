<!--
  @deprecated 观看历史已迁至独立页 ViewHistory.vue。
  保留本文件供类型检查与旧路径兼容，实现复用 ViewHistoryPanel。
-->
<template>
  <ViewHistoryPanel v-bind="$attrs" />
</template>

<script>
import ViewHistoryPanel from "./ViewHistoryPanel.vue";

export default {
  name: "PersonalCenterHistory",
  components: { ViewHistoryPanel },
  inheritAttrs: true
};
</script>
