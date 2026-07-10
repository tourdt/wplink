<template>
  <view class="growth-page">
    <view class="hero-card">
      <text class="hero-title">{{ activeCampaign.title || '新手发布权益' }}</text>
      <text class="hero-desc">{{ campaignHint }}</text>
    </view>

    <view class="section-card quota-card">
      <view class="section-head">
        <text class="section-title">权益余额</text>
        <text class="section-subtitle">当前可用次数</text>
      </view>
      <view class="quota-grid">
        <view v-for="item in quotaCards" :key="item.key" class="quota-item">
          <text class="quota-value">{{ item.value }}</text>
          <text class="quota-label">{{ item.label }}</text>
        </view>
      </view>
    </view>

    <view class="section-card progress-card">
      <view class="section-head">
        <text class="section-title">新手成长进度</text>
        <text class="section-subtitle">主线 {{ taskSummary.starterCompletedCount || 0 }}/{{ taskSummary.starterTotalCount || 0 }}</text>
      </view>
      <view class="progress-track">
        <view class="progress-fill" :style="{ width: starterProgressPercent }"></view>
      </view>
    </view>

    <view class="section-card">
      <view class="section-head">
        <text class="section-title">新手主线任务</text>
        <text class="section-subtitle">完成一次，跑通发布审核</text>
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
          <button v-if="task.actionType !== 'none'" class="task-action" @click="openTaskAction(task)">{{ task.actionText || '去完成' }}</button>
        </view>
      </view>
      <text v-else class="empty-text">{{ taskEmptyText }}</text>
    </view>

    <view class="section-card">
      <view class="section-head">
        <text class="section-title">每日曝光任务</text>
        <text class="section-subtitle">每日刷新，只看有效结果</text>
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
          <button v-if="task.actionType !== 'none'" class="task-action secondary" @click="openTaskAction(task)">{{ task.actionText || '去完成' }}</button>
        </view>
      </view>
      <text v-else class="empty-text">{{ dailyEmptyText }}</text>
    </view>

    <view class="section-card">
      <view class="section-head">
        <text class="section-title">规则说明</text>
      </view>
      <view class="rule-note-list">
        <text class="rule-note">达标自动到账，无需领取。</text>
        <text class="rule-note">发布类不每日重置。</text>
        <text class="rule-note">分享需产生有效浏览或联系。</text>
        <text class="rule-note">权益到期未用自动失效。</text>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'

import { requireLogin } from '../../common/auth'
import { getSession } from '../../store/session'
import { getMerchantEntitlements } from '../../api/entitlement'
import { getActiveGrowthCampaigns, getGrowthTasks } from '../../api/growthCampaign'

const merchantId = ref('')
const entitlements = ref([])
const campaigns = ref([])
const growthTaskData = ref(null)
const tasksLoadFailed = ref(false)

const activeCampaign = computed(() => growthTaskData.value?.campaign?.code ? growthTaskData.value.campaign : campaigns.value[0] || {})
const campaignHint = computed(() => '发资源、做分享，自动得权益')
const taskSummary = computed(() => growthTaskData.value?.summary || fallbackTaskSummary.value)
const allTasks = computed(() => growthTaskData.value?.tasks || [])
const starterTasks = computed(() => allTasks.value.filter((task) => task.group === 'starter'))
const dailyTasks = computed(() => allTasks.value.filter((task) => task.group === 'daily'))
const quotaCards = computed(() => [
  { key: 'publish', label: '发布次数', value: Number(taskSummary.value.publishQuotaRemaining || 0) },
  { key: 'refresh', label: '刷新次数', value: Number(taskSummary.value.refreshQuotaRemaining || 0) },
])
const fallbackTaskSummary = computed(() => ({
  publishQuotaRemaining: entitlementRemaining('publish_quota'),
  refreshQuotaRemaining: entitlementRemaining('refresh_quota'),
  starterCompletedCount: 0,
  starterTotalCount: 0,
}))
const starterProgressPercent = computed(() => {
  const total = Number(taskSummary.value.starterTotalCount || 0)
  if (!total) return '0%'
  const completed = Math.min(Number(taskSummary.value.starterCompletedCount || 0), total)
  return `${Math.round((completed / total) * 100)}%`
})
const taskEmptyText = computed(() => (tasksLoadFailed.value ? '成长任务暂不可用，请稍后重试。' : '暂无可参与的新手任务。'))
const dailyEmptyText = computed(() => (tasksLoadFailed.value ? '每日任务暂不可用，请稍后重试。' : '暂无每日任务。'))

onLoad((options = {}) => {
  merchantId.value = options.merchantId || getSession().merchantId || ''
  loadGrowthPageData()
})

onShow(() => {
  const sessionMerchantId = getSession().merchantId
  if (!merchantId.value && sessionMerchantId) merchantId.value = sessionMerchantId
  loadGrowthPageData()
})

async function loadGrowthPageData() {
  if (!requireLogin()) return
  await Promise.all([loadGrowthTasks(), loadEntitlements(), loadCampaigns()])
}

async function loadGrowthTasks() {
  if (!merchantId.value) {
    growthTaskData.value = null
    return
  }
  try {
    const resp = await getGrowthTasks(merchantId.value, { suppressErrorToast: true })
    growthTaskData.value = resp || null
    tasksLoadFailed.value = false
  } catch (err) {
    growthTaskData.value = null
    tasksLoadFailed.value = true
  }
}

async function loadEntitlements() {
  if (!merchantId.value) {
    entitlements.value = []
    return
  }
  try {
    const resp = await getMerchantEntitlements(merchantId.value, { suppressErrorToast: true })
    entitlements.value = resp.items || []
  } catch (err) {
    entitlements.value = []
  }
}

async function loadCampaigns() {
  try {
    const resp = await getActiveGrowthCampaigns({ suppressErrorToast: true })
    campaigns.value = resp.items || []
  } catch (err) {
    campaigns.value = []
  }
}

function entitlementRemaining(type) {
  return entitlements.value
    .filter((item) => item.type === type && item.type !== 'top_voucher')
    .reduce((sum, item) => sum + Number(item.remainingAmount || 0), 0)
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

function openTaskAction(task = {}) {
  const action = task.actionType
  if (action === 'publish') {
    navigateTo('/pages/publish/index')
    return
  }
  if (action === 'share' || action === 'manage_resources') {
    navigateTo('/pages/my-resources/index')
    return
  }
  if (action === 'use_entitlement') {
    navigateTo(task.rewardType === 'refresh_quota' ? '/pages/my-resources/index' : '/pages/publish/index')
  }
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

.progress-track {
  position: relative;
  overflow: hidden;
  height: 14rpx;
  border-radius: 999rpx;
  background: $wplink-line;
}

.progress-fill {
  height: 100%;
  border-radius: inherit;
  background: $wplink-accent;
}

.task-list,
.rule-note-list {
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
  align-items: center;
  gap: 12rpx;
  min-width: 0;
}

.task-title {
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
