<template>
  <view class="resource-page">
    <view class="resource-card">
      <text class="title">{{ resource.title }}</text>
      <text class="meta">{{ resource.category }} · {{ resource.quantityText }}</text>
      <text class="price">{{ resource.priceText }}</text>
      <text class="desc">{{ resource.description }}</text>
    </view>

    <view class="merchant-card" @click="openMerchant">
      <MerchantBadge :merchant="resource.merchant || {}" />
    </view>

    <view class="contact-bar">
      <button @click="copyWechat">复制微信</button>
      <button class="primary-button" @click="callPhone">联系商家</button>
      <button open-type="share" @click="shareResource">分享</button>
    </view>
  </view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad, onShareAppMessage } from '@dcloudio/uni-app'
import MerchantBadge from '../../components/MerchantBadge.vue'
import { getResource, recordResourceContact, recordResourceDetailView } from '../../api/resource'

const resource = ref({})

onLoad(async (options) => {
  if (!options.id) return
  resource.value = await getResource(options.id)
  await recordResourceDetailView(options.id)
})

async function recordContact(action) {
  if (!resource.value.id) return
  await recordResourceContact(resource.value.id, action)
}

async function openMerchant() {
  const merchantId = resource.value.merchant?.id
  if (!merchantId) return
  await recordContact('merchant_home')
  uni.navigateTo({ url: `/pages/merchant/detail?id=${merchantId}` })
}

async function callPhone() {
  await recordContact('phone')
  const phone = resource.value.contact?.phoneMasked || ''
  if (phone && !phone.includes('*')) {
    uni.makePhoneCall({ phoneNumber: phone })
    return
  }
  uni.showToast({ title: '已记录联系，完整电话由平台保护', icon: 'none' })
}

async function copyWechat() {
  await recordContact('wechat')
  const wechat = resource.value.contact?.wechatMasked || ''
  if (wechat && !wechat.includes('*')) {
    uni.setClipboardData({ data: wechat })
    return
  }
  uni.showToast({ title: '已记录联系，完整微信由平台保护', icon: 'none' })
}

async function shareResource() {
  await recordContact('share')
}

onShareAppMessage(() => ({
  title: resource.value.title || '服链通资源',
  path: resource.value.id ? `/pages/resource/detail?id=${resource.value.id}` : '/pages/home/index',
}))
</script>

<style scoped>
.resource-page {
  min-height: 100vh;
  padding: 24rpx;
  background: #f4f6f8;
}

.resource-card,
.merchant-card {
  display: grid;
  gap: 12rpx;
  margin-bottom: 20rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: #ffffff;
}

.title {
  color: #1f2933;
  font-size: 38rpx;
  font-weight: 700;
}

.meta,
.desc,
.merchant-status {
  color: #697586;
  font-size: 28rpx;
}

.price {
  color: #c2410c;
  font-size: 32rpx;
  font-weight: 700;
}

.contact-bar {
  position: fixed;
  right: 24rpx;
  bottom: 24rpx;
  left: 24rpx;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16rpx;
}

.primary-button {
  background: #0f766e;
  color: #ffffff;
}
</style>
