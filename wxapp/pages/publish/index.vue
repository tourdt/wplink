<template>
  <ResourcePublishForm :initial-options="initialOptions" mode="create" :reserve-bottom-safe-area="false" />
</template>

<script setup>
import { ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import ResourcePublishForm from '../../components/ResourcePublishForm.vue'

const PUBLISH_TYPE_KEY = 'wplink_pending_publish_type_code'
const initialOptions = ref({})

onLoad(applyPendingPublishType)
onShow(applyPendingPublishType)

function applyPendingPublishType() {
  const pendingTypeCode = uni.getStorageSync(PUBLISH_TYPE_KEY)
  if (!pendingTypeCode) return
  uni.removeStorageSync(PUBLISH_TYPE_KEY)
  initialOptions.value = { typeCode: pendingTypeCode }
}
</script>
