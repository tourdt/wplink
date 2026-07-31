<template>
  <button class="recent-merchant-card" @click="$emit('open', merchant)">
    <view class="merchant-card-top">
      <image
        v-if="merchant.logoUrl"
        class="merchant-logo"
        :src="merchant.logoUrl"
        mode="aspectFill"
      />
      <view v-else class="merchant-initial">{{ merchantInitial }}</view>

      <view class="merchant-heading">
        <text class="merchant-name">{{ merchant.name }}</text>
        <text class="merchant-type">{{ merchantTypeLabel }}</text>
      </view>
    </view>

    <view class="merchant-meta-row">
      <text class="new-badge">新入驻</text>
      <text class="onboarded-date">{{ onboardedLabel }}</text>
    </view>

    <view class="category-row">
      <text v-for="category in visibleCategories" :key="category" class="category-tag">{{ category }}</text>
      <text v-if="!visibleCategories.length" class="category-tag muted">主营待补充</text>
    </view>

    <text class="merchant-address">{{ merchant.addressText || '地址资料待完善' }}</text>
  </button>
</template>

<script setup>
import { computed } from 'vue'
import { formatListFreshnessDate } from '../common/date'
import { merchantTypeText } from '../common/enums'

const props = defineProps({
  merchant: {
    type: Object,
    required: true,
  },
})

defineEmits(['open'])

const merchantInitial = computed(() => String(props.merchant.name || '商').trim().slice(0, 1) || '商')
const merchantTypeLabel = computed(() => merchantTypeText[props.merchant.merchantType] || props.merchant.merchantType || '商家')
const visibleCategories = computed(() => (props.merchant.mainCategories || []).filter(Boolean).slice(0, 2))
const onboardedLabel = computed(() => `${formatListFreshnessDate(props.merchant.onboardedAt)}入驻`)
</script>

<style lang="scss" scoped>
.recent-merchant-card {
  position: relative;
  display: block;
  box-sizing: border-box;
  min-width: 0;
  min-height: 288rpx;
  margin: 0;
  padding: 22rpx;
  overflow: hidden;
  border: 1rpx solid rgba($wplink-primary, 0.14);
  border-radius: 16rpx;
  background: $wplink-card;
  color: $wplink-primary;
  text-align: left;
  line-height: 1.2;
  box-shadow: 0 10rpx 28rpx rgba(15, 23, 42, 0.05);
  transition: transform 120ms ease-out, box-shadow 120ms ease-out;
}

.recent-merchant-card:active {
  box-shadow: 0 5rpx 16rpx rgba(15, 23, 42, 0.07);
  transform: translateY(1rpx);
}

.recent-merchant-card::before {
  position: absolute;
  top: 0;
  right: 0;
  left: 0;
  height: 5rpx;
  background: linear-gradient(90deg, $wplink-primary 0 68%, $wplink-warning 68% 100%);
  content: '';
}

.recent-merchant-card::after {
  border: 0;
}

.merchant-card-top {
  display: flex;
  align-items: center;
  gap: 16rpx;
  min-width: 0;
}

.merchant-logo,
.merchant-initial {
  flex: 0 0 72rpx;
  width: 72rpx;
  height: 72rpx;
  border-radius: 12rpx;
}

.merchant-logo {
  display: block;
  background: #eef1f5;
}

.merchant-initial {
  display: grid;
  place-items: center;
  background:
    linear-gradient(145deg, rgba($wplink-primary, 0.96), rgba(38, 64, 83, 0.9)),
    $wplink-primary;
  color: #ffffff;
  font-size: 32rpx;
  font-weight: 800;
}

.merchant-heading {
  min-width: 0;
}

.merchant-name,
.merchant-type,
.merchant-address {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.merchant-name {
  color: $wplink-primary;
  font-size: 28rpx;
  font-weight: 800;
}

.merchant-type {
  margin-top: 8rpx;
  color: $wplink-muted;
  font-size: 21rpx;
}

.merchant-meta-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
  margin-top: 18rpx;
}

.new-badge {
  padding: 5rpx 11rpx;
  border-radius: 5rpx;
  background: $wplink-warning;
  color: #ffffff;
  font-size: 19rpx;
  font-weight: 800;
  letter-spacing: 1rpx;
}

.onboarded-date {
  color: $wplink-muted;
  font-family: "DIN Alternate", Arial, sans-serif;
  font-size: 20rpx;
  font-weight: 700;
}

.category-row {
  display: flex;
  gap: 7rpx;
  min-width: 0;
  margin-top: 16rpx;
  overflow: hidden;
}

.category-tag {
  max-width: 128rpx;
  padding: 5rpx 9rpx;
  overflow: hidden;
  border-radius: 6rpx;
  background: #edf3f5;
  color: #365564;
  font-size: 19rpx;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.category-tag.muted {
  color: $wplink-muted;
}

.merchant-address {
  margin-top: 16rpx;
  color: $wplink-muted;
  font-size: 20rpx;
  line-height: 1.35;
}
</style>
