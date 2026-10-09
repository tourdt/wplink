<template>
  <view class="my-page">
    <view class="page-heading">
      <text class="eyebrow">我的业务</text>
      <text class="page-title">让客户了解你的业务</text>
      <text class="page-intro">管理业务资料、主页公开和地图展示。</text>
    </view>

    <view v-if="pageState === 'guest'" class="state-card">
      <text class="state-title">登录后管理业务主页</text>
      <text class="state-copy">浏览无需登录；商家和服务者可登录入驻、管理业务主页。</text>
      <button class="primary-button" @click="openLogin">微信登录</button>
    </view>
    <view v-else-if="pageState === 'loading'" class="state-card" aria-live="polite">
      <text class="state-title">正在读取业务资料…</text>
    </view>
    <view v-else-if="pageState === 'error'" class="state-card" aria-live="polite">
      <text class="state-title">业务资料暂时无法读取</text>
      <text class="state-copy">请检查网络后重新加载业务资料。</text>
      <button class="primary-button" @click="loadMerchantProfile">重新加载</button>
    </view>
    <view v-else-if="pageState === 'unregistered' || pageState === 'incomplete'" class="state-card">
      <text class="state-title">{{ pageState === 'incomplete' ? '继续完善资料' : '还没有建立业务主页' }}</text>
      <text class="state-copy">选择业务身份，填写对应资料。有固定接待地点时，可另外开启地图展示。</text>
      <text v-if="hasLocalDraft" class="draft-note">这台设备有尚未提交的草稿。</text>
      <button class="primary-button" @click="openMerchantEditor">{{ pageState === 'incomplete' ? '继续填写' : '开始入驻' }}</button>
    </view>
    <view v-else class="store-card">
      <view class="store-topline">
        <text class="eyebrow">我的业务主页</text>
        <text class="status-pill" :class="pageState">{{ pageState === 'paused' ? '主页已暂停' : pageState === 'profile_only' ? '主页已公开' : pageState === 'location_error' ? '地图位置待处理' : '地图展示中' }}</text>
      </view>
      <view class="store-identity">
        <image v-if="merchantProfile.logoUrl" class="store-logo" :src="merchantProfile.logoUrl" mode="aspectFill" />
        <view v-else class="store-logo store-initial">{{ merchantInitial }}</view>
        <view class="store-name-block">
          <text class="store-name">{{ merchantProfile.name || '我的业务主页' }}</text>
          <text v-if="merchantProfile.merchantNo" class="store-number" @click="copyMerchantNo">编号 {{ merchantProfile.merchantNo }} · 复制</text>
        </view>
      </view>
      <view class="store-feature">
        <text class="feature-label">{{ businessRoleText || '经营业务' }}</text>
        <text class="feature-value">{{ categoryText }}</text>
      </view>
      <text v-if="businessProfile.serviceArea" class="store-position">服务区域：{{ businessProfile.serviceArea }}</text>
      <text v-if="place.position" class="store-position">{{ place.position }}</text>
      <text v-if="pageState === 'paused'" class="status-message">主页目前仅你可预览，公开后其他人才能查看。</text>
      <text v-else-if="pageState === 'profile_only'" class="status-message">主页已公开，未展示地图地点。客户可通过分享链接了解业务。</text>
      <text v-else-if="pageState === 'location_error'" class="status-message">主页可以访问，地图位置待处理。请编辑资料检查接待地点。</text>
      <view class="store-actions">
        <button class="primary-button" @click="openMerchantEditor">编辑资料</button>
        <button class="outline-button" @click="openMerchantHome">{{ pageState === 'paused' ? '预览主页' : '查看主页' }}</button>
      </view>
      <view class="store-secondary-actions">
        <button v-if="pageState !== 'paused'" class="text-button" open-type="share">分享主页</button>
        <button class="text-button" :disabled="visibilityBusy" :loading="visibilityBusy" @click="toggleVisibility">{{ pageState === 'paused' ? '公开主页' : '暂停主页' }}</button>
      </view>
    </view>

    <view v-if="pageState !== 'guest'" class="account-section">
      <text class="section-label">账号</text>
      <view class="account-row" @click="openAccountSettings">
        <text>账号与隐私</text><text class="chevron">›</text>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShow, onShareAppMessage } from '@dcloudio/uni-app'
