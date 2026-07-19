<template>
  <view class="merchant-badge">
    <text class="merchant-name">{{ merchant.name || '商家' }}</text>
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

</style>
