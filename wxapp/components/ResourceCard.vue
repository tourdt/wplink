<template>
  <view :class="['resource-card', variantClass]" @click="$emit('open', resource)">
    <view class="thumb-wrap">
      <image class="resource-thumb" :src="coverUrl || DEFAULT_RESOURCE_COVER" mode="aspectFill" />
      <text v-if="resourceTypeLabel" class="type-corner">{{ resourceTypeLabel }}</text>
    </view>
    <view class="card-main">
      <text class="resource-title">{{ resource.title || '供应标题待完善' }}</text>
      <text class="resource-meta">{{ resourceSummaryText }}</text>
      <text v-if="resource.priceText" class="resource-price">{{ resource.priceText }}</text>
      <view v-if="resourceLabels.length" class="resource-labels">
        <text v-for="label in resourceLabels" :key="label" class="resource-label">{{ label }}</text>
      </view>
      <view class="merchant-line">
        <text v-if="isVIPMerchant" class="vip-badge">VIP</text>
        <text class="merchant-name">{{ merchantName }}</text>
        <text class="refresh-time">{{ formatRefreshedAt(resource.refreshedAt) }}</text>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed } from 'vue'

import { formatListFreshnessDate } from '../common/date'
import { resourceTypeLabel as resolveResourceTypeLabel } from '../common/resourceCategories'

const DEFAULT_RESOURCE_COVER = '/static/resource/default-resource-cover.png'

const props = defineProps({
  resource: {
    type: Object,
    required: true,
  },
  variant: {
    type: String,
    default: '',
  },
})

defineEmits(['open'])

const variantClass = computed(() => {
  if (props.variant === 'home') return 'resource-card-home'
  if (props.variant === 'compact') return 'resource-card-compact'
  return ''
})
const coverUrl = computed(() => {
  const images = props.resource.images || []
  return props.resource.coverUrl || images[0] || ''
})
const isVIPMerchant = computed(() => (props.resource.merchant || {}).vipStatus === 'active')
const merchantName = computed(() => (props.resource.merchant || {}).name || '商家待确认')
const resourceTypeLabel = computed(() => resolveResourceTypeLabel(props.resource))
const resourceSummaryText = computed(() => buildResourceSummaryText(props.resource, resourceTypeLabel.value || '供应信息待完善'))
const resourceLabels = computed(() => normalizeResourceLabels(props.resource.tags).slice(0, 3))

function buildResourceSummaryText(resource, fallbackText) {
  const parts = [resource.category, resource.quantityText]
    .map((item) => String(item || '').trim())
    .filter(Boolean)
  return parts.join(' · ') || fallbackText
}

function formatRefreshedAt(value) {
  return formatListFreshnessDate(value)
}

function normalizeResourceLabels(tags = []) {
  if (!Array.isArray(tags)) return []
  return tags
    .map((tag) => String(tag || '').trim())
    .filter(Boolean)
}
</script>

<style lang="scss" scoped>
.resource-card {
  display: flex;
  align-items: stretch;
  gap: 12rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.04);
  overflow: hidden;
}

.thumb-wrap {
  position: relative;
  align-self: flex-start;
  box-sizing: border-box;
  flex: 0 0 168rpx;
  width: 168rpx;
  height: 168rpx;
  min-height: 0;
  overflow: hidden;
  border-radius: 10rpx;
  background: #edf2f7;
}

.resource-thumb {
  display: block;
  width: 100%;
  height: 100%;
}

.type-corner {
  position: absolute;
  top: 10rpx;
  left: 10rpx;
  max-width: calc(100% - 20rpx);
  box-sizing: border-box;
  padding: 3rpx 8rpx;
  border-radius: 7rpx;
  background: rgba(15, 23, 42, 0.76);
  color: #fff;
  font-size: 20rpx;
  font-weight: 700;
  line-height: 1.3;
}

.card-main {
  display: grid;
  flex: 1;
  align-content: start;
  gap: 14rpx;
  min-width: 0;
}

.merchant-line {
  display: flex;
  align-items: center;
  gap: 12rpx;
  justify-content: space-between;
}

.vip-badge {
  flex: 0 0 auto;
  padding: 4rpx 10rpx;
  border-radius: 8rpx;
  background: rgba(194, 58, 0, 0.1);
  color: $wplink-warning;
  font-size: 22rpx;
  font-weight: 700;
  line-height: 1.3;
}

.merchant-name {
  flex: 1;
  min-width: 0;
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.refresh-time {
  flex: 0 0 auto;
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.35;
}

.resource-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  word-break: break-word;
}

.resource-meta {
  color: $wplink-muted;
  font-size: 28rpx;
  line-height: 1.45;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  word-break: break-word;
}

.resource-price {
  color: $wplink-warning;
  font-size: 30rpx;
  font-weight: 700;
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.resource-labels {
  display: flex;
  flex-wrap: wrap;
  gap: 8rpx;
  min-width: 0;
  max-height: 52rpx;
  overflow: hidden;
}

.resource-label {
  max-width: 160rpx;
  padding: 4rpx 10rpx;
  border-radius: 7rpx;
  background: #f8fafc;
  color: $wplink-muted;
  font-size: 22rpx;
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.resource-card-home {
  gap: 24rpx;
  padding: 24rpx;
  border-radius: 16rpx;
  box-shadow: 0 16rpx 48rpx rgba(15, 23, 42, 0.06);
}

.resource-card-home .thumb-wrap {
  flex-basis: 160rpx;
  width: 160rpx;
  height: 160rpx;
  min-height: 0;
  border-radius: 16rpx;
}

.resource-card-home .card-main {
  gap: 10rpx;
}

.resource-card-home .type-corner {
  top: 8rpx;
  left: 8rpx;
  max-width: calc(100% - 16rpx);
  font-size: 20rpx;
}

.resource-card-home .resource-title {
  font-size: 30rpx;
  line-height: 1.32;
}

.resource-card-home .resource-meta {
  font-size: 26rpx;
  line-height: 1.4;
}

.resource-card-home .resource-price {
  font-size: 30rpx;
  line-height: 1.3;
}

.resource-card-home .vip-badge,
.resource-card-home .merchant-name,
.resource-card-home .refresh-time {
  font-size: 24rpx;
  line-height: 1.35;
}

.resource-card-compact {
  gap: 16rpx;
  padding: 20rpx;
}

.resource-card-compact .thumb-wrap {
  flex-basis: 144rpx;
  width: 144rpx;
  height: 144rpx;
  min-height: 0;
}

.resource-card-compact .card-main {
  gap: 8rpx;
}

.resource-card-compact .type-corner {
  top: 8rpx;
  left: 8rpx;
  max-width: calc(100% - 16rpx);
  font-size: 18rpx;
}

.resource-card-compact .resource-title {
  font-size: 30rpx;
}

.resource-card-compact .resource-meta {
  font-size: 26rpx;
}

.resource-card-compact .resource-price {
  font-size: 28rpx;
}

.resource-card-compact .vip-badge,
.resource-card-compact .merchant-name,
.resource-card-compact .refresh-time {
  font-size: 22rpx;
}
</style>
