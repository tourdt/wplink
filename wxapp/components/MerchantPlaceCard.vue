<template>
  <view class="merchant-place-card" :class="{ selected }" @click="$emit('select', place)">
    <view class="doorplate">
      <text class="doorplate-label">档口</text>
      <text class="doorplate-code">{{ place.code || '--' }}</text>
    </view>

    <view class="place-main">
      <view class="place-title-row">
        <text class="place-name">{{ place.name }}</text>
        <text :class="['source-badge', { claimed: place.claimed }]">{{ sourceLabel }}</text>
      </view>
      <text class="market-location">{{ locationText }}</text>
      <view v-if="visibleTags.length" class="tag-row">
        <text v-for="tag in visibleTags" :key="tag" class="place-tag">{{ tag }}</text>
      </view>
      <view class="place-foot">
        <text class="distance-text">{{ hasLocation ? (place.distanceText || '可导航到店') : '位置待完善' }}</text>
        <view class="place-actions">
          <button v-if="hasLocation" class="navigate-button" @click.stop="$emit('navigate', place)">导航</button>
          <button v-if="!place.claimed" class="claim-button" @click.stop="$emit('claim', place)">这是我的档口</button>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed } from 'vue'
import { hasValidLocation } from '../pages/sourcing-map/merchantPlaceState'

const props = defineProps({
  place: {
    type: Object,
    required: true,
  },
  selected: {
    type: Boolean,
    default: false,
  },
})

defineEmits(['select', 'navigate', 'claim'])

const hasLocation = computed(() => hasValidLocation(props.place))
const sourceLabel = computed(() => props.place.claimed ? '已入驻' : '待认领')
const visibleTags = computed(() => [
  ...(props.place.categoryCodes || []),
  ...(props.place.serviceTags || []),
].slice(0, 3))
const locationText = computed(() => [
  props.place.marketName,
  props.place.buildingName,
  props.place.floorNo,
  props.place.address,
].filter(Boolean).join(' · ') || '档口地址待完善')
</script>

<style lang="scss" scoped>
.merchant-place-card {
  display: grid;
  grid-template-columns: 112rpx minmax(0, 1fr);
  gap: 20rpx;
  padding: 24rpx;
  border: 1rpx solid rgba(148, 163, 184, 0.32);
  border-radius: 16rpx;
  background: #ffffff;
  box-shadow: 0 8rpx 22rpx rgba(15, 23, 42, 0.05);
}

.merchant-place-card.selected {
  border-color: rgba(194, 58, 0, 0.72);
  box-shadow: 0 10rpx 28rpx rgba(194, 58, 0, 0.12);
}

.doorplate {
  align-self: start;
  overflow: hidden;
  border: 2rpx solid #172033;
  border-radius: 8rpx;
  background: #f8fafc;
  text-align: center;
}

.doorplate-label {
  display: block;
  padding: 5rpx 8rpx;
  background: #172033;
  color: #ffffff;
  font-size: 18rpx;
  letter-spacing: 4rpx;
}

.doorplate-code {
  display: block;
  overflow: hidden;
  padding: 15rpx 8rpx;
  color: #172033;
  font-size: 25rpx;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.place-main {
  min-width: 0;
}

.place-title-row,
.place-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14rpx;
}

.place-name {
  overflow: hidden;
  color: #172033;
  font-size: 30rpx;
  font-weight: 750;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.source-badge {
  flex: none;
  padding: 5rpx 12rpx;
  border: 1rpx solid #8190a5;
  border-radius: 999rpx;
  color: #607086;
  font-size: 20rpx;
}

.source-badge.claimed {
  border-color: #172033;
  background: #172033;
  color: #ffffff;
}

.market-location {
  display: block;
  margin-top: 10rpx;
  color: #667085;
  font-size: 23rpx;
  line-height: 1.55;
}

.tag-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8rpx;
  margin-top: 14rpx;
}

.place-tag {
  padding: 5rpx 10rpx;
  border-radius: 6rpx;
  background: #f1f4f8;
  color: #526176;
  font-size: 20rpx;
}

.place-foot {
  margin-top: 16rpx;
  min-height: 50rpx;
}

.distance-text {
  color: #8190a5;
  font-size: 22rpx;
}

.place-actions {
  display: flex;
  flex: none;
  gap: 8rpx;
}

.navigate-button,
.claim-button {
  min-width: 92rpx;
  margin: 0;
  padding: 0 14rpx;
  border-radius: 8rpx;
  font-size: 22rpx;
  line-height: 50rpx;
}

.navigate-button {
  background: #c23a00;
  color: #ffffff;
}

.claim-button {
  border: 1rpx solid #c23a00;
  background: #fff7f2;
  color: #a83200;
}

.navigate-button::after,
.claim-button::after {
  border: 0;
}
</style>
