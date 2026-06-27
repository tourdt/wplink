<template>
  <view class="success-page">
    <view class="success-card">
      <text class="success-title">采购需求已提交</text>
      <text class="success-desc">运营会根据需求进行人工撮合，后续进展可在消息中心查看。</text>
      <button class="primary-button" @click="openMessages">查看消息</button>
      <button class="secondary-button" @click="backHome">返回首页</button>
    </view>
  </view>
</template>

<script setup>
function openMessages() {
  uni.switchTab({ url: '/pages/messages/index' })
}

function backHome() {
  uni.switchTab({ url: '/pages/home/index' })
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
