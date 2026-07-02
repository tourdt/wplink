<template>
  <view class="sourcing-map-page">
    <view class="map-empty-card">
      <text class="map-empty-title">拿货地图</text>
      <text class="map-empty-desc">正在加载织里童装拿货地图。</text>
    </view>
  </view>
</template>

<script setup>
</script>

<style lang="scss" scoped>
.sourcing-map-page {
  min-height: 100vh;
  padding: 28rpx;
  background: $wplink-bg;
}

.map-empty-card {
  display: grid;
  gap: 14rpx;
  padding: 40rpx 32rpx;
  border-radius: 20rpx;
  background: $wplink-card;
  box-shadow: 0 16rpx 48rpx rgba(15, 23, 42, 0.06);
}

.map-empty-title {
  color: $wplink-primary;
  font-size: 34rpx;
  font-weight: 800;
}

.map-empty-desc {
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.5;
}
</style>
