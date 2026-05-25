<template>
  <div class="adm-layout">
    <header class="adm-header">
      <div class="adm-header__brand">
        <img src="@/assets/cakelogo.png" alt="" />
        <span>|</span>
        <strong>cakecake 运营中心</strong>
      </div>
      <div class="adm-header__right">
        <span v-if="me" class="adm-header__user">{{ me.display_name || me.username }}</span>
        <a href="javascript:;" class="adm-header__link" @click.prevent="logout">退出</a>
        <router-link to="/" class="adm-header__link">返回主站</router-link>
      </div>
    </header>
    <div class="adm-body">
      <aside class="adm-side">
        <router-link
          :to="{ name: 'adminBanners' }"
          class="adm-side__item"
          active-class="adm-side__item--on"
        >
          首页轮播
        </router-link>
        <router-link
          :to="{ name: 'adminHotSearch' }"
          class="adm-side__item"
          active-class="adm-side__item--on"
        >
          热搜运营
        </router-link>
        <router-link
          :to="{ name: 'adminVideoReview' }"
          class="adm-side__item"
          active-class="adm-side__item--on"
        >
          视频审核
        </router-link>
        <router-link
          :to="{ name: 'adminArticleReview' }"
          class="adm-side__item"
          active-class="adm-side__item--on"
        >
          专栏审核
        </router-link>
        <router-link
          :to="{ name: 'adminDynamicManage' }"
          class="adm-side__item"
          active-class="adm-side__item--on"
        >
          动态管理
        </router-link>
        <router-link
          :to="{ name: 'adminAgent' }"
          class="adm-side__item"
          active-class="adm-side__item--on"
        >
          AI 角色
        </router-link>
      </aside>
      <main class="adm-main">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script>
import { adminMe } from "@/api/admin";
import { clearAdminTokens } from "@/utils/adminAuth";

export default {
  data() {
    return {
      me: null
    };
  },
  created() {
    this.loadMe();
  },
  methods: {
    async loadMe() {
      try {
        const body = await adminMe();
        this.me = body.data;
      } catch {
        this.$router.replace({ name: "adminLogin" });
      }
    },
    logout() {
      clearAdminTokens();
      this.$router.replace({ name: "adminLogin" });
    }
  }
};
</script>

<style lang="scss" scoped>
@import "@/style/mixin";

.adm-layout {
  min-height: 100vh;
  background: #f4f5f7;
}
.adm-header {
  height: 50px;
  background: $white;
  border-bottom: 1px solid #e3e5e7;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
}
.adm-header__brand {
  display: flex;
  align-items: center;
  gap: 10px;
  img {
    height: 24px;
  }
  strong {
    @include sc(15px, $blue);
  }
}
.adm-header__right {
  display: flex;
  align-items: center;
  gap: 16px;
  @include sc(13px, #61666d);
}
.adm-header__link {
  color: $blue;
  &:hover {
    color: #00b5e5;
  }
}
.adm-body {
  display: flex;
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px 16px 40px;
  gap: 16px;
}
.adm-side {
  width: 160px;
  flex-shrink: 0;
  background: $white;
  border-radius: 8px;
  padding: 8px 0;
  border: 1px solid #e3e5e7;
  height: fit-content;
}
.adm-side__item {
  display: block;
  padding: 12px 20px;
  @include sc(14px, #61666d);
  &:hover {
    color: $blue;
    background: #f6f7f8;
  }
}
.adm-side__item--on {
  color: $blue;
  font-weight: 600;
  background: #e3f3ff;
  border-right: 3px solid $blue;
}
.adm-main {
  flex: 1;
  min-width: 0;
}
</style>
