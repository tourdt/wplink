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
          <text class="account-desc">{{ accountDesc }}</text>
        </view>
      </view>
      <view v-if="quotaSummaryVisible" class="quota-summary">
        <text class="quota-title">{{ quotaSummaryTitle }}</text>
        <text class="quota-desc">{{ quotaSummaryDesc }}</text>
      </view>
      <button v-if="!isLoggedIn" class="login-button" @click.stop="openLogin">微信登录</button>
    </view>

    <view v-if="entitlementCards.length" class="entitlement-section section-card">
      <view class="section-head">
        <text class="section-title">权益余额</text>
        <text class="section-subtitle">剩余次数与有效期</text>
      </view>
      <view class="entitlement-list">
        <view v-for="item in entitlementCards" :key="item.key" class="entitlement-item" @click="openEntitlementDetail(item)">
          <view class="entitlement-main">
            <text class="entitlement-title">{{ item.title }}</text>
            <text class="entitlement-meta">{{ item.sourceText }} · {{ item.expiresText }}</text>
          </view>
          <view class="entitlement-count">
            <text class="entitlement-count-value">{{ item.remainingAmount }}</text>
            <text class="entitlement-count-label">剩余</text>
          </view>
        </view>
      </view>
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
            <text class="action-meta">供给、商家、搜索</text>
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

    <view v-if="selectedEntitlement" class="entitlement-detail-mask" @click="closeEntitlementDetail">
      <view class="entitlement-detail-panel" @click.stop>
        <view class="detail-head">
          <view class="detail-title-wrap">
            <text class="detail-title">{{ selectedEntitlement.title }}</text>
            <text class="detail-subtitle">{{ selectedEntitlement.sourceText }}</text>
          </view>
          <button class="detail-close" @click="closeEntitlementDetail">关闭</button>
        </view>
        <view class="detail-stats">
          <view class="detail-stat">
            <text class="detail-stat-value">{{ selectedEntitlement.remainingAmount }}</text>
            <text class="detail-stat-label">剩余次数</text>
          </view>
          <view class="detail-stat">
            <text class="detail-stat-value">{{ selectedEntitlement.totalAmount }}</text>
            <text class="detail-stat-label">发放次数</text>
          </view>
        </view>
        <view class="detail-expire-row">
          <text class="detail-label">过期时间</text>
          <text class="detail-value">{{ selectedEntitlement.expiresText }}</text>
        </view>
        <view class="usage-list">
          <view class="usage-head">
            <text class="usage-title">使用记录</text>
            <text class="usage-count">{{ entitlementUsageRecords.length }} 条</text>
          </view>
          <text v-if="usageRecordsLoading" class="usage-empty">加载中</text>
          <text v-else-if="!entitlementUsageRecords.length" class="usage-empty">暂无使用记录</text>
          <template v-else>
            <view v-for="record in entitlementUsageRecords" :key="record.id" class="usage-item">
              <view class="usage-main">
                <text class="usage-action">{{ usageActionText(record.actionType) }}</text>
                <text class="usage-time">{{ formatDateToMinute(record.usedAt) }}</text>
              </view>
              <text class="usage-balance">{{ record.beforeRemainingAmount }} 到 {{ record.afterRemainingAmount }}</text>
            </view>
          </template>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { buildLoginUrl, requireLogin } from '../../common/auth'
import { ensureMerchantProfileReady } from '../../common/merchantProfileGuard'
import { getSession } from '../../store/session'
import { getMerchantEntitlements, getMerchantEntitlementUsageRecords } from '../../api/entitlement'
import { getMerchant } from '../../api/merchant'
import { getMerchantMetricsSummary } from '../../api/metrics'

const token = ref('')
const merchantId = ref('')
const merchantProfile = ref({})
const merchantMetricsSummary = ref(null)
const merchantEntitlements = ref([])
const selectedEntitlement = ref(null)
const entitlementUsageRecords = ref([])
const usageRecordsLoading = ref(false)

