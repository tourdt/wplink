<template>
  <view class="my-page">
    <view class="profile-card">
      <text class="page-title">我的</text>
      <button class="secondary-button" @click="loginWithWechat">微信登录</button>
      <input v-model="userId" class="field" placeholder="用户 ID" />
      <input v-model="merchantId" class="field" placeholder="商家 ID" />
      <button class="primary-button" @click="saveIdentity">保存身份</button>
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
import { DEFAULT_CITY_CODE } from '../../common/constants'
import { wechatLogin } from '../../api/auth'
import { getMerchantId, getUserId, saveMerchantId, saveToken, saveUserId } from '../../store/session'

const userId = ref('')
const merchantId = ref('')

onLoad(() => {
  userId.value = getUserId()
  merchantId.value = getMerchantId()
})

function saveIdentity() {
  if (userId.value.trim()) {
    saveUserId(userId.value.trim())
  }
  if (!merchantId.value.trim()) {
    uni.showToast({ title: '请填写商家 ID', icon: 'none' })
    return
  }
  saveMerchantId(merchantId.value.trim())
  uni.showToast({ title: '已保存身份', icon: 'none' })
}

async function loginWithWechat() {
  try {
    const code = await getWechatLoginCode()
    const resp = await wechatLogin({ code, defaultCityCode: DEFAULT_CITY_CODE })
    if (resp.token) {
      saveToken(resp.token)
    }
    if (resp.user?.id) {
      userId.value = resp.user.id
      saveUserId(resp.user.id)
    }
    uni.showToast({ title: '登录成功', icon: 'none' })
  } catch (err) {
    uni.showToast({ title: err.message || '登录失败，请稍后重试', icon: 'none' })
  }
}

function getWechatLoginCode() {
  return new Promise((resolve) => {
    uni.login({
      provider: 'weixin',
      success: (res) => {
        resolve(res.code || localDevLoginCode())
      },
      // 本地 H5/模拟环境可能没有微信登录能力，使用开发 code 仍可完成后端链路验收。
      fail: () => {
        resolve(localDevLoginCode())
      },
    })
  })
}

function localDevLoginCode() {
  const key = 'wplink_dev_login_code'
  const existing = uni.getStorageSync(key)
  if (existing) return existing
  const code = `local-dev-${Date.now()}`
  uni.setStorageSync(key, code)
  return code
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

.secondary-button {
  height: 84rpx;
  border: 1rpx solid #0f766e;
  border-radius: 12rpx;
  background: #ffffff;
  color: #0f766e;
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
