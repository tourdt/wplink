<template>
  <view class="growth-page">
    <view class="hero-card">
      <text class="hero-title">{{ activeCampaign.title || '新手发布权益' }}</text>
      <text class="hero-desc">{{ campaignHint }}</text>
    </view>

    <view class="section-card quota-card">
      <view class="section-head">
        <text class="section-title">权益余额</text>
        <text class="section-subtitle">当前可用次数 · 主线进度 {{ taskSummary.starterCompletedCount || 0 }}/{{ taskSummary.starterTotalCount || 0 }}</text>
      </view>
      <view class="quota-grid">
        <view v-for="item in quotaCards" :key="item.key" class="quota-item">
          <text class="quota-value">{{ item.value }}</text>
          <text class="quota-label">{{ item.label }}</text>
        </view>
      </view>
      <text v-if="nearestExpiryText" class="quota-expiry">{{ nearestExpiryText }}</text>
    </view>

    <view class="section-card">
      <view class="section-head">
        <text class="section-title">新手主线任务</text>
        <text class="section-subtitle">优先完成发布和审核</text>
      </view>
      <view v-if="starterTasks.length" class="task-list">
        <view v-for="task in starterTasks" :key="task.taskCode" class="task-item">
          <view class="task-main">
            <view class="task-title-row">
              <text class="task-title">{{ task.title }}</text>
              <text :class="['task-status', `status-${task.status}`]">{{ taskStatusText(task.status) }}</text>
            </view>
            <text class="task-desc">{{ task.description }}</text>
            <view class="task-progress-row">
              <text class="task-progress">{{ taskProgressText(task) }}</text>
              <text class="task-reward">{{ taskRewardMeta(task) }}</text>
            </view>
            <text v-if="task.hint" class="task-hint">{{ task.hint }}</text>
          </view>
          <button v-if="shouldShowTaskAction(task)" class="task-action" @click="openTaskAction(task)">{{ taskActionText(task) }}</button>
        </view>
      </view>
      <text v-else class="empty-text">{{ taskEmptyText }}</text>
    </view>

    <view class="section-card">
      <view class="section-head">
        <text class="section-title">每日曝光任务</text>
        <text class="section-subtitle">分享带来有效结果后奖励</text>
      </view>
      <view v-if="dailyTasks.length" class="task-list">
        <view v-for="task in dailyTasks" :key="task.taskCode" class="task-item">
          <view class="task-main">
            <view class="task-title-row">
              <text class="task-title">{{ task.title }}</text>
              <text :class="['task-status', `status-${task.status}`]">{{ taskStatusText(task.status) }}</text>
            </view>
            <text class="task-desc">{{ task.description }}</text>
            <view class="task-progress-row">
              <text class="task-progress">今日 {{ taskProgressText(task) }}</text>
              <text class="task-reward">{{ taskRewardMeta(task) }}</text>
            </view>
            <text v-if="task.hint" class="task-hint">{{ task.hint }}</text>
          </view>
          <button v-if="shouldShowTaskAction(task)" class="task-action secondary" @click="openTaskAction(task)">{{ taskActionText(task) }}</button>
        </view>
      </view>
      <text v-else class="empty-text">{{ dailyEmptyText }}</text>
    </view>

    <view class="section-card">
      <text class="rule-note">规则说明：达标自动到账；分享需产生有效浏览或联系；权益到期未用自动失效。</text>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'

import { requireLogin } from '../../common/auth'
import { formatDateToDay } from '../../common/date'
import { getSession } from '../../store/session'
import { getMerchantEntitlements } from '../../api/entitlement'
import { getActiveGrowthCampaigns, getGrowthTasks } from '../../api/growthCampaign'

const merchantId = ref('')
const entitlements = ref([])
const campaigns = ref([])
const growthTaskData = ref(null)
const tasksLoaded = ref(false)
const tasksLoadFailed = ref(false)
let growthPageLoadSeq = 0
let skipNextShowLoad = false

