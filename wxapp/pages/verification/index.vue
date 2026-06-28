<template>
  <view class="verification-page">
    <view class="status-card">
      <text class="section-title">认证状态</text>
      <text class="status-text">{{ statusLabel(latestVerification.status) }}</text>
      <text class="status-meta" v-if="latestVerification.verificationType">
        {{ typeLabel(latestVerification.verificationType) }} · {{ latestVerification.reviewedAt || '等待审核' }}
      </text>
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
import { getLatestVerification, submitVerification } from '../../api/verification'
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

const currentTypeLabel = computed(() => typeOptions.find((item) => item.value === form.verificationType)?.label || '工厂认证')
const latestVerification = ref({ status: 'none' })

onLoad(async (options) => {
  form.merchantId = options.merchantId || getMerchantId()
  await loadLatestVerification()
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

function changeType(event) {
  form.verificationType = typeOptions[Number(event.detail.value)]?.value || 'factory'
}

function typeLabel(type) {
  return typeOptions.find((item) => item.value === type)?.label || type || '-'
}

function statusLabel(status) {
  const statusText = {
    none: '未提交认证',
    pending: '审核中',
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
</script>

<style scoped>
.verification-page {
  min-height: 100vh;
  padding: 24rpx;
  background: #f4f6f8;
}

.status-card,
.form-card {
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

.section-title,
.status-meta {
  color: #697586;
  font-size: 26rpx;
}

.status-text {
  color: #0f766e;
  font-size: 34rpx;
  font-weight: 700;
}

.picker-field {
  display: flex;
  align-items: center;
  color: #1f2933;
}

.primary-button {
  height: 88rpx;
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
</style>
