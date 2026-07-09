<template>
  <view class="demand-card" @click="$emit('open', resource)">
    <view class="demand-head">
      <text class="direction-badge">需求</text>
      <text v-if="resourceTypeLabel" class="type-badge">{{ resourceTypeLabel }}</text>
    </view>
    <text class="demand-title">{{ resource.title || '需求标题待完善' }}</text>
    <text class="demand-meta">{{ resourceSummaryText }}</text>
    <view class="demand-foot">
      <text v-if="resource.priceText" class="budget-text">{{ resource.priceText }}</text>
      <view class="merchant-line">
        <text v-if="isVerifiedMerchant" class="verified-badge">已认证</text>
        <text class="merchant-name">{{ merchantName }}</text>
        <text class="refresh-time">{{ formatRefreshedAt(resource.refreshedAt) }}</text>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed } from 'vue'

import { formatListFreshnessDate } from '../common/date'
import { resourceTypeText } from '../common/enums'

const props = defineProps({
  resource: {
    type: Object,
    required: true,
  },
})

defineEmits(['open'])

const isVerifiedMerchant = computed(() => (props.resource.merchant || {}).verificationStatus === 'verified')
const merchantName = computed(() => (props.resource.merchant || {}).name || '采购方待确认')
const resourceTypeLabel = computed(() => resourceTypeText[props.resource.typeCode] || '')
const resourceSummaryText = computed(() => buildResourceSummaryText(props.resource, resourceTypeLabel.value || '需求信息待完善'))

function buildResourceSummaryText(resource, fallbackText) {
  const parts = [resource.category, resource.quantityText]
    .map((item) => String(item || '').trim())
    .filter(Boolean)
  return parts.join(' · ') || fallbackText
}

function formatRefreshedAt(value) {
  return formatListFreshnessDate(value)
}
</script>

<style lang="scss" scoped>
.demand-card {
  display: grid;
  gap: 14rpx;
  padding: 24rpx;
  border: 1rpx solid rgba($wplink-warning, 0.18);
  border-radius: 12rpx;
  background: $wplink-card;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.04);
}

.demand-head {
  display: flex;
  align-items: center;
  gap: 10rpx;
}

.direction-badge,
.type-badge {
  display: inline-flex;
  align-items: center;
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
  background: $wplink-primary-soft;
  color: $wplink-primary;
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

.demand-foot {
  display: grid;
  gap: 12rpx;
}

.budget-text {
  color: $wplink-warning;
  font-size: 30rpx;
  font-weight: 700;
  line-height: 1.35;
}

.merchant-line {
  display: flex;
  align-items: center;
  gap: 12rpx;
  justify-content: space-between;
}

.verified-badge {
  flex: 0 0 auto;
  padding: 4rpx 10rpx;
  border-radius: 8rpx;
  background: $wplink-success-soft;
  color: $wplink-success;
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
</style>
