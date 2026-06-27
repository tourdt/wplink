<template>
  <view class="my-page">
    <view class="profile-card">
      <text class="page-title">我的</text>
      <input v-model="merchantId" class="field" placeholder="商家 ID" />
      <button class="primary-button" @click="saveMerchant">保存商家</button>
    </view>

    <view class="action-list">
      <view class="action-item" @click="openMyResources">
        <text>我的发布</text>
        <text class="action-meta">资源状态和效果数据</text>
      </view>
      <view class="action-item" @click="openVerification">
        <text>商家认证</text>
        <text class="action-meta">认证状态和提交资料</text>
      </view>
      <view class="action-item" @click="openPublish">
        <text>发布资源</text>
        <text class="action-meta">新增库存、货源、工厂或服务</text>
      </view>
    </view>
  </view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getMerchantId, saveMerchantId } from '../../store/session'

const merchantId = ref('')

onLoad(() => {
  merchantId.value = getMerchantId()
})

function saveMerchant() {
  if (!merchantId.value.trim()) {
    uni.showToast({ title: '请填写商家 ID', icon: 'none' })
    return
  }
  saveMerchantId(merchantId.value.trim())
  uni.showToast({ title: '已保存', icon: 'none' })
}

function openMyResources() {
  uni.navigateTo({ url: `/pages/my-resources/index?merchantId=${merchantId.value}` })
}

function openVerification() {
  uni.navigateTo({ url: '/pages/verification/index' })
}

function openPublish() {
  uni.switchTab({ url: '/pages/publish/index' })
}
</script>

<style scoped>
.my-page {
  min-height: 100vh;
  padding: 24rpx;
  background: #f4f6f8;
}

.profile-card,
.action-list {
  display: grid;
  gap: 18rpx;
  margin-bottom: 20rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: #ffffff;
}

.page-title {
  color: #1f2933;
  font-size: 36rpx;
  font-weight: 700;
}

.field {
  min-height: 80rpx;
  padding: 0 20rpx;
  border: 1rpx solid #d8dde6;
  border-radius: 10rpx;
}

.primary-button {
  height: 84rpx;
  border-radius: 12rpx;
  background: #0f766e;
  color: #ffffff;
}

.action-item {
  display: grid;
  gap: 8rpx;
  padding: 18rpx 0;
  border-bottom: 1rpx solid #edf2f7;
  color: #1f2933;
  font-size: 32rpx;
}

.action-item:last-child {
  border-bottom: 0;
}

.action-meta {
  color: #697586;
  font-size: 26rpx;
}
</style>
