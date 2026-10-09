<template>
  <view class="merchant-place-card" role="button" :aria-label="'查看' + merchant.name + '店铺主页'" @click="$emit('detail', merchant)">
    <image v-if="merchant.logoUrl && failedLogo !== merchant.logoUrl" class="shop-cover" :src="merchant.logoUrl" mode="aspectFill" @error="failedLogo = merchant.logoUrl" />
    <view v-else class="shop-cover shop-placeholder"><text>{{ primaryRoleLabel || '经营地点' }}</text><text>{{ merchant.boothNo || '实拍待补充' }}</text></view>
    <view class="shop-content">
      <view class="shop-title-row"><text class="shop-title">{{ merchant.name }}</text><text class="shop-arrow">›</text></view>
      <text v-if="businessRoleText" class="business-role">{{ businessRoleText }}</text>
      <view v-if="merchant.mainCategories.length" class="shop-tags"><text v-for="category in merchant.mainCategories.slice(0, 3)" :key="category">{{ category }}</text></view>
      <text class="coordinate-strip">{{ coordinateText }}</text>
      <text v-if="merchant.marketName && merchant.addressText" class="shop-street">{{ merchant.addressText }}</text>
    </view>
  </view>
</template>
<script setup>
import { computed, ref } from 'vue'
import { merchantDirectoryCoordinateText } from '../pages/sourcing-map/merchantDirectoryState'
import { getBusinessRoleLabel } from '../common/merchantBusinessProfile'
const props = defineProps({ merchant: { type: Object, required: true } })
defineEmits(['detail'])
const failedLogo = ref('')
const coordinateText = computed(() => merchantDirectoryCoordinateText(props.merchant))
// 目录身份只使用业务档案投影，不根据其他字段推断经营身份。
const primaryRoleLabel = computed(() => getBusinessRoleLabel(props.merchant.primaryRole))
const businessRoleText = computed(() => {
  const roles = props.merchant.businessRoles || []
  return roles.map(getBusinessRoleLabel).filter(Boolean).join(' · ') || primaryRoleLabel.value
})
</script>
<style lang="scss" scoped>
.merchant-place-card { display: flex; gap: 22rpx; padding: 24rpx; background: #fff; border: 1rpx solid #e8eaed; border-radius: 16rpx; }
.merchant-place-card:active { background: #fffaf7; }
.shop-cover { flex: 0 0 140rpx; width: 140rpx; height: 150rpx; border-radius: 10rpx; background: #f2f3f4; }
.shop-placeholder { display: flex; flex-direction: column; justify-content: center; align-items: center; gap: 10rpx; color: #727b86; padding: 10rpx; }
.shop-placeholder text:first-child { font-size: 30rpx; font-weight: 600; }
.shop-placeholder text:last-child { font-size: 18rpx; word-break: break-all; text-align: center; }
.shop-content { min-width: 0; flex: 1; display: grid; align-content: start; gap: 14rpx; }
.shop-title-row { display: flex; align-items: center; justify-content: space-between; gap: 12rpx; }
.shop-title { font-size: 31rpx; font-weight: 700; color: #1e2833; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.shop-arrow { color: #a3a9b1; font-size: 36rpx; }
.shop-tags { display: flex; gap: 10rpx; flex-wrap: wrap; }
.business-role { color: #845038; font-size: 21rpx; }
.shop-tags text { color: #845038; background: #fff5ef; padding: 5rpx 10rpx; border-radius: 5rpx; font-size: 21rpx; }
.coordinate-strip { display: block; padding-left: 12rpx; border-left: 4rpx solid #c23a00; color: #404b57; font-size: 24rpx; line-height: 1.5; word-break: break-all; }
.shop-street { color: #8a929c; font-size: 21rpx; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
