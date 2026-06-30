<template>
  <view class="success-page">
    <view class="success-card">
      <view class="success-icon">✓</view>
      <text class="success-title">资源已提交审核</text>
      <text class="success-desc">审核通过后，资源会进入搜索、推荐和商家主页，后续进展会通过消息中心通知。</text>

      <view class="success-result-list">
        <view class="success-result-item">
          <text class="result-label">审核结果</text>
          <text class="result-value">消息中心通知</text>
        </view>
        <view class="success-result-item">
          <text class="result-label">通过后曝光</text>
          <text class="result-value">搜索、推荐和商家主页</text>
        </view>
      </view>

      <view class="success-actions">
        <button class="wplink-primary-button action-button" @click="openMyResources">查看我的发布</button>
        <button class="wplink-secondary-button action-button" @click="openMessages">查看消息</button>
      </view>
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

<style lang="scss" scoped>
.success-page {
  min-height: 100vh;
  padding: 48rpx 24rpx 24rpx;
  background: $wplink-bg;
}

.success-card {
  display: grid;
  justify-items: center;
  gap: 18rpx;
  padding: 56rpx 28rpx 32rpx;
  border-radius: 12rpx;
  background: $wplink-card;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.04);
  text-align: center;
}

.success-icon {
  display: grid;
  place-items: center;
  width: 112rpx;
  height: 112rpx;
  border-radius: 999rpx;
  background: $wplink-success;
  color: #ffffff;
  font-size: 58rpx;
  font-weight: 800;
  line-height: 1;
}

.success-title {
  color: $wplink-text;
  font-size: 38rpx;
  font-weight: 700;
  line-height: 1.3;
}

.success-desc {
  max-width: 612rpx;
  color: $wplink-muted;
  font-size: 28rpx;
  line-height: 1.6;
}

.success-result-list {
  display: grid;
  gap: 0;
  width: 100%;
  margin: 14rpx 0 8rpx;
  border-top: 1rpx solid $wplink-line;
  border-bottom: 1rpx solid $wplink-line;
}

.success-result-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24rpx;
  min-height: 88rpx;
  padding: 18rpx 0;
  text-align: left;
}

.success-result-item + .success-result-item {
  border-top: 1rpx solid $wplink-line;
}

.result-label {
  flex: 0 0 auto;
  color: $wplink-text;
  font-size: 27rpx;
  font-weight: 700;
}

.result-value {
  min-width: 0;
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.45;
  text-align: right;
}

.success-actions {
  display: grid;
  gap: 16rpx;
  width: 100%;
}

.action-button {
  width: 100%;
}
</style>
