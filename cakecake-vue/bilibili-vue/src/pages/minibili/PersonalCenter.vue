<template>
  <div class="mb-pc">
    <!-- 全宽蓝：中间 <img> 撑出行高，左右翼片 stretch 等高 — 蓝底底缘与图源底自动对齐（同 B 站顶栏思路） -->
    <div class="mb-pc-hero">
      <div class="mb-pc-hero__body">
        <div class="mb-pc-hero__row">
          <div class="mb-pc-hero__wing mb-pc-hero__wing--left" aria-hidden="true" />
          <div class="mb-pc-hero__center">
            <div class="mb-pc-hero__under" aria-hidden="true" />
            <img
              class="mb-pc-hero__img"
              src="@/assets/rl_top.png"
              alt=""
              width="6000"
              height="648"
              decoding="async"
            />
          </div>
          <div class="mb-pc-hero__wing mb-pc-hero__wing--right" aria-hidden="true" />
        </div>
      </div>
    </div>
    <div class="mb-pc__body">
      <div class="bili-wrapper mb-pc__wrap">
        <div class="mb-pc__shell">
          <div class="mb-pc__cols">
            <aside class="mb-pc__side" aria-label="个人中心导航">
              <div class="mb-pc__side-title">个人中心</div>
              <div class="mb-pc-security-left">
                <ul id="ser-ul" class="mb-pc-security-ul">
                  <li
                    v-for="item in primaryNav"
                    :key="item.id"
                    class="mb-pc-security-list"
                    :class="{
                      active: item.id === activeNavId,
                      'mb-pc-security-list--home': item.id === 'home'
                    }"
                    @click="selectPrimaryNav(item.id)"
                  >
                    <i :class="['mb-pc-security-icon', item.biliIconClass]" />
                    <span class="mb-pc-security-nav-name">{{ item.label }}</span>
                  </li>
                </ul>
                <ul class="mb-pc-security-ul mb-pc-security-ul--foot">
                  <li
                    v-for="item in bottomNav"
                    :key="item.id"
                    class="mb-pc-security-list mb-pc-security-list--foot"
                    role="button"
                    tabindex="0"
                    @click="onBottomNavClick(item.id)"
                    @keydown.enter.prevent="onBottomNavClick(item.id)"
                  >
                    <span class="mb-pc-security-nav-name">{{ item.label }}</span>
                    <span class="mb-pc-security-chev" aria-hidden="true">›</span>
                  </li>
                </ul>
              </div>
            </aside>
            <main class="mb-pc__main">
              <template v-if="activeNavId === 'home'">
                <header class="mb-pc__profile-hd">
                  <div class="mb-pc__profile-hd-left">
                    <div
                      class="mb-pc__profile-avatar"
                      :style="homeAvatarBgStyle"
                    />
                    <div class="mb-pc__profile-meta">
                      <div class="mb-pc__profile-name-row">
                        <div class="mb-pc__profile-name">{{ displayNickname }}</div>
                        <span class="mb-pc__badge mb-pc__badge--official"
                          >正式会员</span
                        >
                      </div>
                      <div class="mb-pc__profile-lv-row">
                        <div class="mb-pc__lv-cluster">
                          <div class="mb-pc__lv-bar-shell">
                            <div class="mb-pc__lv-bar-trackgroup">
                              <span class="mb-pc__lv-head mb-pc__lv-head--active"
                                >LV{{ lvTier.lv }}</span
                              >
                              <div class="mb-pc__lv-track">
                                <div
                                  class="mb-pc__lv-fill"
                                  :style="lvFillTransform"
                                />
                              </div>
                            </div>
                            <span class="mb-pc__lv-exp"
                              >{{ lvTier.cur }}/{{ lvTier.max }}</span
                            >
                          </div>
                        </div>
                        <div class="mb-pc__profile-actions">
                          <button
                            type="button"
                            class="mb-pc__hd-act"
                            @click="selectPrimaryNav('info')"
                          >
                            修改资料
                          </button>
                          <router-link
                            v-if="isMinibiliMode && minibiliMe && minibiliMe.user_id"
                            class="mb-pc__hd-act"
                            :to="minibiliSpaceLinkTo"
                          >
                            个人空间
                            <span class="mb-pc__hd-act-chev" aria-hidden="true"
                              >›</span
                            >
                          </router-link>
                          <button
                            v-else
                            type="button"
                            class="mb-pc__hd-act"
                          >
                            个人空间
                            <span class="mb-pc__hd-act-chev" aria-hidden="true"
                              >›</span
                            >
                          </button>
                        </div>
                      </div>
                      <div class="mb-pc__profile-wallet">
                        <span class="mb-pc__wallet-row">
                          <i
                            class="mb-pc__wallet-ico mb-pc__wallet-ico--b"
                            aria-hidden="true"
                          />
                          <span class="mb-pc__wallet-label">B币</span>
                          <span class="mb-pc__wallet-num">{{ walletBcoinDisplay }}</span>
                        </span>
                        <span class="mb-pc__wallet-gap" />
                        <span class="mb-pc__wallet-row">
                          <i
                            class="mb-pc__wallet-ico mb-pc__wallet-ico--coin"
                            aria-hidden="true"
                          />
                          <span class="mb-pc__wallet-label">硬币</span>
                          <span class="mb-pc__wallet-num">{{ walletCoinDisplay }}</span>
                        </span>
                      </div>
                    </div>
                  </div>
                </header>
                <div class="mb-pc__main-body">
                  <section class="mb-pc__section mb-pc__section--daily">
                    <div class="mb-pc-daily-task">
                      <div class="mb-pc-daily-task__hd">
                        <span
                          class="mb-pc-daily-task__icon"
                          aria-hidden="true"
                        />
                        <span class="mb-pc-daily-task__title">每日奖励</span>
                      </div>
                      <div class="mb-pc-daily-exp">
                        <div
                          v-for="(task, index) in dailyRewardItems"
                          :key="task.id"
                          class="mb-pc-daily-exp__item"
                        >
                          <div
                            class="mb-pc-daily-exp__icon"
                            :class="
                              task.done
                                ? 'mb-pc-daily-exp__icon--ok'
                                : 'mb-pc-daily-exp__icon--rest'
                            "
                          >
                            <template v-if="!task.done">
                              {{ task.exp
                              }}<span class="mb-pc-daily-exp__exp-label"
                                >EXP</span
                              >
                            </template>
                          </div>
                          <p class="mb-pc-daily-exp__label">{{ task.label }}</p>
                          <p
                            v-if="task.done"
                            class="mb-pc-daily-exp__status mb-pc-daily-exp__status--badge"
                          >
                            {{ task.exp }}经验值到手
                          </p>
                          <p
                            v-else-if="task.id === 'coin'"
                            class="mb-pc-daily-exp__status mb-pc-daily-exp__status--progress"
                          >
                            已获得{{ task.progress }}/{{ task.max }}
                          </p>
                          <p
                            v-else
                            class="mb-pc-daily-exp__status mb-pc-daily-exp__status--empty"
                            aria-hidden="true"
                          >
                            &nbsp;
                          </p>
                          <div
                            v-if="index < dailyRewardItems.length - 1"
                            class="mb-pc-daily-exp__divider"
                            aria-hidden="true"
                          />
                        </div>
                      </div>
                    </div>
                  </section>
                </div>
              </template>
              <div v-else-if="activeNavId === 'info'" class="mb-pc-myinfo">
                <div class="mb-pc-subpage-hd">
                  <h2 class="mb-pc-subpage-hd__title mb-pc-subpage-hd__title--accent">
                    我的信息
                  </h2>
                </div>
                <div class="mb-pc-myinfo__body">
                  <div class="mb-pc-myinfo__form">
                  <div class="mb-pc-myinfo__row">
                    <label class="mb-pc-myinfo__label" for="mb-pc-nick"
                      >昵称</label
                    >
                    <div class="mb-pc-myinfo__field">
                      <div class="mb-pc-myinfo__nick-line">
                        <input
                          id="mb-pc-nick"
                          v-model="formNickname"
                          type="text"
                          class="mb-pc-myinfo__input mb-pc-myinfo__input--nick"
                          maxlength="30"
                          autocomplete="nickname"
                        />
                        <span class="mb-pc-myinfo__hint"
                          >注：修改一次昵称需要消耗6个硬币</span
                        >
                      </div>
                    </div>
                  </div>
                  <div class="mb-pc-myinfo__row">
                    <span class="mb-pc-myinfo__label">用户名</span>
                    <div class="mb-pc-myinfo__field">
                      <span class="mb-pc-myinfo__readonly">{{
                        displayCakeId
                      }}</span>
                    </div>
                  </div>
                  <div class="mb-pc-myinfo__row mb-pc-myinfo__row--top">
                    <label class="mb-pc-myinfo__label" for="mb-pc-sign"
                      >我的签名</label
                    >
                    <div class="mb-pc-myinfo__field">
                      <textarea
                        id="mb-pc-sign"
                        v-model="formSign"
                        class="mb-pc-myinfo__textarea"
                        rows="5"
                        maxlength="150"
                        :placeholder="defaultSignPlaceholder"
                      />
                    </div>
                  </div>
                  <div class="mb-pc-myinfo__row">
                    <span class="mb-pc-myinfo__label">性别</span>
                    <div class="mb-pc-myinfo__field">
                      <div class="mb-pc-myinfo__seg">
                        <button
                          type="button"
                          class="mb-pc-myinfo__seg-btn"
                          :class="{
                            'mb-pc-myinfo__seg-btn--on': formGender === 'male'
                          }"
                          @click="formGender = 'male'"
                        >
                          男
                        </button>
                        <button
                          type="button"
                          class="mb-pc-myinfo__seg-btn"
                          :class="{
                            'mb-pc-myinfo__seg-btn--on': formGender === 'female'
                          }"
                          @click="formGender = 'female'"
                        >
                          女
                        </button>
                        <button
                          type="button"
                          class="mb-pc-myinfo__seg-btn"
                          :class="{
                            'mb-pc-myinfo__seg-btn--on':
                              formGender === 'secret'
                          }"
                          @click="formGender = 'secret'"
                        >
                          保密
                        </button>
                      </div>
                    </div>
                  </div>
                  <div class="mb-pc-myinfo__row">
                    <label class="mb-pc-myinfo__label" for="mb-pc-birth"
                      >出生日期</label
                    >
                    <div class="mb-pc-myinfo__field">
                      <div class="mb-pc-myinfo__date-wrap">
                        <span
                          class="mb-pc-myinfo__date-ico"
                          aria-hidden="true"
                        />
                        <input
                          id="mb-pc-birth"
                          v-model="formBirthday"
                          type="date"
                          class="mb-pc-myinfo__input mb-pc-myinfo__input--date"
                        />
                      </div>
                    </div>
                  </div>
                  <div class="mb-pc-myinfo__actions">
                    <button
                      type="button"
                      class="mb-pc-myinfo__save"
                      :disabled="profileSaving"
                      @click="onSaveMyInfo"
                    >
                      {{ profileSaving ? "保存中…" : "保存" }}
                    </button>
                  </div>
                </div>
                </div>
              </div>
              <div v-else-if="activeNavId === 'avatar'" class="mb-pc-myavatar">
                <div class="mb-pc-subpage-hd">
                  <h2 class="mb-pc-subpage-hd__title mb-pc-subpage-hd__title--accent">
                    我的头像
                  </h2>
                </div>
                <div class="mb-pc-myavatar__body">
                  <div class="mb-pc-myavatar__stage">
                    <div class="mb-pc-myavatar__orbit">
                      <div class="mb-pc-myavatar__hub">
                        <div class="mb-pc-myavatar__rings">
                          <div class="mb-pc-myavatar__ring-outer">
                            <div class="mb-pc-myavatar__ring-inner">
                              <img
                                class="mb-pc-myavatar__img"
                                :src="avatarDisplayUrl"
                                alt=""
                                width="122"
                                height="122"
                                decoding="async"
                              />
                            </div>
                          </div>
                        </div>
                      </div>
                      <button
                        type="button"
                        class="mb-pc-myavatar__change"
                        :disabled="avatarUploading"
                        @click="triggerAvatarFile"
                      >
                        <span class="mb-pc-myavatar__change-line">更换</span>
                        <span class="mb-pc-myavatar__change-line">头像</span>
                      </button>
                      <input
                        ref="avatarFileInput"
                        class="mb-pc-myavatar__file"
                        type="file"
                        accept=".jpg,.jpeg,.png,.gif,.bmp,.webp,image/jpeg,image/png,image/gif,image/webp"
                        @change="onAvatarFileChange"
                      />
                    </div>
                    <p
                      v-if="avatarInlineHint"
                      class="mb-pc-myavatar__inline-hint"
                      role="status"
                    >
                      {{ avatarInlineHint }}
                    </p>
                  </div>
                </div>
              </div>
              <div
                v-else-if="activeNavId === 'security'"
                class="mb-pc-mysecurity"
              >
                <div class="mb-pc-subpage-hd">
                  <h2 class="mb-pc-subpage-hd__title mb-pc-subpage-hd__title--accent">
                    账号安全
                  </h2>
                </div>
                <div class="mb-pc-myinfo__body">
                  <div class="mb-pc-myinfo__form">
                    <p
                      v-if="!isMinibiliMode"
                      class="mb-pc-mysecurity__mode-hint"
                    >
                      修改密码与申请注销账号需登录后使用。
                    </p>
                    <p
                      v-if="isMinibiliMode"
                      class="mb-pc-mysecurity__policy"
                    >
                      账号注销后，个人可识别数据将被删除或匿名化，但部分与公共内容关联的信息（如评论、弹幕）仍以「已注销用户」形式存在。提交注销申请后将进入
                      7 至 30 天的冷静期，期间可登录撤销操作，冷静期结束后账号才被系统永久删除，该过程不可逆。
                    </p>
                    <div
                      v-if="
                        isMinibiliMode &&
                          minibiliMe &&
                          minibiliMe.pending_deletion
                      "
                      class="mb-pc-mysecurity__cooling-banner"
                    >
                      <p class="mb-pc-mysecurity__cooling-title">
                        注销申请处理中（冷静期）
                      </p>
                      <p class="mb-pc-mysecurity__cooling-meta">
                        预计截止时间：<strong>{{ formattedDeletionEffective }}</strong>
                      </p>
                      <p class="mb-pc-mysecurity__cooling-hint">
                        在截止前你可随时点击下方按钮撤销；期满后未撤销将按上述规则永久注销。
                      </p>
                      <button
                        type="button"
                        class="mb-pc-mysecurity__revoke"
                        :disabled="revocationRevoking"
                        @click="onRevokeDeletionRequest"
                      >
                        {{
                          revocationRevoking ? "处理中…" : "撤销注销申请"
                        }}
                      </button>
                    </div>
                    <div class="mb-pc-myinfo__row">
                      <label class="mb-pc-myinfo__label" for="mb-pc-sec-old"
                        >旧密码</label
                      >
                      <div class="mb-pc-myinfo__field">
                        <div class="mb-pc-myinfo__pwd-wrap">
                          <input
                            id="mb-pc-sec-old"
                            v-model="formSecOldPwd"
                            :type="showSecOldPwd ? 'text' : 'password'"
                            class="mb-pc-myinfo__input mb-pc-myinfo__input--pwd"
                            maxlength="128"
                            autocomplete="current-password"
                            @input="securityFormError = ''"
                          />
                          <button
                            type="button"
                            class="mb-pc-myinfo__pwd-toggle"
                            :aria-pressed="showSecOldPwd"
                            :aria-label="
                              showSecOldPwd ? '隐藏旧密码' : '显示旧密码'
                            "
                            @click="showSecOldPwd = !showSecOldPwd"
                          >
                            <svg
                              v-if="!showSecOldPwd"
                              class="mb-pc-myinfo__pwd-ico"
                              viewBox="0 0 24 24"
                              width="18"
                              height="18"
                              aria-hidden="true"
                            >
                              <path
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"
                              />
                              <circle
                                cx="12"
                                cy="12"
                                r="3"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                              />
                            </svg>
                            <svg
                              v-else
                              class="mb-pc-myinfo__pwd-ico"
                              viewBox="0 0 24 24"
                              width="18"
                              height="18"
                              aria-hidden="true"
                            >
                              <path
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"
                              />
                              <line
                                x1="1"
                                y1="1"
                                x2="23"
                                y2="23"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                              />
                            </svg>
                          </button>
                        </div>
                      </div>
                    </div>
                    <div class="mb-pc-myinfo__row">
                      <label class="mb-pc-myinfo__label" for="mb-pc-sec-new"
                        >新密码</label
                      >
                      <div class="mb-pc-myinfo__field">
                        <div class="mb-pc-myinfo__pwd-wrap">
                          <input
                            id="mb-pc-sec-new"
                            v-model="formSecNewPwd"
                            :type="showSecNewPwd ? 'text' : 'password'"
                            class="mb-pc-myinfo__input mb-pc-myinfo__input--pwd"
                            maxlength="128"
                            autocomplete="new-password"
                            @input="securityFormError = ''"
                          />
                          <button
                            type="button"
                            class="mb-pc-myinfo__pwd-toggle"
                            :aria-pressed="showSecNewPwd"
                            :aria-label="
                              showSecNewPwd ? '隐藏新密码' : '显示新密码'
                            "
                            @click="showSecNewPwd = !showSecNewPwd"
                          >
                            <svg
                              v-if="!showSecNewPwd"
                              class="mb-pc-myinfo__pwd-ico"
                              viewBox="0 0 24 24"
                              width="18"
                              height="18"
                              aria-hidden="true"
                            >
                              <path
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"
                              />
                              <circle
                                cx="12"
                                cy="12"
                                r="3"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                              />
                            </svg>
                            <svg
                              v-else
                              class="mb-pc-myinfo__pwd-ico"
                              viewBox="0 0 24 24"
                              width="18"
                              height="18"
                              aria-hidden="true"
                            >
                              <path
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"
                              />
                              <line
                                x1="1"
                                y1="1"
                                x2="23"
                                y2="23"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                              />
                            </svg>
                          </button>
                        </div>
                      </div>
                    </div>
                    <div class="mb-pc-myinfo__row">
                      <label class="mb-pc-myinfo__label" for="mb-pc-sec-confirm"
                        >确认新密码</label
                      >
                      <div class="mb-pc-myinfo__field">
                        <div class="mb-pc-myinfo__pwd-wrap">
                          <input
                            id="mb-pc-sec-confirm"
                            v-model="formSecConfirmPwd"
                            :type="showSecConfirmPwd ? 'text' : 'password'"
                            class="mb-pc-myinfo__input mb-pc-myinfo__input--pwd"
                            maxlength="128"
                            autocomplete="new-password"
                            @input="securityFormError = ''"
                          />
                          <button
                            type="button"
                            class="mb-pc-myinfo__pwd-toggle"
                            :aria-pressed="showSecConfirmPwd"
                            :aria-label="
                              showSecConfirmPwd ? '隐藏确认密码' : '显示确认密码'
                            "
                            @click="showSecConfirmPwd = !showSecConfirmPwd"
                          >
                            <svg
                              v-if="!showSecConfirmPwd"
                              class="mb-pc-myinfo__pwd-ico"
                              viewBox="0 0 24 24"
                              width="18"
                              height="18"
                              aria-hidden="true"
                            >
                              <path
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"
                              />
                              <circle
                                cx="12"
                                cy="12"
                                r="3"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                              />
                            </svg>
                            <svg
                              v-else
                              class="mb-pc-myinfo__pwd-ico"
                              viewBox="0 0 24 24"
                              width="18"
                              height="18"
                              aria-hidden="true"
                            >
                              <path
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"
                              />
                              <line
                                x1="1"
                                y1="1"
                                x2="23"
                                y2="23"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                              />
                            </svg>
                          </button>
                        </div>
                      </div>
                    </div>
                    <p
                      v-if="securityFormError"
                      class="mb-pc-mysecurity__form-err"
                      role="alert"
                    >
                      {{ securityFormError }}
                    </p>
                    <div class="mb-pc-myinfo__actions">
                      <button
                        type="button"
                        class="mb-pc-myinfo__save"
                        :disabled="securityPwdSaving || !isMinibiliMode"
                        @click="onSubmitSecurityPassword"
                      >
                        {{ securityPwdSaving ? "提交中…" : "修改密码" }}
                      </button>
                    </div>
                    <div
                      v-if="
                        isMinibiliMode &&
                          (!minibiliMe || !minibiliMe.pending_deletion)
                      "
                      class="mb-pc-mysecurity__danger"
                    >
                      <h3 class="mb-pc-mysecurity__subhd">申请注销账号</h3>
                      <p class="mb-pc-mysecurity__warn">
                        提交申请即进入冷静期；期满且未撤销时，系统将注销账号并删除或匿名化个人可识别数据。与公开评论、弹幕等关联的内容可能仍以「已注销用户」展示。请在下方操作前于上方填写<strong>旧密码</strong>以验证身份。
                      </p>
                      <button
                        type="button"
                        class="mb-pc-mysecurity__del"
                        :disabled="!isMinibiliMode"
                        @click="onAccountDeregister"
                      >
                        提交注销申请
                      </button>
                    </div>
                  </div>
                </div>
              </div>
              <PersonalCenterCoin
                v-else-if="activeNavId === 'coin'"
                ref="coinPage"
                :coin-balance="coinPageBalance"
                :is-minibili-mode="isMinibiliMode"
              />
              <div v-else class="mb-pc__main-body mb-pc__main-body--other">
                <p class="mb-pc__other-hint">该功能即将开放</p>
              </div>
            </main>
          </div>
        </div>
      </div>
    </div>
    <Teleport to="body">
      <div
        v-show="mbPcTipModal.visible"
        class="mb-pc-success-dim"
        :aria-hidden="!mbPcTipModal.visible"
        @click.self="closeMbPcTipModal"
      >
        <div
          class="mb-pc-success-modal"
          role="dialog"
          aria-modal="true"
          aria-labelledby="mb-pc-tip-title"
          @click.stop
        >
          <header class="mb-pc-success-modal__hd">
            <h2 id="mb-pc-tip-title" class="mb-pc-success-modal__title">
              {{ mbPcTipModal.title }}
            </h2>
            <button
              type="button"
              class="mb-pc-success-modal__close"
              aria-label="关闭"
              @click="closeMbPcTipModal"
            >
              ×
            </button>
          </header>
          <div
            class="mb-pc-success-modal__bd"
            :class="{
              'mb-pc-success-modal__bd--textOnly': !mbPcTipModal.showSuccessImage
            }"
          >
            <img
              v-if="mbPcTipModal.showSuccessImage"
              class="mb-pc-success-modal__img"
              :src="profileSaveSuccessImg"
              alt=""
              width="280"
              height="200"
              decoding="async"
            />
            <p class="mb-pc-success-modal__msg">
              {{ mbPcTipModal.message }}
            </p>
          </div>
          <footer
            v-if="mbPcTipModal.footerKind === 'ok'"
            class="mb-pc-success-modal__ft"
          >
            <button
              type="button"
              class="mb-pc-success-modal__ok"
              @click="closeMbPcTipModal"
            >
              确定
            </button>
          </footer>
          <footer
            v-else-if="mbPcTipModal.footerKind === 'deregister'"
            class="mb-pc-success-modal__ft mb-pc-success-modal__ft--split"
          >
            <button
              type="button"
              class="mb-pc-success-modal__btn-cancel"
              :disabled="accountDeleting"
              @click="closeMbPcTipModal"
            >
              取消
            </button>
            <button
              type="button"
              class="mb-pc-success-modal__ok mb-pc-success-modal__ok--danger"
              :disabled="accountDeleting"
              @click="onMbPcTipModalDeregisterConfirm"
            >
              {{ accountDeleting ? "提交中…" : "确认提交" }}
            </button>
          </footer>
        </div>
      </div>
    </Teleport>
    <MbAvatarCropDialog
      :visible="avatarCropVisible"
      :src="avatarCropSrc"
      :file-name="avatarCropFileName"
      @cancel="onAvatarCropCancel"
      @confirm="onAvatarCropConfirm"
      @error="onAvatarCropError"
    />
  </div>
