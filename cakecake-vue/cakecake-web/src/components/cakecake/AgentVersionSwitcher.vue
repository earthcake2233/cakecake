<template>
  <div v-if="count > 1" class="msg-chat-versions">
    <button
      type="button"
      class="msg-chat-versions__btn"
      :disabled="index <= 0"
      @click="$emit('switch', index - 1)"
    >
      ‹
    </button>
    <span class="msg-chat-versions__count">{{ index + 1 }} / {{ count }}</span>
    <button
      type="button"
      class="msg-chat-versions__btn"
      :disabled="index >= count - 1"
      @click="$emit('switch', index + 1)"
    >
      ›
    </button>
  </div>
</template>

<script>
export default {
  name: "AgentVersionSwitcher",
  props: {
    index: { type: Number, required: true },
    count: { type: Number, required: true }
  },
  emits: ["switch"]
};
</script>
