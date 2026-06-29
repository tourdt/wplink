<template>
  <view class="my-page">
    <view class="account-card" @click="openAccountCard">
      <view class="account-shell">
        <view class="account-side">
          <view class="avatar" :class="{ 'avatar-guest': !isLoggedIn }">{{ avatarText }}</view>
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

    <view class="quick-section">
      <view class="section-head quick-section-head">
        <text class="section-title">发布和推广</text>
      </view>
      <view class="quick-entry-grid">
        <button class="quick-entry primary" @click="openPublish">
          <view class="quick-icon publish-icon">
            <text></text>
          </view>
          <view class="quick-copy">
            <text class="quick-title">发布资源</text>
            <text class="quick-desc">库存、货源、服务</text>
          </view>
        </button>
        <button class="quick-entry" @click="openMyResources">
          <view class="quick-icon resource-icon">
            <text></text>
            <text></text>
          </view>
          <view class="quick-copy">
            <text class="quick-title">我的发布</text>
            <text class="quick-desc">状态、数据、推广</text>
          </view>
        </button>
      </view>
    </view>

    <view class="common-service-section section-card">
      <view class="section-head">
        <text class="section-title">常用服务</text>
      </view>
      <view class="action-list">
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
        <view class="action-item" @click="openMyDemands">
          <view class="action-main">
            <text class="action-title">我的需求</text>
            <text class="action-meta">需求和进展</text>
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
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { buildLoginUrl, requireLogin } from '../../common/auth'
import { getSession } from '../../store/session'
import { getLatestVerification } from '../../api/verification'

const token = ref('')
const merchantId = ref('')
const latestVerification = ref({ status: 'none' })

const isLoggedIn = computed(() => Boolean(token.value))
const avatarText = computed(() => (isLoggedIn.value ? '我' : '游'))
const accountName = computed(() => (isLoggedIn.value ? '我的账号' : '未登录'))
const accountDesc = computed(() => (isLoggedIn.value ? '已登录，可管理需求、收藏和消息' : '登录后管理需求、收藏和发布记录'))
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
  }
  return actionMeta[verificationStatus.value] || actionMeta.none
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
  await loadVerificationStatus()
}

function openLogin() {
  uni.navigateTo({ url: buildLoginUrl('/pages/my/index') })
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

function openAccountCard() {
  if (!isLoggedIn.value) {
    openLogin()
    return
  }
  openMerchantHome()
}

function openMyDemands() {
  if (!requireLogin()) return
  uni.navigateTo({ url: '/pages/my-demands/index' })
}

function openFavorites() {
  if (!requireLogin()) return
  uni.navigateTo({ url: '/pages/favorites/index' })
}

function openMessages() {
  if (!requireLogin()) return
  uni.switchTab({ url: '/pages/messages/index' })
}

function openPublish() {
  if (!requireLogin()) return
  uni.switchTab({ url: '/pages/publish/index' })
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
.status-revoked {
  background: rgba(194, 58, 0, 0.1);
  color: $wplink-warning;
}

.quick-section {
  display: grid;
  gap: 14rpx;
  margin-bottom: 20rpx;
}

.quick-section-head {
  padding: 0 4rpx;
}

.quick-entry-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16rpx;
}

.quick-entry {
  display: grid;
  grid-template-columns: 54rpx minmax(0, 1fr);
  gap: 16rpx;
  align-items: center;
  min-height: 128rpx;
  padding: 20rpx;
  border-radius: 12rpx;
  background: $wplink-card;
  color: $wplink-primary;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.04);
  text-align: left;
}

.quick-entry.primary {
  background: $wplink-card;
  color: $wplink-primary;
}

.quick-icon {
  position: relative;
  width: 54rpx;
  height: 54rpx;
  border-radius: 12rpx;
  background: $wplink-primary-soft;
  color: $wplink-primary;
}

.quick-entry.primary .quick-icon {
  background: $wplink-warning-soft;
  color: $wplink-warning;
}

.publish-icon::before,
.publish-icon::after {
  position: absolute;
  top: 25rpx;
  left: 15rpx;
  width: 24rpx;
  height: 4rpx;
  border-radius: 999rpx;
  background: currentColor;
  content: '';
}

.publish-icon::after {
  transform: rotate(90deg);
}

.resource-icon text:first-child {
  position: absolute;
  top: 13rpx;
  left: 13rpx;
  width: 28rpx;
  height: 8rpx;
  border-radius: 999rpx;
  background: currentColor;
}

.resource-icon text:last-child {
  position: absolute;
  right: 13rpx;
  bottom: 13rpx;
  left: 13rpx;
  height: 22rpx;
  border: 4rpx solid currentColor;
  border-radius: 6rpx;
}

.quick-copy {
  display: grid;
  gap: 6rpx;
  min-width: 0;
}

.quick-title {
  font-size: 30rpx;
  font-weight: 700;
  line-height: 1.25;
}

.quick-desc {
  color: $wplink-muted;
  font-size: 22rpx;
  line-height: 1.35;
  word-break: break-word;
}

.quick-entry.primary .quick-desc {
  color: $wplink-muted;
}

.section-head {
  display: grid;
  gap: 6rpx;
  min-width: 0;
}

.section-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
  line-height: 1.3;
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
