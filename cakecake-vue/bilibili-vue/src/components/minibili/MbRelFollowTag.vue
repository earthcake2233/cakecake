<template>
  <div
    ref="root"
    class="mb-rel-follow-tag"
    :class="{ 'is-interactive': interactive }"
    @mouseenter="onWrapEnter"
    @mouseleave="onWrapLeave"
  >
    <button
      ref="trigger"
      type="button"
      class="mb-rel-follow-tag__btn"
      :class="{ 'is-open': menuOpen }"
      :disabled="!interactive"
      @click.prevent
    >
      <span v-if="interactive" class="mb-rel-follow-tag__bars" aria-hidden="true">
        <i /><i /><i />
      </span>
      <span class="mb-rel-follow-tag__label">{{ label }}</span>
    </button>

    <Teleport to="body">
      <div
        v-if="interactive && menuOpen"
        class="mb-rel-follow-tag__menu"
        :style="menuStyle"
        role="menu"
        @mouseenter="onMenuEnter"
        @mouseleave="onMenuLeave"
      >
        <button
          type="button"
          class="mb-rel-follow-tag__menu-row"
          role="menuitem"
          @click.stop="onOpenAssignDialog"
        >
          {{ rel.setGroup }}
        </button>
        <div class="mb-rel-follow-tag__menu-divider" aria-hidden="true" />
        <button
          type="button"
          class="mb-rel-follow-tag__menu-row"
          role="menuitem"
          :disabled="unfollowBusy"
          @click.stop="onUnfollow"
        >
          {{ rel.unfollow }}
        </button>
      </div>
    </Teleport>

    <MbFollowGroupAssignDialog
      v-model="assignDialogOpen"
      :nickname="nickname"
      :groups="followGroups"
      :selected-ids="assignSelectedIds"
      :loading="assignSaving"
      @update:selected-ids="assignSelectedIds = $event"
      @confirm="onAssignConfirm"
      @cancel="assignDialogOpen = false"
      @create-group="onAssignCreateGroup"
    />
    <MbFollowGroupCreateDialog
      v-model="createDialogOpen"
      :loading="createDialogSaving"
      @submit="onCreateGroupSubmit"
    />
  </div>
</template>

<script>
import { ElMessage } from "element-plus";
import MbFollowGroupAssignDialog from "@/components/minibili/MbFollowGroupAssignDialog.vue";
import MbFollowGroupCreateDialog from "@/components/minibili/MbFollowGroupCreateDialog.vue";
import {
  mbAddFollowGroupMember,
  mbCreateFollowGroup,
  mbListFolloweeGroupIds,
  mbRemoveFollowGroupMember,
  mbToggleUserFollow
} from "@/api/minibili";
import { personalSpaceZhCN } from "@/i18n/personalSpace.zh-CN";
import { showMbDarkToast } from "@/utils/mbToast";

const MENU_W = 112;
const GAP = 8;

