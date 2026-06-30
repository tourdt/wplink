<template>
  <view class="success-page">
    <view class="success-card">
      <view class="success-icon">✓</view>
      <text class="success-title">采购需求已提交</text>
      <text class="success-desc">运营会根据你的采购需求继续跟进，匹配进展和处理结果会通过消息中心通知。</text>

      <view class="success-result-list">
        <view class="success-result-item">
          <text class="result-label">跟进通知</text>
          <text class="result-value">消息中心查看</text>
        </view>
        <view class="success-result-item">
          <text class="result-label">后续处理</text>
          <text class="result-value">运营继续对接合适资源</text>
        </view>
      </view>

      <view class="success-actions">
        <button class="wplink-primary-button action-button" @click="openMessages">查看消息</button>
        <view class="secondary-action-row">
          <button class="wplink-secondary-button action-button" @click="openMyDemands">我的需求</button>
          <button class="wplink-secondary-button action-button" @click="backHome">返回首页</button>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup>
function openMessages() {
  uni.switchTab({ url: '/pages/messages/index' })
}

function openMyDemands() {
  uni.navigateTo({ url: '/pages/my-demands/index' })
}

function backHome() {
  uni.switchTab({ url: '/pages/home/index' })
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

.secondary-action-row {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14rpx;
}

.action-button {
  width: 100%;
}
</style>
