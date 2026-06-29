<template>
  <view class="my-resources-page">
    <view class="resource-manager-head">
      <view>
        <text class="manager-title">我的发布</text>
        <text class="manager-desc">管理资源状态、效果数据和推广权益。</text>
      </view>
      <button @click="openPublish">发布</button>
    </view>

    <view class="lifecycle-note">
      <text>已发布资源可刷新、置顶、成交或下架；即将过期和已过期资源建议及时再发类似。</text>
    </view>

    <scroll-view class="filter-row" scroll-x>
      <button
        v-for="item in statusOptions"
        :key="item.value"
        :class="['filter-button', filters.status === item.value ? 'active' : '']"
        @click="selectStatus(item.value)"
      >
        {{ item.label }}
      </button>
    </scroll-view>

    <view class="resource-list">
      <view v-for="item in rows" :key="item.id" class="resource-card">
        <view class="card-head">
          <view class="tag-row">
            <text :class="['status-tag', item.status]">{{ statusText[item.status] || item.status }}</text>
            <text v-if="canTopResource(item)" class="top-tag">可置顶</text>
          </view>
          <text class="expire-text">{{ expireText(item) }}</text>
        </view>
        <text class="resource-title">{{ item.title }}</text>
        <text class="resource-meta">{{ item.category }} · {{ item.typeCode }}</text>
        <text class="resource-meta">发布 {{ item.publishedAt || '-' }} · 到期 {{ item.expiresAt || '-' }}</text>
        <MetricStrip :items="metricItems(item)" />
        <text class="effect-advice">{{ effectAdvice(item) }}</text>
        <view class="action-row">
          <button v-if="item.status === 'published'" @click="refresh(item)">刷新</button>
          <button v-if="item.status === 'published'" @click="topResource(item)">置顶</button>
          <button v-if="item.status === 'published'" @click="markDealt(item)">成交</button>
          <button v-if="item.status === 'published'" @click="takeDown(item)">下架</button>
          <button v-if="item.status === 'draft'" @click="submitDraft(item)">提交审核</button>
          <button v-if="canRepost(item)" @click="repost(item)">再发类似</button>
          <button @click="openResource(item)">详情</button>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import MetricStrip from '../../components/MetricStrip.vue'
import { getMerchantId } from '../../store/session'
import { redeemTopVoucher, listTopVouchers } from '../../api/entitlement'
import { listMyResources, markResourceDeal, refreshResource, repostSimilarResource, submitResource, takeDownResource } from '../../api/resource'

const statusOptions = [
  { label: '全部', value: '' },
  { label: '草稿', value: 'draft' },
  { label: '待审核', value: 'pending' },
  { label: '已发布', value: 'published' },
  { label: '即将过期', value: 'expiring_soon' },
  { label: '已过期', value: 'expired' },
  { label: '已成交', value: 'dealt' },
  { label: '已下架', value: 'taken_down' },
]
const statusText = {
  draft: '草稿',
  pending: '待审核',
  published: '已发布',
  rejected: '已驳回',
  taken_down: '已下架',
  expired: '已过期',
}

const rows = ref([])
const merchantId = ref('')
const filters = reactive({ status: '' })

onLoad((options) => {
  // 我的发布必须绑定当前商家；路由参数用于后台调试，正常用户流程使用我的页保存的商家 ID。
  merchantId.value = options.merchantId || getMerchantId()
  loadRows()
})

async function loadRows() {
  if (!merchantId.value) {
    uni.showToast({ title: '请先填写商家 ID', icon: 'none' })
    return
  }
  const resp = await listMyResources({ merchantId: merchantId.value, status: filters.status, page: 1, pageSize: 20 })
  rows.value = resp.items || []
}

function selectStatus(status) {
  filters.status = status
  loadRows()
}

async function refresh(item) {
  await refreshResource(item.id, merchantId.value)
  uni.showToast({ title: '已刷新', icon: 'none' })
  await loadRows()
}

async function topResource(item) {
  const resp = await listTopVouchers(merchantId.value)
  const voucher = (resp.items || []).find((entry) => entry.status === 'unused')
  if (!voucher) {
    uni.showToast({ title: '暂无可用置顶券', icon: 'none' })
    return
  }
  await redeemTopVoucher(voucher.id, item.id, merchantId.value)
  uni.showToast({ title: '已置顶', icon: 'none' })
  await loadRows()
}

async function markDealt(item) {
  await markResourceDeal(item.id, { merchantId: merchantId.value, isDealt: true, isReal: true, responseTimely: true, willingToCooperateAgain: true })
  uni.showToast({ title: '已标记成交', icon: 'none' })
  await loadRows()
}

