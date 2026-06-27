<template>
  <view class="merchant-page">
    <view class="merchant-head">
      <view>
        <text class="merchant-name">{{ merchant.name }}</text>
        <view class="tag-row">
          <text class="tag">{{ merchantTypeText[merchant.merchantType] || merchant.merchantType }}</text>
          <text class="tag verified" v-if="merchant.verificationStatus === 'verified'">已认证</text>
        </view>
      </view>
    </view>

    <view class="section">
      <text class="section-title">主营品类</text>
      <text class="section-content">{{ (merchant.mainCategories || []).join('、') }}</text>
    </view>

    <view class="section" v-if="merchant.creditTags?.length">
      <text class="section-title">信用标签</text>
      <view class="tag-row">
        <text v-for="tag in merchant.creditTags" :key="tag.code" class="tag verified">{{ tag.label }}</text>
      </view>
    </view>

    <view class="section">
      <text class="section-title">商家简介</text>
      <text class="section-content">{{ merchant.description || '暂无简介' }}</text>
    </view>

    <view class="section">
      <text class="section-title">发布概况</text>
      <text class="section-content">
        当前发布 {{ merchant.resourcesSummary?.publishedCount || 0 }} 条，成交反馈 {{ merchant.resourcesSummary?.dealtCount || 0 }} 条
      </text>
    </view>

    <view class="contact-bar">
      <button class="contact-button" @click="copyWechat">复制微信</button>
      <button class="primary-button" @click="callPhone">拨打电话</button>
    </view>
  </view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getMerchant } from '../../api/merchant'

const merchant = ref({})
const merchantTypeText = {
  factory: '工厂',
  stall: '档口',
  stockist: '库存商',
  service_provider: '服务商',
  buyer: '采购商',
}

onLoad(async (options) => {
  if (!options.id) return
  merchant.value = await getMerchant(options.id)
})

function callPhone() {
  uni.showToast({ title: '请在资源详情页查看完整电话', icon: 'none' })
}

function copyWechat() {
  uni.showToast({ title: '请在资源详情页查看完整微信', icon: 'none' })
}
</script>

<style scoped>
.merchant-page {
  min-height: 100vh;
  padding: 24rpx;
  background: #f4f6f8;
}

.merchant-head,
.section {
  margin-bottom: 20rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: #ffffff;
}

.merchant-name {
  display: block;
  margin-bottom: 12rpx;
  color: #1f2933;
  font-size: 36rpx;
  font-weight: 700;
}

.tag-row {
  display: flex;
  flex-wrap: wrap;
  gap: 12rpx;
}

.tag {
  padding: 6rpx 12rpx;
  border-radius: 8rpx;
  background: #edf2f7;
  color: #4a5568;
  font-size: 24rpx;
}

.tag.verified {
  background: #e6f4f1;
  color: #0f766e;
}

.section-title {
  display: block;
  margin-bottom: 12rpx;
  color: #697586;
  font-size: 26rpx;
}

.section-content {
  color: #1f2933;
  font-size: 30rpx;
  line-height: 1.6;
}

.contact-bar {
  position: fixed;
  right: 24rpx;
  bottom: 24rpx;
  left: 24rpx;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16rpx;
}

.contact-button,
.primary-button {
  height: 88rpx;
  border-radius: 12rpx;
  font-size: 30rpx;
}

.primary-button {
  background: #0f766e;
  color: #ffffff;
}
</style>
