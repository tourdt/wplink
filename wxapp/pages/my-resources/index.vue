<template>
  <view class="my-resources-page">
    <view class="resource-manager-head">
      <view>
        <text class="manager-title">我的发布</text>
        <text class="manager-desc">管理资源状态和推广效果</text>
      </view>
      <button @click="openPublish">发布</button>
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
            <text :class="['status-tag', statusClass(item)]">{{ displayStatusText(item) }}</text>
            <text v-if="isExpiringSoon(item)" class="expire-tag">即将过期</text>
            <text v-if="canTopResource(item)" class="top-tag">可置顶</text>
          </view>
          <text class="expire-text">{{ expireText(item) }}</text>
        </view>
        <text class="resource-title">{{ item.title }}</text>
        <text class="resource-meta">{{ item.category }} · {{ item.typeCode }}</text>
        <text class="resource-meta">发布 {{ formatDateToDay(item.publishedAt) }} · 到期 {{ formatDateToDay(item.expiresAt) }}</text>
        <text v-if="item.status === 'rejected' && item.rejectReason" class="reject-reason">驳回原因：{{ item.rejectReason }}</text>
        <MetricStrip :items="metricItems(item)" />
        <view class="action-row">
          <button v-if="isActivePublished(item)" @click="refresh(item)">刷新</button>
          <button v-if="isActivePublished(item)" @click="topResource(item)">置顶</button>
          <button v-if="isActivePublished(item)" @click="takeDown(item)">下架</button>
          <button v-if="item.status === 'draft'" @click="openDraftEditor(item)">编辑</button>
          <button v-if="item.status === 'rejected'" @click="openRejectedEditor(item)">编辑</button>
          <button v-if="canRepost(item)" @click="repost(item)">再发类似</button>
          <button v-if="canDeleteTakenDown(item)" class="danger-button" @click="deleteTakenDown(item)">删除</button>
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
import { deleteTakenDownResource, getOwnResource, listMyResources, refreshResource, takeDownResource } from '../../api/resource'
import { formatDateToDay } from '../../common/date'

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

async function takeDown(item) {
  await takeDownResource(item.id, merchantId.value, '商家主动下架')
  uni.showToast({ title: '已下架', icon: 'none' })
  await loadRows()
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
  await loadRows()
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

function expireText(item) {
  if (item.status === 'pending') return '审核中'
  if (item.dealtAt) return `成交 ${formatDateToDay(item.dealtAt)}`
  if (isExpiredResource(item)) return '已过期'
  if (item.expiresAt) return `到期 ${formatDateToDay(item.expiresAt)}`
  return '有效期待确认'
}

function openPublish() {
  uni.switchTab({ url: '/pages/publish/index' })
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

.action-row .danger-button {
  background: #fff1f2;
  color: #be123c;
}
</style>
