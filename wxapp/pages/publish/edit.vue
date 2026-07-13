<template>
  <ResourcePublishForm v-if="routeReady" :mode="publishFormMode" :initial-options="routeOptions" />
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { requireLogin } from '../../common/auth'
import { getMerchantId } from '../../store/session'
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
  if (!requireLogin()) return
  routeOptions.merchantId = options.merchantId || getMerchantId()
  routeOptions.resourceId = options.resourceId || ''
  routeOptions.typeCode = options.typeCode || ''
  routeOptions.direction = options.direction || ''
  routeOptions.repost = options.repost || ''
  if (!routeOptions.resourceId) {
    uni.setNavigationBarTitle({
      title: routeOptions.direction === 'demand' ? '发布需求' : '发布供应',
    })
  }
  routeReady.value = true
})
</script>
