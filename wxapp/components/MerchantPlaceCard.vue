<template>
  <MerchantListItem
    :title="merchant.name"
    :tags="merchant.mainCategories"
    @activate="$emit('detail', merchant)"
  >
    <template #leading>
      <view class="doorplate">
        <text class="doorplate-label">档口</text>
        <text class="doorplate-code">{{ merchant.boothNo || '--' }}</text>
      </view>
    </template>

    <template #badge>
      <text v-if="merchant.merchantNo" class="merchant-number">{{ merchant.merchantNo }}</text>
    </template>

    <template #meta>
      <text class="coordinate-strip">{{ coordinateText }}</text>
    </template>

    <template #actions>
      <button class="detail-button" @click.stop="$emit('detail', merchant)">进入主页</button>
      <button v-if="hasLocation" class="location-button" @click.stop="$emit('location', merchant)">查看位置</button>
    </template>
  </MerchantListItem>
</template>

<script setup>
import { computed } from 'vue'
import MerchantListItem from './MerchantListItem.vue'
import {
  hasValidMerchantLocation,
  merchantDirectoryCoordinateText,
} from '../pages/sourcing-map/merchantDirectoryState'

const props = defineProps({
  merchant: {
    type: Object,
    required: true,
  },
})

defineEmits(['detail', 'location'])

const hasLocation = computed(() => hasValidMerchantLocation(props.merchant))
const coordinateText = computed(() => merchantDirectoryCoordinateText(props.merchant))
</script>

<style lang="scss" scoped>
.doorplate {
  width: 112rpx;
  overflow: hidden;
  border: 2rpx solid $wplink-primary;
  border-radius: 8rpx;
  background: #f8fafc;
  text-align: center;
}

.doorplate-label {
  display: block;
  padding: 5rpx 8rpx;
  background: $wplink-primary;
  color: $wplink-card;
  font-size: 18rpx;
  letter-spacing: 4rpx;
}

.doorplate-code {
  display: block;
  overflow: hidden;
  padding: 15rpx 8rpx;
  color: $wplink-primary;
  font-size: 25rpx;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.merchant-number {
  color: $wplink-muted;
  font-size: 20rpx;
  font-weight: 650;
}

.coordinate-strip {
  display: block;
  overflow: hidden;
  padding-left: 12rpx;
  border-left: 4rpx solid $wplink-warning;
  color: $wplink-primary;
  font-size: 23rpx;
  font-weight: 650;
  letter-spacing: 0.5rpx;
  line-height: 1.45;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.detail-button,
.location-button {
  min-width: 104rpx;
  margin: 0;
  padding: 0 16rpx;
  border-radius: 8rpx;
  font-size: 22rpx;
  line-height: 52rpx;
}

.detail-button {
  border: 1rpx solid $wplink-primary;
  background: $wplink-card;
  color: $wplink-primary;
}

.location-button {
  background: $wplink-warning;
  color: $wplink-card;
}

.detail-button::after,
.location-button::after {
  border: 0;
}
</style>