import { buildLoginUrl } from '../../common/auth'
import { getMerchant, setMerchantVisibility } from '../../api/merchant'
import { getMerchantDraft } from '../../common/merchantDraft'
import { getMyStorefrontState, getStorefrontPlace } from '../../common/merchantStorefront'
import { getBusinessProfile, getBusinessRoleLabels, getBusinessProfileSummary } from '../../common/merchantBusinessProfile'
import { getSession } from '../../store/session'

const token = ref('')
const userId = ref('')
const merchantId = ref('')
const merchantProfile = ref({})
const loading = ref(false)
const loadError = ref(false)
const visibilityBusy = ref(false)
const requestId = ref(0)
const hasLocalDraft = ref(false)
const pageState = computed(() => getMyStorefrontState({ token: token.value, merchantId: merchantId.value, loading: loading.value, error: loadError.value, merchant: merchantProfile.value }))
const place = computed(() => getStorefrontPlace(merchantProfile.value))
const merchantInitial = computed(() => String(merchantProfile.value.name || '店').slice(0, 1))
const businessProfile = computed(() => getBusinessProfile(merchantProfile.value))
const businessRoleText = computed(() => getBusinessRoleLabels(businessProfile.value).join(' · '))
const categoryText = computed(() => getBusinessProfileSummary(merchantProfile.value) || '业务资料待补充')

onLoad(syncSession)
onShow(syncSession)

function syncSession() {
  const session = getSession()
  token.value = session.token
  userId.value = session.userId
  merchantId.value = session.merchantId
  hasLocalDraft.value = Boolean(session.token && session.userId && getMerchantDraft(session.userId, session.merchantId))
  return loadMerchantProfile()
}

async function loadMerchantProfile() {
  const id = ++requestId.value
  if (!token.value || !merchantId.value) {
    merchantProfile.value = {}
    loadError.value = false
    loading.value = false
    return
  }
  loading.value = true
  loadError.value = false
  try {
    const detail = await getMerchant(merchantId.value, { suppressErrorToast: true })
    if (id !== requestId.value) return
    if (!detail?.id) throw new Error('业务资料不完整')
    merchantProfile.value = detail || {}
  } catch (error) {
    if (id !== requestId.value) return
    merchantProfile.value = {}
    loadError.value = true
  } finally {
    if (id === requestId.value) loading.value = false
  }
}

function openLogin() {
  uni.navigateTo({ url: buildLoginUrl('/pages/my/index') })
}

function openMerchantEditor() {
  if (!token.value) return openLogin()
  const query = merchantId.value ? `?merchantId=${encodeURIComponent(merchantId.value)}` : ''
  uni.navigateTo({ url: `/pages/merchant/profile${query}` })
}

function openMerchantHome() {
  if (!token.value || !merchantId.value || !['active', 'profile_only', 'paused', 'location_error'].includes(pageState.value)) return
  uni.navigateTo({ url: `/pages/merchant/detail?id=${encodeURIComponent(merchantId.value)}` })
}

async function toggleVisibility() {
  if (!token.value || !merchantId.value || visibilityBusy.value || !['active', 'profile_only', 'paused', 'location_error'].includes(pageState.value)) return
  const visible = pageState.value === 'paused'
  visibilityBusy.value = true
  try {
    await setMerchantVisibility(merchantId.value, visible)
    await loadMerchantProfile()
    if (loadError.value) return
    uni.showToast({ title: visible ? '已公开主页' : '已暂停主页', icon: 'none' })
  } catch (error) {
    uni.showToast({ title: error.message || '展示状态修改失败，请重试', icon: 'none' })
  } finally {
    visibilityBusy.value = false
  }
}

function copyMerchantNo() {
  if (!merchantProfile.value.merchantNo) return
  uni.setClipboardData({ data: merchantProfile.value.merchantNo })
}

function openAccountSettings() {
  if (!token.value) return openLogin()
  uni.navigateTo({ url: '/pages/account/settings' })
}

