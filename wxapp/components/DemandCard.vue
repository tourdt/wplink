<template>
  <view class="demand-card" @click="$emit('open', resource)">
    <view class="thumb-wrap">
      <image class="resource-thumb" :src="coverUrl || DEFAULT_RESOURCE_COVER" mode="aspectFill" />
    </view>
    <view class="card-main">
      <view class="demand-head">
        <view class="badge-row">
          <text class="direction-badge">需求</text>
          <text v-if="resourceTypeLabel" class="type-badge">{{ resourceTypeLabel }}</text>
        </view>
        <text v-if="freshnessText" class="refresh-time">{{ freshnessText }}</text>
      </view>
      <text class="demand-title">{{ resource.title || '需求标题待完善' }}</text>
      <text class="demand-meta">{{ resourceSummaryText }}</text>
      <view v-if="resource.priceText || locationText" class="value-line">
        <text v-if="resource.priceText" class="budget-text">{{ resource.priceText }}</text>
        <text v-if="locationText" class="location-text">{{ locationText }}</text>
      </view>
      <view v-if="resourceLabels.length" class="resource-labels">
        <text v-for="label in resourceLabels" :key="label" class="resource-label">{{ label }}</text>
      </view>
      <view class="merchant-line">
        <text class="merchant-name">{{ merchantName }}</text>
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
})

defineEmits(['open'])

const coverUrl = computed(() => {
  const images = props.resource.images || []
  return props.resource.coverUrl || images[0] || ''
})
const merchantName = computed(() => (props.resource.merchant || {}).name || '采购方待确认')
const resourceTypeLabel = computed(() => resolveResourceTypeLabel(props.resource))
const resourceSummaryText = computed(() => buildResourceSummaryText(props.resource, resourceTypeLabel.value || '需求信息待完善'))
const resourceLabels = computed(() => normalizeResourceLabels(props.resource.tags).slice(0, 3))
const locationText = computed(() => String(props.resource.district || '').trim())
const freshnessText = computed(() => formatRefreshedAt(props.resource.refreshedAt))

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
.demand-card {
  display: flex;
  align-items: stretch;
  gap: 12rpx;
  padding: 24rpx;
  border: 1rpx solid rgba($wplink-warning, 0.18);
  border-radius: 12rpx;
  background: $wplink-card;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.04);
  overflow: hidden;
}

.thumb-wrap {
  align-self: flex-start;
  box-sizing: border-box;
  flex: 0 0 168rpx;
  width: 168rpx;
  height: 168rpx;
  min-height: 0;
  overflow: hidden;
  border-radius: 10rpx;
  background: #fff7ed;
}

.resource-thumb {
  display: block;
  width: 100%;
  height: 100%;
}

.card-main {
  display: grid;
  flex: 1;
  align-content: start;
  gap: 12rpx;
  min-width: 0;
}

.demand-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
}

.badge-row {
  display: flex;
  align-items: center;
  gap: 8rpx;
  min-width: 0;
}

.direction-badge,
.type-badge {
  display: inline-flex;
  align-items: center;
  flex: 0 0 auto;
  height: 38rpx;
  padding: 0 12rpx;
  border-radius: 8rpx;
  font-size: 22rpx;
  font-weight: 700;
  line-height: 1;
}

.direction-badge {
  background: $wplink-warning-soft;
  color: $wplink-warning;
}

.type-badge {
  max-width: 220rpx;
  overflow: hidden;
  background: $wplink-primary-soft;
  color: $wplink-primary;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.demand-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  word-break: break-word;
}

.demand-meta {
  color: $wplink-muted;
  font-size: 28rpx;
  line-height: 1.45;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  word-break: break-word;
}

.value-line,
.merchant-line {
  display: flex;
  align-items: center;
  gap: 12rpx;
  justify-content: space-between;
}

.budget-text {
  flex: 1;
  min-width: 0;
  color: $wplink-warning;
  font-size: 30rpx;
  font-weight: 700;
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.location-text {
  flex: 0 1 auto;
  min-width: 0;
  color: $wplink-muted;
  font-size: 24rpx;
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
  max-width: 180rpx;
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
</style>
