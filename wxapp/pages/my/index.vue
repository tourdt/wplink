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

    <view v-if="benefitOverviewVisible" class="benefit-overview-card section-card">
      <view class="benefit-head">
        <view class="benefit-title-wrap">
          <text class="benefit-title">我的权益</text>
          <text class="benefit-desc">剩余次数可用于发布和刷新供需信息</text>
        </view>
        <text class="benefit-detail-action" @click="openGrowthEntitlement">权益明细 ›</text>
      </view>
      <view class="benefit-stats">
        <view
          class="benefit-stat"
          :class="{
            'benefit-stat-low': entitlementQuotaReady && publishQuotaLow,
            'benefit-stat-empty': entitlementQuotaReady && publishQuotaRemaining === 0,
          }"
        >
          <view class="benefit-stat-head">
            <text class="benefit-label">可发布</text>
            <text
              v-if="entitlementQuotaReady"
              class="quota-purchase-action"
              @click="openQuotaPurchase(QUOTA_TYPE_PUBLISH)"
            >{{ publishQuotaRemaining === 0 ? '购买 ›' : '补充 ›' }}</text>
          </view>
          <view class="benefit-value-row">
            <text class="benefit-value">{{ publishQuotaDisplay }}</text>
            <text v-if="entitlementQuotaReady" class="benefit-unit">次</text>
          </view>
        </view>
        <view
          class="benefit-stat"
          :class="{
            'benefit-stat-low': entitlementQuotaReady && refreshQuotaLow,
            'benefit-stat-empty': entitlementQuotaReady && refreshQuotaRemaining === 0,
          }"
        >
          <view class="benefit-stat-head">
            <text class="benefit-label">可刷新</text>
            <text
              v-if="entitlementQuotaReady"
              class="quota-purchase-action"
              @click="openQuotaPurchase(QUOTA_TYPE_REFRESH)"
            >{{ refreshQuotaRemaining === 0 ? '购买 ›' : '补充 ›' }}</text>
          </view>
          <view class="benefit-value-row">
            <text class="benefit-value">{{ refreshQuotaDisplay }}</text>
            <text v-if="entitlementQuotaReady" class="benefit-unit">次</text>
          </view>
        </view>
      </view>
      <view v-if="activeGrowthCampaign.code" class="benefit-growth-banner">
        <view class="benefit-growth-main">
          <text class="benefit-growth-title">{{ benefitGrowthTitle }}</text>
          <text class="benefit-growth-desc">完成任务，奖励自动到账</text>
        </view>
        <text class="benefit-growth-button" @click="openGrowthEntitlement">去完成</text>
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
        <view class="action-item" @click="openAccountSettings">
          <view class="action-main">
            <text class="action-title">账号与隐私</text>
            <text class="action-meta">协议、隐私和账号注销</text>
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
import { QUOTA_TYPE_PUBLISH, QUOTA_TYPE_REFRESH, buildQuotaPurchaseUrl } from '../../common/entitlementPurchase'
import { getSession } from '../../store/session'
import { getMerchantEntitlements } from '../../api/entitlement'
import { getActiveGrowthCampaigns } from '../../api/growthCampaign'
import { getMerchant } from '../../api/merchant'

const BENEFIT_EXPIRY_SOON_DAYS = 7
const QUOTA_LOW_THRESHOLD = 2

