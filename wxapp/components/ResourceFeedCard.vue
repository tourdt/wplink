<template>
  <view :class="['resource-feed-card', { demand: cardModel.isDemand }]" @click="$emit('open', resource)">
    <view class="feed-thumb-wrap">
      <image
        class="feed-thumb"
        :src="cardModel.coverUrl || DEFAULT_RESOURCE_COVER"
        mode="aspectFill"
      />
      <text
        :class="['feed-type-badge', { demand: cardModel.isDemand, 'with-completed': cardModel.isCompleted }]"
      >
        {{ cardModel.resourceTypeLabel || cardModel.directionLabel }}
      </text>
      <text v-if="cardModel.isCompleted" class="feed-completed-badge">已完成</text>
    </view>

    <view class="feed-card-main">
      <text class="feed-title">{{ cardModel.titleText }}</text>
      <view v-if="cardModel.hasTradeInfo" class="feed-decision-line">
        <text v-if="cardModel.quantityText" class="feed-quantity">{{ cardModel.quantityText }}</text>
        <text v-if="cardModel.priceText" class="feed-price">{{ cardModel.priceText }}</text>
      </view>
      <text v-else class="feed-trade-empty">交易信息待完善</text>
      <view class="feed-merchant-line">
        <text class="feed-merchant-name">{{ cardModel.merchantName }}</text>
        <text v-if="cardModel.freshnessText" class="feed-refresh-time">{{ cardModel.freshnessText }}</text>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed } from 'vue'

import { buildResourceFeedCardModel } from './resourceFeedCardState.js'

const DEFAULT_RESOURCE_COVER = '/static/resource/default-resource-cover.png'

const props = defineProps({
  resource: {
    type: Object,
    required: true,
  },
})

defineEmits(['open'])

const cardModel = computed(() => buildResourceFeedCardModel(props.resource))
</script>

<style lang="scss" scoped>
.resource-feed-card {
  display: flex;
  align-items: flex-start;
  gap: 12rpx;
  padding: 20rpx;
  overflow: hidden;
  border: 1rpx solid rgba(148, 163, 184, 0.22);
  border-radius: 12rpx;
  background: $wplink-card;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.04);
}

.feed-thumb-wrap {
  position: relative;
  box-sizing: border-box;
  flex: 0 0 152rpx;
  width: 152rpx;
  height: 152rpx;
  overflow: hidden;
  border-radius: 10rpx;
  background: #edf2f7;
}

.feed-thumb {
  display: block;
  width: 100%;
  height: 100%;
}

.feed-type-badge,
.feed-completed-badge {
  position: absolute;
  top: 8rpx;
  z-index: 1;
  display: inline-flex;
  align-items: center;
  height: 36rpx;
  padding: 0 10rpx;
  border-radius: 8rpx;
  color: #ffffff;
  font-size: 20rpx;
  font-weight: 700;
  line-height: 1;
}

.feed-type-badge {
  left: 8rpx;
  max-width: calc(100% - 16rpx);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.feed-type-badge.with-completed {
  max-width: 68rpx;
}

.feed-type-badge:not(.demand) {
  background: $wplink-primary;
}

.feed-type-badge.demand {
  background: $wplink-warning;
}

.feed-completed-badge {
  right: 8rpx;
  background: rgba(71, 85, 105, 0.92);
}

.feed-card-main {
  display: grid;
  flex: 1;
  align-self: stretch;
  align-content: space-between;
  min-width: 0;
}

.feed-title,
.feed-quantity,
.feed-price,
.feed-trade-empty,
.feed-merchant-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.feed-title {
  color: $wplink-primary;
  font-size: 30rpx;
  font-weight: 700;
  line-height: 1.35;
}

.feed-decision-line,
.feed-merchant-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10rpx;
  min-width: 0;
}

.feed-quantity {
  min-width: 0;
  color: #475569;
  font-size: 25rpx;
  font-weight: 600;
}

.feed-price {
  min-width: 0;
  margin-left: auto;
  color: $wplink-warning;
  font-size: 28rpx;
  font-weight: 700;
  text-align: right;
}

.feed-trade-empty,
.feed-merchant-name,
.feed-refresh-time {
  color: $wplink-muted;
  font-size: 24rpx;
}

.feed-merchant-name {
  flex: 1;
  min-width: 0;
}

.feed-refresh-time {
  flex: 0 0 auto;
}
</style>