</template>

<script>
import { createNamespacedHelpers } from "vuex";
import MbAvatarCropDialog from "@/components/minibili/MbAvatarCropDialog.vue";
import {
  mbGetMe,
  mbGetMeDailyRewards,
  mbPostMeAvatar,
  mbPutMePassword,
  mbPutMeProfile,
  mbRequestAccountDeletion,
  mbRevokeAccountDeletion
} from "@/api/minibili";
import defaultAvatarImg from "@/assets/akari.jpg";
import profileSaveSuccessImg from "@/assets/mb-profile-save-success.png";
import { minibiliErrorMessage, validateMinibiliRegisterPassword } from "@/utils/minibiliAuthRules";
import { resolveUserAvatarUrl } from "@/utils/imageCacheBust";
import { minibiliUserSpaceRoute } from "@/utils/minibiliRoutes";
import {
  getAccessToken,
  getMinibiliDisplayName,
  setMinibiliDisplayName,
  getUserId
} from "@/utils/authTokens";
import { formatCoinBalance, coinBalanceNumber } from "@/utils/coinBalance";
import { levelFillPct } from "@/utils/userLevel";
import PersonalCenterCoin from "./PersonalCenterCoin.vue";
const { mapState, mapActions } = createNamespacedHelpers("login");

const DEFAULT_SIGN_PLACEHOLDER = "这个家伙很懒，什么都没有写";

