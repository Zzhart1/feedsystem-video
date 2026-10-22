<script setup>
import { useRouter } from 'vue-router'

const props = defineProps({
  name: { type: String, default: '' },
  size: { type: Number, default: 36 },
})

const router = useRouter()
function goProfile() {
  router.push(`/profile/${props.name}`)
}
</script>

<template>
  <button class="uavatar" :style="{ width: size + 'px', height: size + 'px', fontSize: size * 0.42 + 'px' }" @click="goProfile" :title="name">
    {{ name.slice(0, 1).toUpperCase() }}
  </button>
</template>

<style scoped>
.uavatar {
  border: none; border-radius: 50%;
  background: linear-gradient(135deg, #0ea5e9, #22c55e);
  color: #fff; font-weight: 700;
  display: inline-flex; align-items: center; justify-content: center;
  cursor: pointer; flex-shrink: 0;
}
</style>
