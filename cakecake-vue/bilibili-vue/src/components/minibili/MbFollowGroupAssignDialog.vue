<template>
  <Teleport to="body">
    <div
      v-if="modelValue"
      class="mb-follow-group-assign-overlay"
      role="presentation"
      @click.self="onCancel"
    >
      <div
        class="mb-follow-group-assign"
        role="dialog"
        aria-modal="true"
        aria-labelledby="mb-follow-group-assign-title"
        @click.stop
      >
        <button
          type="button"
          class="mb-follow-group-assign__close"
          aria-label="关闭"
          :disabled="loading"
          @click="onCancel"
        >
          ×
        </button>
        <h2 id="mb-follow-group-assign-title" class="mb-follow-group-assign__title">
          {{ rel.setGroup }}
        </h2>
        <p class="mb-follow-group-assign__hint">
          {{ rel.assignHintPrefix }}
          <span class="mb-follow-group-assign__hint-name">{{ displayName }}</span>
          {{ rel.assignHintSuffix }}
        </p>

        <button
          type="button"
          class="mb-follow-group-assign__new"
          :disabled="loading"
          @click="$emit('create-group')"
        >
          <span class="mb-follow-group-assign__new-icon" aria-hidden="true">+</span>
          {{ rel.assignNewGroup }}
        </button>

        <ul v-if="groups.length" class="mb-follow-group-assign__list">
          <li v-for="g in groups" :key="'assign-g-' + g.id" role="none">
            <button
              type="button"
              class="mb-follow-group-assign__row"
              :class="{ 'is-on': isSelected(g.id) }"
              role="checkbox"
              :aria-checked="isSelected(g.id)"
              :disabled="loading"
              @click="toggleGroup(g.id)"
            >
              <span class="mb-follow-group-assign__check" aria-hidden="true" />
              <span class="mb-follow-group-assign__name">{{ g.name }}</span>
              <span class="mb-follow-group-assign__count">{{ g.member_count || 0 }}</span>
            </button>
          </li>
        </ul>
        <p v-else class="mb-follow-group-assign__empty">{{ rel.groupEmpty }}</p>

        <div class="mb-follow-group-assign__actions">
          <button
            type="button"
            class="mb-follow-group-assign__btn mb-follow-group-assign__btn--cancel"
            :disabled="loading"
            @click="onCancel"
          >
            {{ rel.assignCancel }}
          </button>
          <button
            type="button"
            class="mb-follow-group-assign__btn mb-follow-group-assign__btn--ok"
            :disabled="loading"
            @click="onConfirm"
          >
            {{ loading ? rel.assignSaving : rel.assignConfirm }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script>
import { personalSpaceZhCN } from "@/i18n/personalSpace.zh-CN";

export default {
  name: "MbFollowGroupAssignDialog",
  props: {
    modelValue: { type: Boolean, default: false },
    loading: { type: Boolean, default: false },
    nickname: { type: String, default: "" },
    groups: { type: Array, default: () => [] },
    selectedIds: { type: Array, default: () => [] }
  },
  emits: ["update:modelValue", "confirm", "cancel", "create-group", "update:selectedIds"],
  data() {
    return {
      rel: personalSpaceZhCN.relations,
      draftIds: []
    };
  },
  computed: {
    displayName() {
      const n = String(this.nickname || "").trim();
      return n || "Ta";
    }
  },
  watch: {
    modelValue(open) {
      if (open) {
        this.syncDraftFromProp();
      }
    },
    selectedIds: {
      handler() {
        if (this.modelValue) {
          this.syncDraftFromProp();
        }
      },
      deep: true
    }
  },
  methods: {
    syncDraftFromProp() {
      const ids = Array.isArray(this.selectedIds) ? this.selectedIds : [];
      this.draftIds = ids.map((id) => Number(id)).filter((id) => id > 0);
    },
    isSelected(groupId) {
      return this.draftIds.includes(Number(groupId));
    },
    toggleGroup(groupId) {
      const gid = Number(groupId) || 0;
      if (!gid || this.loading) return;
      if (this.isSelected(gid)) {
        this.draftIds = this.draftIds.filter((id) => id !== gid);
      } else {
        this.draftIds = [...this.draftIds, gid];
      }
      this.$emit("update:selectedIds", [...this.draftIds]);
    },
    onCancel() {
      if (this.loading) return;
      this.$emit("update:modelValue", false);
      this.$emit("cancel");
    },
    onConfirm() {
      if (this.loading) return;
      this.$emit("confirm", [...this.draftIds]);
    }
  }
};
</script>

<style lang="scss" scoped>
@import "@/styles/mb-follow-group-assign-dialog.scss";
</style>