export default {
  name: "MinibiliPersonalCenter",
  components: { PersonalCenterCoin, MbAvatarCropDialog },
  data() {
    return {
      activeNavId: "home",
      primaryNav: [
        { id: "home", label: "首页", biliIconClass: "mb-pc-icon-home" },
        { id: "info", label: "我的信息", biliIconClass: "mb-pc-icon-3" },
        { id: "avatar", label: "我的头像", biliIconClass: "mb-pc-icon-4" },
        {
          id: "security",
          label: "账号安全",
          biliIconClass: "mb-pc-icon-7"
        },
        { id: "block", label: "黑名单管理", biliIconClass: "mb-pc-icon-8" },
        { id: "coin", label: "我的硬币", biliIconClass: "mb-pc-icon-9" },
        { id: "record", label: "我的记录", biliIconClass: "mb-pc-icon-10" }
      ],
      bottomNav: [
        { id: "space", label: "个人空间" },
        { id: "creator", label: "创作中心" }
      ],
      lvCurrent: 0,
      lvMax: 20,
      lvLevel: 1,
      /** 经验条动画：挂载后由 0 过渡到 lvFillPct */
      lvFillAnimPct: 0,
      formNickname: "",
      formSign: "",
      formGender: "secret",
      formBirthday: "2006-03-04",
      profileSaving: false,
      defaultSignPlaceholder: DEFAULT_SIGN_PLACEHOLDER,
      mbPcTipModal: {
        visible: false,
        title: "提示",
        message: "",
        showSuccessImage: true,
        footerKind: "ok"
      },
      profileSaveSuccessImg,
      avatarInlineHint: "",
      avatarUploading: false,
      avatarPreviewBlobUrl: null,
      avatarCropVisible: false,
      avatarCropSrc: "",
      avatarCropFileName: "avatar.jpg",
      _avatarCropObjectUrl: null,
      formSecOldPwd: "",
      formSecNewPwd: "",
      formSecConfirmPwd: "",
      securityPwdSaving: false,
      securityFormError: "",
      showSecOldPwd: false,
      showSecNewPwd: false,
      showSecConfirmPwd: false,
      accountDeleting: false,
      revocationRevoking: false,
      dailyRewardDone: {
        login: false,
        watch: false,
        share: false
      },
      dailyRewardCoinProgress: 0,
      dailyRewardsLoading: false
    };
  },
  computed: {
    ...mapState({
      proInfo: state => state.proInfo,
      minibiliMe: state => state.minibiliMe,
      avatarCacheBust: state => state.avatarCacheBust
    }),
    isMinibiliMode() {
      return (
        import.meta.env.VITE_MINIBILI_API === "true" ||
        import.meta.env.VITE_MINIBILI_API === "1"
      );
    },
    minibiliSpaceLinkTo() {
      if (!this.isMinibiliMode || !this.minibiliMe || this.minibiliMe.user_id == null) {
        return { path: "/" };
      }
      const r = minibiliUserSpaceRoute(this.minibiliMe.user_id);
      return r || { path: "/" };
    },
    profileStorageKey() {
      if (this.isMinibiliMode) {
        const id = getUserId();
        if (id != null && !Number.isNaN(id)) {
          return `mb_pc_myinfo_${id}`;
        }
      }
      const p = this.proInfo;
      if (p && typeof p === "object" && !Array.isArray(p) && p.mid != null) {
        return `mb_pc_myinfo_mid_${p.mid}`;
      }
      return "mb_pc_myinfo_local";
    },
    displayCakeId() {
      if (this.isMinibiliMode && this.minibiliMe && this.minibiliMe.cake_id) {
        return this.minibiliMe.cake_id;
      }
      const p = this.proInfo;
      if (p && typeof p === "object" && !Array.isArray(p) && p.mid != null) {
        const s = String(Math.abs(Math.trunc(Number(p.mid))));
        return `cake_${s.padStart(11, "0")}`;
      }
      return "cake_00000000000";
    },
    proInfoObj() {
      const p = this.proInfo;
      return p && typeof p === "object" && !Array.isArray(p) ? p : null;
    },
    displayNickname() {
      if (this.isMinibiliMode && this.minibiliMe) {
        const m = this.minibiliMe;
        const n = (m.nickname && String(m.nickname).trim()) || "";
        if (n) {
          return n;
        }
        if (m.username) {
          return String(m.username);
        }
        return getMinibiliDisplayName() || "用户";
      }
      const p = this.proInfoObj;
      if (p && p.uname) {
        return p.uname;
      }
      return "用户名";
    },
    lvTier() {
      const li = this.proInfoObj && this.proInfoObj.level_info;
      if (li && typeof li.current_level === "number") {
        return {
          lv: li.current_level,
          min:
            typeof li.current_min === "number" ? li.current_min : 0,
          cur:
            typeof li.current_exp === "number"
              ? li.current_exp
              : this.lvCurrent,
          max:
            typeof li.next_exp === "number" ? li.next_exp : this.lvMax,
          levelInfo: li
        };
      }
      return {
        lv: this.lvLevel,
        min: 0,
        cur: this.lvCurrent,
        max: this.lvMax,
        levelInfo: null
      };
    },
    lvFillPct() {
      const li = this.lvTier.levelInfo;
      if (li) {
        return levelFillPct(li);
      }
      const min = this.lvTier.min;
      const m = this.lvTier.max;
      const c = this.lvTier.cur;
      if (m <= min) {
        return 100;
      }
      return Math.min(100, Math.round(((c - min) / (m - min)) * 1000) / 10);
    },
    lvFillTransform() {
      const s = Math.max(0, Math.min(1, this.lvFillAnimPct / 100));
      return {
        transform: `scaleX(${s})`
      };
    },
    avatarDisplayUrl() {
      if (this.avatarPreviewBlobUrl) {
        return this.avatarPreviewBlobUrl;
      }
      if (this.isMinibiliMode && this.minibiliMe) {
        const u = String(this.minibiliMe.avatar_url || "").trim();
        if (u) {
          return resolveUserAvatarUrl(u, this.avatarCacheBust);
        }
      }
      return defaultAvatarImg;
    },
    formattedDeletionEffective() {
      const me = this.minibiliMe;
      const iso = me && me.deletion_effective_at;
      return this.formatDeletionZh(iso);
    },
    homeAvatarBgStyle() {
      const u = this.avatarDisplayUrl;
      if (!u) {
        return {};
      }
      return {
        backgroundImage: `url(${u})`,
        backgroundSize: "cover",
        backgroundPosition: "center center",
        backgroundRepeat: "no-repeat"
      };
    },
    walletBcoinDisplay() {
      const p = this.proInfoObj;
      const w = p && p.wallet;
      const b = w && typeof w.bcoin_balance === "number" ? w.bcoin_balance : 0;
      return Number(b).toFixed(1);
    },
    walletCoinDisplay() {
      if (this.isMinibiliMode && this.minibiliMe) {
        return formatCoinBalance(this.minibiliMe.coin_balance);
      }
      const p = this.proInfoObj;
      const m = p && typeof p.money === "number" ? p.money : 0;
      return formatCoinBalance(m);
    },
    coinPageBalance() {
      if (this.isMinibiliMode && this.minibiliMe) {
        return coinBalanceNumber(this.minibiliMe.coin_balance);
      }
      const p = this.proInfoObj;
      return p && typeof p.money === "number" ? p.money : 0;
    },
    dailyRewardItems() {
      const coinProgress = this.dailyRewardCoinProgress;
      const coinDone = coinProgress >= 50;
      return [
        {
          id: "login",
          label: "每日登录",
          exp: 5,
          done: this.dailyRewardDone.login
        },
        {
          id: "watch",
          label: "每日观看视频",
          exp: 5,
          done: this.dailyRewardDone.watch
        },
        {
          id: "coin",
          label: "每日投币",
          exp: 50,
          done: coinDone,
          progress: coinProgress,
          max: 50
        },
        {
          id: "share",
          label: "每日分享视频(客户端)",
          exp: 5,
          done: this.dailyRewardDone.share
        }
      ];
    }
  },
  watch: {
    "$route.query.tab"(v) {
      this.applyTabQuery(v);
      if (this.activeNavId === "info") {
        this.hydrateMyInfoForm();
      }
      if (this.activeNavId === "avatar") {
        void this.refreshMinibiliMe();
      }
      if (this.activeNavId === "security") {
        this.resetSecurityForm();
        this.securityFormError = "";
      }
      if (this.activeNavId === "coin") {
        void this.refreshMinibiliMe().then(() => {
          const page = this.$refs.coinPage;
          if (page && typeof page.refresh === "function") {
            page.refresh();
          }
        });
      }
    }
  },
  mounted() {
    this.applyTabQuery(this.$route.query.tab);
    void this.bootstrapProfile().finally(() => {
      this.$nextTick(() => {
        requestAnimationFrame(() => {
          void this.$el.offsetWidth;
          this.lvFillAnimPct = this.lvFillPct;
        });
      });
    });
  },
  beforeUnmount() {
    if (this.avatarPreviewBlobUrl) {
      URL.revokeObjectURL(this.avatarPreviewBlobUrl);
    }
    this.closeAvatarCrop();
  },
  methods: {
    ...mapActions(["refreshMinibiliMe"]),
    applyTabQuery(tab) {
      const t = typeof tab === "string" ? tab : "";
      const allowed = new Set([
        "home",
        "info",
        "avatar",
        "security",
        "block",
        "coin",
        "record"
      ]);
      if (allowed.has(t)) {
        this.activeNavId = t;
      }
      if (t === "home" && this.isMinibiliMode) {
        void this.loadDailyRewards();
      }
    },
    closeMbPcTipModal() {
      this.mbPcTipModal.visible = false;
    },
    openMbPcTipModal(partial) {
      const base = {
        title: "提示",
        message: "",
        showSuccessImage: false,
        footerKind: "ok"
      };
      const o = { ...base, ...partial };
      this.mbPcTipModal.title = o.title;
      this.mbPcTipModal.message = o.message;
      this.mbPcTipModal.showSuccessImage = o.showSuccessImage;
      this.mbPcTipModal.footerKind = o.footerKind;
      this.mbPcTipModal.visible = true;
    },
    formatDeletionZh(iso) {
      if (!iso || typeof iso !== "string") {
        return "";
      }
      try {
        const d = new Date(iso);
        if (Number.isNaN(d.getTime())) {
          return iso;
        }
        return d.toLocaleString("zh-CN", {
          year: "numeric",
          month: "2-digit",
          day: "2-digit",
          hour: "2-digit",
          minute: "2-digit"
        });
      } catch {
        return String(iso);
      }
    },
    async onRevokeDeletionRequest() {
      if (!this.isMinibiliMode || this.revocationRevoking) {
        return;
      }
      this.revocationRevoking = true;
      try {
        await mbRevokeAccountDeletion();
        const me = await mbGetMe();
        this.$store.commit("login/SYNC_MINIBILI_ME", me);
        this.openMbPcTipModal({
          title: "提示",
          message: "已撤销注销申请，账号将继续保留。",
          showSuccessImage: true,
          footerKind: "ok"
        });
      } catch (e) {
        this.securityFormError = minibiliErrorMessage(e, "撤销失败");
      } finally {
        this.revocationRevoking = false;
      }
    },
    onBottomNavClick(id) {
      if (id === "space") {
        const to = this.minibiliSpaceLinkTo;
        if (to && to.name === "minibiliUserSpace") {
          void this.$router.push(to);
        } else if (this.isMinibiliMode) {
          void this.refreshMinibiliMe().then(() => {
            const next = this.minibiliSpaceLinkTo;
            if (next && next.name === "minibiliUserSpace") {
              void this.$router.push(next);
            }
          });
        }
        return;
      }
      if (id === "creator") {
        void this.$router.push({ name: "upload" });
      }
    },
    selectPrimaryNav(id) {
      this.activeNavId = id;
      if (id === "home" && this.isMinibiliMode) {
        void this.loadDailyRewards();
      }
      if (id === "info") {
        this.hydrateMyInfoForm();
      }
      if (id === "avatar") {
        void this.refreshMinibiliMe();
      }
      if (id === "security") {
        this.resetSecurityForm();
        this.securityFormError = "";
      }
      if (id === "coin") {
        void this.refreshMinibiliMe().then(() => {
          const page = this.$refs.coinPage;
          if (page && typeof page.refresh === "function") {
            page.refresh();
          }
        });
      }
      this.$router
        .replace({
          path: this.$route.path,
          query: { ...this.$route.query, tab: id }
        })
        .catch(() => {});
    },
    profileDefaultsFromServer() {
      if (this.isMinibiliMode && this.minibiliMe) {
        const m = this.minibiliMe;
        const nick =
          (m.nickname && String(m.nickname).trim()) ||
          m.username ||
          getMinibiliDisplayName() ||
          "";
        return {
          nickname: nick,
          sign: (m.sign && String(m.sign)) || "",
          gender:
            m.gender === "male" || m.gender === "female" || m.gender === "secret"
              ? m.gender
              : "secret",
          birthday:
            (m.birthday && String(m.birthday).trim()) || "2006-03-04"
        };
      }
      if (this.isMinibiliMode) {
        const nick = getMinibiliDisplayName() || "";
        return {
          nickname: nick,
          sign: "",
          gender: "secret",
          birthday: "2006-03-04"
        };
      }
      const p = this.proInfo;
      if (p && typeof p === "object" && !Array.isArray(p)) {
        return {
          nickname: p.uname || "",
          sign: "",
          gender: "secret",
          birthday: "2006-03-04"
        };
      }
      return {
        nickname: "",
        sign: "",
        gender: "secret",
        birthday: "2006-03-04"
      };
    },
    readStoredProfile() {
      try {
        const raw = localStorage.getItem(this.profileStorageKey);
        if (!raw) return null;
        const o = JSON.parse(raw);
        if (!o || typeof o !== "object") return null;
        return o;
      } catch {
        return null;
      }
    },
    hydrateMyInfoForm() {
      if (this.isMinibiliMode && this.minibiliMe) {
        const d = this.profileDefaultsFromServer();
        this.formNickname = d.nickname;
        this.formSign = d.sign;
        this.formGender = d.gender;
        this.formBirthday = d.birthday;
        return;
      }
      const stored = this.readStoredProfile();
      const d = this.profileDefaultsFromServer();
      this.formNickname =
        stored && typeof stored.nickname === "string"
          ? stored.nickname
          : d.nickname;
      this.formSign =
        stored && typeof stored.sign === "string" ? stored.sign : d.sign;
      const g = stored && stored.gender;
      this.formGender =
        g === "male" || g === "female" || g === "secret" ? g : d.gender;
      this.formBirthday =
        stored && typeof stored.birthday === "string"
          ? stored.birthday
          : d.birthday;
    },
    applyDailyRewardsSnapshot(snap) {
      if (!snap || typeof snap !== "object") {
        return;
      }
      this.dailyRewardDone = {
        login: !!(snap.login && snap.login.done),
        watch: !!(snap.watch && snap.watch.done),
        share: !!(snap.share && snap.share.done)
      };
      const coin = snap.coin;
      this.dailyRewardCoinProgress =
        coin && typeof coin.progress === "number"
          ? Math.max(0, Math.min(50, coin.progress))
          : 0;
    },
    async loadDailyRewards() {
      if (!this.isMinibiliMode) {
        return;
      }
      this.dailyRewardsLoading = true;
      try {
        const snap = await mbGetMeDailyRewards();
        this.applyDailyRewardsSnapshot(snap);
        await this.refreshMinibiliMe();
      } catch {
        /* 保留上次状态，避免首页空白 */
      } finally {
        this.dailyRewardsLoading = false;
      }
    },
    async bootstrapProfile() {
      if (this.isMinibiliMode) {
        await this.refreshMinibiliMe();
        if (this.activeNavId === "home") {
          await this.loadDailyRewards();
        }
      }
      if (this.activeNavId === "info") {
        this.hydrateMyInfoForm();
      }
      if (this.activeNavId === "avatar") {
        void this.refreshMinibiliMe();
      }
      if (this.activeNavId === "security") {
        this.resetSecurityForm();
        this.securityFormError = "";
      }
    },
    triggerAvatarFile() {
      this.avatarInlineHint = "";
      const el = this.$refs.avatarFileInput;
      if (el && typeof el.click === "function") {
        el.click();
      }
    },
    closeAvatarCrop() {
      if (this._avatarCropObjectUrl) {
        URL.revokeObjectURL(this._avatarCropObjectUrl);
        this._avatarCropObjectUrl = null;
      }
      this.avatarCropVisible = false;
      this.avatarCropSrc = "";
      this.avatarCropFileName = "avatar.jpg";
    },
    onAvatarCropCancel() {
      this.closeAvatarCrop();
      this.avatarInlineHint = "";
    },
    onAvatarCropError() {
      this.avatarInlineHint = "图片处理失败，请换一张图片重试";
    },
    async onAvatarCropConfirm(file) {
      this.closeAvatarCrop();
      if (!file || this.avatarUploading) {
        return;
      }
      this.avatarUploading = true;
      let preview = null;
      try {
        preview = URL.createObjectURL(file);
        this.avatarPreviewBlobUrl = preview;
        const { avatar_url: uploadedUrl } = await mbPostMeAvatar(file);
        await this.refreshMinibiliMe();
        this.$store.commit("login/BUMP_AVATAR_BUST");
        if (uploadedUrl && this.minibiliMe) {
          this.$store.commit("login/SYNC_MINIBILI_ME", {
            ...this.minibiliMe,
            avatar_url: uploadedUrl
          });
        }
        this.avatarInlineHint = "";
        this.openMbPcTipModal({
          title: "提示",
          message: "已经成功更新你的头像",
          showSuccessImage: true,
          footerKind: "ok"
        });
      } catch (e) {
        this.openMbPcTipModal({
          title: "提示",
          message: minibiliErrorMessage(e, "上传失败"),
          showSuccessImage: false,
          footerKind: "ok"
        });
      } finally {
        if (preview) {
          URL.revokeObjectURL(preview);
        }
        this.avatarPreviewBlobUrl = null;
        this.avatarUploading = false;
      }
    },
    onAvatarFileChange(ev) {
      const input = ev.target;
      const file = input && input.files && input.files[0];
      if (input) {
        input.value = "";
      }
      if (!file) {
        return;
      }
      this.avatarInlineHint = "";
      if (!this.isMinibiliMode || !getAccessToken()) {
        this.avatarInlineHint = "请先登录后再更换头像";
        return;
      }
      const name = String(file.name || "");
      const dot = name.lastIndexOf(".");
      const ext = dot >= 0 ? name.slice(dot).toLowerCase() : "";
      const okExt = new Set([".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp"]);
      if (!okExt.has(ext)) {
        this.avatarInlineHint = "仅支持 jpg、png、gif、bmp、webp 格式";
        return;
      }
      const maxB = 5 * 1024 * 1024;
      if (file.size > maxB) {
        this.avatarInlineHint = "图片大小不能超过 5MB";
        return;
      }
      if (this.avatarUploading || this.avatarCropVisible) {
        return;
      }
      this.closeAvatarCrop();
      const url = URL.createObjectURL(file);
      this._avatarCropObjectUrl = url;
      this.avatarCropSrc = url;
      this.avatarCropFileName = name || "avatar.jpg";
      this.avatarCropVisible = true;
    },
    resetSecurityForm() {
      this.formSecOldPwd = "";
      this.formSecNewPwd = "";
      this.formSecConfirmPwd = "";
      this.securityFormError = "";
      this.showSecOldPwd = false;
      this.showSecNewPwd = false;
      this.showSecConfirmPwd = false;
    },
    async onSubmitSecurityPassword() {
      this.securityFormError = "";
      if (!this.isMinibiliMode) {
        return;
      }
      const oldPwd = String(this.formSecOldPwd || "");
      const newPwd = String(this.formSecNewPwd || "");
      const confirmPwd = String(this.formSecConfirmPwd || "");
      if (!oldPwd) {
        this.securityFormError = "请输入旧密码";
        return;
      }
      const errNew = validateMinibiliRegisterPassword(newPwd);
      if (errNew) {
        this.securityFormError = errNew;
        return;
      }
      if (newPwd !== confirmPwd) {
        this.securityFormError = "两次输入的新密码不一致";
        return;
      }
      if (oldPwd === newPwd) {
        this.securityFormError = "新密码不能与旧密码相同";
        return;
      }
      if (this.securityPwdSaving) {
        return;
      }
      this.securityPwdSaving = true;
      try {
        await mbPutMePassword(oldPwd, newPwd);
        this.resetSecurityForm();
        this.openMbPcTipModal({
          title: "提示",
          message: "已经成功更新你的登录密码",
          showSuccessImage: true,
          footerKind: "ok"
        });
      } catch (e) {
        this.securityFormError = minibiliErrorMessage(e, "修改密码失败");
      } finally {
        this.securityPwdSaving = false;
      }
    },
    onAccountDeregister() {
      if (!this.isMinibiliMode) {
        return;
      }
      this.openMbPcTipModal({
        title: "申请注销账号",
        message:
          "账号注销后，个人可识别数据将被删除或匿名化，但部分与公共内容关联的信息（如评论、弹幕）仍以「已注销用户」形式存在。提交注销申请后将进入 7 至 30 天的冷静期，期间可登录撤销操作，冷静期结束后账号才被系统永久删除，该过程不可逆。请确认已在上方填写登录密码（旧密码）。",
        showSuccessImage: false,
        footerKind: "deregister"
      });
    },
    async onMbPcTipModalDeregisterConfirm() {
      const pwd = String(this.formSecOldPwd || "").trim();
      if (!pwd) {
        this.closeMbPcTipModal();
        this.securityFormError =
          "提交注销申请需先在上方填写「旧密码」以验证身份";
        return;
      }
      if (this.accountDeleting) {
        return;
      }
      this.accountDeleting = true;
      try {
        const res = await mbRequestAccountDeletion(pwd);
        this.closeMbPcTipModal();
        let me = null;
        try {
          me = await mbGetMe();
        } catch {
          /* 忽略，仍用 res 展示截止时间 */
        }
        if (me) {
          this.$store.commit("login/SYNC_MINIBILI_ME", me);
        } else if (this.minibiliMe && typeof this.minibiliMe === "object") {
          this.$store.commit("login/SYNC_MINIBILI_ME", {
            ...this.minibiliMe,
            pending_deletion: true,
            deletion_effective_at:
              res.deletion_effective_at ||
              this.minibiliMe.deletion_effective_at ||
              null
          });
        }
        const effZh =
          this.formatDeletionZh(res.deletion_effective_at) ||
          this.formattedDeletionEffective;
        this.openMbPcTipModal({
          title: "提示",
          message: `注销申请已提交。系统已为你设置 7～30 日的冷静期，预计截止时间为：${effZh || "请稍后在「账号安全」查看"}。期间可登录并在此页撤销申请。期满后未撤销的，账号将被永久注销：个人可识别数据将被删除或匿名化；与公开评论、弹幕等仍可能以「已注销用户」展示。该过程不可逆。`,
          showSuccessImage: true,
          footerKind: "ok"
        });
      } catch (e) {
        this.closeMbPcTipModal();
        this.securityFormError = minibiliErrorMessage(e, "提交失败");
      } finally {
        this.accountDeleting = false;
      }
    },
    async onSaveMyInfo() {
      if (this.profileSaving) return;
      this.profileSaving = true;
      try {
        const payload = {
          nickname: String(this.formNickname || "").trim(),
          sign: String(this.formSign || "").trim(),
          gender: this.formGender,
          birthday: String(this.formBirthday || "").trim()
        };
        if (this.isMinibiliMode) {
          const d = this.profileDefaultsFromServer();
          const nickChanged = payload.nickname !== String(d.nickname || "").trim();
          if (nickChanged) {
            const bal =
              this.minibiliMe && typeof this.minibiliMe.coin_balance === "number"
                ? this.minibiliMe.coin_balance
                : 0;
            if (bal < 6) {
              this.openMbPcTipModal({
                title: "提示",
                message: "硬币不足，修改昵称需要消耗 6 个硬币",
                showSuccessImage: false,
                footerKind: "ok"
              });
              return;
            }
          }
          await mbPutMeProfile(payload);
          await this.refreshMinibiliMe();
          setMinibiliDisplayName(payload.nickname);
          this.hydrateMyInfoForm();
          this.openMbPcTipModal({
            title: "提示",
            message: "已经成功更新你的资料",
            showSuccessImage: true,
            footerKind: "ok"
          });
          return;
        }
        localStorage.setItem(this.profileStorageKey, JSON.stringify(payload));
        this.openMbPcTipModal({
          title: "提示",
          message: "已经成功更新你的资料",
          showSuccessImage: true,
          footerKind: "ok"
        });
      } catch (e) {
        this.openMbPcTipModal({
          title: "提示",
          message: minibiliErrorMessage(e, "保存失败"),
          showSuccessImage: false,
          footerKind: "ok"
        });
      } finally {
        this.profileSaving = false;
      }
    }
  }
};
</script>

