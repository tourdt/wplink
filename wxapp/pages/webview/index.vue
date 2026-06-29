<template>
  <view class="webview-page">
    <view v-if="allowedUrl" class="activity-shell">
      <view class="webview-bar">
        <text>web-view</text>
        <text class="url-text">{{ allowedUrl }}</text>
      </view>
      <view class="activity-card">
        <view class="activity-cover">
          <text>活动封面</text>
        </view>
        <view class="activity-body">
          <text class="activity-tag">城市站活动</text>
          <text class="activity-title">织里童装夏款供需对接会</text>
          <text class="activity-desc">活动页由已配置业务域名承载，返回小程序后可继续查看相关平台资源。</text>
          <button class="secondary-button" @click="openTopic">查看相关资源</button>
        </view>
      </view>
      <web-view :src="allowedUrl" />
    </view>
    <view v-else class="blocked-card">
      <text class="blocked-title">链接不可访问</text>
      <text class="blocked-desc">该活动链接不在平台允许范围内。</text>
      <text class="blocked-rule">URL 必须属于已配置业务域名；活动页内容需和服装产业相关，并保留返回小程序路径。</text>
      <button class="secondary-button" @click="openTopic">查看平台资源</button>
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

function openTopic() {
  uni.navigateTo({ url: '/pages/topic/index?id=default-topic' })
}
</script>

<style lang="scss" scoped>
.webview-page {
  min-height: 100vh;
  background: $wplink-bg;
}

.activity-shell {
  min-height: 100vh;
  padding: 24rpx;
  background: $wplink-bg;
}

.webview-bar {
  display: grid;
  gap: 8rpx;
  margin-bottom: 20rpx;
  padding: 18rpx 20rpx;
  border-radius: 12rpx;
  background: $wplink-card;
}

.webview-bar text:first-child {
  color: $wplink-primary;
  font-size: 24rpx;
  font-weight: 700;
}

.url-text {
  color: $wplink-muted;
  font-size: 22rpx;
  line-height: 1.4;
  word-break: break-all;
}

.activity-card {
  overflow: hidden;
  margin-bottom: 20rpx;
  border-radius: 12rpx;
  background: $wplink-card;
}

.activity-cover {
  display: flex;
  align-items: flex-end;
  height: 260rpx;
  padding: 24rpx;
  background:
    radial-gradient(circle at 32% 24%, rgba(255, 255, 255, 0.26), transparent 28%),
    #7b8fc7;
  color: $wplink-card;
  font-size: 30rpx;
  font-weight: 700;
}

.activity-body {
  display: grid;
  gap: 12rpx;
  padding: 24rpx;
}

.activity-tag {
  color: $wplink-warning;
  font-size: 24rpx;
  font-weight: 700;
}

.activity-title {
  color: $wplink-primary;
  font-size: 36rpx;
  font-weight: 700;
  line-height: 1.3;
}

.activity-desc,
.blocked-rule {
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.55;
}

.blocked-card {
  display: grid;
  gap: 12rpx;
  margin: 24rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
}

.blocked-title {
  color: $wplink-primary;
  font-size: 34rpx;
  font-weight: 700;
}

.blocked-desc {
  color: $wplink-muted;
  font-size: 28rpx;
}

.secondary-button {
  height: 84rpx;
  border-radius: 12rpx;
  background: $wplink-primary-soft;
  color: $wplink-primary;
  font-size: 28rpx;
  font-weight: 700;
}
</style>
