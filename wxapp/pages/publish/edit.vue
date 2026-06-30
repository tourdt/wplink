<template>
  <ResourcePublishForm v-if="routeReady" mode="edit" :initial-options="routeOptions" />
</template>

<script setup>
import { reactive, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import ResourcePublishForm from '../../components/ResourcePublishForm.vue'

const routeReady = ref(false)
const routeOptions = reactive({
  merchantId: '',
  resourceId: '',
  typeCode: '',
  repost: '',
})

onLoad((options) => {
  routeOptions.merchantId = options.merchantId || ''
  routeOptions.resourceId = options.resourceId || ''
  routeOptions.typeCode = options.typeCode || ''
  routeOptions.repost = options.repost || ''
  routeReady.value = true
})
</script>
