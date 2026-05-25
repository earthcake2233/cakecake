<template>
  <div class="login">
    <div class="complain-mask" @click="setLoginShow()"></div>
    <div class="login-form">
      <div class="login-close" @click="setLoginShow()">
        <i class="iconfont icon-close"></i>
      </div>
      <div class="login-logo"></div>
      <div class="login-title">
        <a
          v-for="(item, index) in tab"
          :key="`login_tab_${index}`"
          :class="{ active: index == nowindex }"
          href="#"
          @click.prevent="onTabClick(index)"
          >{{ item.name }}</a
        >
      </div>
      <div class="login-user" v-if="nowindex == 0">
        <div class="login-content">
          <div class="user" :class="{ on: user !== '' }">
            <input
              v-model="user"
              type="text"
              value=""
              :placeholder="isMinibiliMode ? '用户名（支持中文）' : '你的手机号/邮箱'"
              maxlength="50"
              autocomplete="off"
              class="username"
            />
            <p v-if="!isMinibiliMode" class="error">{{ userError.errorText }}</p>
          </div>
          <div
            class="password password--with-toggle"
            :class="{ on: password !== '' }"
          >
            <input
              v-model="password"
              :type="passwordRevealed ? 'text' : 'password'"
              :placeholder="
                isMinibiliMode ? '密码' : '密码'
              "
              id="login-passwd"
              class="userpassword userpassword--padded"
              autocomplete="current-password"
            />
            <button
              type="button"
              class="pwd-toggle"
              :aria-pressed="passwordRevealed"
              :aria-label="passwordRevealed ? '隐藏密码' : '显示密码'"
              @click.prevent="passwordRevealed = !passwordRevealed"
            >
              <svg
                v-if="!passwordRevealed"
                class="pwd-toggle__svg"
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.75"
                  stroke-linecap="round"
                  d="M2 12s4-6 10-6 10 6 10 6-4 6-10 6S2 12 2 12z"
                />
                <circle cx="12" cy="12" r="2.75" fill="currentColor" />
              </svg>
              <svg
                v-else
                class="pwd-toggle__svg"
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.75"
                  stroke-linecap="round"
                  d="M3 3l18 18M10.6 10.6a2 2 0 002.8 2.8M9.9 5.3C10.6 5.1 11.3 5 12 5c6 0 10 7 10 7a18.9 18.9 0 01-3.5 4.2M6.2 6.2A18.5 18.5 0 002 12s4 7 10 7c1.1 0 2.1-.2 3.1-.5"
                />
              </svg>
            </button>
            <p v-if="!isMinibiliMode" class="error">{{ passError.errorText }}</p>
          </div>
        </div>
        <div class="login-forget">
          <a href="javascript:;" class="lff-password">忘记密码？</a>
        </div>
        <div
          class="login-btn"
          :class="{
            on: loginSubmitLooksReady,
            'is-disabled': !loginSubmitLooksReady
          }"
          @click="onLogin()"
        >
          登录
        </div>
        <div class="btn-error">{{ btnErrorText }}</div>
      </div>
      <div class="register-user" v-else>
        <p v-if="isMinibiliMode" class="register-rule-hint">
          用户名：{{ MINIBILI_USERNAME_RULE_HINT }}；密码 {{ MINIBILI_REGISTER_PASSWORD_HINT }}。
        </p>
        <div class="register-content">
          <div class="user" :class="{ on: user !== '' }">
            <input
              v-model="user"
              type="text"
              value=""
              :placeholder="
                isMinibiliMode ? MINIBILI_USERNAME_PLACEHOLDER : '昵称（例：哔哩哔哩）'
              "
              :maxlength="isMinibiliMode ? 32 : 50"
              autocomplete="off"
              class="username"
            />
            <p
              v-if="isMinibiliMode"
              class="field-hint"
              :class="{ 'field-hint--error': registerUsernameFieldError }"
            >
              {{ registerUsernameFieldError || MINIBILI_USERNAME_RULE_HINT }}
            </p>
          </div>
          <div
            class="password password--with-toggle"
            :class="{ on: password !== '' }"
          >
            <input
              v-model="password"
              :type="registerPasswordRevealed ? 'text' : 'password'"
              :placeholder="
                isMinibiliMode ? '密码（至少 8 位）' : '密码（6-16个字符组成，区分大小写）'
              "
              class="userpassword userpassword--padded"
              autocomplete="new-password"
            />
            <button
              type="button"
              class="pwd-toggle"
              :aria-pressed="registerPasswordRevealed"
              :aria-label="registerPasswordRevealed ? '隐藏密码' : '显示密码'"
              @click.prevent="
                registerPasswordRevealed = !registerPasswordRevealed
              "
            >
              <svg
                v-if="!registerPasswordRevealed"
                class="pwd-toggle__svg"
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.75"
                  stroke-linecap="round"
                  d="M2 12s4-6 10-6 10 6 10 6-4 6-10 6S2 12 2 12z"
                />
                <circle cx="12" cy="12" r="2.75" fill="currentColor" />
              </svg>
              <svg
                v-else
                class="pwd-toggle__svg"
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.75"
                  stroke-linecap="round"
                  d="M3 3l18 18M10.6 10.6a2 2 0 002.8 2.8M9.9 5.3C10.6 5.1 11.3 5 12 5c6 0 10 7 10 7a18.9 18.9 0 01-3.5 4.2M6.2 6.2A18.5 18.5 0 002 12s4 7 10 7c1.1 0 2.1-.2 3.1-.5"
                />
              </svg>
            </button>
          </div>
        </div>
        <div
          class="register-btn"
          :class="{
            on: registerSubmitLooksReady,
            'is-disabled': !registerSubmitLooksReady
          }"
          @click="onRegister()"
        >
          立即注册
        </div>
        <div class="register-login">
          <a href="javascript:;" @click.prevent="onTabClick(0)"
            >已有账号，直接登录>></a
          >
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { ElMessage } from "element-plus";
import { createNamespacedHelpers } from "vuex";
import { getUserInfo } from "../../api";
import { mbLogin, mbRegisterThenLogin } from "@/api/minibili";
import { consumeMinibiliPostLoginRedirect } from "@/utils/authTokens";
import {
  MINIBILI_REGISTER_PASSWORD_HINT,
  MINIBILI_USERNAME_PLACEHOLDER,
  MINIBILI_USERNAME_RULE_HINT,
  validateMinibiliUsername,
  validateMinibiliRegisterPassword,
  minibiliErrorMessage,
  mapMinibiliLoginFailureMessage
} from "@/utils/minibiliAuthRules";