export default {
  name: "MbRelFollowTag",
  components: {
    MbFollowGroupAssignDialog,
    MbFollowGroupCreateDialog
  },
  props: {
    userId: { type: Number, required: true },
    nickname: { type: String, default: "" },
    label: { type: String, required: true },
    interactive: { type: Boolean, default: false },
    followGroups: { type: Array, default: () => [] }
  },
  emits: ["unfollow", "group-change", "groups-updated"],
  data() {
    return {
      rel: personalSpaceZhCN.relations,
      menuOpen: false,
      menuStyle: {},
      assignDialogOpen: false,
      assignSelectedIds: [],
      assignBaselineIds: [],
      assignSaving: false,
      createDialogOpen: false,
      createDialogSaving: false,
      unfollowBusy: false,
      _closeTimer: null
    };
  },
  mounted() {
    window.addEventListener("scroll", this.onViewportChange, true);
    window.addEventListener("resize", this.onViewportChange);
  },
  beforeUnmount() {
    window.removeEventListener("scroll", this.onViewportChange, true);
    window.removeEventListener("resize", this.onViewportChange);
    this.clearCloseTimer();
  },
  methods: {
    onViewportChange() {
      if (this.menuOpen) {
        this.updateMenuPosition();
      }
    },
    updateMenuPosition() {
      const el = this.$refs.trigger;
      if (!el || typeof el.getBoundingClientRect !== "function") {
        return;
      }
      const r = el.getBoundingClientRect();
      const menuLeft = Math.round(r.left + r.width / 2 - MENU_W / 2);
      const menuTop = Math.round(r.bottom + GAP);
      this.menuStyle = {
        position: "fixed",
        top: `${menuTop}px`,
        left: `${menuLeft}px`,
        width: `${MENU_W}px`,
        zIndex: 3000
      };
    },
    clearCloseTimer() {
      if (this._closeTimer) {
        clearTimeout(this._closeTimer);
        this._closeTimer = null;
      }
    },
    scheduleClose() {
      if (this.assignDialogOpen || this.createDialogOpen) {
        return;
      }
      this.clearCloseTimer();
      this._closeTimer = setTimeout(() => {
        this.menuOpen = false;
        this._closeTimer = null;
      }, 150);
    },
    openMenu() {
      this.clearCloseTimer();
      this.menuOpen = true;
      this.$nextTick(() => this.updateMenuPosition());
    },
    onWrapEnter() {
      if (!this.interactive) return;
      this.openMenu();
    },
    onWrapLeave() {
      if (!this.interactive) return;
      this.scheduleClose();
    },
    onMenuEnter() {
      this.clearCloseTimer();
      this.menuOpen = true;
    },
    onMenuLeave() {
      this.scheduleClose();
    },
    async loadAssignSelection() {
      if (!this.userId) {
        this.assignSelectedIds = [];
        this.assignBaselineIds = [];
        return;
      }
      try {
        const res = await mbListFolloweeGroupIds(this.userId);
        const ids = Array.isArray(res.group_ids) ? res.group_ids : [];
        const normalized = ids.map((id) => Number(id)).filter((id) => id > 0);
        this.assignSelectedIds = [...normalized];
        this.assignBaselineIds = [...normalized];
      } catch {
        this.assignSelectedIds = [];
        this.assignBaselineIds = [];
      }
    },
    async onOpenAssignDialog() {
      this.menuOpen = false;
      this.clearCloseTimer();
      await this.loadAssignSelection();
      this.assignDialogOpen = true;
    },
    onAssignCreateGroup() {
      this.createDialogOpen = true;
    },
    async onCreateGroupSubmit(payload) {
      if (this.createDialogSaving) return;
      this.createDialogSaving = true;
      try {
        const row = await mbCreateFollowGroup({ name: payload.name });
        this.createDialogOpen = false;
        this.$emit("groups-updated");
        if (row && row.id && !this.assignSelectedIds.includes(row.id)) {
          this.assignSelectedIds = [...this.assignSelectedIds, row.id];
        }
        showMbDarkToast("分组创建成功");
      } catch (e) {
        const msg =
          (e && e.response && e.response.data && e.response.data.message) ||
          (e && e.message) ||
          "操作失败，请稍后重试";
        ElMessage.error(String(msg));
      } finally {
        this.createDialogSaving = false;
      }
    },
    async onAssignConfirm(draftIds) {
      if (this.assignSaving || !this.userId) return;
      const after = new Set(
        (Array.isArray(draftIds) ? draftIds : [])
          .map((id) => Number(id))
          .filter((id) => id > 0)
      );
      const before = new Set(this.assignBaselineIds);
      const toAdd = [...after].filter((id) => !before.has(id));
      const toRemove = [...before].filter((id) => !after.has(id));
      if (!toAdd.length && !toRemove.length) {
        this.assignDialogOpen = false;
        return;
      }
      this.assignSaving = true;
      try {
        for (const gid of toAdd) {
          await mbAddFollowGroupMember(gid, this.userId);
        }
        for (const gid of toRemove) {
          await mbRemoveFollowGroupMember(gid, this.userId);
        }
        this.assignBaselineIds = [...after];
        this.assignDialogOpen = false;
        showMbDarkToast(this.rel.assignSaved);
        this.$emit("groups-updated");
        for (const gid of toAdd) {
          this.$emit("group-change", {
            userId: this.userId,
            groupId: gid,
            inGroup: true
          });
        }
        for (const gid of toRemove) {
          this.$emit("group-change", {
            userId: this.userId,
            groupId: gid,
            inGroup: false
          });
        }
      } catch (e) {
        const msg =
          (e && e.response && e.response.data && e.response.data.message) ||
          (e && e.message) ||
          "操作失败，请稍后重试";
        ElMessage.error(String(msg));
      } finally {
        this.assignSaving = false;
      }
    },
    async onUnfollow() {
      if (this.unfollowBusy || !this.userId) return;
      this.unfollowBusy = true;
      try {
        const res = await mbToggleUserFollow(this.userId);
        if (res && res.followed === false) {
          showMbDarkToast(this.rel.unfollowDone);
          this.menuOpen = false;
          this.assignDialogOpen = false;
          this.$emit("unfollow", { userId: this.userId });
        }
      } catch (e) {
        const msg =
          (e && e.response && e.response.data && e.response.data.message) ||
          (e && e.message) ||
          "操作失败，请稍后重试";
        ElMessage.error(String(msg));
      } finally {
        this.unfollowBusy = false;
      }
    }
  }
};
</script>

