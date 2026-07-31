<template>
  <view
    :class="['merchant-list-item', { selected }]"
    role="button"
    @click="$emit('activate')"
  >
    <view class="merchant-list-leading">
      <slot name="leading" />
    </view>

    <view class="merchant-list-main">
      <view class="merchant-list-title-row">
        <text class="merchant-list-title">{{ title }}</text>
        <slot name="badge" />
      </view>

      <text v-if="subtitle" class="merchant-list-subtitle">{{ subtitle }}</text>

      <view v-if="tags.length" class="merchant-list-tags">
        <text v-for="tag in tags" :key="tag" class="merchant-list-tag">{{ tag }}</text>
      </view>

      <view class="merchant-list-foot">
        <slot name="meta" />
        <view class="merchant-list-actions">
          <slot name="actions" />
        </view>
      </view>
    </view>
  </view>
</template>

<script setup>
defineProps({
  title: {
    type: String,
    required: true,
  },
  subtitle: {
    type: String,
    default: '',
  },
  tags: {
    type: Array,
    default: () => [],
  },
  selected: {
    type: Boolean,
    default: false,
  },
})

defineEmits(['activate'])
</script>

<style lang="scss" scoped>
.merchant-list-item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 20rpx;
  padding: 24rpx;
  border: 1rpx solid rgba(148, 163, 184, 0.32);
  border-radius: 16rpx;
  background: #ffffff;
  box-shadow: 0 8rpx 22rpx rgba(15, 23, 42, 0.05);
  transition: border-color 120ms ease-out, box-shadow 120ms ease-out, transform 120ms ease-out;
}

.merchant-list-item:active {
  transform: translateY(1rpx);
}

.merchant-list-item.selected {
  border-color: rgba(194, 58, 0, 0.72);
  box-shadow: 0 10rpx 28rpx rgba(194, 58, 0, 0.12);
}

.merchant-list-leading {
  align-self: start;
}

.merchant-list-main {
  min-width: 0;
}

.merchant-list-title-row,
.merchant-list-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14rpx;
}

.merchant-list-title {
  min-width: 0;
  overflow: hidden;
  color: $wplink-primary;
  font-size: 30rpx;
  font-weight: 750;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.merchant-list-subtitle {
  display: block;
  margin-top: 10rpx;
  overflow: hidden;
  color: $wplink-muted;
  font-size: 23rpx;
  line-height: 1.55;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.merchant-list-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8rpx;
  margin-top: 14rpx;
}

.merchant-list-tag {
  max-width: 180rpx;
  padding: 5rpx 10rpx;
  overflow: hidden;
  border-radius: 6rpx;
  background: #f1f4f8;
  color: #526176;
  font-size: 20rpx;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.merchant-list-foot {
  min-height: 50rpx;
  margin-top: 16rpx;
}

.merchant-list-actions {
  display: flex;
  flex: none;
  gap: 8rpx;
}
</style>
