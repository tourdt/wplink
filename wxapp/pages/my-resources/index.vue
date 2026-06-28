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

    <view class="resource-list">
      <view v-for="item in rows" :key="item.id" class="resource-card">
        <view class="card-head">
          <text class="resource-title">{{ item.title }}</text>
          <text class="status-tag">{{ statusText[item.status] || item.status }}</text>
        </view>
        <text class="resource-meta">{{ item.category }} · {{ item.typeCode }}</text>
        <text class="resource-meta">发布 {{ item.publishedAt || '-' }} · 到期 {{ item.expiresAt || '-' }}</text>
        <MetricStrip :items="metricItems(item)" />
        <view class="action-row">
          <button v-if="item.status === 'published'" @click="refresh(item)">刷新</button>
          <button v-if="item.status === 'published'" @click="topResource(item)">置顶</button>
          <button v-if="item.status === 'published'" @click="markDealt(item)">成交</button>
          <button v-if="item.status === 'published'" @click="takeDown(item)">下架</button>
          <button v-if="canRepost(item)" @click="repost(item)">再发类似</button>
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
import { listMyResources, markResourceDeal, refreshResource, repostSimilarResource, takeDownResource } from '../../api/resource'

const statusOptions = [
  { label: '全部', value: '' },
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
  await redeemTopVoucher(voucher.id, item.id)
  uni.showToast({ title: '已置顶', icon: 'none' })
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

async function repost(item) {
  await repostSimilarResource(item.id, merchantId.value)
  uni.showToast({ title: '已复制为草稿', icon: 'none' })
  await loadRows()
}

function canRepost(item) {
  return item.status === 'expired' || Boolean(item.dealtAt)
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

<style scoped>
.my-resources-page {
  min-height: 100vh;
  padding: 24rpx;
  background: #f4f6f8;
}

.filter-row {
  display: flex;
  gap: 12rpx;
  margin-bottom: 20rpx;
  overflow-x: auto;
}

.filter-button {
  min-width: 128rpx;
  height: 72rpx;
  padding: 0 20rpx;
  border-radius: 10rpx;
  background: #ffffff;
  color: #364152;
}

.filter-button.active {
  background: #d9f3ef;
  color: #0f766e;
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
  background: #ffffff;
}

.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
}

.resource-title {
  color: #1f2933;
  font-size: 32rpx;
  font-weight: 700;
}

.status-tag {
  color: #0f766e;
  font-size: 24rpx;
}

.resource-meta {
  color: #697586;
  font-size: 26rpx;
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
  color: #1f2933;
  font-size: 26rpx;
}
</style>
