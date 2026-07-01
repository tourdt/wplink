<template>
  <view class="verification-page">
    <view class="status-card">
      <text class="section-title">认证状态</text>
      <text class="status-text">{{ statusLabel(latestVerification.status) }}</text>
      <text class="status-meta" v-if="latestVerification.verificationType">
        {{ typeLabel(latestVerification.verificationType) }} · {{ latestVerification.reviewedAt || '等待审核' }}
      </text>
      <text class="status-meta">{{ billingSummary }}</text>
      <button v-if="canPayVerification" class="primary-button" :loading="paying" @click="payVerification">支付认证费</button>
    </view>

    <view class="form-card">
      <text class="page-title">商家认证</text>
      <input v-model="form.merchantId" class="field" placeholder="商家 ID" />
      <picker :range="typeOptions" range-key="label" @change="changeType">
        <view class="field picker-field">{{ currentTypeLabel }}</view>
      </picker>
      <input v-model="form.businessName" class="field" placeholder="营业主体名称" />
      <input v-model="form.licenseUrl" class="field" placeholder="营业执照图片 URL" />
      <button class="secondary-button" @click="uploadLicense">上传营业执照</button>
      <input v-model="form.storefrontUrl" class="field" placeholder="门头/场地图片 URL" />
      <button class="secondary-button" @click="uploadStorefront">上传门头/场地</button>
      <button class="primary-button" @click="submit">提交认证</button>
    </view>
  </view>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getMerchantId, getUserId } from '../../store/session'
import { createVerificationPayment, getLatestVerification, getVerificationBillingConfig, submitVerification } from '../../api/verification'
import { chooseAndUploadImage } from '../../common/upload'

const typeOptions = [
  { label: '工厂认证', value: 'factory' },
  { label: '档口认证', value: 'stall' },
  { label: '库存商认证', value: 'stockist' },
  { label: '服务商认证', value: 'service_provider' },
]

const form = reactive({
  merchantId: '',
  verificationType: 'factory',
  businessName: '',
  licenseUrl: '',
  storefrontUrl: '',
  materials: {},
})

const currentTypeLabel = computed(() => {
  const matched = typeOptions.find((item) => item.value === form.verificationType) || {}
  return matched.label || '工厂认证'
})
const latestVerification = ref({ status: 'none' })
const billingConfig = ref({ chargeEnabled: false, feeAmount: 0, currency: 'CNY', freeEnabled: false })
const paying = ref(false)
const billingSummary = computed(() => {
  if (!billingConfig.value.chargeEnabled) return '当前认证免费，审核通过后直接生效'
  const feeText = `认证费 ¥${(Number(billingConfig.value.feeAmount || 0) / 100).toFixed(2)}`
  if (isFreeWindowActive()) return `${feeText}，当前限时免费`
  if (latestVerification.value.status === 'payment_pending') return `${feeText}，支付成功后认证生效`
  return `${feeText}，资料审核通过后在线支付`
})
const canPayVerification = computed(() => {
  return latestVerification.value.status === 'payment_pending' && billingConfig.value.chargeEnabled && !isFreeWindowActive() && latestVerification.value.id
})

onLoad(async (options) => {
  form.merchantId = options.merchantId || getMerchantId()
  await Promise.all([loadLatestVerification(), loadBillingConfig()])
})

async function loadLatestVerification() {
  if (!form.merchantId) {
    latestVerification.value = { status: 'none' }
    return
  }
  try {
    latestVerification.value = await getLatestVerification(form.merchantId)
  } catch (err) {
    latestVerification.value = { status: 'none' }
  }
}

async function loadBillingConfig() {
  try {
    billingConfig.value = await getVerificationBillingConfig('zhili')
  } catch (err) {
    billingConfig.value = { chargeEnabled: false, feeAmount: 0, currency: 'CNY', freeEnabled: false }
  }
}

function changeType(event) {
  const selected = typeOptions[Number(event.detail.value)] || {}
  form.verificationType = selected.value || 'factory'
}

function typeLabel(type) {
  const matched = typeOptions.find((item) => item.value === type) || {}
  return matched.label || type || '-'
}