const isLoggedIn = computed(() => Boolean(token.value))
const merchantLogo = computed(() => merchantProfile.value.logoUrl || '')
const merchantName = computed(() => merchantProfile.value.name || '')
const avatarText = computed(() => merchantName.value.slice(0, 1) || (isLoggedIn.value ? '我' : '游'))
const accountName = computed(() => merchantName.value || (isLoggedIn.value ? '我的账号' : '未登录'))
const accountDesc = computed(() => (merchantName.value ? '已录入商户资料，可管理发布和 VIP 权益' : isLoggedIn.value ? '已登录，可管理收藏和消息' : '登录后管理收藏和发布记录'))
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
const quotaSummaryVisible = computed(() => Boolean(isLoggedIn.value && merchantId.value))
const publishQuotaRemaining = computed(() => entitlementRemaining('publish_quota'))
const refreshQuotaRemaining = computed(() => entitlementRemaining('refresh_quota'))
const quotaSummaryTitle = computed(() => {
  if (merchantProfile.value.vipStatus === 'active') {
    return `VIP 权益：发布 ${publishQuotaRemaining.value} 条 · 刷新 ${refreshQuotaRemaining.value} 次`
  }
  if (merchantProfile.value.profileStatus === 'completed') {
    return `本月剩余：发布 ${publishQuotaRemaining.value} 条 · 刷新 ${refreshQuotaRemaining.value} 次`
  }
  return `免费额度：本月可发布 ${publishQuotaRemaining.value} 条`
})
const quotaSummaryDesc = computed(() => {
  if (merchantProfile.value.vipStatus === 'active') {
    return '置顶功能暂未开放，当前可使用发布和刷新权益'
  }
  if (merchantProfile.value.profileStatus === 'completed') {
    return '需要更多发布和曝光，可查看 VIP 权益'
  }
  return '完善资料后每月可发布 10 条，并获得 3 次刷新'
})
const merchantEffectVisible = computed(() => Boolean(token.value && merchantId.value && merchantMetricsSummary.value))
const visibleEntitlements = computed(() => merchantEntitlements.value.filter((item) => item.type !== 'top_voucher'))
const entitlementCards = computed(() => visibleEntitlements.value.map((item, index) => ({
  ...item,
  key: item.id || `${item.type}-${item.sourceType}-${index}`,
  title: entitlementTitle(item.type),
  sourceText: entitlementSourceText(item.sourceType),
  remainingAmount: Number(item.remainingAmount || 0),
  totalAmount: Number(item.totalAmount || 0),
  expiresText: formatDateToMinute(item.expiresAt) || '长期有效',
})))
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
  await Promise.all([loadMerchantProfile(), loadMerchantMetricsSummary(), loadMerchantEntitlements()])
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

