<template>
  <view class="messages-page">
    <view class="message-hero">
      <text class="hero-title">消息和效果</text>
      <text class="hero-desc">关注审核、过期、需求跟进和资源表现，及时处理会影响资源曝光。</text>
    </view>

    <scroll-view class="filter-row" scroll-x>
      <button
        v-for="item in messageTabs"
        :key="item.label"
        :class="['filter-button', activeMessageTab === item.label ? 'active' : '']"
        @click="selectMessageTab(item)"
      >
        {{ item.label }}
      </button>
    </scroll-view>

    <view class="message-list">
      <view v-for="item in rows" :key="item.id" :class="['message-card', item.status === 'read' ? 'read' : 'unread']" @click="openMessageTarget(item)">
        <text :class="['message-dot', item.status === 'read' ? 'read' : '']"></text>
        <view class="card-head">
          <text class="message-title">{{ item.title }}</text>
          <text class="status-tag">{{ item.status === 'read' ? '已读' : '未读' }}</text>
        </view>
        <text class="message-content">{{ item.content }}</text>
        <text class="message-time">{{ item.createdAt }}</text>
        <text v-if="item.targetUrl" class="target-hint">查看详情</text>
      </view>
    </view>

    <view class="effect-card">
      <text class="effect-title">商家本周效果</text>
      <view class="effect-grid">
        <view class="effect-item">
          <text class="effect-value">386</text>
          <text class="effect-label">曝光</text>
        </view>
        <view class="effect-item">
          <text class="effect-value">52</text>
          <text class="effect-label">浏览</text>
        </view>
        <view class="effect-item">
          <text class="effect-value">9</text>
          <text class="effect-label">联系</text>
        </view>
      </view>
      <text class="effect-tip">联系率高的资源可刷新或使用置顶券延长曝光。</text>
      <button @click="openMyResources">查看我的资源</button>
    </view>
  </view>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { getSession } from '../../store/session'
import { listMessages, readMessage } from '../../api/message'

const rows = ref([])
const userId = ref('')
const roleCode = ref('')
const activeMessageTab = ref('全部')
const filters = reactive({ status: '' })
const messageTabs = [
  { label: '全部', status: '' },
  { label: '未读', status: 'unread', action: () => selectStatus('unread') },
  { label: '已读', status: 'read', action: () => selectStatus('read') },
  { label: '审核', status: '' },
  { label: '效果', status: '' },
]
const tabPagePaths = ['/pages/home/index', '/pages/search/index', '/pages/publish/index', '/pages/messages/index', '/pages/my/index']

onLoad((options) => {
  const session = getSession()
  userId.value = options.userId || session.userId
  roleCode.value = options.roleCode || (session.merchantId ? `merchant:${session.merchantId}` : '')
})

onShow(loadRows)

async function loadRows() {
  const resp = await listMessages({
    userId: userId.value,
    roleCode: roleCode.value,
    status: filters.status,
    page: 1,
    pageSize: 30,
  })
  rows.value = resp.items || []
}

function selectStatus(status) {
  filters.status = status
  loadRows()
}

function selectMessageTab(item) {
  activeMessageTab.value = item.label
  if (item.action) {
    item.action()
    return
  }
  selectStatus(item.status)
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
  const targetUrl = normalizeTargetUrl(item.targetUrl)
  if (!targetUrl) return
  if (isTabPage(targetUrl)) {
    uni.switchTab({ url: stripQuery(targetUrl) })
    return
  }
  uni.navigateTo({ url: targetUrl })
}

function normalizeTargetUrl(targetUrl) {
  const url = String(targetUrl || '').trim()
  if (!url || !url.startsWith('/pages/')) return ''
  return url
}

function isTabPage(targetUrl) {
  return tabPagePaths.includes(stripQuery(targetUrl))
}

function stripQuery(targetUrl) {
  return targetUrl.split('?')[0]
}

function openMyResources() {
  uni.navigateTo({ url: '/pages/my-resources/index' })
}
</script>

<style scoped>
.messages-page {
  min-height: 100vh;
  padding: 24rpx;
  background: #f4f6f8;
}

.message-hero {
  display: grid;
  gap: 8rpx;
  margin-bottom: 20rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background:
    linear-gradient(135deg, rgba(15, 118, 110, 0.08), rgba(183, 121, 31, 0.08)),
    #ffffff;
}

.hero-title {
  color: #1f2933;
  font-size: 36rpx;
  font-weight: 700;
}

.hero-desc {
  color: #697586;
  font-size: 26rpx;
  line-height: 1.5;
}

.filter-row {
  width: 100%;
  margin-bottom: 20rpx;
  white-space: nowrap;
}

.filter-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 112rpx;
  height: 72rpx;
  margin-right: 12rpx;
  border-radius: 10rpx;
  background: #ffffff;
  color: #364152;
  font-size: 26rpx;
}

.filter-button.active {
  background: #d9f3ef;
  color: #0f766e;
}

.message-list {
  display: grid;
  gap: 18rpx;
  margin-bottom: 20rpx;
}

.message-card {
  position: relative;
  display: grid;
  gap: 10rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: #ffffff;
}

.message-card.unread {
  border: 1rpx solid #e6f4f1;
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
  background: #dc6b4a;
}

.message-dot.read {
  background: #0f766e;
}

.card-head {
  display: flex;
  justify-content: space-between;
  gap: 18rpx;
  padding-left: 18rpx;
}

.message-title {
  color: #1f2933;
  font-size: 32rpx;
  font-weight: 700;
}

.status-tag {
  color: #0f766e;
  font-size: 24rpx;
}

.message-content,
.message-time {
  color: #697586;
  font-size: 26rpx;
  line-height: 1.5;
}

.target-hint {
  color: #0f766e;
  font-size: 24rpx;
}

.effect-card {
  display: grid;
  gap: 16rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: #ffffff;
}

.effect-title {
  color: #1f2933;
  font-size: 32rpx;
  font-weight: 700;
}

.effect-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12rpx;
}

.effect-item {
  display: grid;
  gap: 6rpx;
  padding: 18rpx;
  border-radius: 10rpx;
  background: #f8fafc;
  text-align: center;
}

.effect-value {
  color: #1f2933;
  font-size: 34rpx;
  font-weight: 700;
}

.effect-label,
.effect-tip {
  color: #697586;
  font-size: 24rpx;
  line-height: 1.5;
}

.effect-card button {
  height: 80rpx;
  border-radius: 10rpx;
  background: #e6f4f1;
  color: #0f766e;
  font-size: 28rpx;
  font-weight: 700;
}
</style>