const activeCampaign = computed(() => growthTaskData.value?.campaign?.code ? growthTaskData.value.campaign : campaigns.value[0] || {})
const campaignHint = computed(() => activeCampaign.value.hint || '发资源、做分享，自动得权益')
const taskSummary = computed(() => growthTaskData.value?.summary || fallbackTaskSummary.value)
const allTasks = computed(() => growthTaskData.value?.tasks || [])
const starterTasks = computed(() => allTasks.value.filter((task) => task.group === 'starter'))
const dailyTasks = computed(() => allTasks.value.filter((task) => task.group === 'daily'))
const hasTaskCampaign = computed(() => Boolean(growthTaskData.value?.campaign?.code))
const quotaCards = computed(() => [
  { key: 'publish', label: '发布次数', value: Number(taskSummary.value.publishQuotaRemaining || 0) },
  { key: 'refresh', label: '刷新次数', value: Number(taskSummary.value.refreshQuotaRemaining || 0) },
])
const nearestExpiryText = computed(() => {
  const candidates = entitlements.value
    .filter((item) => item.type !== 'top_voucher' && Number(item.remainingAmount || 0) > 0 && item.expiresAt)
    .map((item) => ({ ...item, expiresAtTime: Date.parse(item.expiresAt) }))
    .filter((item) => !Number.isNaN(item.expiresAtTime))
    .sort((left, right) => left.expiresAtTime - right.expiresAtTime)
  if (!candidates.length) return ''
  const nearest = candidates[0]
  const amount = Number(nearest.remainingAmount || 0)
  return `最近到期：${entitlementLabel(nearest.type)} ${amount} 次，${formatDateToDay(nearest.expiresAt, '')} 到期`
})
const fallbackTaskSummary = computed(() => ({
  publishQuotaRemaining: entitlementRemaining('publish_quota'),
  refreshQuotaRemaining: entitlementRemaining('refresh_quota'),
  starterCompletedCount: 0,
  starterTotalCount: 0,
}))
const taskEmptyText = computed(() => {
  if (!merchantId.value) return '请先完善发布者资料。'
  if (tasksLoadFailed.value) return '成长任务暂不可用，请稍后重试。'
  if (!tasksLoaded.value) return '任务加载中。'
  if (!hasTaskCampaign.value) return '当前暂无活动。'
  if (!allTasks.value.length) return '活动规则未开启。'
  return '暂无主线任务。'
})
const dailyEmptyText = computed(() => {
  if (!merchantId.value) return '请先完善发布者资料。'
  if (tasksLoadFailed.value) return '每日任务暂不可用，请稍后重试。'
  if (!tasksLoaded.value) return '任务加载中。'
  if (!hasTaskCampaign.value) return '当前暂无活动。'
  if (!allTasks.value.length) return '活动规则未开启。'
  return '暂无每日任务。'
})

onLoad((options = {}) => {
  merchantId.value = options.merchantId || getSession().merchantId || ''
  skipNextShowLoad = true
  loadGrowthPageData()
})

onShow(() => {
  if (skipNextShowLoad) {
    // uni-app 首次进入页面会在 onLoad 后立即触发 onShow；首屏请求已在 onLoad 发起，这里跳过一次避免重复拉取。
    skipNextShowLoad = false
    return
  }
  const sessionMerchantId = getSession().merchantId
  if (!merchantId.value && sessionMerchantId) merchantId.value = sessionMerchantId
  loadGrowthPageData()
})

async function loadGrowthPageData() {
  if (!requireLogin()) return
  const loadSeq = ++growthPageLoadSeq
  await Promise.all([loadGrowthTasks(loadSeq), loadEntitlements(loadSeq), loadCampaigns(loadSeq)])
}

async function loadGrowthTasks(loadSeq = growthPageLoadSeq) {
  tasksLoaded.value = false
  if (!merchantId.value) {
    // 任务进度按商户统计；缺少商户 ID 时不请求接口，直接给出可理解的空态。
    growthTaskData.value = null
    tasksLoadFailed.value = false
    tasksLoaded.value = true
    return
  }
  try {
    const resp = await getGrowthTasks(merchantId.value, { suppressErrorToast: true })
    if (isStaleGrowthPageLoad(loadSeq)) return
    growthTaskData.value = resp || null
    tasksLoadFailed.value = false
  } catch (err) {
    if (isStaleGrowthPageLoad(loadSeq)) return
    growthTaskData.value = null
    tasksLoadFailed.value = true
  } finally {
    if (!isStaleGrowthPageLoad(loadSeq)) tasksLoaded.value = true
  }
}

async function loadEntitlements(loadSeq = growthPageLoadSeq) {
  if (!merchantId.value) {
    entitlements.value = []
    return
  }
  try {
    const resp = await getMerchantEntitlements(merchantId.value, { suppressErrorToast: true })
    if (isStaleGrowthPageLoad(loadSeq)) return
    entitlements.value = resp.items || []
  } catch (err) {
    if (isStaleGrowthPageLoad(loadSeq)) return
    entitlements.value = []
  }
}

async function loadCampaigns(loadSeq = growthPageLoadSeq) {
  try {
    const resp = await getActiveGrowthCampaigns({ suppressErrorToast: true })
    if (isStaleGrowthPageLoad(loadSeq)) return
    campaigns.value = resp.items || []
  } catch (err) {
    if (isStaleGrowthPageLoad(loadSeq)) return
    campaigns.value = []
  }
}

function isStaleGrowthPageLoad(loadSeq) {
  return loadSeq !== growthPageLoadSeq
}

function entitlementRemaining(type) {
  return entitlements.value
    .filter((item) => item.type === type && item.type !== 'top_voucher')
    .reduce((sum, item) => sum + Number(item.remainingAmount || 0), 0)
}

function entitlementLabel(type) {
  if (type === 'refresh_quota') return '刷新次数'
  if (type === 'publish_quota') return '发布次数'
  return '权益'
}

