<template>
  <view class="webview-page">
    <web-view v-if="allowedUrl" :src="allowedUrl" />
    <view v-else class="blocked-card">
      <text class="blocked-title">链接不可访问</text>
      <text class="blocked-desc">该活动链接不在平台允许范围内。</text>
    </view>
  </view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { validateWebview } from '../../api/discovery'

const allowedUrl = ref('')

onLoad(async (options) => {
  const rawUrl = decodeURIComponent(options.url || '')
  if (!rawUrl) return
  const resp = await validateWebview(rawUrl)
  allowedUrl.value = resp.url
})
</script>

<style scoped>
.webview-page {
  min-height: 100vh;
  background: #f4f6f8;
}

.blocked-card {
  display: grid;
  gap: 12rpx;
  margin: 24rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: #ffffff;
}

.blocked-title {
  color: #1f2933;
  font-size: 34rpx;
  font-weight: 700;
}

.blocked-desc {
  color: #697586;
  font-size: 28rpx;
}
</style>
