<template>
  <view class="merchant-badge">
    <text class="merchant-name">{{ merchant.name || '商家' }}</text>
    <text v-if="merchant.verificationStatus === 'verified'" class="verified-tag">已认证</text>
  </view>
</template>

<script setup>
defineProps({
  merchant: {
    type: Object,
    default: () => ({}),
  },
})
</script>

<style scoped>
.merchant-badge {
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.merchant-name {
  color: #1f2933;
  font-size: 32rpx;
  font-weight: 700;
}

.verified-tag {
  padding: 6rpx 12rpx;
  border-radius: 8rpx;
  background: #e6f4f1;
  color: #0f766e;
  font-size: 24rpx;
}
</style>
