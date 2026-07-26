<template>
  <view class="settings-page">
    <view class="settings-card">
      <view class="settings-item" @click="openLegal('agreement')">
        <text>用户协议</text>
        <text class="arrow">›</text>
      </view>
      <view class="settings-item" @click="openLegal('privacy')">
        <text>隐私政策</text>
        <text class="arrow">›</text>
      </view>
      <button class="settings-item service-item" open-type="contact">
        <text>联系客服与申诉</text>
        <text class="arrow">›</text>
      </button>
    </view>

    <button class="secondary-button" @click="logout">退出登录</button>
    <button class="danger-button" :disabled="deleting" @click="confirmDeleteAccount">
      {{ deleting ? '正在注销...' : '注销账号' }}
    </button>
    <text class="danger-hint">注销后无法恢复；个人资料将匿名化，必要的安全和交易记录依法保留。</text>
  </view>
</template>

<script setup>
import { ref } from 'vue'
import { deleteAccount } from '../../api/auth'
import { clearSession } from '../../store/session'

const deleting = ref(false)

function openLegal(type) {
  uni.navigateTo({ url: `/pages/legal/index?type=${type}` })
}

function logout() {
  clearSession()
  uni.showToast({ title: '已退出登录', icon: 'none' })
  setTimeout(() => uni.reLaunch({ url: '/pages/home/index' }), 300)
}

function confirmDeleteAccount() {
  if (deleting.value) return
  uni.showModal({
    title: '确认注销账号？',
    content: '注销后账号会立即停用，发布内容停止管理，个人资料将匿名化且无法恢复。',
    confirmText: '继续注销',
    confirmColor: '#b42318',
    success: (result) => {
      if (result.confirm) deleteCurrentAccount()
    },
  })
}

async function deleteCurrentAccount() {
  try {
    deleting.value = true
    await deleteAccount({ confirmation: '确认注销', reason: '用户主动申请' })
    clearSession()
    uni.showToast({ title: '账号已注销', icon: 'none' })
    setTimeout(() => uni.reLaunch({ url: '/pages/home/index' }), 500)
  } catch (err) {
    uni.showToast({ title: err.message || '注销失败，请稍后重试', icon: 'none' })
  } finally {
    deleting.value = false
  }
}
</script>

<style lang="scss" scoped>
.settings-page {
  min-height: 100vh;
  padding: 24rpx;
  background: $wplink-bg;
}

.settings-card {
  overflow: hidden;
  margin-bottom: 28rpx;
  border-radius: 12rpx;
  background: $wplink-card;
}

.settings-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  min-height: 92rpx;
  padding: 0 28rpx;
  border-bottom: 1rpx solid $wplink-line;
  border-radius: 0;
  background: transparent;
  color: $wplink-primary;
  font-size: 28rpx;
  text-align: left;
}

.settings-item:last-child {
  border-bottom: 0;
}

.service-item::after {
  border: 0;
}

.arrow {
  color: $wplink-muted;
  font-size: 36rpx;
}

.secondary-button,
.danger-button {
  height: 84rpx;
  margin-top: 18rpx;
  border-radius: 12rpx;
  font-size: 28rpx;
  line-height: 84rpx;
}

.secondary-button {
  border: 1rpx solid $wplink-line;
  background: $wplink-card;
  color: $wplink-primary;
}

.danger-button {
  border: 1rpx solid #fecdca;
  background: #fff5f4;
  color: #b42318;
}

.danger-hint {
  display: block;
  padding: 18rpx 10rpx;
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.6;
}
</style>
