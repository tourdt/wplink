<template>
  <view :class="['resource-exposure', observerClass]">
    <slot />
  </view>
</template>

<script setup>
import { computed, getCurrentInstance, onMounted, onUnmounted, ref } from 'vue'
import { enqueueResourceExposure } from '../common/resourceExposure'

const props = defineProps({
  resourceId: {
    type: [String, Number],
    required: true,
  },
  source: {
    type: String,
    required: true,
  },
})

const instance = getCurrentInstance()
const uniqueKey = `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 9)}`
const observerClass = computed(() => `resource-exposure-${uniqueKey}`)
const visibleSince = ref(0)
let observer = null
let exposureTimer = null
let recorded = false

onMounted(() => {
  if (!uni.createIntersectionObserver) return
  observer = uni.createIntersectionObserver(instance?.proxy, {
    thresholds: [0, 0.5, 1],
  })
  observer.relativeToViewport().observe(`.${observerClass.value}`, handleIntersection)
})

onUnmounted(() => {
  clearExposureTimer()
  observer?.disconnect()
})

function handleIntersection(result = {}) {
  if (recorded) return
  if (Number(result.intersectionRatio) >= 0.5) {
    if (!visibleSince.value) visibleSince.value = Date.now()
    if (!exposureTimer) {
      exposureTimer = setTimeout(recordExposure, 800)
    }
    return
  }
  clearExposureTimer()
  visibleSince.value = 0
}

function recordExposure() {
  exposureTimer = null
  if (!visibleSince.value || recorded) return
  const visibleDurationMs = Date.now() - visibleSince.value
  if (visibleDurationMs < 800) return
  recorded = true
  enqueueResourceExposure({
    resourceId: props.resourceId,
    source: props.source,
    visibleDurationMs,
  })
  observer?.disconnect()
}

function clearExposureTimer() {
  if (!exposureTimer) return
  clearTimeout(exposureTimer)
  exposureTimer = null
}
</script>
