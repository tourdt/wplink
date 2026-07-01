<template>
  <view class="my-page">
    <view class="account-card" @click="openAccountCard">
      <view class="account-shell">
        <view class="account-side">
          <image v-if="merchantLogo" class="avatar avatar-image" :src="merchantLogo" mode="aspectFill" />
          <view v-else class="avatar" :class="{ 'avatar-guest': !isLoggedIn }">{{ avatarText }}</view>
        </view>
        <view class="account-main">
          <view class="account-title-row">
            <view class="account-title-main">
              <text class="account-name">{{ accountName }}</text>
              <text v-if="isLoggedIn" class="verification-status" :class="verificationStatusClass">{{ verificationStatusText }}</text>
            </view>
            <text class="entry-arrow"></text>
          </view>
          <text class="account-desc">{{ accountDesc }}</text>
        </view>
      </view>
      <button v-if="!isLoggedIn" class="login-button" @click.stop="openLogin">微信登录</button>
    </view>

    <view v-if="merchantEffectVisible" class="merchant-effect-card section-card">
      <view class="section-head effect-head">
        <text class="section-title">商家本周效果</text>
        <text class="section-subtitle">近 7 天</text>
      </view>
      <view class="merchant-effect-grid">
        <view v-for="item in merchantEffectItems" :key="item.label" class="merchant-effect-item">
          <text class="merchant-effect-value">{{ item.value }}</text>
          <text class="merchant-effect-label">{{ item.label }}</text>
        </view>
      </view>
    </view>

    <view class="common-service-section section-card">
      <view class="action-list">
        <view class="action-item" @click="openMyResources">
          <view class="action-main">
            <text class="action-title">我的发布</text>
            <text class="action-meta">状态、数据、推广</text>
          </view>
          <text class="entry-arrow"></text>
        </view>
        <view class="action-item" @click="openMerchantHome">
          <view class="action-main">
            <text class="action-title">商家主页</text>
            <text class="action-meta">查看自己的公开页</text>
          </view>
          <text class="entry-arrow"></text>
        </view>
        <view class="action-item" @click="openMerchantVerification">
          <view class="action-main">
            <text class="action-title">商家认证</text>
            <text class="action-meta">{{ verificationActionMeta }}</text>
          </view>
          <text class="entry-arrow"></text>
        </view>
        <view class="action-item" @click="openFavorites">
          <view class="action-main">
            <text class="action-title">收藏关注</text>
            <text class="action-meta">资源、商家、搜索</text>
          </view>
          <text class="entry-arrow"></text>
        </view>
        <view class="action-item" @click="openMessages">
          <view class="action-main">
            <text class="action-title">消息</text>
            <text class="action-meta">审核和联系提醒</text>
          </view>
          <text class="entry-arrow"></text>
        </view>
        <button class="action-item customer-service-button" open-type="contact">
          <view class="action-main">
            <text class="action-title">联系客服</text>
            <text class="action-meta">平台问题和使用咨询</text>
          </view>
          <text class="entry-arrow"></text>
        </button>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { buildLoginUrl, requireLogin } from '../../common/auth'
import { getSession } from '../../store/session'
import { getMerchant } from '../../api/merchant'
import { getMerchantMetricsSummary } from '../../api/metrics'
import { getLatestVerification } from '../../api/verification'

const token = ref('')
const merchantId = ref('')
const merchantProfile = ref({})
const latestVerification = ref({ status: 'none' })
const merchantMetricsSummary = ref(null)