<style lang="scss" scoped>
/* 个人中心：布局/尺寸参考 https://github.com/xunlu129/teriteri-client/blob/main/src/views/account/AccountView.vue */
$pc-blue: #00a1d6;
$pc-orange: #fba053;
/* teriteri 边框与底 */
$tt-border: #e1e2e5;
$tt-shell-bg: #fafafa;
$tt-hover: #e1e4ea;
$pc-text: #18191c;
$pc-text-sub: #9499a0;
$pc-bg-page: #fff;
$pc-hover: #f6f7f8;
/* 顶图 rl_top.png 为 6000×648；<img width/height> 便于布局占位，实际显示随容器宽度缩放 */
$mb-pc-content-max-w: 980px;
/* 与 .mb-pc__body 左右 padding 一致，用于顶栏中间列与主卡对齐 */
$mb-pc-page-pad-x: 16px;
/*
 * 顶栏总高怎么调：
 * 1) 中间图最高：$mb-pc-hero-img-max-height 如 106px（cover 从顶裁）；null 不限制。
 * 2) 两翼高度：$mb-pc-hero-wing-height 如 88px、120px 固定翼高；null 则两翼与中间列等高。
 * 3) 外边留白：.mb-pc-hero__body 的 padding-top / padding-bottom。
 * 4) 主卡与顶图同宽：$mb-pc-content-max-w（慎用）。
 */
