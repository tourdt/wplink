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
              <text v-if="isLoggedIn" class="verification-status" :class="accountStatusClass">{{ accountStatusText }}</text>
            </view>
            <text class="entry-arrow"></text>
          </view>
          <view v-if="merchantNoVisible" class="account-number-row" @click.stop="copyMerchantNo">
            <text class="account-number-label">编号：</text>
            <text class="account-number-value">{{ merchantNo }}</text>
            <text class="account-number-copy">复制</text>
          </view>
        </view>
      </view>
      <button v-if="!isLoggedIn" class="login-button" @click.stop="openLogin">微信登录</button>
    </view>

    <view v-if="benefitOverviewVisible" class="benefit-overview-card section-card" @click="openBenefitOverview">
      <view class="benefit-head">
        <view class="benefit-title-wrap">
          <text class="benefit-title">我的权益</text>
          <text class="benefit-desc">{{ benefitOverviewDesc }}</text>
        </view>
        <text class="benefit-action">查看</text>
      </view>
      <view class="benefit-stats">
        <view class="benefit-stat">
          <text class="benefit-value">{{ publishQuotaRemaining }}</text>
          <text class="benefit-label">发布次数</text>
        </view>
        <view class="benefit-stat">
          <text class="benefit-value">{{ refreshQuotaRemaining }}</text>
          <text class="benefit-label">刷新次数</text>
        </view>
      </view>
      <text v-if="benefitExpiryReminder" class="benefit-expiry">{{ benefitExpiryReminder }}</text>
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
            <text class="action-title">我的主页</text>
            <text class="action-meta">查看对外展示资料</text>
          </view>
          <text class="entry-arrow"></text>
        </view>
        <view class="action-item" @click="openVIP">
          <view class="action-main">
            <text class="action-title">VIP 权益</text>
            <text class="action-meta">查看额度和限时特价</text>
          </view>
          <text class="entry-arrow"></text>
        </view>
        <view class="action-item" @click="openFavorites">
          <view class="action-main">
            <text class="action-title">收藏关注</text>
            <text class="action-meta">供应、商家、搜索</text>
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
import { ensureMerchantProfileReady } from '../../common/merchantProfileGuard'
import { formatDateToDay } from '../../common/date'
import { getSession } from '../../store/session'
import { getMerchantEntitlements } from '../../api/entitlement'
import { getActiveGrowthCampaigns } from '../../api/growthCampaign'
import { getMerchant } from '../../api/merchant'

const BENEFIT_EXPIRY_SOON_DAYS = 7

const token = ref('')
const merchantId = ref('')
const merchantProfile = ref({})
const merchantEntitlements = ref([])
const growthCampaigns = ref([])