function statusLabel(status) {
  const statusText = {
    none: '未提交认证',
    pending: '审核中',
    payment_pending: '待支付',
    verified: '已认证',
    rejected: '未通过',
  }
  return statusText[status] || status || '未提交认证'
}

async function uploadLicense() {
  await uploadVerificationImage('license')
}

async function uploadStorefront() {
  await uploadVerificationImage('storefront')
}

async function uploadVerificationImage(kind) {
  try {
    // 认证材料上传成功后只提交 CDN URL，后台审核无需接收二进制文件。
    const url = await chooseAndUploadImage(`verification-${kind}`)
    if (kind === 'license') {
      form.licenseUrl = url
    } else {
      form.storefrontUrl = url
    }
    uni.showToast({ title: '图片已上传', icon: 'none' })
  } catch (err) {
    uni.showToast({ title: err.message || '图片上传失败，请重试', icon: 'none' })
  }
}

async function submit() {
  const userId = getUserId()
  if (!userId) {
    uni.showToast({ title: '请先在我的页保存用户 ID', icon: 'none' })
    return
  }
  if (!form.merchantId || !form.businessName) {
    uni.showToast({ title: '请填写商家和主体名称', icon: 'none' })
    return
  }
  // 认证申请需要记录提交人，便于后台审核留痕和后续消息通知。
  await submitVerification(form.merchantId, {
    applicantUserId: userId,
    verificationType: form.verificationType,
    businessName: form.businessName,
    licenseUrl: form.licenseUrl,
    storefrontUrl: form.storefrontUrl,
    materials: form.materials,
  })
  uni.showToast({ title: '认证已提交', icon: 'none' })
  await loadLatestVerification()
}

async function payVerification() {
  const userId = getUserId()
  if (!userId) {
    uni.showToast({ title: '请先登录后支付', icon: 'none' })
    return
  }
  paying.value = true
  try {
    // 小程序端只负责调起收银台，认证生效必须以后端收到微信支付成功通知为准。
    const resp = await createVerificationPayment(form.merchantId, latestVerification.value.id, { userId })
    const payment = resp.payment || {}
    await requestWechatPayment(payment)
    uni.showToast({ title: '支付成功，正在更新认证状态', icon: 'none' })
    await loadLatestVerification()
  } catch (err) {
    uni.showToast({ title: err.message || '支付未完成，请稍后重试', icon: 'none' })
  } finally {
    paying.value = false
  }
}

function requestWechatPayment(payment) {
  return new Promise((resolve, reject) => {
    uni.requestPayment({
      timeStamp: payment.timeStamp,
      nonceStr: payment.nonceStr,
      package: payment.package,
      signType: payment.signType || 'RSA',
      paySign: payment.paySign,
      success: resolve,
      fail: reject,
    })
  })
}

function isFreeWindowActive() {
  const config = billingConfig.value || {}
  if (!config.freeEnabled) return false
  const now = Date.now()
  const start = config.freeStartAt ? Date.parse(config.freeStartAt) : null
  const end = config.freeEndAt ? Date.parse(config.freeEndAt) : null
  if (start && now < start) return false
  if (end && now > end) return false
  return true
}
</script>

<style lang="scss" scoped>
.verification-page {
  min-height: 100vh;
  padding: 24rpx;
  background: $wplink-bg;
}

.status-card,
.form-card {
  display: grid;
  gap: 18rpx;
  margin-bottom: 20rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
}

.page-title {
  color: $wplink-primary;
  font-size: 36rpx;
  font-weight: 700;
}

.field {
  min-height: 80rpx;
  padding: 0 20rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 10rpx;
}

.section-title,
.status-meta {
  color: $wplink-muted;
  font-size: 26rpx;
}

.status-text {
  color: $wplink-primary;
  font-size: 34rpx;
  font-weight: 700;
}

.picker-field {
  display: flex;
  align-items: center;
  color: $wplink-primary;
}

.primary-button {
  height: 88rpx;
  border-radius: 12rpx;
  background: $wplink-primary;
  color: $wplink-card;
}

.secondary-button {
  height: 84rpx;
  border: 1rpx solid $wplink-primary;
  border-radius: 12rpx;
  background: $wplink-card;
  color: $wplink-primary;
}
</style>
