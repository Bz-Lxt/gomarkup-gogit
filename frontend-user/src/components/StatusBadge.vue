<template>
  <span class="text-[10px] uppercase tracking-widest px-1.5 py-0.5 border" :class="cls">{{ status }}</span>
</template>
<script setup>
import { computed } from 'vue'
const props = defineProps({ status: { type: String, default: '' } })
const cls = computed(() => {
  switch (props.status) {
    case 'added':
      return 'border-brass text-brass'
    case 'modified':
      return 'border-danger text-danger'
    case 'deleted':
      return 'border-danger text-danger'
    default:
      return 'border-cyan text-cyan'
  }
})
</script>