const token = ref('')
const merchantId = ref('')
const merchantProfile = ref({})
const merchantEntitlements = ref([])
const entitlementLoadState = ref('idle')
const entitlementRequestId = ref(0)
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
// 只有确认了商户身份且权益接口成功返回时，额度才可作为购买引导的依据；未知状态不能按零额度营销。
const entitlementQuotaReady = computed(() => Boolean(isLoggedIn.value && merchantId.value && entitlementLoadState.value === 'loaded'))
const publishQuotaDisplay = computed(() => entitlementQuotaReady.value ? publishQuotaRemaining.value : '--')
const refreshQuotaDisplay = computed(() => entitlementQuotaReady.value ? refreshQuotaRemaining.value : '--')
const publishQuotaLow = computed(() => publishQuotaRemaining.value > 0 && publishQuotaRemaining.value <= QUOTA_LOW_THRESHOLD)
const refreshQuotaLow = computed(() => refreshQuotaRemaining.value > 0 && refreshQuotaRemaining.value <= QUOTA_LOW_THRESHOLD)
const activeGrowthCampaign = computed(() => growthCampaigns.value[0] || {})
const benefitGrowthTitle = computed(() => {
  // 权益未知时使用中性免费任务文案，不能把接口故障误表达为“次数已用完”。
  if (!entitlementQuotaReady.value) return '做任务，免费得次数'
  return publishQuotaRemaining.value === 0 || refreshQuotaRemaining.value === 0
    ? '也可以免费获取'
    : '做任务，免费得次数'
})
const benefitExpiryReminder = computed(() => {
  if (!entitlementQuotaReady.value) return ''
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
  // onLoad 与 onShow 可能并发；仅允许最后一次请求写回，避免切换商户后展示旧商户的权益。
  const requestId = entitlementRequestId.value + 1
  entitlementRequestId.value = requestId
  if (!token.value || !merchantId.value) {
    merchantEntitlements.value = []
    entitlementLoadState.value = 'unavailable'
    return
  }
  entitlementLoadState.value = 'loading'
  try {
    const resp = await getMerchantEntitlements(merchantId.value, { suppressErrorToast: true })
    if (requestId !== entitlementRequestId.value) return
    merchantEntitlements.value = resp.items || []
    entitlementLoadState.value = 'loaded'
  } catch (err) {
    if (requestId !== entitlementRequestId.value) return
    // 请求失败时保留中性额度展示，避免把接口故障误导为用户的真实零额度。
    merchantEntitlements.value = []
    entitlementLoadState.value = 'error'
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
  // 消息页退出 TabBar 后仍从“我的”进入，使用普通页面跳转保留返回路径。
  uni.navigateTo({ url: '/pages/messages/index' })
}

function openAccountSettings() {
  if (!requireLogin()) return
  uni.navigateTo({ url: '/pages/account/settings' })
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

function openQuotaPurchase(quotaType) {
  if (!requireLogin()) return
  uni.navigateTo({ url: buildQuotaPurchaseUrl(quotaType) })
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
  align-items: flex-start;
  justify-content: space-between;
  gap: 18rpx;
  min-width: 0;
}

.benefit-title-wrap {
  display: grid;
  flex: 1 1 auto;
  gap: 6rpx;
  min-width: 0;
}

.benefit-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
  line-height: 1.3;
}

.benefit-detail-action {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  min-height: 72rpx;
  color: $wplink-muted;
  font-size: 24rpx;
  font-weight: 700;
}

.benefit-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12rpx;
}

.benefit-stat {
  position: relative;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 136rpx;
  padding: 16rpx 18rpx;
  border: 1rpx solid transparent;
  border-radius: 10rpx;
  background: $wplink-primary-soft;
}

.benefit-stat-head {
  display: flex;
  align-items: center;
  min-width: 0;
  min-height: 32rpx;
  padding-right: 74rpx;
}

.benefit-stat-low {
  border-color: rgba(194, 58, 0, 0.16);
  background: rgba(194, 58, 0, 0.04);
}

.benefit-stat-empty {
  border-color: rgba(194, 58, 0, 0.24);
  background: $wplink-warning-soft;
}

.benefit-value-row {
  display: flex;
  align-items: baseline;
  gap: 6rpx;
  margin-top: 10rpx;
}

.benefit-value {
  color: $wplink-primary;
  font-size: 40rpx;
  font-weight: 800;
  line-height: 1.1;
}

.benefit-unit,
.benefit-label {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.3;
}

.quota-purchase-action {
  position: absolute;
  top: 2rpx;
  right: 8rpx;
  display: inline-flex;
  align-items: center;
  min-height: 64rpx;
  padding: 0 10rpx;
  color: $wplink-muted;
  font-size: 22rpx;
  font-weight: 700;
  line-height: 1.4;
}

.benefit-stat-low .quota-purchase-action {
  color: rgba(194, 58, 0, 0.78);
}

.benefit-stat-empty .quota-purchase-action {
  color: $wplink-warning;
}

.benefit-growth-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18rpx;
  padding: 18rpx;
  border-radius: 10rpx;
  background: $wplink-success-soft;
}

.benefit-growth-main {
  display: grid;
  flex: 1 1 auto;
  gap: 6rpx;
  min-width: 0;
}

.benefit-growth-title {
  color: $wplink-success;
  font-size: 26rpx;
  font-weight: 700;
  line-height: 1.35;
}

.benefit-growth-desc {
  color: $wplink-muted;
  font-size: 22rpx;
  line-height: 1.4;
}

.benefit-growth-button {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  min-height: 72rpx;
  padding: 0 20rpx;
  border-radius: 999rpx;
  background: $wplink-success;
  color: $wplink-card;
  font-size: 22rpx;
  font-weight: 700;
}

.benefit-expiry {
  padding: 14rpx 18rpx;
  border-radius: 10rpx;
  background: $wplink-primary-soft;
  color: $wplink-muted;
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
