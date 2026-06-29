<template>
  <view :class="['resource-card', variantClass]" @click="$emit('open', resource)">
    <image v-if="coverUrl" class="resource-thumb" :src="coverUrl" mode="aspectFill" />
    <view v-else class="resource-thumb placeholder-thumb">
      <text>{{ typeLabel }}</text>
    </view>
    <view class="card-main">
      <view class="tag-row">
        <text v-if="isVerifiedMerchant" class="tag verified">已认证</text>
        <text v-if="hasCreditTags" class="tag verified">平台核实</text>
        <text v-if="resource.typeCode" class="tag">{{ resource.typeCode }}</text>
      </view>
      <text class="resource-title">{{ resource.title || '资源标题待完善' }}</text>
      <text class="resource-meta">{{ resource.category || '品类待沟通' }} · {{ resource.quantityText || '数量待沟通' }}</text>
      <view class="card-foot">
        <text class="resource-price">{{ resource.priceText || '价格面议' }}</text>
        <text class="resource-action">查看详情</text>
      </view>
      <view class="merchant-row">
        <text class="merchant-name">{{ merchantName }}</text>
        <text class="refresh-time">{{ formatRefreshedAt(resource.refreshedAt) }}</text>
      </view>
      <text class="decision-tip">{{ decisionTip }}</text>
    </view>
  </view>
</template>

<script setup>
import { computed } from 'vue'

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

const hasCreditTags = computed(() => Array.isArray(props.resource.creditTags) && props.resource.creditTags.length > 0)
const variantClass = computed(() => (props.variant === 'home' ? 'resource-card-home' : ''))
const coverUrl = computed(() => {
  const images = props.resource.images || []
  return props.resource.coverUrl || images[0] || ''
})
const isVerifiedMerchant = computed(() => (props.resource.merchant || {}).verificationStatus === 'verified')
const merchantName = computed(() => (props.resource.merchant || {}).name || '商家待确认')
const typeLabel = computed(() => props.resource.typeCode || props.resource.category || '资源')
const decisionTip = computed(() => {
  if (hasCreditTags.value) return '平台已补充核实信息，联系前仍建议确认实物、价格和交付时间。'
  if (isVerifiedMerchant.value) return '认证商家发布，建议进入详情查看规格和联系方式。'
  return '建议进入详情确认数量、价格、看样方式和刷新时间。'
})

function formatRefreshedAt(value) {
  if (!value) return '近期更新'
  if (value.includes('T')) return value.slice(0, 10)
  return value
}
</script>

<style scoped>
.resource-card {
  display: flex;
  align-items: stretch;
  gap: 12rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: #ffffff;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.04);
  overflow: hidden;
}

.resource-thumb {
  flex: 0 0 168rpx;
  width: 168rpx;
  min-height: 168rpx;
  border-radius: 10rpx;
  background: #edf2f7;
}

.placeholder-thumb {
  display: flex;
  align-items: flex-end;
  justify-content: flex-start;
  padding: 14rpx;
  background:
    linear-gradient(140deg, rgba(255, 255, 255, 0.22), transparent 38%),
    repeating-linear-gradient(45deg, rgba(255, 255, 255, 0.18) 0 12rpx, transparent 12rpx 24rpx),
    #5c8a72;
  color: #ffffff;
  font-size: 24rpx;
  font-weight: 700;
}

.card-main {
  display: grid;
  flex: 1;
  gap: 12rpx;
  min-width: 0;
}

.tag-row,
.merchant-row,
.card-foot {
  display: flex;
  align-items: center;
  gap: 12rpx;
  justify-content: space-between;
}

.tag-row,
.merchant-row {
  flex-wrap: wrap;
}

.tag {
  padding: 6rpx 12rpx;
  border-radius: 8rpx;
  background: #edf2f7;
  color: #4a5568;
  font-size: 22rpx;
}

.tag.verified {
  background: #e6f4f1;
  color: #0f766e;
}

.resource-title {
  color: #1f2933;
  font-size: 32rpx;
  font-weight: 700;
  line-height: 1.35;
  word-break: break-word;
}

.resource-meta {
  color: #697586;
  font-size: 28rpx;
  line-height: 1.45;
  word-break: break-word;
}

.resource-price {
  flex: 1;
  color: #c2410c;
  font-size: 30rpx;
  font-weight: 700;
}

.resource-action {
  flex: 0 0 auto;
  color: #0f766e;
  font-size: 26rpx;
  font-weight: 700;
}

.merchant-name,
.refresh-time {
  color: #697586;
  font-size: 24rpx;
}

.decision-tip {
  padding: 12rpx 14rpx;
  border-radius: 10rpx;
  background: #f8fafc;
  color: #4b5565;
  font-size: 23rpx;
  line-height: 1.45;
  word-break: break-word;
}

.resource-card-home {
  gap: 24rpx;
  padding: 24rpx;
  border-radius: 16rpx;
  box-shadow: 0 16rpx 48rpx rgba(15, 23, 42, 0.06);
}

.resource-card-home .resource-thumb {
  flex-basis: 160rpx;
  width: 160rpx;
  min-height: 160rpx;
  border-radius: 16rpx;
}

.resource-card-home .placeholder-thumb {
  padding: 18rpx;
  font-size: 24rpx;
}

.resource-card-home .card-main {
  gap: 8rpx;
}

.resource-card-home .tag-row {
  gap: 10rpx;
}

.resource-card-home .tag {
  min-height: 40rpx;
  padding: 0 14rpx;
  border-radius: 10rpx;
  font-size: 22rpx;
  line-height: 40rpx;
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

.resource-card-home .resource-action {
  font-size: 26rpx;
  white-space: nowrap;
}

.resource-card-home .merchant-name,
.resource-card-home .refresh-time {
  font-size: 24rpx;
  line-height: 1.35;
}

.resource-card-home .merchant-row,
.resource-card-home .decision-tip {
  display: none;
}
</style>
