<template>
  <div class="adm-login">
    <div class="adm-login__card">
      <div class="adm-login__logo">
        <img src="@/assets/cakelogo.png" alt="logo" />
        <span class="adm-login__split">|</span>
        <span class="adm-login__title">运营后台</span>
      </div>
      <p class="adm-login__hint">内部账号登录，与主站用户体系分离</p>
      <el-form label-position="top" @submit.prevent="onSubmit">
        <el-form-item label="用户名">
          <el-input v-model="username" autocomplete="username" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input
            v-model="password"
            type="password"
            show-password
            autocomplete="current-password"
          />
        </el-form-item>
        <el-button
          type="primary"
          class="adm-login__btn"
          :loading="loading"
          native-type="submit"
          @click="onSubmit"
        >
          登录
        </el-button>
      </el-form>
      <p v-if="error" class="adm-login__error">{{ error }}</p>
    </div>
  </div>
</template>

<script>
import { adminLogin } from "@/api/admin";
import { setAdminTokens } from "@/utils/adminAuth";

export default {
  data() {
    return {
      username: "",
      password: "",
      loading: false,
      error: ""
    };
  },
  methods: {
    async onSubmit() {
      this.error = "";
      const u = String(this.username || "").trim();
      const p = this.password;
      if (!u || !p) {
        this.error = "请输入用户名和密码";
        return;
      }
      this.loading = true;
      try {
        const body = await adminLogin(u, p);
        setAdminTokens(body.data.access_token, body.data.refresh_token);
        this.$router.replace({ name: "adminBanners" });
      } catch (e) {
        this.error = (e && e.message) || "登录失败";
      } finally {
        this.loading = false;
      }
    }
  }
};
</script>

<style lang="scss" scoped>
@import "@/style/mixin";

.adm-login {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f4f5f7;
}
.adm-login__card {
  position: relative;
  z-index: 10;
  width: 400px;
  padding: 32px 36px 28px;
  background: $white;
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
}
.adm-login__btn {
  width: 100%;
  margin-top: 8px;
  --el-button-bg-color: #{$blue};
  --el-button-border-color: #{$blue};
  --el-button-hover-bg-color: #008ebd;
  --el-button-hover-border-color: #008ebd;
}
.adm-login__logo {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
  img {
    height: 28px;
  }
}
.adm-login__split {
  margin: 0 10px;
  color: #e3e5e7;
}
.adm-login__title {
  @include sc(18px, $blue);
  font-weight: 600;
}
.adm-login__hint {
  margin: 0 0 20px;
  @include sc(12px, #9499a0);
}
.adm-login__error {
  margin: 12px 0 0;
  @include sc(12px, #f25d8e);
  text-align: center;
}
</style>
