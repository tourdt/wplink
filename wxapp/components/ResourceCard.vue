<template>
  <view class="resource-card" @click="$emit('open', resource)">
    <image v-if="coverUrl" class="resource-thumb" :src="coverUrl" mode="aspectFill" />
    <view class="card-main">
      <view class="tag-row">
        <text v-if="isVerifiedMerchant" class="tag verified">已认证</text>
        <text v-if="hasCreditTags" class="tag verified">平台核实</text>
        <text v-if="resource.typeCode" class="tag">{{ resource.typeCode }}</text>
      </view>
      <text class="resource-title">{{ resource.title }}</text>
      <text class="resource-meta">{{ resource.category || '品类待沟通' }} · {{ resource.quantityText || '数量待沟通' }}</text>
      <view class="card-foot">
        <text class="resource-price">{{ resource.priceText || '价格面议' }}</text>
        <text class="resource-action">查看详情</text>
      </view>
      <view class="merchant-row">
        <text class="merchant-name">{{ merchantName }}</text>
        <text class="refresh-time">{{ formatRefreshedAt(resource.refreshedAt) }}</text>
      </view>
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
})

defineEmits(['open'])

const hasCreditTags = computed(() => Array.isArray(props.resource.creditTags) && props.resource.creditTags.length > 0)
const coverUrl = computed(() => {
  const images = props.resource.images || []
  return props.resource.coverUrl || images[0] || ''
})
const isVerifiedMerchant = computed(() => (props.resource.merchant || {}).verificationStatus === 'verified')
const merchantName = computed(() => (props.resource.merchant || {}).name || '商家待确认')

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
}

.resource-thumb {
  flex: 0 0 168rpx;
  width: 168rpx;
  min-height: 168rpx;
  border-radius: 10rpx;
  background: #edf2f7;
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
}

.resource-meta {
  color: #697586;
  font-size: 28rpx;
  line-height: 1.45;
}

.resource-price {
  flex: 1;
  color: #c2410c;
  font-size: 30rpx;
  font-weight: 700;
}

.resource-action {
  color: #0f766e;
  font-size: 26rpx;
  font-weight: 700;
}

.merchant-name,
.refresh-time {
  color: #697586;
  font-size: 24rpx;
}
</style>
