<template>
  <div class="mb-page bili-wrapper">
    <div class="mb-card">
      <h1 class="mb-card__title">cakecake 注册</h1>
      <p v-if="!isMinibiliEnv" class="mb-card__warn">
        当前未连接 cakecake 后端（请在 .env.local 中设置 VITE_MINIBILI_API=true）。
      </p>
      <p class="mb-card__hint">
        注册前请确认用户名符合规范；密码 {{ MINIBILI_REGISTER_PASSWORD_HINT }}。
      </p>
      <el-form label-width="72px" @submit.prevent="onSubmit">
        <el-form-item label="用户名">
          <el-input
            v-model="username"
            autocomplete="username"
            maxlength="32"
            show-word-limit
            :placeholder="MINIBILI_USERNAME_PLACEHOLDER"
            @keyup.enter="onSubmit"
          />
          <p
            class="mb-field-hint"
            :class="{ 'mb-field-hint--error': usernameFieldError }"
          >
            {{ usernameFieldError || MINIBILI_USERNAME_RULE_HINT }}
          </p>
        </el-form-item>
        <el-form-item label="密码">
          <el-input
            v-model="password"
            type="password"
            show-password
            autocomplete="new-password"
            :placeholder="'密码（' + MINIBILI_REGISTER_PASSWORD_HINT + '）'"
            @keyup.enter="onSubmit"
          />
        </el-form-item>
        <el-form-item>
          <el-button
            type="primary"
            :loading="loading"
            :disabled="loading || !registerSubmitReady"
            native-type="submit"
            @click="onSubmit"
          >
            注册并登录
          </el-button>
          <a href="#" class="mb-card__link" @click.prevent="openLoginModal">已有账号</a>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script>
import { ElMessage } from "element-plus";
import { mbRegisterThenLogin } from "@/api/minibili";
import {
  MINIBILI_REGISTER_PASSWORD_HINT,
  MINIBILI_USERNAME_PLACEHOLDER,
  MINIBILI_USERNAME_RULE_HINT,
  validateMinibiliUsername,
  validateMinibiliRegisterPassword,
  minibiliErrorMessage
} from "@/utils/minibiliAuthRules";
import { openMinibiliLoginModal } from "@/utils/minibiliLoginModal";

export default {
  name: "MinibiliRegister",
  data() {
    return {
      username: "",
      password: "",
      loading: false,
      MINIBILI_USERNAME_RULE_HINT,
      MINIBILI_USERNAME_PLACEHOLDER,
      MINIBILI_REGISTER_PASSWORD_HINT
    };
  },
  computed: {
    isMinibiliEnv() {
      return (
        import.meta.env.VITE_MINIBILI_API === "true" ||
        import.meta.env.VITE_MINIBILI_API === "1"
      );
    },
    usernameFieldError() {
      const u = this.username.trim();
      if (!u) return "";
      return validateMinibiliUsername(u);
    },
    registerSubmitReady() {
      const u = this.username.trim();
      const p = this.password;
      if (!u || !p) return false;
      if (validateMinibiliUsername(u)) return false;
      if (validateMinibiliRegisterPassword(p)) return false;
      return true;
    }
  },
  methods: {
    openLoginModal() {
      openMinibiliLoginModal({ tab: 0 });
    },
    async onSubmit() {
      if (!this.registerSubmitReady) return;
      const u = this.username.trim();
      if (!u || !this.password) {
        ElMessage.warning("请输入用户名和密码");
        return;
      }
      const nameErr = validateMinibiliUsername(u);
      if (nameErr) {
        ElMessage.warning(nameErr);
        return;
      }
      const passErr = validateMinibiliRegisterPassword(this.password);
      if (passErr) {
        ElMessage.warning(passErr);
        return;
      }
      this.loading = true;
      try {
        await mbRegisterThenLogin(u, this.password);
        localStorage.setItem("signIn", "1");
        this.$store.dispatch("login/setSignIn", { signIn: "1" });
        await this.$store.dispatch("login/refreshMinibiliMe").catch(() => {});
        this.$router.replace("/");
      } catch (e) {
        ElMessage.error(minibiliErrorMessage(e, "注册失败"));
      } finally {
        this.loading = false;
      }
    }
  }
};
</script>

<style scoped lang="scss">
.mb-page {
  padding: 48px 16px;
  min-height: 60vh;
}
.mb-card {
  max-width: 420px;
  margin: 0 auto;
  padding: 24px;
  border: 1px solid #e3e5e7;
  border-radius: 4px;
  background: #fff;
}
.mb-card__title {
  margin: 0 0 12px;
  font-size: 20px;
}
.mb-card__hint {
  margin: 0 0 16px;
  font-size: 13px;
  color: #61666d;
  line-height: 1.5;
}
.mb-card__warn {
  margin: 0 0 12px;
  font-size: 13px;
  color: #e6a23c;
  line-height: 1.5;
}
.mb-card__link {
  margin-left: 16px;
  font-size: 14px;
  color: #00aeec;
}
.mb-field-hint {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.5;
  color: #9499a0;
}
.mb-field-hint--error {
  color: #f85a54;
}
</style>
