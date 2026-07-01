<template>
  <view class="my-demands-page">
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

    <view v-if="rows.length === 0" class="empty-state">
      <text class="empty-title">暂无采购需求</text>
      <text class="empty-desc">提交需求后，运营处理进展会在这里同步。</text>
      <button class="primary-button" @click="openDemand">提交需求</button>
    </view>

    <view v-else class="demand-list">
      <view v-for="item in rows" :key="item.id" class="demand-card">
        <view class="card-head">
          <text class="demand-title">{{ item.title }}</text>
          <text class="status-tag">{{ statusLabel(item.status) }}</text>
        </view>
        <text class="demand-meta">{{ typeLabel(item.demandType) }} · {{ item.category || '-' }}</text>
        <text class="demand-meta">联系人 {{ item.contactName || '-' }} · {{ formatDate(item.createdAt) }}</text>
      </view>
      <button class="secondary-button" @click="openMessages">查看消息</button>
      <button class="primary-button" @click="openDemand">继续提交需求</button>
    </view>
  </view>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { onLoad, onPullDownRefresh } from '@dcloudio/uni-app'
import { listMyDemands } from '../../api/demand'
import { resourceTypeText } from '../../common/enums'
import { getUserId } from '../../store/session'

const statusOptions = [
  { label: '全部', value: '' },
  { label: '待处理', value: 'pending' },
  { label: '跟进中', value: 'matching' },
  { label: '已联系', value: 'contacted' },
  { label: '已关闭', value: 'closed' },
]

const statusText = {
  pending: '待处理',
  matching: '跟进中',
  contacted: '已联系',
  closed: '已关闭',
}

const rows = ref([])
const filters = reactive({ status: '' })
const userId = ref('')

onLoad((options) => {
  userId.value = options.userId || getUserId()
  loadRows()
})

onPullDownRefresh(async () => {
  await loadRows()
  uni.stopPullDownRefresh()
})

async function loadRows() {
  if (!userId.value) {
    rows.value = []
    uni.showToast({ title: '请先登录', icon: 'none' })
    return
  }
  const resp = await listMyDemands({ userId: userId.value, status: filters.status, page: 1, pageSize: 20 })
  rows.value = resp.items || []
}

function selectStatus(status) {
  filters.status = status
  loadRows()
}

function statusLabel(status) {
  return statusText[status] || status || '-'
}

function typeLabel(type) {
  return resourceTypeText[type] || type || '-'
}

function formatDate(value) {
  if (!value) return '-'
  return String(value).slice(0, 10)
}

function openDemand() {
  uni.navigateTo({ url: '/pages/demand/index' })
}

function openMessages() {
  uni.switchTab({ url: '/pages/messages/index' })
}
</script>

<style lang="scss" scoped>
.my-demands-page {
  min-height: 100vh;
  padding: 24rpx;
  background: $wplink-bg;
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
  background: $wplink-card;
  color: #364152;
}

.filter-button.active {
  background: $wplink-warning-soft;
  color: $wplink-primary;
}

.empty-state,
.demand-card {
  display: grid;
  gap: 14rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
}

.empty-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
}

.empty-desc,
.demand-meta {
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.5;
}

.demand-list {
  display: grid;
  gap: 18rpx;
}

.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
}

.demand-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
}

.status-tag {
  color: $wplink-primary;
  font-size: 24rpx;
}

.primary-button,
.secondary-button {
  height: 84rpx;
  border-radius: 12rpx;
}

.primary-button {
  background: $wplink-primary;
  color: $wplink-card;
}

.secondary-button {
  background: #edf2f7;
  color: #364152;
}
</style>
