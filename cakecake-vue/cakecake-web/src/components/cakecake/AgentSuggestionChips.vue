<template>
  <div v-if="suggestions.length" class="msg-chat-chips">
    <button
      v-for="s in suggestions"
      :key="s"
      type="button"
      class="msg-chat-chip"
      @click="$emit('pick', s)"
    >
      {{ s }}
    </button>
  </div>
</template>

<script>
export default {
  name: "AgentSuggestionChips",
  props: {
    suggestions: { type: Array, default: () => [] }
  },
  emits: ["pick"]
};
</script>
