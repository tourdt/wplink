<template>
  <view class="messages-page">
    <view class="filter-row">
      <button :class="['filter-button', filters.status === '' ? 'active' : '']" @click="selectStatus('')">全部</button>
      <button :class="['filter-button', filters.status === 'unread' ? 'active' : '']" @click="selectStatus('unread')">未读</button>
      <button :class="['filter-button', filters.status === 'read' ? 'active' : '']" @click="selectStatus('read')">已读</button>
    </view>

    <view class="message-list">
      <view v-for="item in rows" :key="item.id" class="message-card" @click="openMessageTarget(item)">
        <view class="card-head">
          <text class="message-title">{{ item.title }}</text>
          <text class="status-tag">{{ item.status === 'read' ? '已读' : '未读' }}</text>
        </view>
        <text class="message-content">{{ item.content }}</text>
        <text class="message-time">{{ item.createdAt }}</text>
        <text v-if="item.targetUrl" class="target-hint">查看详情</text>
      </view>
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
const filters = reactive({ status: '' })
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
</script>

<style scoped>
.messages-page {
  min-height: 100vh;
  padding: 24rpx;
  background: #f4f6f8;
}

.filter-row {
  display: flex;
  gap: 12rpx;
  margin-bottom: 20rpx;
}

.filter-button {
  min-width: 112rpx;
  height: 72rpx;
  border-radius: 10rpx;
  background: #ffffff;
  color: #364152;
}

.filter-button.active {
  background: #d9f3ef;
  color: #0f766e;
}

.message-list {
  display: grid;
  gap: 18rpx;
}

.message-card {
  display: grid;
  gap: 10rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: #ffffff;
}

.card-head {
  display: flex;
  justify-content: space-between;
  gap: 12rpx;
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
</style>
