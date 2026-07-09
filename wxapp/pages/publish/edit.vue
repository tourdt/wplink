<template>
  <ResourcePublishForm v-if="routeReady" :mode="publishFormMode" :initial-options="routeOptions" />
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { ensureMerchantProfileReady } from '../../common/merchantProfileGuard'
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

onLoad(async (options) => {
  routeOptions.merchantId = options.merchantId || getMerchantId()
  routeOptions.resourceId = options.resourceId || ''
  routeOptions.typeCode = options.typeCode || ''
  routeOptions.direction = options.direction || ''
  routeOptions.repost = options.repost || ''
  if (!(await ensureMerchantProfileReady(routeOptions.merchantId))) return
  if (!routeOptions.resourceId) {
    uni.setNavigationBarTitle({
      title: routeOptions.direction === 'demand' ? '发布需求' : '发布供给',
    })
  }
  routeReady.value = true
})
</script>
