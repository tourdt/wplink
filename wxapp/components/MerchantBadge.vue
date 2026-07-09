<template>
  <view class="merchant-badge">
    <text class="merchant-name">{{ merchant.name || '商家' }}</text>
    <text v-if="merchant.vipStatus === 'active'" class="vip-tag">VIP</text>
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

<style lang="scss" scoped>
.merchant-badge {
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.merchant-name {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
}

.vip-tag {
  padding: 6rpx 12rpx;
  border-radius: 8rpx;
  background: rgba(194, 58, 0, 0.1);
  color: $wplink-warning;
  font-size: 24rpx;
}
</style>
