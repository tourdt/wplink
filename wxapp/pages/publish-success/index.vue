<template>
  <view class="success-page">
    <view class="success-card">
      <text class="success-title">资源已提交审核</text>
      <text class="success-desc">审核结果会通过消息中心通知，审核通过后资源将进入搜索和推荐流量。</text>
      <button class="primary-button" @click="openMessages">查看消息</button>
      <button class="secondary-button" @click="openMyResources">查看我的发布</button>
    </view>
  </view>
</template>

<script setup>
import { getMerchantId } from '../../store/session'

function openMessages() {
  uni.switchTab({ url: '/pages/messages/index' })
}

function openMyResources() {
  uni.navigateTo({ url: `/pages/my-resources/index?merchantId=${getMerchantId()}` })
}
</script>

<style scoped>
.success-page {
  min-height: 100vh;
  padding: 24rpx;
  background: #f4f6f8;
}

.success-card {
  display: grid;
  gap: 18rpx;
  padding: 32rpx 24rpx;
  border-radius: 12rpx;
  background: #ffffff;
}

.success-title {
  color: #1f2933;
  font-size: 38rpx;
  font-weight: 700;
}

.success-desc {
  color: #697586;
  font-size: 28rpx;
  line-height: 1.6;
}

.primary-button,
.secondary-button {
  height: 88rpx;
  border-radius: 12rpx;
}

.primary-button {
  background: #0f766e;
  color: #ffffff;
}

.secondary-button {
  background: #edf2f7;
  color: #364152;
}
</style>
