<template>
  <div class="mb-pc-coin">
    <div class="mb-pc-subpage-hd">
      <h2 class="mb-pc-subpage-hd__title mb-pc-subpage-hd__title--accent">
        我的硬币
      </h2>
    </div>
    <div class="mb-pc-coin__body">
      <div class="mb-pc-coin__tabs" role="tablist">
        <button
          type="button"
          class="mb-pc-coin__tab"
          :class="{ 'mb-pc-coin__tab--on': pageTab === 'home' }"
          role="tab"
          :aria-selected="pageTab === 'home'"
          @click="switchTab('home')"
        >
          首页
        </button>
        <button
          type="button"
          class="mb-pc-coin__tab"
          :class="{ 'mb-pc-coin__tab--on': pageTab === 'records' }"
          role="tab"
          :aria-selected="pageTab === 'records'"
          @click="switchTab('records')"
        >
          硬币记录
        </button>
      </div>

      <p v-if="pageTab === 'home'" class="coin-rest-p">
        硬币余额：<i class="coin-num">{{ balanceLabel }}</i>
      </p>

      <div
        class="mb-pc-coin__content"
        :class="{ 'mb-pc-coin__content--records': pageTab === 'records' }"
      >
        <div class="mb-pc-coin__ledger-col">
          <h3 class="mb-pc-coin__block-hd">
            硬币记录
            <span class="mb-pc-coin__block-sub">您最近一周的变化情况</span>
          </h3>
          <div class="mb-pc-coin__table-wrap">
            <table class="mb-pc-coin__table">
              <thead>
                <tr>
                  <th scope="col">时间</th>
                  <th scope="col">变化</th>
                  <th scope="col">原因</th>
                </tr>
              </thead>
              <tbody>
                <tr v-if="ledgerLoading">
                  <td colspan="3" class="mb-pc-coin__loading">加载中…</td>
                </tr>
                <tr v-else-if="!ledgerRows.length">
                  <td colspan="3" class="mb-pc-coin__empty">暂无硬币记录</td>
                </tr>
                <tr
                  v-for="(row, i) in ledgerRows"
                  v-else
                  :key="pageTab + '-' + i"
                >
                  <td class="mb-pc-coin__cell-time">{{ row.created_at }}</td>
                  <td class="mb-pc-coin__cell-change">
                    {{ formatChange(row.change) }}
                  </td>
                  <td class="mb-pc-coin__cell-reason">{{ row.reason }}</td>
                </tr>
              </tbody>
            </table>
            <button
              v-if="pageTab === 'home' && homeHasMore && !ledgerLoading"
              type="button"
              class="mb-pc-coin__more"
              @click="switchTab('records')"
            >
              查看更多记录 &gt;
            </button>
          </div>
        </div>

        <aside
          v-if="pageTab === 'home'"
          class="mb-pc-coin__side-col"
          aria-label="硬币说明"
        >
          <section class="mb-pc-coin__help">
            <div class="mb-pc-coin__help-hd">
              <h4 class="mb-pc-coin__help-title">硬币有什么用？</h4>
              <a
                class="mb-pc-coin__help-link"
                href="https://www.bilibili.com/blackboard/help.html#/coins"
                target="_blank"
                rel="noopener noreferrer"
                >更多硬币帮助 &gt;&gt;</a
              >
            </div>
            <p class="mb-pc-coin__help-intro">
              硬币是bilibili世界中非常重要的物品
            </p>
            <ul class="mb-pc-coin__help-list">
              <li>
                硬币可用于对优秀的视频作品进行投币支持，这是对UP主的一种肯定
              </li>
              <li>硬币还可用于修改昵称、购买标识、参与活动等</li>
            </ul>
          </section>
          <section class="mb-pc-coin__help">
            <h4 class="mb-pc-coin__help-title">如何获得硬币？</h4>
            <ul class="mb-pc-coin__help-list">
              <li>会员等级=Lv0时，将无法获得硬币登录奖励</li>
              <li>
                会员等级&gt;=Lv1且绑定手机时，每天登录后可领取登录硬币奖励
              </li>
              <li>
                UP主可通过投稿视频来获得更多硬币（观众投币的百分之十将作为UP主的硬币收入结算）
              </li>
            </ul>
          </section>
        </aside>
      </div>
    </div>
  </div>
</template>

<script>
import { mbGetMeCoinLedger } from "@/api/minibili";
import { formatCoinBalance } from "@/utils/coinBalance";

export default {
  name: "PersonalCenterCoin",
  props: {
    coinBalance: {
      type: Number,
      default: 0
    },
    isMinibiliMode: {
      type: Boolean,
      default: false
    }
  },
  data() {
    return {
      pageTab: "home",
      homeItems: [],
      homeHasMore: false,
      homeLoading: false,
      recordsItems: [],
      recordsLoading: false
    };
  },
  computed: {
    balanceLabel() {
      return formatCoinBalance(this.coinBalance);
    },
    ledgerRows() {
      return this.pageTab === "home" ? this.homeItems : this.recordsItems;
    },
    ledgerLoading() {
      return this.pageTab === "home" ? this.homeLoading : this.recordsLoading;
    }
  },
  mounted() {
    void this.loadHome();
  },
  methods: {
    formatChange(change) {
      const n = Number(change);
      if (!Number.isFinite(n)) {
        return "0";
      }
      const rounded = Math.round(n * 10) / 10;
      if (Math.abs(rounded - Math.trunc(rounded)) < 1e-6) {
        return String(Math.trunc(rounded));
      }
      return rounded.toFixed(1);
    },
    switchTab(tab) {
      this.pageTab = tab;
      if (tab === "records" && !this.recordsItems.length && !this.recordsLoading) {
        void this.loadRecords();
      }
    },
    async loadHome() {
      if (!this.isMinibiliMode) {
        return;
      }
      this.homeLoading = true;
      try {
        const res = await mbGetMeCoinLedger({ range: "week", limit: 10, offset: 0 });
        this.homeItems = res.items || [];
        this.homeHasMore = !!res.has_more;
      } catch {
        this.homeItems = [];
        this.homeHasMore = false;
      } finally {
        this.homeLoading = false;
      }
    },
    async loadRecords() {
      if (!this.isMinibiliMode) {
        return;
      }
      this.recordsLoading = true;
      try {
        const res = await mbGetMeCoinLedger({ range: "week", limit: 50, offset: 0 });
        this.recordsItems = res.items || [];
      } catch {
        this.recordsItems = [];
      } finally {
        this.recordsLoading = false;
      }
    },
    refresh() {
      void this.loadHome();
      if (this.pageTab === "records") {
        void this.loadRecords();
      }
    }
  }
};
</script>

<style lang="scss" scoped>
@import "./personal-center-coin.scss";
</style>