onShareAppMessage(() => {
  if (!merchantId.value || !['active', 'profile_only', 'location_error'].includes(pageState.value)) {
    return { title: '找档口', path: '/pages/sourcing-map/index' }
  }
  return {
    title: merchantProfile.value.name || '业务主页',
    path: `/pages/merchant/detail?id=${encodeURIComponent(merchantId.value)}`,
    imageUrl: merchantProfile.value.logoUrl || undefined,
  }
})
</script>

<style lang="scss" scoped>
.my-page { min-height: 100vh; padding: 42rpx 28rpx 60rpx; background: #fff; color: #202020; }
.page-heading { display: grid; gap: 12rpx; margin-bottom: 42rpx; }
.eyebrow, .section-label { color: #c23a00; font-size: 22rpx; font-weight: 700; letter-spacing: 3rpx; }
.page-title { font-size: 50rpx; font-weight: 800; line-height: 1.15; }
.page-intro { max-width: 610rpx; color: #666; font-size: 26rpx; line-height: 1.55; }
.state-card, .store-card { padding: 32rpx; border: 1rpx solid #e9e6e3; border-radius: 20rpx; background: #fff; box-shadow: 0 12rpx 40rpx rgba(34, 27, 21, .05); }
.state-card { display: grid; gap: 22rpx; padding-top: 60rpx; padding-bottom: 56rpx; }
.state-title { font-size: 38rpx; font-weight: 750; }
.state-copy, .draft-note, .status-message { color: #666; font-size: 27rpx; line-height: 1.55; }
.draft-note { color: #a83200; }
button { margin: 0; }
button::after { border: 0; }
.primary-button, .outline-button { display: flex; align-items: center; justify-content: center; min-height: 84rpx; padding: 0 28rpx; border-radius: 10rpx; font-size: 27rpx; font-weight: 700; }
.primary-button { background: #c23a00; color: #fff; }
.outline-button { border: 1rpx solid #c23a00; background: #fff; color: #c23a00; }
.store-topline, .store-identity, .store-actions, .store-secondary-actions { display: flex; align-items: center; gap: 18rpx; }
.store-topline { justify-content: space-between; }
.status-pill { padding: 8rpx 16rpx; border-radius: 100rpx; background: #eef6f0; color: #2a7040; font-size: 22rpx; }
.status-pill.paused { background: #f4f1ef; color: #665d57; }
.status-pill.location_error { background: #fff0e9; color: #b53a08; }
.store-identity { margin: 34rpx 0; }
.store-logo { width: 112rpx; height: 112rpx; border-radius: 14rpx; background: #f4e6df; }
.store-initial { display: flex; align-items: center; justify-content: center; color: #c23a00; font-size: 44rpx; font-weight: 800; }
.store-name-block { display: grid; gap: 8rpx; min-width: 0; }
.store-name { font-size: 38rpx; font-weight: 800; line-height: 1.25; }
.store-number { color: #777; font-size: 23rpx; }
.store-feature { display: grid; gap: 8rpx; padding: 25rpx 0; border-top: 1rpx solid #eee; border-bottom: 1rpx solid #eee; }
.feature-label { color: #777; font-size: 22rpx; }
.feature-value { color: #c23a00; font-size: 35rpx; font-weight: 800; line-height: 1.4; }
.store-position { display: block; padding: 22rpx 0; color: #292929; font-size: 28rpx; font-weight: 700; }
.status-message { display: block; margin-bottom: 25rpx; padding: 20rpx; background: #fff5ef; color: #873518; }
.store-actions { margin-top: 25rpx; }
.store-actions button { flex: 1; }
.store-secondary-actions { justify-content: space-between; margin-top: 18rpx; }
.text-button { min-height: 66rpx; padding: 0 8rpx; background: transparent; color: #c23a00; font-size: 25rpx; }
.account-section { display: grid; gap: 10rpx; margin-top: 56rpx; }
.account-row { display: flex; justify-content: space-between; align-items: center; min-height: 100rpx; border-bottom: 1rpx solid #eaeaea; font-size: 29rpx; }
.chevron { color: #999; font-size: 40rpx; }
</style>
