<template>
  <view class="verification-page">
    <view class="form-card">
      <text class="page-title">商家认证</text>
      <input v-model="form.merchantId" class="field" placeholder="商家 ID" />
      <picker :range="typeOptions" range-key="label" @change="changeType">
        <view class="field picker-field">{{ currentTypeLabel }}</view>
      </picker>
      <input v-model="form.businessName" class="field" placeholder="营业主体名称" />
      <input v-model="form.licenseUrl" class="field" placeholder="营业执照图片 URL" />
      <input v-model="form.storefrontUrl" class="field" placeholder="门头/场地图片 URL" />
      <button class="primary-button" @click="submit">提交认证</button>
    </view>
  </view>
</template>

<script setup>
import { computed, reactive } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getMerchantId } from '../../store/session'
import { submitVerification } from '../../api/verification'

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

onLoad((options) => {
  form.merchantId = options.merchantId || getMerchantId()
})

function changeType(event) {
  form.verificationType = typeOptions[Number(event.detail.value)]?.value || 'factory'
}

async function submit() {
  if (!form.merchantId || !form.businessName) {
    uni.showToast({ title: '请填写商家和主体名称', icon: 'none' })
    return
  }
  await submitVerification(form.merchantId, {
    verificationType: form.verificationType,
    businessName: form.businessName,
    licenseUrl: form.licenseUrl,
    storefrontUrl: form.storefrontUrl,
    materials: form.materials,
  })
  uni.showToast({ title: '认证已提交', icon: 'none' })
}
</script>

<style scoped>
.verification-page {
  min-height: 100vh;
  padding: 24rpx;
  background: #f4f6f8;
}

.form-card {
  display: grid;
  gap: 18rpx;
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
</style>
