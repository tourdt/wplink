<template>
  <view class="messages-page">
    <view class="filter-row">
      <button
        v-for="item in messageTabs"
        :key="item.label"
        :class="['filter-button', filters.status === item.status ? 'active' : '']"
        @click="selectStatusFromTab(item)"
      >
        {{ item.label }}
      </button>
    </view>

    <view class="message-list">
      <view v-for="item in rows" :key="item.id" :class="['message-card', item.status === 'read' ? 'read' : 'unread']" @click="openMessageTarget(item)">
        <text :class="['message-dot', item.status === 'read' ? 'read' : '']"></text>
        <view class="card-head">
          <text class="message-title">{{ item.title }}</text>
          <text class="status-tag">{{ item.status === 'read' ? '已读' : '未读' }}</text>
        </view>
        <text class="message-content">{{ item.content }}</text>
        <text class="message-time">{{ item.createdAt }}</text>
        <text v-if="canOpenTarget(item)" class="target-hint">查看详情</text>
      </view>
      <view v-if="!loading && !rows.length" class="empty-placeholder">
        <text class="empty-title">{{ emptyTitle }}</text>
      </view>
      <text v-if="rows.length" class="load-more-text">{{ loading ? '加载中...' : hasMore ? '上拉加载更多' : '没有更多了' }}</text>
    </view>

  </view>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { onLoad, onPullDownRefresh, onReachBottom, onShow } from '@dcloudio/uni-app'
import { getSession } from '../../store/session'
import { listMessages, readMessage } from '../../api/message'

const rows = ref([])
const userId = ref('')
const roleCode = ref('')
const filters = reactive({ status: '' })
const page = ref(1)
const pageSize = 20
const total = ref(0)
const hasMore = ref(true)
const loading = ref(false)
const messageTabs = [
  { label: '全部', status: '' },
  { label: '未读', status: 'unread' },
  { label: '已读', status: 'read' },
]
const resourceMessageTypes = new Set([
  'resource_review',
  'resource_lifecycle',
  'resource_expired',
  'resource_expiring',
  'effect_feedback',
])
const tabPagePaths = ['/pages/home/index', '/pages/search/index', '/pages/publish/index', '/pages/messages/index', '/pages/my/index']
const emptyTitle = computed(() => {
  if (filters.status === 'unread') return '暂无未读消息'
  if (filters.status === 'read') return '暂无已读消息'
  return '暂无消息'
})

onLoad((options) => {
  const session = getSession()
  userId.value = options.userId || session.userId
  roleCode.value = options.roleCode || (session.merchantId ? `merchant:${session.merchantId}` : '')
})

onShow(() => {
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
  if (!reset && !hasMore.value) return
  loading.value = true
  try {
    const nextPage = reset ? 1 : page.value + 1
    const resp = await listMessages({
      userId: userId.value,
      roleCode: roleCode.value,
      status: filters.status,
      page: nextPage,
      pageSize,
    })
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

function selectStatusFromTab(item) {
  if (item.status === 'unread') {
    selectStatus('unread')
    return
  }
  if (item.status === 'read') {
    selectStatus('read')
    return
  }
  selectStatus('')
}

async function markRead(item) {
  if (item.status === 'read' || !userId.value) return
  await readMessage(item.id, userId.value, roleCode.value)
  item.status = 'read'
}

async function openMessageTarget(item) {
  try {
    await markRead(item)
  } catch (err) {
    uni.showToast({ title: err.message || '消息已读状态更新失败', icon: 'none' })
  }
  const targetUrl = normalizeTargetUrl(buildMessageTargetUrl(item))
  if (!targetUrl) return
  if (isTabPage(targetUrl)) {
    uni.switchTab({ url: stripQuery(targetUrl) })
    return
  }
  uni.navigateTo({ url: targetUrl })
}

function buildMessageTargetUrl(item) {
  if (resourceMessageTypes.has(item.messageType) && item.triggerId) {
    const merchantId = getMessageMerchantId(item)
    if (!merchantId) return item.targetUrl
    return `/pages/resource/detail?id=${encodeURIComponent(item.triggerId)}&merchantId=${encodeURIComponent(merchantId)}&from=my-resources`
  }
  return item.targetUrl
}

function getMessageMerchantId(item) {
  if (roleCode.value.startsWith('merchant:')) return roleCode.value.slice('merchant:'.length)
  return getQueryParam(item.targetUrl, 'merchantId')
}

function getQueryParam(targetUrl, key) {
  const query = String(targetUrl || '').split('?')[1] || ''
  const pairs = query.split('&').filter(Boolean)
  for (const pair of pairs) {
    const [rawKey, rawValue = ''] = pair.split('=')
    if (decodeURIComponent(rawKey) === key) return decodeURIComponent(rawValue)
  }
  return ''
}

function normalizeTargetUrl(targetUrl) {
  const url = String(targetUrl || '').trim()
  if (!url || !url.startsWith('/pages/')) return ''
  if (stripQuery(url) === '/pages/my-demands/index') return ''
  return url
}

function canOpenTarget(item) {
  return Boolean(normalizeTargetUrl(buildMessageTargetUrl(item)))
}

function isTabPage(targetUrl) {
  return tabPagePaths.includes(stripQuery(targetUrl))
}

function stripQuery(targetUrl) {
  return targetUrl.split('?')[0]
}

</script>

<style lang="scss" scoped>
.messages-page {
  min-height: 100vh;
  padding: 24rpx;
  padding-top: 132rpx;
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
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12rpx;
  padding: 24rpx 24rpx 16rpx;
  overflow: hidden;
  background: $wplink-card;
  box-shadow: 0 8rpx 20rpx rgba(15, 23, 42, 0.06);
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

.message-list {
  display: grid;
  gap: 18rpx;
  margin-bottom: 20rpx;
}

.load-more-text {
  padding: 8rpx 0 18rpx;
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.5;
  text-align: center;
}

.empty-placeholder {
  display: flex;
  min-height: 360rpx;
  align-items: center;
  justify-content: center;
  border-radius: 12rpx;
  background: $wplink-card;
}

.empty-title {
  color: $wplink-muted;
  font-size: 28rpx;
  line-height: 1.5;
}

.message-card {
  position: relative;
  display: grid;
  gap: 10rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
}

.message-card.unread {
  border: 1rpx solid $wplink-primary-soft;
}

.message-card.read {
  opacity: 0.86;
}

.message-dot {
  position: absolute;
  top: 30rpx;
  left: 20rpx;
  width: 12rpx;
  height: 12rpx;
  border-radius: 50%;
  background: $wplink-warning;
}

.message-dot.read {
  background: $wplink-primary;
}

.card-head {
  display: flex;
  justify-content: space-between;
  gap: 18rpx;
  padding-left: 18rpx;
  min-width: 0;
}

.message-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
  line-height: 1.35;
  min-width: 0;
  word-break: break-word;
}

.status-tag {
  color: $wplink-primary;
  font-size: 24rpx;
}

.message-content,
.message-time {
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.5;
}

.target-hint {
  color: $wplink-primary;
  font-size: 24rpx;
}

</style>