const isLoggedIn = computed(() => Boolean(token.value))
const merchantLogo = computed(() => merchantProfile.value.logoUrl || '')
const merchantName = computed(() => merchantProfile.value.name || '')
const avatarText = computed(() => merchantName.value.slice(0, 1) || (isLoggedIn.value ? '我' : '游'))
const accountName = computed(() => merchantName.value || (isLoggedIn.value ? '我的账号' : '未登录'))
const accountDesc = computed(() => (merchantName.value ? '已录入商户资料，可管理发布和认证' : isLoggedIn.value ? '已登录，可管理收藏和消息' : '登录后管理收藏和发布记录'))
const verificationStatus = computed(() => {
  if (!merchantId.value) return 'unconfigured'
  return latestVerification.value.status || 'none'
})
const verificationStatusText = computed(() => {
  const statusText = {
    unconfigured: '待完善',
    none: '未认证',
    pending: '审核中',
    verified: '已认证',
    rejected: '未通过',
    revoked: '已撤销',
    expired: '已过期',
  }
  return statusText[verificationStatus.value] || '未认证'
})
const verificationStatusClass = computed(() => `status-${verificationStatus.value}`)
const verificationActionMeta = computed(() => {
  const actionMeta = {
    unconfigured: '先完善商家资料',
    none: '提交主体资料',
    pending: '查看审核进度',
    verified: '查看认证状态',
    rejected: '修改后重提',
    revoked: '可重新提交',
    expired: '重新认证',
  }
  return actionMeta[verificationStatus.value] || actionMeta.none
})
const merchantEffectVisible = computed(() => Boolean(token.value && merchantId.value && merchantMetricsSummary.value))
const merchantEffectItems = computed(() => {
  const last7Days = merchantMetricsSummary.value?.last7Days || {}
  return [
    { label: '曝光', value: last7Days.exposureCount || 0 },
    { label: '浏览', value: last7Days.detailViewCount || 0 },
    { label: '联系', value: last7Days.contactClickCount || 0 },
  ]
})

onLoad(() => {
  syncSession()
})

onShow(() => {
  syncSession()
})

async function syncSession() {
  const session = getSession()
  token.value = session.token
  merchantId.value = session.merchantId
  await Promise.all([loadMerchantProfile(), loadVerificationStatus(), loadMerchantMetricsSummary()])
}

function openLogin() {
  uni.navigateTo({ url: buildLoginUrl('/pages/my/index') })
}

async function loadMerchantProfile() {
  if (!token.value || !merchantId.value) {
    merchantProfile.value = {}
    return
  }
  try {
    merchantProfile.value = await getMerchant(merchantId.value, { suppressErrorToast: true })
  } catch (err) {
    // 账号卡展示商户身份即可；详情加载失败时回落到普通账号，不影响我的页入口。
    merchantProfile.value = {}
  }
}

async function loadVerificationStatus() {
  if (!token.value || !merchantId.value) {
    latestVerification.value = { status: 'none' }
    return
  }
  try {
    latestVerification.value = await getLatestVerification(merchantId.value)
  } catch (err) {
    latestVerification.value = { status: 'none' }
  }
}

async function loadMerchantMetricsSummary() {
  if (!token.value || !merchantId.value) {
    merchantMetricsSummary.value = null
    return
  }
  try {
    merchantMetricsSummary.value = await getMerchantMetricsSummary(merchantId.value, { suppressErrorToast: true })
  } catch (err) {
    merchantMetricsSummary.value = null
  }
}

function openAccountCard() {
  if (!isLoggedIn.value) {
    openLogin()
    return
  }
  openMerchantHome()
}

function openFavorites() {
  if (!requireLogin()) return
  uni.navigateTo({ url: '/pages/favorites/index' })
}

function openMessages() {
  if (!requireLogin()) return
  uni.switchTab({ url: '/pages/messages/index' })
}

function openMyResources() {
  if (!requireLogin()) return
  if (!merchantId.value) {
    uni.showToast({ title: '请先完善商家资料', icon: 'none' })
    uni.navigateTo({ url: '/pages/merchant/profile' })
    return
  }
  uni.navigateTo({ url: `/pages/my-resources/index?merchantId=${merchantId.value}` })
}

function openMerchantHome() {
  if (!requireLogin()) return
  if (!merchantId.value) {
    uni.navigateTo({ url: '/pages/merchant/profile' })
    return
  }
  uni.navigateTo({ url: `/pages/merchant/detail?id=${merchantId.value}` })
}

function openMerchantVerification() {
  if (!requireLogin()) return
  if (!merchantId.value) {
    uni.showToast({ title: '请先完善商家资料', icon: 'none' })
    uni.navigateTo({ url: '/pages/merchant/profile' })
    return
  }
  uni.navigateTo({ url: `/pages/verification/index?merchantId=${merchantId.value}` })
}
</script>

<style lang="scss" scoped>
.my-page {
  min-height: 100vh;
  padding: 24rpx 24rpx 40rpx;
  background: $wplink-bg;
}