function taskStatusText(status) {
  const texts = {
    todo: '待完成',
    in_progress: '进行中',
    pending_review: '等待审核',
    completed: '已达标',
    granted: '已到账',
    locked: '未开启',
  }
  return texts[status] || '待完成'
}

function taskProgressText(task = {}) {
  const current = Number(task.progressCurrent || 0)
  const target = Number(task.progressTarget || 0)
  if (!target) return '按规则完成'
  return `${Math.min(current, target)}/${target}`
}

function taskRewardMeta(task = {}) {
  if (task.rewardText) return task.rewardText
  const amount = Number(task.rewardAmount || 0)
  const typeText = task.rewardType === 'refresh_quota' ? '刷新次数' : '发布次数'
  return `奖励 ${amount} 次${typeText}`
}

function shouldShowTaskAction(task = {}) {
  if (!task || task.actionType === 'none') return false
  return !['completed', 'locked'].includes(task.status)
}

function taskActionText(task = {}) {
  if (task.status === 'granted') return task.actionText || '去使用'
  if (task.status === 'pending_review') return task.actionText || '查看进度'
  return task.actionText || '去完成'
}

function openTaskAction(task = {}) {
  if (!shouldShowTaskAction(task)) return
  const action = task.actionType
  if (action === 'publish') {
    navigateTo('/pages/publish/index')
    return
  }
  if (action === 'share' || action === 'manage_resources') {
    navigateTo(withMerchantQuery('/pages/my-resources/index'))
    return
  }
  if (action === 'use_entitlement') {
    navigateTo(task.rewardType === 'refresh_quota' ? withMerchantQuery('/pages/my-resources/index') : '/pages/publish/index')
  }
}

function withMerchantQuery(url) {
  return merchantId.value ? `${url}?merchantId=${merchantId.value}` : url
}

function navigateTo(url) {
  uni.navigateTo({ url })
}
</script>

<style lang="scss" scoped>
.growth-page {
  min-height: 100vh;
  padding: 24rpx 24rpx 44rpx;
  background: $wplink-bg;
}

.hero-card,
.section-card {
  display: grid;
  gap: 18rpx;
  margin-bottom: 20rpx;
  padding: 26rpx;
  border-radius: 12rpx;
  background: $wplink-card;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.04);
}

.hero-card {
  gap: 10rpx;
  background: $wplink-primary;
}

.hero-title {
  color: $wplink-card;
  font-size: 38rpx;
  font-weight: 800;
  line-height: 1.25;
}

.hero-desc {
  color: rgba(255, 255, 255, 0.82);
  font-size: 26rpx;
  line-height: 1.5;
}

.section-head {
  display: grid;
  gap: 6rpx;
}

.section-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
  line-height: 1.3;
}

.section-subtitle,
.empty-text,
.task-desc,
.task-progress,
.task-hint,
.rule-note {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.45;
}

.quota-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16rpx;
}

.quota-item {
  display: grid;
  gap: 6rpx;
  min-width: 0;
  padding: 18rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 12rpx;
  background: $wplink-primary-soft;
}

.quota-value {
  color: $wplink-primary;
  font-size: 44rpx;
  font-weight: 800;
  line-height: 1.1;
}

.quota-label {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.3;
}

.quota-expiry {
  color: $wplink-warning;
  font-size: 24rpx;
  line-height: 1.45;
}

.task-list {
  display: grid;
  gap: 14rpx;
}

.task-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18rpx;
  min-width: 0;
  padding: 18rpx 0;
  border-bottom: 1rpx solid $wplink-line;
}

.task-item:last-child {
  border-bottom: 0;
}

.task-main {
  display: grid;
  gap: 8rpx;
  min-width: 0;
}

.task-title-row,
.task-progress-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12rpx;
  min-width: 0;
}

.task-title {
  flex: 1 1 220rpx;
  min-width: 0;
  color: $wplink-primary;
  font-size: 28rpx;
  font-weight: 700;
  line-height: 1.35;
}

.task-status,
.task-reward {
  flex: 0 0 auto;
  padding: 6rpx 12rpx;
  border-radius: 999rpx;
  font-size: 22rpx;
  font-weight: 700;
  line-height: 1.3;
}

.task-status {
  background: $wplink-primary-soft;
  color: $wplink-primary;
}

.status-granted,
.status-completed {
  background: rgba(34, 197, 94, 0.12);
  color: #15803d;
}

.status-pending_review {
  background: rgba(245, 158, 11, 0.14);
  color: #a16207;
}

.task-reward {
  background: rgba(249, 115, 22, 0.12);
  color: $wplink-accent;
}

.task-action {
  flex: 0 0 auto;
  min-width: 132rpx;
  margin: 0;
  padding: 0 18rpx;
  border: 0;
  border-radius: 999rpx;
  background: $wplink-primary;
  color: $wplink-card;
  font-size: 24rpx;
  font-weight: 700;
  line-height: 58rpx;
}

.task-action.secondary {
  background: $wplink-accent;
}

.rule-note {
  display: block;
}
</style>
