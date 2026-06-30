<template>
  <view class="my-resources-page">
    <view class="filter-row">
      <button
        v-for="item in statusOptions"
        :key="item.value"
        :class="['filter-button', filters.status === item.value ? 'active' : '']"
        @click="selectStatus(item.value)"
      >
        {{ item.label }}
      </button>
    </view>

    <view v-if="!loading && rows.length === 0" class="empty-state">
      <text class="empty-title">暂无发布资源</text>
      <text class="empty-desc">发布资源后，可在这里查看审核进度、曝光数据和推广效果。</text>
      <button class="empty-action" @click="openPublish">继续发布</button>
    </view>

    <view v-else class="resource-list">
      <view v-for="item in rows" :key="item.id" class="resource-card">
        <view class="card-head">
          <view class="tag-row">
            <text :class="['status-tag', statusClass(item)]">{{ displayStatusText(item) }}</text>
            <text v-if="isExpiringSoon(item)" class="expire-tag">即将过期</text>
            <text v-if="canTopResource(item)" class="top-tag">可置顶</text>
          </view>
          <text class="expire-text">{{ expireText(item) }}</text>
        </view>
        <view class="resource-summary">
          <view class="resource-thumb-wrap">
            <image class="resource-thumb" :src="item.coverUrl || DEFAULT_RESOURCE_COVER" mode="aspectFill" @error="handleResourceCoverError(item)" />
          </view>
          <view class="resource-body">
            <text class="resource-title">{{ item.title }}</text>
            <text class="resource-meta">{{ item.category }} · {{ displayResourceTypeText(item) }}</text>
            <text v-if="shouldShowResourceDates(item)" class="resource-meta">发布 {{ displayDateOrPlaceholder(item.publishedAt) }} · 到期 {{ displayDateOrPlaceholder(item.expiresAt) }}</text>
          </view>
        </view>
        <text v-if="item.status === 'rejected' && item.rejectReason" class="reject-reason">驳回原因：{{ item.rejectReason }}</text>
        <MetricStrip :items="metricItems(item)" />
        <view class="action-row">
          <button v-if="isActivePublished(item)" class="primary-action" @click="refresh(item)">刷新</button>
          <button v-if="isActivePublished(item)" @click="topResource(item)">置顶</button>
          <button v-if="isActivePublished(item)" @click="takeDown(item)">下架</button>
          <button v-if="item.status === 'draft'" class="primary-action" @click="openDraftEditor(item)">编辑</button>
          <button v-if="item.status === 'rejected'" class="primary-action" @click="openRejectedEditor(item)">编辑</button>
          <button v-if="canRepost(item)" class="primary-action" @click="repost(item)">再发类似</button>
          <button v-if="canDeleteTakenDown(item)" class="danger-button" @click="deleteTakenDown(item)">删除</button>
          <button @click="openResource(item)">详情</button>
        </view>
      </view>
      <text v-if="rows.length" class="load-more-text">{{ loading ? '加载中...' : hasMore ? '上拉加载更多' : '没有更多了' }}</text>
    </view>

    <button class="publish-fab" @click="openPublish">发布</button>
  </view>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { onLoad, onPullDownRefresh, onReachBottom } from '@dcloudio/uni-app'
import MetricStrip from '../../components/MetricStrip.vue'
import { getMerchantId } from '../../store/session'
import { redeemTopVoucher, listTopVouchers } from '../../api/entitlement'
import { deleteTakenDownResource, getOwnResource, listMyResources, refreshResource, takeDownResource } from '../../api/resource'
import { formatDateToDay } from '../../common/date'
import { resourceTypeText } from '../../common/enums'

const DEFAULT_RESOURCE_COVER = '/static/resource/default-resource-cover.png'