function entitlementRemaining(type) {
  const items = merchantEntitlements.value.filter((item) => item.type === type)
  if (items.length > 0) {
    return items.reduce((sum, item) => sum + Number(item.remainingAmount || 0), 0)
  }
  if (merchantProfile.value.vipStatus === 'active') return 0
  if (type === 'publish_quota') {
    return merchantProfile.value.profileStatus === 'completed' ? 10 : 3
  }
  if (type === 'refresh_quota') {
    return merchantProfile.value.profileStatus === 'completed' ? 3 : 0
  }
  return 0
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

async function openEntitlementDetail(item) {
  selectedEntitlement.value = item
  entitlementUsageRecords.value = []
  if (!item.id || !merchantId.value) return
  usageRecordsLoading.value = true
  try {
    const resp = await getMerchantEntitlementUsageRecords(merchantId.value, item.id, { suppressErrorToast: true })
    entitlementUsageRecords.value = resp.items || []
  } catch (err) {
    entitlementUsageRecords.value = []
  } finally {
    usageRecordsLoading.value = false
  }
}

function closeEntitlementDetail() {
  selectedEntitlement.value = null
  entitlementUsageRecords.value = []
  usageRecordsLoading.value = false
}

function entitlementTitle(type) {
  const titleMap = {
    publish_quota: '发布额度',
    refresh_quota: '刷新次数',
    top_voucher: '置顶券',
  }
  return titleMap[type] || '权益'
}

function entitlementSourceText(sourceType) {
  const sourceMap = {
    vip: 'VIP赠送',
    quota_pack: '次数包',
    verification: '认证赠送',
    profile_monthly: '资料权益',
    manual: '手动发放',
  }
  return sourceMap[sourceType] || '权益发放'
}

function usageActionText(actionType) {
  const actionMap = {
    publish_resource: '消耗发布额度',
    refresh_resource: '刷新资源',
    top_resource: '置顶资源',
  }
  return actionMap[actionType] || '使用权益'
}

function formatDateToMinute(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const pad = (num) => String(num).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
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
.action-meta,
.quota-desc {
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.5;
}

.quota-summary {
  display: grid;
  gap: 6rpx;
  min-width: 0;
  padding-top: 18rpx;
  border-top: 1rpx solid $wplink-line;
}

.quota-title {
  color: $wplink-primary;
  font-size: 28rpx;
  font-weight: 700;
  line-height: 1.35;
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

.entitlement-list {
  display: grid;
  gap: 12rpx;
}

.entitlement-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18rpx;
  min-width: 0;
  padding: 18rpx 0;
  border-bottom: 1rpx solid $wplink-line;
}

.entitlement-item:last-child {
  border-bottom: 0;
}

.entitlement-main {
  display: grid;
  gap: 8rpx;
  min-width: 0;
}

.entitlement-title {
  color: $wplink-primary;
  font-size: 29rpx;
  font-weight: 700;
  line-height: 1.35;
}

.entitlement-meta {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.4;
}

.entitlement-count {
  display: grid;
  gap: 2rpx;
  flex: 0 0 96rpx;
  text-align: right;
}

.entitlement-count-value {
  color: $wplink-primary;
  font-size: 34rpx;
  font-weight: 700;
  line-height: 1.2;
}

.entitlement-count-label {
  color: $wplink-muted;
  font-size: 22rpx;
  line-height: 1.3;
}

.entitlement-detail-mask {
  position: fixed;
  inset: 0;
  z-index: 30;
  display: flex;
  align-items: flex-end;
  background: rgba(15, 23, 42, 0.36);
}

.entitlement-detail-panel {
  display: grid;
  gap: 22rpx;
  width: 100%;
  max-height: 78vh;
  padding: 28rpx 28rpx calc(34rpx + env(safe-area-inset-bottom));
  border-radius: 18rpx 18rpx 0 0;
  background: $wplink-card;
  overflow-y: auto;
}

.detail-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18rpx;
}

.detail-title-wrap {
  display: grid;
  gap: 6rpx;
  min-width: 0;
}

.detail-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
  line-height: 1.3;
}

.detail-subtitle,
.detail-label,
.usage-count,
.usage-time {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.4;
}

.detail-close {
  flex: 0 0 auto;
  min-width: 112rpx;
  height: 58rpx;
  padding: 0 18rpx;
  border-radius: 10rpx;
  background: #f4f7fd;
  color: $wplink-primary;
  font-size: 24rpx;
  line-height: 58rpx;
}

.detail-close::after {
  border: 0;
}

.detail-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12rpx;
}

.detail-stat {
  display: grid;
  gap: 6rpx;
  padding: 18rpx;
  border-radius: 10rpx;
  background: #f8fafc;
}

.detail-stat-value {
  color: $wplink-primary;
  font-size: 34rpx;
  font-weight: 700;
  line-height: 1.2;
}

.detail-stat-label {
  color: $wplink-muted;
  font-size: 23rpx;
  line-height: 1.4;
}

.detail-expire-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
  padding: 18rpx 0;
  border-top: 1rpx solid $wplink-line;
  border-bottom: 1rpx solid $wplink-line;
}

.detail-value {
  color: $wplink-primary;
  font-size: 25rpx;
  line-height: 1.4;
  text-align: right;
}

.usage-list {
  display: grid;
  gap: 12rpx;
}

.usage-head,
.usage-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
}

.usage-title,
.usage-action {
  color: $wplink-primary;
  font-size: 27rpx;
  font-weight: 700;
  line-height: 1.35;
}

.usage-main {
  display: grid;
  gap: 4rpx;
  min-width: 0;
}

.usage-item {
  padding: 14rpx 0;
  border-bottom: 1rpx solid $wplink-line;
}

.usage-item:last-child {
  border-bottom: 0;
}

.usage-balance {
  flex: 0 0 auto;
  color: $wplink-primary;
  font-size: 24rpx;
  line-height: 1.4;
}

.usage-empty {
  padding: 20rpx 0;
  color: $wplink-muted;
  font-size: 25rpx;
  line-height: 1.5;
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