const isLoggedIn = computed(() => Boolean(token.value))
const merchantLogo = computed(() => merchantProfile.value.logoUrl || '')
const merchantName = computed(() => merchantProfile.value.name || '')
const merchantNo = computed(() => merchantProfile.value.merchantNo || '')
const merchantNoVisible = computed(() => Boolean(isLoggedIn.value && merchantNo.value))
const avatarText = computed(() => merchantName.value.slice(0, 1) || (isLoggedIn.value ? '我' : '游'))
const accountName = computed(() => merchantName.value || (isLoggedIn.value ? '我的账号' : '未登录'))
const accountStatus = computed(() => {
  if (!merchantId.value) return 'unconfigured'
  if (merchantProfile.value.vipStatus === 'active') return 'vip'
  if (merchantProfile.value.profileStatus === 'incomplete') return 'unconfigured'
  return 'normal'
})
const accountStatusText = computed(() => {
  const statusText = {
    unconfigured: '待完善',
    normal: '普通用户',
    vip: 'VIP',
  }
  return statusText[accountStatus.value] || '普通用户'
})
const accountStatusClass = computed(() => `status-${accountStatus.value}`)
const benefitOverviewVisible = computed(() => Boolean(isLoggedIn.value))
const publishQuotaRemaining = computed(() => entitlementRemaining('publish_quota'))
const refreshQuotaRemaining = computed(() => entitlementRemaining('refresh_quota'))
const activeGrowthCampaign = computed(() => growthCampaigns.value[0] || {})
const benefitExpiryReminder = computed(() => {
  const candidates = merchantEntitlements.value
    .filter((item) => ['publish_quota', 'refresh_quota'].includes(item.type) && Number(item.remainingAmount || 0) > 0 && item.expiresAt)
    .map((item) => ({ ...item, expiresAtTime: Date.parse(item.expiresAt) }))
    .filter((item) => !Number.isNaN(item.expiresAtTime) && item.expiresAtTime > Date.now() && item.expiresAtTime - Date.now() <= BENEFIT_EXPIRY_SOON_DAYS * 24 * 60 * 60 * 1000)
    .sort((left, right) => left.expiresAtTime - right.expiresAtTime)
  if (!candidates.length) return ''
  const nearest = candidates[0]
  const amount = Number(nearest.remainingAmount || 0)
  return `最近到期：${entitlementLabel(nearest.type)} ${amount} 次，${formatDateToDay(nearest.expiresAt, '')} 到期`
})
const benefitOverviewDesc = computed(() => {
  if (merchantProfile.value.vipStatus === 'active') {
    return 'VIP 权益已启用，可购买次数包补充'
  }
  if (activeGrowthCampaign.value.code) {
    return activeGrowthCampaign.value.hint || '完成新手任务可获得更多发布和刷新次数'
  }
  return '完成任务、开通 VIP 或购买次数包可获得更多次数'
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
  await Promise.all([loadMerchantProfile(), loadMerchantEntitlements(), loadGrowthCampaigns()])
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

async function loadMerchantEntitlements() {
  if (!token.value || !merchantId.value) {
    merchantEntitlements.value = []
    return
  }
  try {
    const resp = await getMerchantEntitlements(merchantId.value, { suppressErrorToast: true })
    merchantEntitlements.value = resp.items || []
  } catch (err) {
    merchantEntitlements.value = []
  }
}

async function loadGrowthCampaigns() {
  if (!token.value) {
    growthCampaigns.value = []
    return
  }
  try {
    const resp = await getActiveGrowthCampaigns({ suppressErrorToast: true })
    growthCampaigns.value = resp.items || []
  } catch (err) {
    // 活动接口不可用时隐藏入口，不影响我的页核心管理入口。
    growthCampaigns.value = []
  }
}

function entitlementRemaining(type) {
  const items = merchantEntitlements.value.filter((item) => item.type === type)
  return items.reduce((sum, item) => sum + Number(item.remainingAmount || 0), 0)
}

function entitlementLabel(type) {
  if (type === 'refresh_quota') return '刷新次数'
  if (type === 'publish_quota') return '发布次数'
  return '权益'
}

function openAccountCard() {
  if (!isLoggedIn.value) {
    openLogin()
    return
  }
  openMerchantHome()
}

function copyMerchantNo() {
  if (!merchantNo.value) return
  uni.setClipboardData({
    data: merchantNo.value,
    success: () => {
      uni.showToast({ title: '编号已复制', icon: 'none' })
    },
  })
}

function openFavorites() {
  if (!requireLogin()) return
  uni.navigateTo({ url: '/pages/favorites/index' })
}

function openMessages() {
  if (!requireLogin()) return
  uni.switchTab({ url: '/pages/messages/index' })
}

async function openMyResources() {
  if (!requireLogin()) return
  if (!(await ensureMerchantProfileReady(merchantId.value))) return
  uni.navigateTo({ url: `/pages/my-resources/index?merchantId=${merchantId.value}` })
}

async function openMerchantHome() {
  if (!requireLogin()) return
  if (!(await ensureMerchantProfileReady(merchantId.value))) return
  uni.navigateTo({ url: `/pages/merchant/detail?id=${merchantId.value}` })
}

async function openVIP() {
  if (!requireLogin()) return
  uni.navigateTo({ url: `/pages/vip/index?merchantId=${merchantId.value}` })
}

async function openBenefitOverview() {
  if (!requireLogin()) return
  if (activeGrowthCampaign.value.code) {
    await openGrowthEntitlement()
    return
  }
  await openVIP()
}

async function openGrowthEntitlement() {
  if (!requireLogin()) return
  if (!(await ensureMerchantProfileReady(merchantId.value))) return
  uni.navigateTo({ url: `/pages/my/growth-entitlement?merchantId=${merchantId.value}` })
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

.action-meta,
.benefit-desc {
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.5;
}

.account-number-row {
  display: inline-flex;
  align-items: center;
  justify-self: start;
  max-width: 100%;
  min-height: 40rpx;
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.4;
}

.account-number-label,
.account-number-value,
.account-number-copy {
  flex: 0 1 auto;
  min-width: 0;
}

.account-number-value {
  color: $wplink-primary;
  font-weight: 700;
}

.account-number-copy {
  margin-left: 14rpx;
  color: $wplink-accent;
  font-size: 24rpx;
  font-weight: 700;
}

.benefit-overview-card {
  display: grid;
  gap: 20rpx;
}

.benefit-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18rpx;
  min-width: 0;
}

.benefit-title-wrap {
  display: grid;
  gap: 6rpx;
  min-width: 0;
}

.benefit-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
  line-height: 1.3;
}

.benefit-action {
  flex: 0 0 auto;
  color: $wplink-accent;
  font-size: 24rpx;
  font-weight: 700;
}

.benefit-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12rpx;
}

.benefit-stat {
  display: grid;
  gap: 4rpx;
  min-width: 0;
  padding: 18rpx;
  border-radius: 10rpx;
  background: $wplink-primary-soft;
}

.benefit-value {
  color: $wplink-primary;
  font-size: 40rpx;
  font-weight: 800;
  line-height: 1.1;
}

.benefit-label {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.3;
}

.benefit-expiry {
  padding: 14rpx 18rpx;
  border-radius: 10rpx;
  background: $wplink-warning-soft;
  color: $wplink-warning;
  font-size: 24rpx;
  font-weight: 600;
  line-height: 1.4;
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

.status-vip {
  background: rgba(194, 58, 0, 0.1);
  color: $wplink-warning;
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