.account-card,
.section-card {
  display: grid;
  gap: 18rpx;
  margin-bottom: 20rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.04);
}

.account-card {
  gap: 20rpx;
  padding: 32rpx;
  box-shadow: 0 12rpx 32rpx rgba(15, 23, 42, 0.06);
}

.account-shell {
  display: grid;
  grid-template-columns: 104rpx minmax(0, 1fr);
  gap: 24rpx;
  align-items: center;
  min-width: 0;
}

.account-side {
  display: flex;
  align-items: center;
  justify-content: center;
}

.avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 104rpx;
  height: 104rpx;
  border-radius: 14rpx;
  background: $wplink-primary;
  color: $wplink-card;
  font-size: 40rpx;
  font-weight: 700;
}

.avatar-guest {
  background: $wplink-muted;
}

.avatar-image {
  display: block;
  background: $wplink-primary-soft;
}

.account-main {
  display: grid;
  gap: 8rpx;
  min-width: 0;
}

.account-title-row,
.action-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18rpx;
  min-width: 0;
}

.account-title-main {
  display: flex;
  flex: 1 1 auto;
  flex-wrap: wrap;
  align-items: center;
  gap: 12rpx;
  min-width: 0;
}

.account-name {
  flex: 0 1 auto;
  color: $wplink-primary;
  font-size: 38rpx;
  font-weight: 700;
  line-height: 1.25;
}

.account-desc,
.action-meta {
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.5;
}

.login-button {
  width: 100%;
  height: 84rpx;
  border-radius: 12rpx;
  background: $wplink-primary;
  color: $wplink-card;
  font-size: 28rpx;
  font-weight: 700;
}

.verification-status {
  display: inline-flex;
  align-items: center;
  gap: 6rpx;
  flex: 0 0 auto;
  min-height: 32rpx;
  padding: 0 12rpx;
  border-radius: 999rpx;
  background: $wplink-primary-soft;
  color: $wplink-primary;
  font-size: 22rpx;
  font-weight: 700;
  line-height: 1.3;
}

.verification-status::before {
  flex: 0 0 auto;
  width: 8rpx;
  height: 8rpx;
  border-radius: 999rpx;
  background: currentColor;
  content: '';
}

.status-verified {
  background: rgba(22, 163, 106, 0.12);
  color: $wplink-success;
}

.status-pending {
  background: $wplink-warning-soft;
  color: $wplink-warning;
}

.status-rejected,
.status-revoked,
.status-expired {
  background: rgba(194, 58, 0, 0.1);
  color: $wplink-warning;
}

.section-head {
  display: grid;
  gap: 6rpx;
  min-width: 0;
}

.effect-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
}

.section-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
  line-height: 1.3;
}

.section-subtitle {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.4;
}

.merchant-effect-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12rpx;
}

.merchant-effect-item {
  display: grid;
  gap: 6rpx;
  min-width: 0;
  padding: 18rpx 10rpx;
  border-radius: 10rpx;
  background: #f8fafc;
  text-align: center;
}

.merchant-effect-value {
  color: $wplink-primary;
  font-size: 34rpx;
  font-weight: 700;
  line-height: 1.2;
}

.merchant-effect-label {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.4;
}

.action-list {
  display: grid;
}

.action-item {
  min-height: 100rpx;
  padding: 20rpx 0;
  border-bottom: 1rpx solid $wplink-line;
  color: $wplink-primary;
}

.action-item:last-child {
  border-bottom: 0;
}

.customer-service-button {
  width: 100%;
  margin: 0;
  border-radius: 0;
  background: transparent;
  line-height: normal;
  text-align: left;
}

.customer-service-button::after {
  border: 0;
}

.action-main {
  display: grid;
  gap: 8rpx;
  min-width: 0;
}

.action-title {
  color: $wplink-primary;
  font-size: 30rpx;
  font-weight: 700;
  line-height: 1.3;
}

.entry-arrow {
  position: relative;
  flex: 0 0 18rpx;
  width: 18rpx;
  height: 18rpx;
}

.entry-arrow::after {
  position: absolute;
  top: 2rpx;
  right: 2rpx;
  width: 12rpx;
  height: 12rpx;
  border-top: 3rpx solid #aab0ba;
  border-right: 3rpx solid #aab0ba;
  content: '';
  transform: rotate(45deg);
}
</style>
