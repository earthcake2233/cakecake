<template>
  <div v-if="acts.length" :class="['msg-tool-activities', { 'msg-tool-activities--live': live }]">
    <div v-for="act in acts" :key="act.span_id" class="msg-tool-activity">
      <span class="msg-tool-activity__status">{{ act.status === "running" ? "↻" : "✓" }}</span>
      <span class="msg-tool-activity__name">{{ act.tool_name }}</span>
      <span v-if="act.duration_ms" class="msg-tool-activity__dur">{{ act.duration_ms }}ms</span>
    </div>
  </div>
</template>

<script>
export default {
  name: "AgentToolActivities",
  props: {
    acts: { type: Array, default: () => [] },
    live: { type: Boolean, default: false }
  }
};
</script>
