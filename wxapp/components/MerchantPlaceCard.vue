<template>
  <MerchantListItem
    :title="place.name"
    :subtitle="locationText"
    :tags="place.displayTags"
    :selected="selected"
    @activate="$emit('select', place)"
  >
    <template #leading>
      <view class="doorplate">
        <text class="doorplate-label">档口</text>
        <text class="doorplate-code">{{ place.code || '--' }}</text>
      </view>
    </template>

    <template #badge>
      <text :class="['source-badge', { claimed: place.claimed }]">{{ sourceLabel }}</text>
    </template>

    <template #meta>
      <text class="distance-text">{{ hasLocation ? (place.distanceText || '可导航到店') : '位置待完善' }}</text>
    </template>

    <template #actions>
      <button v-if="hasMerchantDetail(place)" class="detail-button" @click.stop="$emit('detail', place)">进入主页</button>
      <button v-if="canOpenMerchantLocation" class="location-button" @click.stop="$emit('location', place)">查看位置</button>
      <button v-else-if="!place.claimed && hasLocation" class="navigate-button" @click.stop="$emit('navigate', place)">导航</button>
      <button v-if="!place.claimed" class="claim-button" @click.stop="$emit('claim', place)">这是我的档口</button>
    </template>
  </MerchantListItem>
</template>

<script setup>
import { computed } from 'vue'
import MerchantListItem from './MerchantListItem.vue'
import { hasMerchantDetail, hasValidLocation } from '../pages/sourcing-map/merchantPlaceState'

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

defineEmits(['select', 'detail', 'location', 'navigate', 'claim'])

const hasLocation = computed(() => hasValidLocation(props.place))
const canOpenMerchantLocation = computed(() => hasMerchantDetail(props.place) && hasLocation.value)
const sourceLabel = computed(() => props.place.claimed ? '已入驻' : '待认领')
const locationText = computed(() => [
  props.place.marketName,
  props.place.buildingName,
  props.place.floorNo,
  props.place.address,
].filter(Boolean).join(' · ') || '档口地址待完善')
</script>

<style lang="scss" scoped>
.doorplate {
  width: 112rpx;
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

.distance-text {
  color: #8190a5;
  font-size: 22rpx;
}

.detail-button,
.location-button,
.navigate-button,
.claim-button {
  min-width: 92rpx;
  margin: 0;
  padding: 0 14rpx;
  border-radius: 8rpx;
  font-size: 22rpx;
  line-height: 50rpx;
}

.detail-button {
  border: 1rpx solid #172033;
  background: #ffffff;
  color: #172033;
}

.location-button,
.navigate-button {
  background: #c23a00;
  color: #ffffff;
}

.claim-button {
  border: 1rpx solid #c23a00;
  background: #fff7f2;
  color: #a83200;
}

.detail-button::after,
.location-button::after,
.navigate-button::after,
.claim-button::after {
  border: 0;
}
</style>