const { mapState, mapMutations, mapActions } = createNamespacedHelpers("login");

export default {
  data() {
    return {
      tab: [
        {
          name: "登录"
        },
        {
          name: "注册"
        }
      ],
      btnErrorText: "",
      userFlag: false,
      passFlag: false,
      /** 登录 / 注册密码框明文切换（分开展示，避免切 Tab 时互相干扰） */
      passwordRevealed: false,
      registerPasswordRevealed: false,
      MINIBILI_USERNAME_RULE_HINT,
      MINIBILI_USERNAME_PLACEHOLDER,
      MINIBILI_REGISTER_PASSWORD_HINT
    };
  },
  computed: {
    ...mapState({
      nowindex: state => state.nowindex
    }),
    isMinibiliMode() {
      return (
        import.meta.env.VITE_MINIBILI_API === "true" ||
        import.meta.env.VITE_MINIBILI_API === "1"
      );
    },
    user: {
      get() {
        return this.$store.state.login.userName;
      },
      set(value) {
        this.updateUserName(value);
      }
    },
    password: {
      get() {
        return this.$store.state.login.password;
      },
      set(value) {
        this.updatePassword(value);
      }
    },
    userError() {
      let status;
      let errorText;
      if (!/^\d{6,}$/g.test(this.user)) {
        status = false;
        errorText = "用户名不足六位";
      } else {
        status = true;
        errorText = "";
      }
      if (!this.userFlag) {
        this.userFlag = true;
        errorText = "";
      }
      return {
        status,
        errorText
      };
    },
    passError() {
      let status;
      let errorText;
      if (!/^\w{1,6}$/g.test(this.password)) {
        status = false;
        errorText = "密码超过六位";
      } else {
        status = true;
        errorText = "";
      }
      if (!this.passFlag) {
        this.passFlag = true;
        errorText = "";
      }
      return {
        status,
        errorText
      };
    },
    /** Mini-Bili 登录：用户名合法 + 密码非空 */
    minibiliLoginBtnReady() {
      const u = String(this.user || "").trim();
      const p = this.password || "";
      if (!u || !p) return false;
      return !validateMinibiliUsername(u);
    },
    registerUsernameFieldError() {
      if (!this.isMinibiliMode) return "";
      const u = String(this.user || "").trim();
      if (!u) return "";
      return validateMinibiliUsername(u);
    },
    /** Mini-Bili 注册：用户名 + 密码均满足后端规则 */
    minibiliRegisterBtnReady() {
      const u = String(this.user || "").trim();
      const p = this.password || "";
      if (!u || !p) return false;
      if (validateMinibiliUsername(u)) return false;
      if (validateMinibiliRegisterPassword(p)) return false;
      return true;
    },
    legacyLoginBtnReady() {
      return this.userError.status && this.passError.status;
    },
    /** 演示站注册：与占位文案一致 6–16 位 */
    legacyRegisterBtnReady() {
      const u = String(this.user || "").trim();
      const p = this.password || "";
      if (!u || !p) return false;
      const len = p.length;
      return len >= 6 && len <= 16;
    },
    loginSubmitLooksReady() {
      return this.isMinibiliMode
        ? this.minibiliLoginBtnReady
        : this.legacyLoginBtnReady;
    },
    registerSubmitLooksReady() {
      return this.isMinibiliMode
        ? this.minibiliRegisterBtnReady
        : this.legacyRegisterBtnReady;
    }
  },
  watch: {
    nowindex() {
      this.btnErrorText = "";
      this.passwordRevealed = false;
      this.registerPasswordRevealed = false;
    }
  },
  methods: {
    ...mapMutations({
      setLoginShow: "SET_LOGIN_SHOW",
      setLoginTab: "SET_LOGIN_TAB",
      updateUserName: "SET_USERNAME",
      updatePassword: "SET_PASSWORD"
    }),
    ...mapActions(["setSignIn", "setVipInfo", "refreshMinibiliMe"]),
    onTabClick(index) {
      this.setLoginTab(index);
    },
    onLogin() {
      if (!this.loginSubmitLooksReady) {
        return;
      }
      this.btnErrorText = "";
      if (this.isMinibiliMode) {
        const u = String(this.user || "").trim();
        const p = this.password;
        if (!u || !p) {
          this.btnErrorText = "请输入账号和密码";
          return;
        }
        const nameErr = validateMinibiliUsername(u);
        if (nameErr) {
          this.btnErrorText = nameErr;
          return;
        }
        mbLogin(u, p)
          .then(() => {
            localStorage.setItem("signIn", "1");
            this.setSignIn({ signIn: "1" });
            this.$store.commit("login/CLOSE_LOGIN_MODAL");
            this.setVipInfo().catch(() => {});
            void this.refreshMinibiliMe().catch(() => {});
            const nextPath = consumeMinibiliPostLoginRedirect();
            const target = nextPath || this.$route.path;
            this.$router.replace(target).catch(() => {});
          })
          .catch(e => {
            this.btnErrorText = mapMinibiliLoginFailureMessage(e);
          });
        return;
      }
      sessionStorage.setItem("signIn", 0);
      if (!this.userError.status || !this.passError.status) {
        this.btnErrorText = "部分选项未通过";
      } else {
        getUserInfo().then(res => {
          localStorage.setItem("userName", this.user);
          localStorage.setItem("password", this.password);
          localStorage.setItem("signIn", 1);
          this.setSignIn({
            signIn: localStorage.getItem("signIn")
          });
          this.$store.commit("login/SET_USER_INFO", {
            proInfo: res.data
          });
          this.setLoginShow();
          this.setVipInfo();
        });
      }
    },
    async onRegister() {
      if (!this.registerSubmitLooksReady) {
        return;
      }
      if (this.isMinibiliMode) {
        const u = String(this.user || "").trim();
        const p = this.password;
        if (!u || !p) {
          ElMessage.warning("请输入用户名和密码");
          return;
        }
        const nameErr = validateMinibiliUsername(u);
        if (nameErr) {
          this.btnErrorText = nameErr;
          return;
        }
        const passErr = validateMinibiliRegisterPassword(p);
        if (passErr) {
          this.btnErrorText = passErr;
          return;
        }
        this.btnErrorText = "";
        try {
          await mbRegisterThenLogin(u, p);
          localStorage.setItem("signIn", "1");
          this.setSignIn({ signIn: "1" });
          this.$store.commit("login/CLOSE_LOGIN_MODAL");
          this.setVipInfo().catch(() => {});
          await this.refreshMinibiliMe().catch(() => {});
          const nextPath = consumeMinibiliPostLoginRedirect();
          const target = nextPath || this.$route.path;
          this.$router.replace(target).catch(() => {});
        } catch (e) {
          ElMessage.error(minibiliErrorMessage(e, "注册失败"));
        }
        return;
      }
      ElMessage.info(
        "演示模式下暂不支持注册，请配置 cakecake 后端 API 后使用"
      );
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style lang="scss">
@import "../../style/mixin";

.login {
  position: absolute;
  top: 0;
  @include wh(100%, 100%);
}
.complain-mask {
  background: rgba(0, 0, 0, 0.8);
  @include wh(100%, 100%);
  position: fixed;
  z-index: 999;
  display: block;
  top: 0px;
  left: 0px;
}
.login-form {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translateX(-50%) translateY(-50%);
  width: 300px;
  padding: 30px 50px 30px;
  background: $white;
  @include borderRadius(5px);
  z-index: 9999;
  overflow: hidden;
  .login-close {
    position: absolute;
    cursor: pointer;
    right: 20px;
    top: 20px;
    .icon-close {
      @include sc(24px, #909399);
      &:hover {
        color: $blue;
      }
    }
  }
  .login-logo {
    width: 220px;
    height: 72px;
    margin: 0 auto;
    background: url(../../assets/cakelogo.png) center center / contain no-repeat;
  }
  .login-title {
    overflow: hidden;
    text-align: center;
    a {
      display: inline-block;
      @include sc(15px, #969696);
      padding: 8px;
      margin: 0 16px;
      font-weight: 400;
      &.active {
        font-weight: 600;
        color: $blue;
        border-bottom: 2px solid $blue;
      }
    }
  }
  .error {
    position: absolute;
    @include sc(14px, $pink);
    bottom: 15px;
    right: 0;
  }
  .password--with-toggle .error {
    right: 40px;
  }
  .btn-error {
    margin-top: 10px;
    height: 20px;
    line-height: 20px;
    @include sc(12px, $pink);
    text-align: right;
  }
}
.login-user,
.register-user {
  float: left;
  width: 100%;
}
.login-user .login-content,
.register-user .register-content {
  margin-top: 20px;
  box-sizing: border-box;
  width: 100%;
}
.login-form .user,
.login-form .password {
  position: relative;
}
.login-form .password--with-toggle .pwd-toggle {
  position: absolute;
  right: 4px;
  bottom: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  padding: 0;
  border: none;
  background: transparent;
  color: #9499a0;
  cursor: pointer;
  border-radius: 6px;
  z-index: 1;
}
.login-form .password--with-toggle .pwd-toggle:hover {
  color: $blue;
  background: rgba(0, 161, 214, 0.08);
}
.login-form .password--with-toggle .pwd-toggle__svg {
  width: 22px;
  height: 22px;
}
.login-form .password--with-toggle .userpassword--padded {
  padding-right: 44px;
}
.login-user .login-content input,
.register-user .register-content input {
  box-sizing: border-box;
  border: none;
  border-bottom: 1px solid rgba(0, 0, 0, 0.12);
  padding: 10px 10px 0;
  margin: 10px 0 0 0;
  @include wh(100%, 50px);
  font-size: 14px;
}
.login-user .login-content .on input,
.login-user .login-content input:focus,
.register-user .register-content .on input,
.register-user .register-content input:focus {
  border-bottom: 1px solid $blue;
}
.login-user .login-forget {
  margin-top: 8px;
  min-height: 28px;
  line-height: 28px;
  text-align: right;
}
.login-user .login-forget .lff-password,
.register-user .register-login a {
  @include sc(12px, #999);
}
.register-user .register-login a,
.login-user .login-forget .lff-password:hover {
  color: $blue;
}
.register-user {
  .register-rule-hint {
    margin: 12px 0 0;
    padding: 0 2px;
    font-size: 12px;
    line-height: 1.5;
    color: #9499a0;
  }
  .field-hint {
    margin: 6px 0 0;
    padding: 0 2px;
    font-size: 12px;
    line-height: 1.45;
    color: #9499a0;
  }
  .field-hint--error {
    color: $pink;
  }
  .register-login {
    margin-top: 36px;
    a {
      margin-top: 0;
    }
  }
  .register-btn {
    margin-top: 20px;
  }
}
.login-user .login-btn,
.register-user .register-btn {
  cursor: pointer;
  margin-top: 20px;
  background: #d1d1d1;
  @include sc(14px, $white);
  font-weight: 400;
  line-height: 40px;
  text-align: center;
  @include borderRadius(20px);
}
.login-user .login-btn.on,
.register-user .register-btn.on {
  background: $blue;
}
.login-user .login-btn.is-disabled,
.register-user .register-btn.is-disabled {
  cursor: not-allowed;
  pointer-events: none;
}
</style>
