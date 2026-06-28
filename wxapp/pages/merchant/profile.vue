<template>
  <view class="profile-page">
    <view class="form-card">
      <text class="page-title">商家资料</text>
      <input v-model="form.name" class="field" :disabled="Boolean(merchantId)" placeholder="商家名称" />
      <picker :range="merchantTypeOptions" range-key="label" :disabled="Boolean(merchantId)" @change="changeMerchantType">
        <view class="field picker-field">{{ currentMerchantTypeLabel }}</view>
      </picker>
      <input v-model="mainCategoriesText" class="field" placeholder="主营品类，多个用逗号分隔" />
      <input v-model="form.contactName" class="field" :disabled="Boolean(merchantId)" placeholder="联系人" />
      <input v-model="form.contactPhone" class="field" :disabled="Boolean(merchantId)" type="number" placeholder="联系电话" />
      <input v-model="form.contactWechat" class="field" :disabled="Boolean(merchantId)" placeholder="微信号" />
      <input v-model="form.addressText" class="field" :disabled="Boolean(merchantId)" placeholder="地址" />
      <textarea v-model="form.description" class="textarea" placeholder="商家介绍" />
      <button class="secondary-button" :disabled="uploadingImage" @click="uploadMerchantImage">上传商家图片</button>
      <view v-if="imageUrls.length > 0" class="image-list">
        <view v-for="url in imageUrls" :key="url" class="image-item">
          <image class="merchant-image" :src="url" mode="aspectFill" />
          <button class="remove-button" @click="removeMerchantImage(url)">移除</button>
        </view>
      </view>
      <textarea v-model="imagesText" class="textarea" placeholder="也可粘贴图片 URL，多个用逗号分隔" />
      <button class="primary-button" :disabled="submitting" @click="submitMerchantProfile">
        {{ merchantId ? '保存资料' : '提交入驻' }}
      </button>
    </view>
  </view>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import { createMerchant, getMerchant, updateMerchant } from '../../api/merchant'
import { chooseAndUploadImage } from '../../common/upload'
import { getMerchantId, saveMerchantId } from '../../store/session'

const merchantTypeOptions = [
  { label: '工厂', value: 'factory' },
  { label: '档口', value: 'stall' },
  { label: '库存商', value: 'stockist' },
  { label: '服务商', value: 'service_provider' },
]

const merchantId = ref('')
const submitting = ref(false)
const uploadingImage = ref(false)
const mainCategoriesText = ref('')
const imagesText = ref('')
const imageUrls = computed(() => parseList(imagesText.value))
const form = reactive({
  cityCode: DEFAULT_CITY_CODE,
  name: '',
  merchantType: 'factory',
  contactName: '',
  contactPhone: '',
  contactWechat: '',
  addressText: '',
  description: '',
})

const currentMerchantTypeLabel = computed(() => {
  const matched = merchantTypeOptions.find((item) => item.value === form.merchantType) || {}
  return matched.label || '工厂'
})

onLoad((options) => {
  merchantId.value = options.merchantId || getMerchantId()
  loadMerchant()
})

async function loadMerchant() {
  if (!merchantId.value) return
  try {
    const detail = await getMerchant(merchantId.value)
    form.name = detail.name || ''
    form.cityCode = detail.cityCode || DEFAULT_CITY_CODE
    form.merchantType = detail.merchantType || 'factory'
    const contact = detail.contact || {}
    form.contactName = contact.name || ''
    form.contactPhone = contact.phoneMasked || ''
    form.contactWechat = contact.wechatMasked || ''
    form.description = detail.description || ''
    mainCategoriesText.value = (detail.mainCategories || []).join(',')
    imagesText.value = (detail.images || []).join(',')
  } catch (err) {
    uni.showToast({ title: err.message || '商家资料加载失败', icon: 'none' })
  }
}

function changeMerchantType(event) {
  const selected = merchantTypeOptions[Number(event.detail.value)] || {}
  form.merchantType = selected.value || 'factory'
}

async function submitMerchantProfile() {
  const mainCategories = parseList(mainCategoriesText.value)
  const images = imageUrls.value
  if (!form.name.trim()) {
    uni.showToast({ title: '请填写商家名称', icon: 'none' })
    return
  }
  if (mainCategories.length === 0) {
    uni.showToast({ title: '请填写主营品类', icon: 'none' })
    return
  }
  if (!merchantId.value && (!form.contactName.trim() || !form.contactPhone.trim())) {
    uni.showToast({ title: '请填写联系人和电话', icon: 'none' })
    return
  }
  try {
    submitting.value = true
    if (merchantId.value) {
      await updateMerchant(merchantId.value, {
        mainCategories,
        description: form.description.trim(),
        images,
      })
    } else {
      const resp = await createMerchant({
        cityCode: form.cityCode || DEFAULT_CITY_CODE,
        name: form.name.trim(),
        merchantType: form.merchantType,
        mainCategories,
        contactName: form.contactName.trim(),
        contactPhone: form.contactPhone.trim(),
        contactWechat: form.contactWechat.trim(),
        addressText: form.addressText.trim(),
        description: form.description.trim(),
      })
      merchantId.value = resp.id
      saveMerchantId(resp.id)
      if (images.length > 0 || form.description.trim()) {
        await updateMerchant(resp.id, {
          mainCategories,
          description: form.description.trim(),
          images,
        })
      }
    }
    uni.showToast({ title: '商家资料已保存', icon: 'none' })
  } catch (err) {
    uni.showToast({ title: err.message || '商家资料保存失败', icon: 'none' })
  } finally {
    submitting.value = false
  }
}

async function uploadMerchantImage() {
  try {
    uploadingImage.value = true
    // 商家主页图片使用统一直传链路，保存公开 URL，后台和小程序详情页都只读取 URL。
    const url = await chooseAndUploadImage('merchant-profile')
    const nextImages = [...imageUrls.value, url]
    imagesText.value = nextImages.join(',')
    uni.showToast({ title: '图片已上传', icon: 'none' })
  } catch (err) {
    uni.showToast({ title: err.message || '图片上传失败，请重试', icon: 'none' })
  } finally {
    uploadingImage.value = false
  }
}

function removeMerchantImage(url) {
  imagesText.value = imageUrls.value.filter((item) => item !== url).join(',')
}

function parseList(value) {
  return String(value || '')
    .split(/[,，\n]/)
    .map((item) => item.trim())
    .filter(Boolean)
}
</script>

<style scoped>
.profile-page {
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

.field,
.textarea {
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

.textarea {
  min-height: 144rpx;
  padding: 20rpx;
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

.image-list {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12rpx;
}

.image-item {
  display: grid;
  gap: 8rpx;
}

.merchant-image {
  width: 100%;
  height: 160rpx;
  border-radius: 10rpx;
  background: #e3e8ef;
}

.remove-button {
  height: 56rpx;
  border-radius: 8rpx;
  background: #f8fafc;
  color: #697586;
  font-size: 24rpx;
  line-height: 56rpx;
}
</style>
