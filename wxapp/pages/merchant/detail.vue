<template>
  <view class="merchant-page">
    <view class="merchant-head">
      <view>
        <text class="merchant-name">{{ merchant.name }}</text>
        <text class="merchant-subtitle">{{ merchantSubtitle }}</text>
        <view class="tag-row">
          <text class="tag">{{ merchantTypeText[merchant.merchantType] || merchant.merchantType }}</text>
          <text class="tag verified" v-if="merchant.verificationStatus === 'verified'">已认证</text>
          <text class="tag verified" v-if="creditTags.length">平台核实</text>
        </view>
      </view>
      <button class="follow-button" @click="toggleFollow">{{ followed ? '已关注' : '关注' }}</button>
    </view>

    <view class="merchant-stats">
      <view class="stat-item">
        <text class="stat-value">{{ resourcesSummary.publishedCount || merchantResources.length || 0 }}</text>
        <text class="stat-label">当前资源</text>
      </view>
      <view class="stat-item">
        <text class="stat-value">{{ resourcesSummary.totalCount || resourcesSummary.publishedCount || merchantResources.length || 0 }}</text>
        <text class="stat-label">历史发布</text>
      </view>
      <view class="stat-item">
        <text class="stat-value">{{ resourcesSummary.dealtCount || 0 }}</text>
        <text class="stat-label">成交反馈</text>
      </view>
    </view>

    <view class="section">
      <text class="section-title">主营品类</text>
      <text class="section-content">{{ (merchant.mainCategories || []).join('、') }}</text>
    </view>

    <view class="section" v-if="creditTags.length">
      <text class="section-title">信用标签</text>
      <view class="tag-row">
        <text v-for="tag in creditTags" :key="tag.code" class="tag verified">{{ tag.label }}</text>
      </view>
    </view>

    <view class="section">
      <text class="section-title">商家简介</text>
      <text class="section-content">{{ merchant.description || '暂无简介' }}</text>
    </view>

    <view class="section" v-if="merchantImages.length">
      <text class="section-title">商家图片</text>
      <scroll-view class="image-gallery" scroll-x>
        <image v-for="url in merchantImages" :key="url" class="merchant-image" :src="url" mode="aspectFill" />
      </scroll-view>
    </view>

    <view class="section">
      <text class="section-title">发布概况</text>
      <text class="section-content">
        当前发布 {{ resourcesSummary.publishedCount || 0 }} 条，成交反馈 {{ resourcesSummary.dealtCount || 0 }} 条
      </text>
    </view>

    <view class="section benefit-section">
      <text class="section-title">权益提示</text>
      <text class="section-content">认证商家、运营推荐和置顶权益会影响资源曝光，但不会替代平台审核和买家自行确认。</text>
      <text class="section-tip">联系前建议先从资源详情进入，便于平台记录浏览、电话和微信行为。</text>
    </view>

    <view class="section">
      <view class="section-head">
        <text class="section-title">发布记录</text>
        <text class="section-link" v-if="merchantResources.length">{{ merchantResources.length }} 条</text>
      </view>
      <view v-if="merchantResources.length === 0" class="empty-text">暂无公开资源</view>
      <view v-else class="resource-list">
        <ResourceCard v-for="item in merchantResources" :key="item.id" :resource="item" @open="openResource" />
      </view>
    </view>

    <view class="contact-bar">
      <button class="contact-button" @click="copyWechat">复制微信</button>
      <button class="primary-button" @click="callPhone">拨打电话</button>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import ResourceCard from '../../components/ResourceCard.vue'
import { getMerchantFollowState, setMerchantFollow } from '../../api/favorite'
import { getMerchant } from '../../api/merchant'
import { listResources } from '../../api/resource'
import { getSession } from '../../store/session'

const merchant = ref({})
const merchantResources = ref([])
const followed = ref(false)
const creditTags = computed(() => merchant.value.creditTags || [])
const merchantImages = computed(() => merchant.value.images || [])
const resourcesSummary = computed(() => merchant.value.resourcesSummary || {})
const merchantSubtitle = computed(() => {
  const categories = (merchant.value.mainCategories || []).join('、')
  return categories || merchant.value.description || '服装产业资源商家'
})
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
  await loadFollowState(options.id)
  const resp = await listResources({ merchantId: options.id, page: 1, pageSize: 10 })
  merchantResources.value = resp.items || []
})

async function loadFollowState(merchantId) {
  if (!getSession().token) return
  try {
    const resp = await getMerchantFollowState(merchantId)
    followed.value = Boolean(resp.followed)
  } catch (err) {
    followed.value = false
  }
}

async function toggleFollow() {
  if (!merchant.value.id) return
  try {
    // 关注商家用于后续复访和提醒，当前只改变关注列表，不触发营销消息。
    const resp = await setMerchantFollow(merchant.value.id, !followed.value)
    followed.value = Boolean(resp.followed)
    uni.showToast({ title: followed.value ? '已关注商家' : '已取消关注', icon: 'none' })
  } catch (err) {
    uni.showToast({ title: err.message || '关注失败，请稍后重试', icon: 'none' })
  }
}

function openResource(resource) {
  uni.navigateTo({ url: `/pages/resource/detail?id=${resource.id}` })
}

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

.merchant-head {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 136rpx;
  gap: 16rpx;
  align-items: start;
  background:
    linear-gradient(135deg, rgba(15, 118, 110, 0.08), rgba(37, 99, 235, 0.06)),
    #ffffff;
}

.merchant-name {
  display: block;
  margin-bottom: 12rpx;
  color: #1f2933;
  font-size: 36rpx;
  font-weight: 700;
  line-height: 1.25;
  word-break: break-word;
}

.merchant-subtitle {
  display: block;
  margin-bottom: 14rpx;
  color: #697586;
  font-size: 26rpx;
  line-height: 1.5;
}

.merchant-stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12rpx;
  margin-bottom: 20rpx;
}

.stat-item {
  display: grid;
  gap: 6rpx;
  padding: 18rpx 10rpx;
  border-radius: 12rpx;
  background: #ffffff;
  text-align: center;
}

.stat-value {
  color: #1f2933;
  font-size: 34rpx;
  font-weight: 700;
}

.stat-label {
  color: #697586;
  font-size: 24rpx;
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

.follow-button {
  height: 64rpx;
  border-radius: 10rpx;
  background: #fff7e6;
  color: #b7791f;
  font-size: 24rpx;
}

.section-title {
  display: block;
  margin-bottom: 12rpx;
  color: #697586;
  font-size: 26rpx;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12rpx;
}

.section-head .section-title {
  margin-bottom: 0;
}

.section-link,
.empty-text {
  color: #697586;
  font-size: 26rpx;
}

.resource-list {
  display: grid;
  gap: 14rpx;
}

.section-content {
  color: #1f2933;
  font-size: 30rpx;
  line-height: 1.6;
  word-break: break-word;
}

.benefit-section {
  background: #fff7e6;
}

.section-tip {
  color: #7c5a22;
  font-size: 26rpx;
  line-height: 1.5;
}

.image-gallery {
  width: 100%;
  white-space: nowrap;
}

.merchant-image {
  display: inline-block;
  width: 280rpx;
  height: 180rpx;
  margin-right: 12rpx;
  border-radius: 10rpx;
  background: #e3e8ef;
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
  line-height: 1.25;
}

.primary-button {
  background: #0f766e;
  color: #ffffff;
}
</style>