$mb-pc-hero-img-max-height: null;
$mb-pc-hero-wing-height: 86px;
$mb-pc-side-w: 150px;
$mb-pc-shell-gap-top: 10px;

.mb-pc {
  padding: 0;
  margin: 0;
  min-height: calc(100vh - 200px);
  background: $pc-bg-page;
  color: $pc-text;
  font-family: Helvetica Neue, Helvetica, Hiragino Sans GB, Microsoft YaHei,
    Noto Sans CJK SC, WenQuanYi Micro Hei, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
}

.mb-pc-hero {
  position: relative;
  z-index: 0;
  width: 100%;
  background-color: $pc-bg-page;
  overflow: visible;
}

.mb-pc-hero__body {
  position: relative;
  z-index: 0;
  box-sizing: border-box;
  /* 无左右 padding：两翼铺满整页宽；中间列由 grid 与主卡对齐 */
  padding: 0;
  width: 100%;
  overflow-x: clip;
}

/* 三列：左右 1fr 顶到视口边，中间 min(980, 100%-左右 gutter) 与 .mb-pc__body 内主卡同宽 */
.mb-pc-hero__row {
  display: grid;
  grid-template-columns:
    1fr
    #{"min(#{$mb-pc-content-max-w}, calc(100% - #{$mb-pc-page-pad-x * 2}))"}
    1fr;
  width: 100%;
  box-sizing: border-box;
  @if $mb-pc-hero-wing-height != null {
    align-items: start;
  } @else {
    align-items: stretch;
  }
}

