<template>
  <view class="metric-strip">
    <view v-for="item in items" :key="item.label" class="metric-item">
      <text class="metric-value">{{ item.value }}</text>
      <text class="metric-label">{{ item.label }}</text>
    </view>
  </view>
</template>

<script setup>
defineProps({
  items: {
    type: Array,
    default: () => [],
  },
})
</script>

<style scoped>
.metric-strip {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8rpx;
}

.metric-item {
  display: grid;
  gap: 6rpx;
  padding: 12rpx;
  border-radius: 10rpx;
  background: #f8fafc;
  text-align: center;
}

.metric-value {
  color: #1f2933;
  font-size: 28rpx;
  font-weight: 700;
}

.metric-label {
  color: #697586;
  font-size: 22rpx;
}
</style>
