<template>
  <div class="mb-page bili-wrapper" aria-hidden="true" />
</template>

<script>
import { openMinibiliLoginModal } from "@/utils/minibiliLoginModal";

/**
 * 兼容 /minibili/login 深链：打开顶栏登录弹窗并回到首页或 redirect。
 * 独立登录页已下线，保留本文件供路由与类型检查引用。
 */
export default {
  name: "MinibiliLogin",
  mounted() {
    const q = this.$route.query.redirect;
    const redir = typeof q === "string" && q.startsWith("/") ? q : undefined;
    openMinibiliLoginModal({ tab: 0, redirect: redir });
    this.$router.replace(redir || "/");
  }
};
</script>

<style scoped>
.mb-page {
  min-height: 40vh;
}
</style>