/* 两侧：占满侧轨；$mb-pc-hero-wing-height 有值时为固定高度 */
.mb-pc-hero__wing {
  min-width: 0;
  min-height: 0;
  background-color: $pc-blue;
  background-image: repeating-linear-gradient(
    90deg,
    rgba(255, 255, 255, 0.09) 0 1px,
    transparent 1px 72px
  );

  @if $mb-pc-hero-wing-height != null {
    height: $mb-pc-hero-wing-height;
    align-self: start;
  }
}

/* 中间：宽度由 grid 中间列决定 */
.mb-pc-hero__center {
  position: relative;
  min-width: 0;
  width: 100%;
  display: flex;
  flex-direction: column;
  background-color: $pc-blue;
}

.mb-pc-hero__under {
  position: absolute;
  z-index: 0;
  left: 0;
  right: 0;
  top: 0;
  bottom: 0;
  pointer-events: none;
  background-color: rgba(0, 0, 0, 0.05);
  background-image: repeating-linear-gradient(
    90deg,
    rgba(255, 255, 255, 0.09) 0 1px,
    transparent 1px 72px
  );
}

.mb-pc-hero__img {
  position: relative;
  z-index: 1;
  display: block;
  width: 100%;
  height: auto;
  @if $mb-pc-hero-img-max-height != null {
    max-height: $mb-pc-hero-img-max-height;
    object-fit: cover;
    object-position: top center;
  }
}

.mb-pc__body {
  position: relative;
  z-index: 2;
  margin-top: 0;
  padding: 0 $mb-pc-page-pad-x 36px;
  background: transparent;
}

.mb-pc__wrap {
  max-width: $mb-pc-content-max-w;
  width: 100%;
  margin-left: auto;
  margin-right: auto;
  box-sizing: border-box;
}

.mb-pc__shell {
  position: relative;
  z-index: 2;
  margin-top: $mb-pc-shell-gap-top;
  background: $tt-shell-bg;
  border: 1px solid $tt-border;
  border-radius: 4px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.14);
  overflow: hidden;
}

.mb-pc__cols {
  display: flex;
  align-items: stretch;
  min-height: 440px;
}

.mb-pc__side {
  flex: 0 0 #{$mb-pc-side-w};
  width: #{$mb-pc-side-w};
  box-sizing: border-box;
  background: $tt-shell-bg;
  border-right: 1px solid #ddd;
  display: flex;
  flex-direction: column;
  align-self: stretch;
}

.mb-pc__side-title {
  margin: 0;
  flex-shrink: 0;
  box-sizing: border-box;
  width: #{$mb-pc-side-w};
  height: 50px;
  line-height: 50px;
  text-align: center;
  font-size: 16px;
  font-weight: 400;
  color: #99a2aa;
  cursor: default;
  background: $tt-shell-bg;
  border-bottom: 1px solid $tt-border;
}

.mb-pc-security-left {
  flex: 1;
}

.mb-pc-security-ul {
  list-style: none;
  margin: 0;
  padding: 0;
  border-bottom: 1px solid $tt-border;

  &--foot {
    border-top: 0;
    border-bottom: 0;
    padding-top: 0;
    padding-bottom: 0;
    cursor: pointer;
  }
}

.mb-pc-security-list {
  width: #{$mb-pc-side-w};
  height: 48px;
  line-height: 48px;
  display: flex;
  align-items: center;
  box-sizing: border-box;
  font-size: 14px;
  color: $pc-text;
  cursor: default;
  user-select: none;
  transition: background 0.15s ease, color 0.15s ease;

  &:hover:not(.active) {
    background: $tt-hover;
  }

  &.active {
    background-color: $pc-blue;
    color: #fff;

    .mb-pc-security-icon {
      filter: brightness(0) invert(1);
    }
  }

  &:not(.active) .mb-pc-security-icon {
    filter: brightness(0) saturate(100%) invert(77%);
  }

  /* 「首页」：字间距使「页」与其它项第四字对齐；勿居中整行 */
  &--home .mb-pc-security-nav-name {
    letter-spacing: 2em;
    text-align: left;
  }

  &--foot {
    padding: 0 8px 0 0;
    height: 43px;
    line-height: 43px;
    font-size: 16px;
    border-bottom: 1px solid $tt-border;
    position: relative;

    .mb-pc-security-nav-name {
      margin-left: 16px;
    }

    &:hover {
      background: $tt-hover;
    }
  }
}

.mb-pc-security-icon {
  display: inline-block;
  vertical-align: middle;
  flex-shrink: 0;
  width: 18px;
  height: 36px;
  margin-left: 16px;
  margin-right: 8px;
  background-image: url("../../assets/icons_m_2.png");
  background-repeat: no-repeat;
  font-style: normal;
}

.mb-pc-icon-home {
  background-position: -87px -15px;
}

.mb-pc-icon-3 {
  background-position: -23px -80px;
}

.mb-pc-icon-4 {
  background-position: -23px -144px;
}

.mb-pc-icon-7 {
  background-position: -23px -272px;
}

.mb-pc-icon-8 {
  background-position: -727px -237px;
}

.mb-pc-icon-9 {
  background-position: -23px -335px;
}

.mb-pc-icon-10 {
  background-position: -23px -398px;
}

.mb-pc-security-nav-name {
  flex: 1;
  min-width: 0;
  vertical-align: middle;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-left: 12px;
  text-align: left;
}

.mb-pc-security-chev {
  margin-right: 8px;
  color: $pc-text-sub;
  font-size: 16px;
  line-height: 43px;
  font-weight: 300;
}

.mb-pc-security-list--foot .mb-pc-security-chev {
  line-height: 43px;
}

.mb-pc__main {
  flex: 1;
  min-width: 0;
  background: #fff;
  border-left: 1px solid #ddd;
  display: flex;
  flex-direction: column;
  align-self: stretch;
}

.mb-pc__main-body {
  flex: 1;
  padding: 16px 20px 28px;
}

.mb-pc__profile-hd {
  display: flex;
  align-items: flex-start;
  flex-shrink: 0;
  padding: 28px 28px 24px;
  border-bottom: 1px solid $tt-border;
  margin-bottom: 0;
}

.mb-pc__profile-hd-left {
  display: flex;
  align-items: center;
  gap: 20px;
  flex: 1;
  min-width: 0;
}

