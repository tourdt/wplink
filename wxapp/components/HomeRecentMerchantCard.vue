<template>
  <MerchantListItem
    :title="merchant.name"
    :subtitle="merchantTypeLabel"
    :tags="visibleCategories"
    @activate="$emit('open', merchant)"
  >
    <template #leading>
      <image
        v-if="merchant.logoUrl"
        class="merchant-logo"
        :src="merchant.logoUrl"
        mode="aspectFill"
      />
      <view v-else class="merchant-initial">{{ merchantInitial }}</view>
    </template>

    <template #badge>
      <text class="new-badge">新入驻</text>
    </template>

    <template #meta>
      <text class="onboarded-date">{{ onboardedLabel }}</text>
    </template>
  </MerchantListItem>
</template>

<script setup>
import { computed } from 'vue'
import MerchantListItem from './MerchantListItem.vue'
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
const merchantTypeLabel = computed(() => (
  merchantTypeText[props.merchant.merchantType] || props.merchant.merchantType || '商家'
))
const visibleCategories = computed(() => {
  const categories = (props.merchant.mainCategories || []).filter(Boolean).slice(0, 2)
  return categories.length ? categories : ['主营待补充']
})
const onboardedLabel = computed(() => `${formatListFreshnessDate(props.merchant.onboardedAt)}入驻`)
</script>

<style lang="scss" scoped>
.merchant-logo,
.merchant-initial {
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

.new-badge {
  flex: none;
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
</style>