<style scoped lang="scss">
.mb-rel-follow-tag {
  grid-column: 3;
  grid-row: 1 / span 2;
  align-self: center;
  justify-self: end;
  position: relative;
}

.mb-rel-follow-tag__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  min-width: 76px;
  height: 30px;
  margin: 0;
  padding: 0 12px 0 10px;
  border: 1px solid #e3e5e7;
  border-radius: 6px;
  background: #fff;
  color: #61666d;
  font-size: 12px;
  line-height: 1;
  font-family: inherit;
  cursor: default;
  box-sizing: border-box;
  white-space: nowrap;
  outline: none;
  appearance: none;
  -webkit-appearance: none;
  transition:
    border-color 0.15s ease,
    background 0.15s ease,
    color 0.15s ease;

  &:disabled {
    cursor: default;
  }
}

.mb-rel-follow-tag.is-interactive .mb-rel-follow-tag__btn {
  cursor: pointer;

  &:hover,
  &.is-open {
    border-color: #c9ccd0;
    background: #f6f7f8;
    color: #18191c;
  }
}

.mb-rel-follow-tag__bars {
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  width: 10px;
  flex-shrink: 0;

  i {
    display: block;
    width: 10px;
    height: 1.5px;
    border-radius: 1px;
    background: #9499a0;
  }
}

.mb-rel-follow-tag__label {
  line-height: 1.2;
}

.mb-rel-follow-tag__menu {
  padding: 6px 0;
  border-radius: 12px;
  background: #fff;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
  box-sizing: border-box;
  overflow: hidden;
}

.mb-rel-follow-tag__menu-row {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-height: 36px;
  margin: 0;
  padding: 0 16px;
  border: none;
  background: transparent;
  color: #61666d;
  font-size: 14px;
  line-height: 1.3;
  font-family: inherit;
  text-align: center;
  white-space: nowrap;
  cursor: pointer;
  box-sizing: border-box;
  appearance: none;
  -webkit-appearance: none;
  outline: none;
  transition:
    background 0.12s ease,
    color 0.12s ease;

  &:hover:not(:disabled) {
    background: #f1f2f3;
    color: #18191c;
  }

  &:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }
}

.mb-rel-follow-tag__menu-divider {
  height: 1px;
  margin: 2px 12px;
  background: #e3e5e7;
}
</style>
