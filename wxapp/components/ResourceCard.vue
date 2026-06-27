<template>
  <view class="resource-card" @click="$emit('open', resource)">
    <text class="resource-title">{{ resource.title }}</text>
    <text class="resource-meta">{{ resource.category }} · {{ resource.quantityText || '数量待沟通' }}</text>
    <text class="resource-price">{{ resource.priceText || '价格面议' }}</text>
  </view>
</template>

<script setup>
defineProps({
  resource: {
    type: Object,
    required: true,
  },
})

defineEmits(['open'])
</script>

<style scoped>
.resource-card {
  display: grid;
  gap: 12rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: #ffffff;
}

.resource-title {
  color: #1f2933;
  font-size: 32rpx;
  font-weight: 700;
}

.resource-meta {
  color: #697586;
  font-size: 28rpx;
}

.resource-price {
  color: #c2410c;
  font-size: 30rpx;
  font-weight: 700;
}
</style>
