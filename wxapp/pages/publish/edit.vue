<template>
  <ResourcePublishForm v-if="routeReady" :mode="publishFormMode" :initial-options="routeOptions" />
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import ResourcePublishForm from '../../components/ResourcePublishForm.vue'

const routeReady = ref(false)
const routeOptions = reactive({
  merchantId: '',
  resourceId: '',
  typeCode: '',
  direction: '',
  repost: '',
})
const publishFormMode = computed(() => routeOptions.resourceId ? 'edit' : 'create')

onLoad((options) => {
  routeOptions.merchantId = options.merchantId || ''
  routeOptions.resourceId = options.resourceId || ''
  routeOptions.typeCode = options.typeCode || ''
  routeOptions.direction = options.direction || ''
  routeOptions.repost = options.repost || ''
  if (!routeOptions.resourceId) {
    uni.setNavigationBarTitle({
      title: routeOptions.direction === 'demand' ? '发布需求' : '发布资源',
    })
  }
  routeReady.value = true
})
</script>