const statusOptions = [
  { label: '全部', value: '' },
  { label: '待跟进', value: 'needs_action' },
  { label: '展示中', value: 'showing' },
  { label: '已结束', value: 'ended' },
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
const page = ref(1)
const pageSize = 20
const total = ref(0)
const hasMore = ref(true)
const loading = ref(false)

onLoad((options) => {
  // 我的发布必须绑定当前商家；路由参数用于后台调试，正常用户流程使用我的页保存的商家 ID。
  merchantId.value = options.merchantId || getMerchantId()
  loadRows({ reset: true })
})

onPullDownRefresh(async () => {
  try {
    await loadRows({ reset: true })
  } finally {
    uni.stopPullDownRefresh()
  }
})

onReachBottom(() => {
  loadRows({ reset: false })
})

async function loadRows({ reset = true } = {}) {
  if (loading.value) return
  if (!merchantId.value) {
    rows.value = []
    total.value = 0
    hasMore.value = false
    uni.showToast({ title: '请先填写商家 ID', icon: 'none' })
    return
  }
  if (!reset && !hasMore.value) return
  loading.value = true
  try {
    const nextPage = reset ? 1 : page.value + 1
    const resp = await listMyResources({ merchantId: merchantId.value, status: filters.status, page: nextPage, pageSize })
    const items = resp.items || []
    rows.value = reset ? items : [...rows.value, ...items]
    page.value = nextPage
    total.value = resp.total || rows.value.length
    hasMore.value = rows.value.length < total.value
  } finally {
    loading.value = false
  }
}

function selectStatus(status) {
  filters.status = status
  loadRows({ reset: true })
}

async function refresh(item) {
  await refreshResource(item.id, merchantId.value)
  uni.showToast({ title: '已刷新', icon: 'none' })
  await loadRows({ reset: true })
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
  await loadRows({ reset: true })
}

async function takeDown(item) {
  await takeDownResource(item.id, merchantId.value, '商家主动下架')
  uni.showToast({ title: '已下架', icon: 'none' })
  await loadRows({ reset: true })
}

function openDraftEditor(item) {
  openPublishEditor(item)
}

function openRejectedEditor(item) {
  openPublishEditor(item)
}

function openPublishEditor(item) {
  uni.navigateTo({ url: `/pages/publish/edit?merchantId=${merchantId.value}&resourceId=${item.id}` })
}

async function repost(item) {
  const detail = await getOwnResource(item.id, merchantId.value)
  const repostInitialForm = buildRepostInitialForm(detail)
  uni.setStorageSync('publish:repost-initial-form', repostInitialForm)
  uni.navigateTo({ url: `/pages/publish/edit?merchantId=${merchantId.value}&repost=1` })
}

function buildRepostInitialForm(detail) {
  return {
    merchantId: merchantId.value,
    cityCode: detail.cityCode || 'zhili',
    typeCode: detail.typeCode || '',
    title: detail.title || '',
    category: detail.category || '',
    quantityText: detail.quantityText || '',
    priceText: detail.priceText || '',
    description: detail.description || '',
    attributes: detail.attributes || {},
    tags: detail.tags || [],
    images: detail.images || [],
    contact: {
      name: detail.contact?.name || '',
      phone: detail.contact?.phone || detail.contact?.phoneMasked || '',
      wechat: detail.contact?.wechat || detail.contact?.wechatMasked || '',
    },
  }
}

async function deleteTakenDown(item) {
  // 已下架资源删除后会从我的发布隐藏，保留后台统计和审计数据。
  const confirmed = await new Promise((resolve) => {
    uni.showModal({
      title: '删除资源',
      content: '删除后将不再显示在我的发布中，确认删除吗？',
      confirmText: '删除',
      confirmColor: '#c2410c',
      success: (res) => resolve(Boolean(res.confirm)),
      fail: () => resolve(false),
    })
  })
  if (!confirmed) return
  await deleteTakenDownResource(item.id, merchantId.value)
  uni.showToast({ title: '已删除', icon: 'none' })
  await loadRows({ reset: true })
}

function canRepost(item) {
  return isExpiredResource(item) || Boolean(item.dealtAt)
}

function canDeleteTakenDown(item) {
  return item.status === 'taken_down'
}

function isActivePublished(item) {
  return item.status === 'published' && !item.dealtAt
}

function isExpiringSoon(item) {
  if (!isActivePublished(item) || isExpiredResource(item) || !item.expiresAt) return false
  const expiresAt = Date.parse(item.expiresAt)
  if (Number.isNaN(expiresAt)) return false
  const now = Date.now()
  return expiresAt > now && expiresAt - now <= 3 * 24 * 60 * 60 * 1000
}

function isExpiredResource(item) {
  if (item.status === 'expired') return true
  if (!item.expiresAt) return false
  const expiresAt = Date.parse(item.expiresAt)
  return !Number.isNaN(expiresAt) && expiresAt <= Date.now()
}

function canTopResource(item) {
  return isActivePublished(item)
}

function displayStatusText(item) {
  if (item.dealtAt) return '已成交'
  if (isExpiredResource(item)) return '已过期'
  return statusText[item.status] || item.status
}

function statusClass(item) {
  if (item.dealtAt) return 'dealt'
  if (isExpiredResource(item)) return 'expired'
  return item.status
}

function displayResourceTypeText(item) {
  return resourceTypeText[item.typeCode] || item.typeCode || '资源'
}

function shouldShowResourceDates(item) {
  return Boolean(item.publishedAt || item.expiresAt)
}

function displayDateOrPlaceholder(value) {
  return value ? formatDateToDay(value) : '-'
}

function handleResourceCoverError(item) {
  item.coverUrl = ''
}

function expireText(item) {
  if (item.status === 'pending') return '审核中'
  if (item.dealtAt) return `成交 ${formatDateToDay(item.dealtAt)}`
  if (isExpiredResource(item)) return '已过期'
  if (item.expiresAt) return `到期 ${formatDateToDay(item.expiresAt)}`
  return '有效期待确认'
}

function openPublish() {
  uni.navigateTo({ url: `/pages/publish/edit?merchantId=${merchantId.value}` })
}

function openResource(item) {
  uni.navigateTo({ url: `/pages/resource/detail?id=${item.id}&merchantId=${merchantId.value}&from=my-resources` })
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

</script>

<style lang="scss" scoped>
.my-resources-page {
  min-height: 100vh;
  padding: 24rpx;
  padding-top: 132rpx;
  padding-bottom: calc(128rpx + env(safe-area-inset-bottom));
  overflow-x: hidden;
  background: $wplink-bg;
}

.filter-row {
  position: fixed;
  top: 0;
  right: 0;
  left: 0;
  z-index: 10;
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12rpx;
  padding: 24rpx 24rpx 16rpx;
  overflow: hidden;
  background: $wplink-card;
  box-shadow: 0 8rpx 20rpx rgba(15, 23, 42, 0.06);
}

.publish-fab {
  position: fixed;
  right: 24rpx;
  bottom: calc(32rpx + env(safe-area-inset-bottom));
  z-index: 20;
  width: 132rpx;
  height: 76rpx;
  border-radius: 999rpx;
  background: $wplink-primary;
  color: $wplink-card;
  font-size: 26rpx;
  font-weight: 700;
  box-shadow: 0 12rpx 28rpx rgba(6, 22, 37, 0.18);
}

.filter-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-width: 0;
  height: 72rpx;
  padding: 0 8rpx;
  border: 2rpx solid transparent;
  border-radius: 10rpx;
  background: #f4f7fd;
  color: #364152;
  font-size: 25rpx;
  line-height: 1.2;
  white-space: nowrap;
  transition: background 0.18s ease, color 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease;
}

.filter-button.active {
  border-color: $wplink-primary;
  background: $wplink-primary;
  color: $wplink-card;
  font-weight: 700;
  box-shadow: 0 8rpx 18rpx rgba(194, 58, 0, 0.18);
}

.resource-list {
  display: grid;
  gap: 18rpx;
}

.empty-state {
  display: grid;
  align-content: center;
  justify-items: center;
  gap: 16rpx;
  min-height: 360rpx;
  padding: 40rpx 28rpx;
  border-radius: 12rpx;
  background: $wplink-card;
  text-align: center;
}

.empty-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
  line-height: 1.35;
}

