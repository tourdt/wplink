<template>
  <view class="resource-page">
    <view class="resource-card">
      <view class="tag-row">
        <text v-if="isVerifiedMerchant" class="tag verified">认证商家</text>
        <text v-if="resource.status" class="tag">{{ statusText[resource.status] || resource.status }}</text>
      </view>
      <text class="title">{{ resource.title }}</text>
      <text class="meta">{{ resource.category || '品类待沟通' }} · {{ resource.quantityText || '数量待沟通' }}</text>
      <text class="price">{{ resource.priceText || '价格面议' }}</text>
      <text class="desc">{{ resource.description || '商家暂未填写详细描述，建议联系前确认数量、尺码、看样方式和交付时间。' }}</text>
    </view>

    <view class="trust-card">
      <text class="section-title">联系提示</text>
      <text class="section-content">平台已记录联系行为，电话和微信可能因隐私保护展示为脱敏信息。联系时建议确认实物、价格和交付方式。</text>
    </view>

    <view class="merchant-card" @click="openMerchant">
      <MerchantBadge :merchant="resource.merchant || {}" />
      <text class="merchant-hint">查看商家认证、发布记录和信用信息</text>
    </view>

    <view v-if="relatedResources.length" class="related-section">
      <view class="section-head">
        <text class="section-title">同类推荐</text>
        <text class="section-link" @click="openSearch">查看更多</text>
      </view>
      <ResourceCard v-for="item in relatedResources" :key="item.id" :resource="item" @open="openRelatedResource" />
    </view>

    <view class="contact-bar">
      <button @click="copyWechat">复制微信</button>
      <button class="primary-button" @click="callPhone">联系商家</button>
      <button open-type="share" @click="shareResource">分享</button>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShareAppMessage } from '@dcloudio/uni-app'
import MerchantBadge from '../../components/MerchantBadge.vue'
import ResourceCard from '../../components/ResourceCard.vue'
import { getResource, listResources, recordResourceContact, recordResourceDetailView } from '../../api/resource'

const resource = ref({})
const relatedResources = ref([])
const SEARCH_KEY = 'wplink_pending_search_keyword'
const statusText = {
  published: '已发布',
  pending: '待审核',
  expired: '已过期',
  dealt: '已成交',
}
const isVerifiedMerchant = computed(() => (resource.value.merchant || {}).verificationStatus === 'verified')

onLoad(async (options) => {
  if (!options.id) return
  resource.value = await getResource(options.id)
  await recordResourceDetailView(options.id)
  await loadRelatedResources()
})

async function recordContact(action) {
  if (!resource.value.id) return
  await recordResourceContact(resource.value.id, action)
}

async function openMerchant() {
  const merchantId = (resource.value.merchant || {}).id
  if (!merchantId) return
  await recordContact('merchant_home')
  uni.navigateTo({ url: `/pages/merchant/detail?id=${merchantId}` })
}

async function loadRelatedResources() {
  if (!resource.value.typeCode) return
  const resp = await listResources({ typeCode: resource.value.typeCode, page: 1, pageSize: 4 })
  relatedResources.value = (resp.items || []).filter((item) => item.id !== resource.value.id).slice(0, 3)
}

function openRelatedResource(item) {
  uni.navigateTo({ url: `/pages/resource/detail?id=${item.id}` })
}

function openSearch() {
  if (resource.value.category) {
    uni.setStorageSync(SEARCH_KEY, resource.value.category)
  } else {
    uni.removeStorageSync(SEARCH_KEY)
  }
  uni.switchTab({ url: '/pages/search/index' })
}

async function callPhone() {
  await recordContact('phone')
  const phone = (resource.value.contact || {}).phoneMasked || ''
  if (phone && !phone.includes('*')) {
    uni.makePhoneCall({ phoneNumber: phone })
    return
  }
  uni.showToast({ title: '已记录联系，完整电话由平台保护', icon: 'none' })
}

async function copyWechat() {
  await recordContact('wechat')
  const wechat = (resource.value.contact || {}).wechatMasked || ''
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
  padding: 24rpx 24rpx 150rpx;
  background: #f4f6f8;
}

.resource-card,
.merchant-card,
.trust-card,
.related-section {
  display: grid;
  gap: 12rpx;
  margin-bottom: 20rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: #ffffff;
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

.title {
  color: #1f2933;
  font-size: 38rpx;
  font-weight: 700;
  line-height: 1.35;
}

.meta,
.desc,
.merchant-status,
.section-content,
.merchant-hint,
.section-link {
  color: #697586;
  font-size: 28rpx;
  line-height: 1.55;
}

.price {
  color: #c2410c;
  font-size: 32rpx;
  font-weight: 700;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
}

.section-title {
  color: #1f2933;
  font-size: 32rpx;
  font-weight: 700;
}

.section-link {
  color: #0f766e;
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

.contact-bar button {
  height: 88rpx;
  border-radius: 12rpx;
  background: #ffffff;
  color: #364152;
  font-size: 28rpx;
}

.primary-button {
  background: #0f766e;
  color: #ffffff;
}
</style>
