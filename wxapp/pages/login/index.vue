<template>
  <view class="login-page">
    <view class="login-card">
      <view class="brand-mark">衣</view>
      <text class="login-title">衣货通</text>
      <text class="login-desc">登录后同步收藏、消息和发布记录</text>
      <view class="policy-row" @click="agreedToPolicies = !agreedToPolicies">
        <view :class="['policy-check', { checked: agreedToPolicies }]">{{ agreedToPolicies ? '✓' : '' }}</view>
        <text class="policy-text">我已阅读并同意</text>
        <text class="policy-link" @click.stop="openLegal('agreement')">《用户协议》</text>
        <text class="policy-text">和</text>
        <text class="policy-link" @click.stop="openLegal('privacy')">《隐私政策》</text>
      </view>
      <button class="login-button" :disabled="loggingIn" :loading="loggingIn" @click="loginWithWechatAccount">
        {{ loggingIn ? '登录中' : '微信登录' }}
      </button>
    </view>
  </view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import {
  DEFAULT_CITY_CODE,
  PRIVACY_POLICY_VERSION,
  USER_AGREEMENT_VERSION,
} from '../../common/constants'
import { getMe, wechatLogin } from '../../api/auth'
import { saveMerchantId, saveToken, saveUserId } from '../../store/session'

const TAB_PAGE_PATHS = ['/pages/home/index', '/pages/market/index', '/pages/publish/index', '/pages/messages/index', '/pages/my/index']

const redirectUrl = ref('/pages/my/index')
const loggingIn = ref(false)
const agreedToPolicies = ref(false)

onLoad((options = {}) => {
  redirectUrl.value = safeDecode(options.redirect) || '/pages/my/index'
})

async function loginWithWechatAccount() {
  if (loggingIn.value) return
  if (!agreedToPolicies.value) {
    uni.showToast({ title: '请先阅读并同意用户协议和隐私政策', icon: 'none' })
    return
  }
  try {
    loggingIn.value = true
    const code = await getWechatLoginCode()
    const resp = await wechatLogin({
      code,
      defaultCityCode: DEFAULT_CITY_CODE,
      agreedToPolicies: true,
      privacyPolicyVersion: PRIVACY_POLICY_VERSION,
      userAgreementVersion: USER_AGREEMENT_VERSION,
    })
    if (resp.token) {
      saveToken(resp.token)
    }
    const loginUser = resp.user || {}
    if (loginUser.id) {
      saveUserId(loginUser.id)
    }
    await restoreManagedMerchantId(resp)
    uni.showToast({ title: '登录成功', icon: 'none' })
    goAfterLogin()
  } catch (err) {
    uni.showToast({ title: err.message || '登录失败，请稍后重试', icon: 'none' })
  } finally {
    loggingIn.value = false
  }
}

function openLegal(type) {
  uni.navigateTo({ url: `/pages/legal/index?type=${type}` })
}

async function restoreManagedMerchantId(loginResp = {}) {
  try {
    let managedMerchants = loginResp.managedMerchants
    if (!Array.isArray(managedMerchants)) {
      const me = await getMe()
      managedMerchants = me.managedMerchants || []
    }
    const managedMerchant = managedMerchants[0]
    if (managedMerchant?.id) {
      saveMerchantId(managedMerchant.id)
      return
    }
    // 当前账号没有可管理商家时必须清理旧缓存，避免换号后继续用上一个账号的商家 ID 请求业务接口。
    saveMerchantId('')
  } catch (err) {
    // 登录主链路已成功，商户身份恢复失败时保持未配置状态，避免阻断用户进入。
    saveMerchantId('')
  }
}

function goAfterLogin() {
  const targetUrl = redirectUrl.value || '/pages/my/index'
  if (isTabPage(targetUrl)) {
    uni.switchTab({ url: stripQuery(targetUrl) })
    return
  }
  uni.redirectTo({ url: targetUrl })
}

function isTabPage(url) {
  return TAB_PAGE_PATHS.includes(stripQuery(url))
}

function stripQuery(url) {
  return url.split('?')[0]
}

function safeDecode(value) {
  if (!value) return ''
  try {
    return decodeURIComponent(value)
  } catch (err) {
    return ''
  }
}

function getWechatLoginCode() {
  return new Promise((resolve, reject) => {
    uni.login({
      provider: 'weixin',
      success: (res) => {
        if (res.code) {
          resolve(res.code)
          return
        }
        reject(new Error('未获取到微信登录凭证，请重试'))
      },
      fail: () => {
        reject(new Error('微信登录暂不可用，请稍后重试'))
      },
    })
  })
}
</script>

<style lang="scss" scoped>
.login-page {
  min-height: 100vh;
  padding: 32rpx 24rpx;
  background: $wplink-bg;
}

.login-card {
  display: grid;
  gap: 18rpx;
  justify-items: center;
  padding: 48rpx 28rpx;
  border-radius: 12rpx;
  background: $wplink-card;
  text-align: center;
}

.brand-mark {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 96rpx;
  height: 96rpx;
  border-radius: 18rpx;
  background: $wplink-primary;
  color: $wplink-card;
  font-size: 40rpx;
  font-weight: 700;
}

.login-title {
  color: $wplink-primary;
  font-size: 40rpx;
  font-weight: 700;
}

.login-desc {
  color: $wplink-muted;
  font-size: 28rpx;
  line-height: 1.5;
}

.policy-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 4rpx;
  width: 100%;
  margin-top: 8rpx;
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.7;
}

.policy-check {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30rpx;
  height: 30rpx;
  margin-right: 6rpx;
  border: 2rpx solid $wplink-line;
  border-radius: 6rpx;
  color: #fff;
  font-size: 22rpx;
}

.policy-check.checked {
  border-color: $wplink-accent;
  background: $wplink-accent;
}

.policy-link {
  color: $wplink-accent;
}

.login-button {
  width: 100%;
  height: 88rpx;
  margin-top: 10rpx;
  border-radius: 12rpx;
  background: $wplink-primary;
  color: $wplink-card;
  font-size: 30rpx;
  font-weight: 700;
}
</style>