async function takeDown(item) {
  await takeDownResource(item.id, merchantId.value, '商家主动下架')
  uni.showToast({ title: '已下架', icon: 'none' })
  await loadRows()
}

async function submitDraft(item) {
  await submitResource(item.id, merchantId.value)
  uni.showToast({ title: '已提交审核', icon: 'none' })
  await loadRows()
}

async function repost(item) {
  await repostSimilarResource(item.id, merchantId.value)
  uni.showToast({ title: '已复制为草稿', icon: 'none' })
  await loadRows()
}

function canRepost(item) {
  return item.status === 'expired' || Boolean(item.dealtAt)
}

function canTopResource(item) {
  return item.status === 'published'
}

function expireText(item) {
  if (item.status === 'pending') return '审核中'
  if (item.status === 'expired') return '已过期'
  if (item.expiresAt) return `到期 ${item.expiresAt}`
  return '有效期待确认'
}

function openPublish() {
  uni.switchTab({ url: '/pages/publish/index' })
}

function openResource(item) {
  uni.navigateTo({ url: `/pages/resource/detail?id=${item.id}` })
}

function metricItems(item) {
  const metrics = item.metrics || {}
  return [
    { label: '曝光', value: metrics.exposureCount || 0 },
    { label: '浏览', value: metrics.detailViewCount || 0 },
    { label: '电话', value: metrics.phoneClickCount || 0 },
    { label: '微信', value: metrics.wechatCopyCount || 0 },
  ]
}

function effectAdvice(item) {
  if (item.status === 'pending') return '审核通过后会进入搜索、首页分类和商家主页。'
  if (item.status === 'published') return '可根据曝光和联系情况决定是否刷新或使用置顶券。'
  if (canRepost(item)) return '该资源已结束，可再发类似资源继续获取曝光。'
  return '保持信息完整有助于买家快速判断。'
}
</script>

<style lang="scss" scoped>
.my-resources-page {
  min-height: 100vh;
  padding: 24rpx;
  background: $wplink-bg;
}

.filter-row {
  width: 100%;
  margin-bottom: 20rpx;
  white-space: nowrap;
}

.resource-manager-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18rpx;
  margin-bottom: 20rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
  min-width: 0;
}

.manager-title {
  display: block;
  margin-bottom: 8rpx;
  color: $wplink-primary;
  font-size: 38rpx;
  font-weight: 700;
  line-height: 1.25;
}

.manager-desc {
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.5;
}

.resource-manager-head button {
  flex: 0 0 auto;
  width: 116rpx;
  height: 68rpx;
  border-radius: 10rpx;
  background: $wplink-primary;
  color: $wplink-card;
  font-size: 26rpx;
  font-weight: 700;
}

.lifecycle-note {
  margin-bottom: 20rpx;
  padding: 18rpx 20rpx;
  border-radius: 12rpx;
  background: $wplink-warning-soft;
}

.lifecycle-note text {
  color: #7c5a22;
  font-size: 24rpx;
  line-height: 1.5;
}

.filter-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 128rpx;
  height: 72rpx;
  margin-right: 12rpx;
  padding: 0 20rpx;
  border-radius: 10rpx;
  background: $wplink-card;
  color: #364152;
}

.filter-button.active {
  background: $wplink-warning-soft;
  color: $wplink-primary;
}

.resource-list {
  display: grid;
  gap: 18rpx;
}

.resource-card {
  display: grid;
  gap: 12rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
}

.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
  min-width: 0;
}

.tag-row {
  display: flex;
  flex-wrap: wrap;
  gap: 10rpx;
}

.resource-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
  line-height: 1.35;
  word-break: break-word;
}

.status-tag,
.top-tag {
  padding: 6rpx 12rpx;
  border-radius: 8rpx;
  background: #edf2f7;
  color: $wplink-primary;
  font-size: 24rpx;
}

.status-tag.published {
  background: $wplink-primary-soft;
}

.top-tag {
  background: $wplink-warning-soft;
  color: $wplink-warning;
}

.expire-text {
  flex: 0 0 auto;
  color: $wplink-muted;
  font-size: 24rpx;
}

.resource-meta {
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.5;
}

.effect-advice {
  padding: 14rpx 16rpx;
  border-radius: 10rpx;
  background: #f8fafc;
  color: #4b5565;
  font-size: 24rpx;
  line-height: 1.45;
}

.action-row {
  display: flex;
  flex-wrap: wrap;
  gap: 12rpx;
}

.action-row button {
  min-width: 120rpx;
  height: 68rpx;
  border-radius: 10rpx;
  background: #edf2f7;
  color: $wplink-primary;
  font-size: 26rpx;
  line-height: 1.25;
}
</style>
