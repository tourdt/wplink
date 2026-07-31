<template>
  <view :class="['resource-card', variantClass]" @click="$emit('open', resource)">
    <template v-if="isMarketVariant">
      <view class="thumb-wrap">
        <image class="resource-thumb" :src="coverUrl || DEFAULT_RESOURCE_COVER" mode="aspectFill" />
        <text
          v-if="resourceTypeLabel"
          class="market-type-badge supply"
          :class="{ 'with-completed': isCompleted }"
        >
          {{ resourceTypeLabel }}
        </text>
        <text v-if="isCompleted" class="market-completed-badge">已完成</text>
      </view>
      <view class="market-card-main">
        <text class="market-title">{{ resource.title || '供应标题待完善' }}</text>
        <view v-if="hasMarketTradeInfo" class="market-decision-line">
          <text v-if="quantityText" class="market-quantity">{{ quantityText }}</text>
          <text v-if="resource.priceText" class="market-price">{{ resource.priceText }}</text>
        </view>
        <text v-else class="market-trade-empty">交易信息待完善</text>
        <view class="market-merchant-line">
          <text class="merchant-name">{{ merchantName }}</text>
          <text v-if="freshnessText" class="refresh-time">{{ freshnessText }}</text>
        </view>
      </view>
    </template>
    <template v-else>
      <view class="thumb-wrap">
        <image class="resource-thumb" :src="coverUrl || DEFAULT_RESOURCE_COVER" mode="aspectFill" />
      </view>
      <view class="card-main">
        <view class="card-head">
          <view class="badge-row">
            <text class="direction-badge supply">供应</text>
            <text v-if="resourceTypeLabel" class="type-badge">{{ resourceTypeLabel }}</text>
            <text v-if="isCompleted" class="completed-badge">已完成</text>
          </view>
          <text v-if="freshnessText" class="refresh-time">{{ freshnessText }}</text>
        </view>
        <text class="resource-title">{{ resource.title || '供应标题待完善' }}</text>
        <text class="resource-meta">{{ resourceSummaryText }}</text>
        <view v-if="resource.priceText || locationText" class="value-line">
          <text v-if="resource.priceText" class="resource-price">{{ resource.priceText }}</text>
          <text v-if="locationText" class="location-text">{{ locationText }}</text>
        </view>
        <view v-if="resourceLabels.length" class="resource-labels">
          <text v-for="label in resourceLabels" :key="label" class="resource-label">{{ label }}</text>
        </view>
        <view class="merchant-line">
          <text class="merchant-name">{{ merchantName }}</text>
        </view>
      </view>
    </template>
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
  if (props.variant === 'market') return 'resource-card-market'
  return ''
})
const isMarketVariant = computed(() => props.variant === 'market')
const coverUrl = computed(() => {
  const images = props.resource.images || []
  return props.resource.coverUrl || images[0] || ''
})
const merchantName = computed(() => (props.resource.merchant || {}).name || '商家待确认')
const resourceTypeLabel = computed(() => resolveResourceTypeLabel(props.resource))
const resourceSummaryText = computed(() => buildResourceSummaryText(props.resource, resourceTypeLabel.value || '供应信息待完善'))
const resourceLabels = computed(() => normalizeResourceLabels(props.resource.tags).slice(0, 3))
const locationText = computed(() => String(props.resource.district || '').trim())
const quantityText = computed(() => String(props.resource.quantityText || '').trim())
const hasMarketTradeInfo = computed(() => Boolean(quantityText.value || props.resource.priceText))
const freshnessText = computed(() => formatRefreshedAt(props.resource.refreshedAt))
const isCompleted = computed(() => Boolean(props.resource.dealtAt))

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

.card-main {
  display: grid;
  flex: 1;
  align-content: start;
  gap: 12rpx;
  min-width: 0;
}

.card-head,
.value-line,
.merchant-line {
  display: flex;
  align-items: center;
  gap: 12rpx;
  justify-content: space-between;
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
  height: 36rpx;
  padding: 0 10rpx;
  border-radius: 8rpx;
  font-size: 22rpx;
  font-weight: 700;
  line-height: 1;
}

.completed-badge {
  display: inline-flex;
  align-items: center;
  height: 36rpx;
  padding: 0 10rpx;
  border-radius: 8rpx;
  background: #e2e8f0;
  color: #475569;
  font-size: 22rpx;
  font-weight: 700;
}

.direction-badge.supply {
  background: $wplink-primary-soft;
  color: $wplink-primary;
}

.type-badge {
  max-width: 168rpx;
  overflow: hidden;
  background: #f8fafc;
  color: #475569;
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

.value-line {
  min-width: 0;
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

.resource-card-home .direction-badge,
.resource-card-home .type-badge {
  height: 34rpx;
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

.resource-card-home .merchant-name,
.resource-card-home .refresh-time,
.resource-card-home .location-text {
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

.resource-card-compact .direction-badge,
.resource-card-compact .type-badge {
  height: 32rpx;
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

.resource-card-compact .merchant-name,
.resource-card-compact .refresh-time,
.resource-card-compact .location-text {
  font-size: 22rpx;
}

.resource-card-market {
  align-items: flex-start;
  gap: 12rpx;
  padding: 20rpx;
}

.resource-card-market .thumb-wrap {
  flex-basis: 152rpx;
  width: 152rpx;
  height: 152rpx;
}

.market-type-badge,
.market-completed-badge {
  position: absolute;
  top: 8rpx;
  z-index: 1;
  display: inline-flex;
  align-items: center;
  height: 36rpx;
  padding: 0 10rpx;
  border-radius: 8rpx;
  color: #fff;
  font-size: 20rpx;
  font-weight: 700;
  line-height: 1;
}

.market-type-badge {
  left: 8rpx;
  max-width: calc(100% - 16rpx);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.market-type-badge.with-completed {
  max-width: 68rpx;
}

.market-type-badge.supply {
  background: $wplink-primary;
}

.market-completed-badge {
  right: 8rpx;
  background: rgba(71, 85, 105, 0.92);
}

.market-card-main {
  display: grid;
  flex: 1;
  align-self: stretch;
  align-content: space-between;
  min-width: 0;
}

.market-title,
.market-quantity,
.market-price,
.market-trade-empty {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.market-title {
  color: $wplink-primary;
  font-size: 30rpx;
  font-weight: 700;
  line-height: 1.35;
}

.market-decision-line,
.market-merchant-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10rpx;
  min-width: 0;
}

.market-quantity {
  min-width: 0;
  color: #475569;
  font-size: 25rpx;
  font-weight: 600;
}

.market-price {
  min-width: 0;
  margin-left: auto;
  color: $wplink-warning;
  font-size: 28rpx;
  font-weight: 700;
  text-align: right;
}

.market-trade-empty {
  color: $wplink-muted;
  font-size: 24rpx;
}
</style>