.empty-desc {
  max-width: 520rpx;
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.5;
}

.empty-action {
  min-width: 180rpx;
  height: 72rpx;
  border-radius: 10rpx;
  background: $wplink-primary;
  color: $wplink-card;
  font-size: 26rpx;
  font-weight: 700;
}

.load-more-text {
  padding: 8rpx 0 18rpx;
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.5;
  text-align: center;
}

.resource-card {
  display: grid;
  gap: 12rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
}

.resource-summary {
  display: grid;
  grid-template-columns: 112rpx minmax(0, 1fr);
  gap: 16rpx;
  align-items: start;
}

.resource-thumb-wrap {
  width: 112rpx;
  height: 112rpx;
  overflow: hidden;
  border-radius: 10rpx;
  background: #edf2f7;
}

.resource-thumb {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 112rpx;
  height: 112rpx;
}

.resource-body {
  display: grid;
  gap: 8rpx;
  min-width: 0;
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
.expire-tag,
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

.status-tag.dealt {
  background: $wplink-warning-soft;
  color: $wplink-warning;
}

.expire-tag {
  background: #fff7ed;
  color: #c2410c;
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

.reject-reason {
  padding: 12rpx 14rpx;
  border-radius: 8rpx;
  background: #fff7ed;
  color: #9a3412;
  font-size: 24rpx;
  line-height: 1.45;
}

.action-row {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 10rpx;
  padding-top: 14rpx;
  border-top: 1rpx solid #eef2f7;
}

.action-row button {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 108rpx;
  height: 62rpx;
  margin: 0;
  padding: 0 22rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 10rpx;
  background: $wplink-card;
  color: $wplink-primary;
  font-size: 24rpx;
  font-weight: 600;
  line-height: 1.25;
  box-shadow: 0 2rpx 4rpx rgba(15, 23, 42, 0.04);
  transition: transform 0.16s ease, box-shadow 0.16s ease, background 0.16s ease;
}

.action-row button::after {
  border: 0;
}

.action-row button:active {
  transform: translateY(1rpx);
  background: #f4f7fd;
  box-shadow: 0 1rpx 2rpx rgba(15, 23, 42, 0.06);
}

.action-row .primary-action {
  border-color: #b8c4d4;
  background: $wplink-primary-soft;
  color: $wplink-primary;
  box-shadow: 0 2rpx 4rpx rgba(6, 22, 37, 0.05);
}

.action-row .danger-button {
  border-color: #fecdd3;
  background: #fff8f8;
  color: #be123c;
  box-shadow: 0 2rpx 4rpx rgba(190, 18, 60, 0.04);
}
</style>