.mb-pc__profile-avatar {
  flex-shrink: 0;
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: linear-gradient(145deg, #e3e8ef, #cfd6df);
  border: 2px solid #fff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.mb-pc__profile-meta {
  flex: 1;
  min-width: 0;
}

.mb-pc__profile-name-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.mb-pc__profile-name {
  font-size: 18px;
  font-weight: 600;
  color: $pc-blue;
  line-height: 1.3;
  margin: 0;
}

.mb-pc__badge {
  display: inline-block;
  font-size: 12px;
  line-height: 1.4;
  border-radius: 2px;
}

.mb-pc__badge--official {
  padding: 1px 8px;
  color: #99a2aa;
  background: #fff;
  border: 1px solid #ccd0d7;
  font-weight: 400;
}

.mb-pc__profile-lv-row {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 12px 16px;
  margin-bottom: 12px;
  min-width: 0;
}

.mb-pc__lv-cluster {
  display: flex;
  align-items: center;
  flex: 1;
  min-width: 0;
}

/* 主站：LV 头 + 经验条为一组，数值在条右侧 */
.mb-pc__lv-bar-shell {
  display: flex;
  align-items: center;
  flex: 1;
  min-width: 0;
  gap: 10px;
  height: 24px;
  background: transparent;
  overflow: visible;
}

.mb-pc__lv-bar-trackgroup {
  display: flex;
  align-items: stretch;
  flex: 1;
  min-width: 0;
  max-width: 368px;
  height: 24px;
  border-radius: 4px;
  overflow: hidden;
  background: #e5e9ef;
}

.mb-pc__lv-head {
  flex: 0 0 auto;
  display: inline-block;
  width: 48px;
  height: 24px;
  box-sizing: border-box;
  margin: 0;
  padding-left: 10px;
  font-size: 12px;
  font-weight: 700;
  line-height: 24px;
  color: #fff;
  vertical-align: top;
  white-space: nowrap;
  overflow: hidden;
  background-image: url("../../assets/icons_m_2.png");
  background-repeat: no-repeat;
}

.mb-pc__lv-head--active {
  background-position: -264px -852px;
}

.mb-pc__lv-track {
  position: relative;
  flex: 1;
  min-width: 120px;
  height: 100%;
  margin-left: -2px;
  box-sizing: border-box;
  background: #e5e9ef;
  border-radius: 0 4px 4px 0;
  overflow: hidden;
}

.mb-pc__lv-fill {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 100%;
  transform: scaleX(0);
  transform-origin: left center;
  border-radius: 0 4px 4px 0;
  background: #ff905a;
  pointer-events: none;
  transition: transform 0.9s cubic-bezier(0.22, 1, 0.36, 1);
  will-change: transform;
}

.mb-pc__lv-exp {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  font-size: 12px;
  font-weight: 500;
  color: #222;
  white-space: nowrap;
  line-height: 24px;
}

.mb-pc__profile-actions {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.mb-pc__hd-act {
  display: inline-flex;
  align-items: center;
  gap: 0;
  margin: 0;
  padding: 4px 10px;
  font-size: 12px;
  line-height: 1.2;
  cursor: pointer;
  font-family: inherit;
  color: #222;
  background: #fff;
  border: 1px solid #dcdcdc;
  border-radius: 4px;
  box-sizing: border-box;

  &:hover {
    border-color: #c0c4cc;
    color: #222;
  }
}

.mb-pc__hd-act-chev {
  margin-left: 2px;
  font-weight: 300;
  color: inherit;
}

.mb-pc__profile-wallet {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  font-size: 12px;
  line-height: 1.5;
  color: $pc-text-sub;
}

.mb-pc__wallet-row {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.mb-pc__wallet-ico {
  display: inline-block;
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  background-image: url("../../assets/icons_m_2.png");
  background-repeat: no-repeat;
}

/* B 币 / 硬币：与截图 curren-b、coin-link 同源坐标 */
.mb-pc__wallet-ico--b {
  background-position: -150px -534px;
}

.mb-pc__wallet-ico--coin {
  background-position: -150px -470px;
}

.mb-pc__wallet-label {
  color: $pc-text-sub;
}

.mb-pc__wallet-num {
  color: $pc-text;
  font-weight: 500;
}

.mb-pc__wallet-gap {
  display: inline-block;
  width: 20px;
}

.mb-pc__section {
  margin-top: 18px;

  &:first-of-type {
    margin-top: 0;
  }
}

.mb-pc__sec-title {
  margin: 0 0 10px;
  font-size: 15px;
  font-weight: 400;
  color: $pc-text;
  padding-left: 10px;
  border-left: 3px solid $pc-blue;
  line-height: 1.35;
}

.mb-pc__sec-placeholder {
  min-height: 88px;
  border-radius: 5px;
  background: #f6f7f8;
  border: 1px dashed #ccd0d7;
}

/* —— 首页每日奖励（主站 home-dialy-task / home-dialy-exp） —— */
.mb-pc__section--daily {
  margin-top: 0;
}

.mb-pc-daily-task__hd {
  display: flex;
  align-items: center;
  margin: 0 0 14px;
}

.mb-pc-daily-task__icon {
  display: inline-block;
  flex-shrink: 0;
  width: 30px;
  height: 30px;
  margin-left: 10px;
  margin-right: 6px;
  background-image: url("../../assets/icons_m_2.png");
  background-repeat: no-repeat;
  background-position: -145px -18px;
}

.mb-pc-daily-task__title {
  font-size: 15px;
  font-weight: 400;
  color: $pc-text;
  line-height: 30px;
}

.mb-pc-daily-exp {
  display: flex;
  align-items: flex-start;
  width: 100%;
  box-sizing: border-box;
}

.mb-pc-daily-exp__item {
  position: relative;
  flex: 1;
  min-width: 0;
  padding: 4px 8px 8px;
  text-align: center;
  box-sizing: border-box;
}

.mb-pc-daily-exp__icon {
  width: 72px;
  height: 72px;
  margin: 0 auto;
  border-radius: 72px;
  background-image: url("../../assets/icons_m_2.png");
  background-repeat: no-repeat;
  text-align: center;
  color: #fff;
  box-sizing: border-box;
}

.mb-pc-daily-exp__icon--ok {
  background-position: -252px -508px;
}

.mb-pc-daily-exp__icon--rest {
  line-height: 72px;
  background-position: -252px -636px;
  font-size: 24px;
  font-weight: 700;
  white-space: nowrap;
}

.mb-pc-daily-exp__exp-label {
  margin-left: 1px;
  font-size: 12px;
  font-weight: 400;
  vertical-align: middle;
}

.mb-pc-daily-exp__label {
  margin: 10px 0 0;
  font-size: 12px;
  line-height: 1.5;
  color: #717171;
}

.mb-pc-daily-exp__status {
  margin: 8px 0 0;
  font-size: 12px;
  line-height: 1.5;
}

.mb-pc-daily-exp__status--badge {
  display: inline-block;
  padding: 2px 10px;
  color: #fff;
  background: #8a95a8;
  border-radius: 2px;
}

.mb-pc-daily-exp__status--progress {
  color: #99a2aa;
  background: transparent;
  padding: 0;
}

.mb-pc-daily-exp__status--empty {
  visibility: hidden;
  min-height: 22px;
  margin-top: 8px;
  padding: 2px 10px;
  line-height: 1.5;
}

.mb-pc-daily-exp__divider {
  position: absolute;
  top: 16px;
  right: 0;
  bottom: 16px;
  width: 1px;
  background: #e5e9ef;
}

/* 子页顶栏：与侧栏「个人中心」同高 50px，底部分割线对齐 */
.mb-pc-subpage-hd {
  flex-shrink: 0;
  box-sizing: border-box;
  height: 50px;
  display: flex;
  align-items: center;
  padding: 0 28px;
  border-bottom: 1px solid $tt-border;
}

.mb-pc-subpage-hd__title {
  margin: 0;
  padding-left: 10px;
  font-size: 15px;
  font-weight: 400;
  line-height: 1.35;
  color: $pc-text;
}

.mb-pc-subpage-hd__title--accent {
  color: $pc-blue;
  border-left: 3px solid $pc-blue;
}

/* —— 我的信息（主区域，参考主站个人资料页） —— */
.mb-pc-myinfo {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  padding: 0;
  background: #fff;
}

.mb-pc-myinfo__body {
  flex: 1;
  box-sizing: border-box;
  padding: 16px 28px 40px;
}

.mb-pc-myinfo__form {
  max-width: 720px;
}

.mb-pc-myinfo__row {
  display: grid;
  grid-template-columns: 104px 1fr;
  column-gap: 20px;
  row-gap: 8px;
  align-items: center;
  margin-top: 22px;

  &:first-of-type {
    margin-top: 18px;
  }

  &--top {
    align-items: start;

    .mb-pc-myinfo__label {
      padding-top: 10px;
      line-height: 1.4;
    }
  }
}

.mb-pc-myinfo__label {
  text-align: right;
  font-size: 14px;
  color: #222;
  line-height: 32px;
}

.mb-pc-myinfo__field {
  min-width: 0;
}

.mb-pc-myinfo__nick-line {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 14px;
}

.mb-pc-myinfo__input,
.mb-pc-myinfo__textarea {
  border: 1px solid #e5e9ef;
  border-radius: 4px;
  font-size: 14px;
  color: #222;
  box-sizing: border-box;
  font-family: inherit;
  background: #fff;

  &:focus {
    outline: none;
    border-color: #ccd0d7;
  }
}

.mb-pc-myinfo__input--nick {
  width: 220px;
  max-width: 100%;
  height: 34px;
  padding: 0 10px;
  line-height: 32px;
}

.mb-pc-myinfo__pwd-wrap {
  position: relative;
  display: inline-block;
  width: 100%;
  max-width: 280px;
  vertical-align: top;
}

.mb-pc-myinfo__input--pwd {
  display: block;
  width: 100%;
  height: 34px;
  padding: 0 40px 0 10px;
  line-height: 32px;
}

.mb-pc-myinfo__pwd-toggle {
  position: absolute;
  right: 2px;
  top: 50%;
  transform: translateY(-50%);
  display: flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 30px;
  margin: 0;
  padding: 0;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: #9499a0;
  cursor: pointer;
  font-family: inherit;

  &:hover {
    color: #222;
    background: rgba(0, 0, 0, 0.04);
  }
}

.mb-pc-myinfo__pwd-ico {
  display: block;
  flex-shrink: 0;
}

.mb-pc-myinfo__hint {
  font-size: 12px;
  color: #6d757a;
  line-height: 1.4;
  white-space: nowrap;
}

.mb-pc-myinfo__readonly {
  display: inline-block;
  font-size: 14px;
  color: #222;
  line-height: 32px;
}

.mb-pc-myinfo__textarea {
  display: block;
  width: 100%;
  max-width: 520px;
  min-height: 120px;
  padding: 10px 12px;
  line-height: 1.5;
  resize: vertical;
}

.mb-pc-myinfo__seg {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.mb-pc-myinfo__seg-btn {
  margin: 0;
  min-width: 52px;
  padding: 6px 20px;
  font-size: 14px;
  line-height: 1.4;
  color: #222;
  background: #fff;
  border: 1px solid #e5e9ef;
  border-radius: 4px;
  cursor: pointer;
  font-family: inherit;
  transition: background 0.15s ease, color 0.15s ease, border-color 0.15s;

  &:hover:not(.mb-pc-myinfo__seg-btn--on) {
    border-color: #ccd0d7;
  }
}

.mb-pc-myinfo__seg-btn--on {
  color: #fff;
  background: $pc-blue;
  border-color: $pc-blue;
}

.mb-pc-myinfo__date-wrap {
  position: relative;
  display: inline-block;
  max-width: 100%;
  vertical-align: top;
}

.mb-pc-myinfo__date-ico {
  position: absolute;
  left: 10px;
  top: 50%;
  width: 16px;
  height: 16px;
  margin-top: -8px;
  pointer-events: none;
  opacity: 0.55;
  background: no-repeat center / contain;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' fill='none' stroke='%236d757a' stroke-width='1.2'%3E%3Crect x='2.5' y='3.5' width='11' height='10' rx='1'/%3E%3Cpath d='M5 2v3M11 2v3M2.5 6.5h11'/%3E%3C/svg%3E");
}

.mb-pc-myinfo__input--date {
  width: 220px;
  max-width: 100%;
  height: 34px;
  padding: 0 10px 0 34px;
  line-height: 32px;
  color: #222;

  &::-webkit-calendar-picker-indicator {
    opacity: 1;
    cursor: pointer;
  }
}

.mb-pc-myinfo__actions {
  margin-top: 40px;
  display: flex;
  justify-content: center;
}

.mb-pc-myinfo__save {
  margin: 0;
  min-width: 140px;
  padding: 10px 52px;
  font-size: 15px;
  font-weight: 400;
  line-height: 1.3;
  color: #fff;
  background: #00a1d6;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-family: inherit;
  transition: background 0.15s ease;

  &:hover:not(:disabled) {
    background: #0087b8;
  }

  &:disabled {
    opacity: 0.65;
    cursor: not-allowed;
  }
}

/* —— 我的头像（同心圆 + 左侧卫星按钮） —— */
.mb-pc-myavatar {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  padding: 0;
  background: #fff;
}

.mb-pc-myavatar__body {
  flex: 1;
  box-sizing: border-box;
  padding: 0 28px 40px;
}

.mb-pc-myavatar__stage {
  position: relative;
  display: flex;
  justify-content: center;
  padding-top: 48px;
  padding-bottom: 8px;
}

.mb-pc-myavatar__orbit {
  /* 外环为灰色轨道；圆心即头像中心；按钮中心落在圆周上 */
  --mb-avatar-ring: 216px;
  --mb-avatar-orbit-r: calc(var(--mb-avatar-ring) / 2);
  --mb-avatar-sat: 58px;
  position: relative;
  width: calc(var(--mb-avatar-ring) + var(--mb-avatar-sat));
  height: calc(var(--mb-avatar-ring) + var(--mb-avatar-sat));
  flex-shrink: 0;
  margin: 0 auto;
}

.mb-pc-myavatar__hub {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  z-index: 1;
}

.mb-pc-myavatar__rings {
  position: relative;
  z-index: 1;
}

.mb-pc-myavatar__change {
  position: absolute;
  z-index: 3;
  left: 50%;
  top: 50%;
  box-sizing: border-box;
  width: var(--mb-avatar-sat);
  height: var(--mb-avatar-sat);
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1px;
  font-size: 11px;
  font-weight: 400;
  line-height: 1.15;
  letter-spacing: 0.02em;
  color: #fff;
  background: $pc-blue;
  border: 2px solid #fff;
  border-radius: 50%;
  cursor: pointer;
  font-family: inherit;
  box-shadow: 0 2px 8px rgba(0, 161, 214, 0.35);
  transform: translate(
    calc(-50% - var(--mb-avatar-orbit-r)),
    -50%
  );
  transition: background 0.15s ease, transform 0.15s ease;

  &:hover:not(:disabled) {
    background: #0087b8;
    transform: translate(
        calc(-50% - var(--mb-avatar-orbit-r)),
        -50%
      )
      scale(1.04);
  }

  &:disabled {
    opacity: 0.65;
    cursor: not-allowed;
    transform: translate(
      calc(-50% - var(--mb-avatar-orbit-r)),
      -50%
    );
  }
}

.mb-pc-myavatar__change-line {
  display: block;
  text-align: center;
}

.mb-pc-myavatar__ring-outer {
  box-sizing: border-box;
  width: var(--mb-avatar-ring, 216px);
  height: var(--mb-avatar-ring, 216px);
  border-radius: 50%;
  border: 1px solid #e1e4ea;
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
}

.mb-pc-myavatar__ring-inner {
  box-sizing: border-box;
  width: 122px;
  height: 122px;
  border-radius: 50%;
  border: 2px solid #fff;
  box-shadow: none;
  overflow: hidden;
  background: #fff;
}

.mb-pc-myavatar__img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.mb-pc-myavatar__file {
  position: absolute;
  width: 0.1px;
  height: 0.1px;
  opacity: 0;
  overflow: hidden;
  z-index: -1;
}

.mb-pc-myavatar__inline-hint {
  margin: 16px auto 0;
  max-width: 360px;
  font-size: 13px;
  line-height: 1.5;
  color: #e84a4a;
  text-align: center;
}

.mb-pc-mysecurity__mode-hint {
  margin: 0 0 16px;
  padding: 10px 14px;
  font-size: 13px;
  line-height: 1.5;
  color: #9499a0;
  background: #f6f7f8;
  border: 1px solid #e5e9ef;
  border-radius: 4px;

  strong {
    font-weight: 600;
    color: #18191c;
  }
}

.mb-pc-mysecurity__policy {
  margin: 0 0 18px;
  padding: 12px 14px;
  font-size: 13px;
  line-height: 1.65;
  color: #18191c;
  background: #f6fbfd;
  border: 1px solid #ccecf7;
  border-radius: 4px;
}

.mb-pc-mysecurity__cooling-banner {
  margin: 0 0 22px;
  padding: 14px 16px;
  background: #fffaf5;
  border: 1px solid #ffe4cc;
  border-radius: 4px;
}

.mb-pc-mysecurity__cooling-title {
  margin: 0 0 8px;
  font-size: 14px;
  font-weight: 600;
  color: #18191c;
}

.mb-pc-mysecurity__cooling-meta {
  margin: 0 0 8px;
  font-size: 13px;
  line-height: 1.5;
  color: #505259;

  strong {
    font-weight: 600;
    color: #00a1d6;
  }
}

.mb-pc-mysecurity__cooling-hint {
  margin: 0 0 14px;
  font-size: 12px;
  line-height: 1.55;
  color: #9499a0;
}

.mb-pc-mysecurity__revoke {
  padding: 6px 18px;
  font-size: 14px;
  color: #18191c;
  background: #fff;
  border: 1px solid #e5e9ef;
  border-radius: 4px;
  cursor: pointer;
  font-family: inherit;

  &:hover:not(:disabled) {
    border-color: #00a1d6;
    color: #00a1d6;
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
}

.mb-pc-mysecurity__form-err {
  margin: 0 0 12px;
  padding: 0;
  font-size: 13px;
  line-height: 1.5;
  color: #e84a4a;
}

.mb-pc-mysecurity__danger {
  margin-top: 28px;
  padding-top: 20px;
  border-top: 1px solid #e5e9ef;
}

.mb-pc-mysecurity__subhd {
  margin: 0 0 8px;
  font-size: 14px;
  font-weight: 600;
  color: #18191c;
}

.mb-pc-mysecurity__warn {
  margin: 0 0 16px;
  font-size: 13px;
  line-height: 1.6;
  color: #9499a0;

  strong {
    font-weight: 600;
    color: #18191c;
  }
}

.mb-pc-mysecurity__del {
  padding: 6px 18px;
  font-size: 14px;
  color: #e84a4a;
  background: #fff;
  border: 1px solid #f0b4b4;
  border-radius: 4px;
  cursor: pointer;

  &:hover {
    border-color: #e84a4a;
    background: #fff8f8;
  }

  &:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }
}

.mb-pc__main-body--other {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 280px;
}

.mb-pc__other-hint {
  margin: 0;
  font-size: 14px;
  color: #9499a0;
}

/* 个人中心统一提示框（资料 / 头像 / 密码成功、纯文案提示、注销确认；Teleport → body） */
.mb-pc-success-dim {
  position: fixed;
  inset: 0;
  z-index: 10050;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px 16px;
  box-sizing: border-box;
  background: rgba(0, 0, 0, 0.45);
}

.mb-pc-success-modal {
  width: 100%;
  max-width: 416px;
  box-sizing: border-box;
  background: #fff;
  border: 1px solid #e5e9ef;
  border-radius: 4px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);
  font-family: Helvetica Neue, Helvetica, Hiragino Sans GB, Microsoft YaHei,
    Noto Sans CJK SC, WenQuanYi Micro Hei, Arial, sans-serif;
}

.mb-pc-success-modal__hd {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px 12px;
  border-bottom: 1px solid #e5e9ef;
}

.mb-pc-success-modal__title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #18191c;
  line-height: 1.3;
}

.mb-pc-success-modal__close {
  margin: 0;
  padding: 0 4px;
  font-size: 20px;
  line-height: 1;
  font-weight: 300;
  color: #9499a0;
  background: transparent;
  border: none;
  cursor: pointer;
  font-family: inherit;

  &:hover {
    color: #222;
  }
}

.mb-pc-success-modal__bd {
  padding: 20px 28px 8px;
  text-align: center;
}

.mb-pc-success-modal__bd--textOnly {
  padding: 28px 24px 16px;

  .mb-pc-success-modal__msg {
    margin: 0;
  }
}

.mb-pc-success-modal__img {
  display: block;
  margin: 0 auto;
  max-width: 100%;
  width: auto;
  height: auto;
  max-height: 200px;
  object-fit: contain;
}

.mb-pc-success-modal__msg {
  margin: 18px 0 0;
  font-size: 14px;
  line-height: 1.5;
  color: #18191c;
  text-align: center;
}

.mb-pc-success-modal__ft {
  display: flex;
  justify-content: center;
  padding: 16px 24px 22px;
}

.mb-pc-success-modal__ok {
  margin: 0;
  min-width: 120px;
  padding: 8px 36px;
  font-size: 14px;
  line-height: 1.4;
  color: #fff;
  background: #00a1d6;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-family: inherit;
  transition: background 0.15s ease;

  &:hover {
    background: #0087b8;
  }
}

.mb-pc-success-modal__ft--split {
  justify-content: center;
  gap: 20px;
  flex-wrap: wrap;
}

.mb-pc-success-modal__btn-cancel {
  margin: 0;
  min-width: 88px;
  padding: 8px 20px;
  font-size: 14px;
  line-height: 1.4;
  color: #18191c;
  background: #fff;
  border: 1px solid #e5e9ef;
  border-radius: 4px;
  cursor: pointer;
  font-family: inherit;

  &:hover {
    border-color: #00a1d6;
    color: #00a1d6;
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
}

.mb-pc-success-modal__ok--danger {
  background: #e84a4a;

  &:hover:not(:disabled) {
    background: #d04040;
  }

  &:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }
}
</style>
