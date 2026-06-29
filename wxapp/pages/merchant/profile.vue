<template>
  <view class="profile-page">
    <view class="form-card">
      <view class="form-section">
        <view class="section-heading">
          <view class="section-title-row">
            <text class="section-title">基础资料</text>
            <text class="required-badge">必填</text>
          </view>
        </view>
        <view class="section-body">
          <view class="form-field">
            <text class="field-label">商家名称</text>
            <input v-model="form.name" class="field" :disabled="Boolean(merchantId)" placeholder="请输入商家名称" />
          </view>
          <view class="form-field">
            <text class="field-label">商家类型</text>
            <picker :range="merchantTypeOptions" range-key="label" @change="changeMerchantType">
              <view class="field picker-field">{{ currentMerchantTypeLabel }}</view>
            </picker>
            <text v-if="merchantTypeChangeNeedsReverify" class="field-helper warning-helper">
              修改后可能需要重新认证，保存后需重新提交认证。
            </text>
          </view>
          <view class="form-field">
            <text class="field-label">主营品类</text>
            <input v-model="mainCategoriesText" class="field" placeholder="如：童装,面料" />
          </view>
          <view class="form-field">
            <text class="field-label">商家介绍</text>
            <textarea v-model="form.description" class="textarea" placeholder="主营资源、供货能力" />
          </view>
        </view>
      </view>

      <view class="form-section contact-section">
        <button class="section-toggle" @click="toggleContactSection">
          <view>
            <text class="section-title">联系方式</text>
            <text class="section-summary">买家联系和导航</text>
          </view>
          <text class="toggle-mark">{{ contactSectionOpen ? '收起' : '展开' }}</text>
        </button>
        <view v-if="contactSectionOpen" class="section-body">
          <view class="form-field">
            <text class="field-label">主页联系人</text>
            <input v-model="form.contactName" class="field" placeholder="联系人姓名" />
          </view>
          <view v-if="merchantId" class="form-field contact-phone-card">
            <text class="field-label">主页联系电话</text>
            <text v-if="contactPhoneHint" class="field-helper">当前：{{ contactPhoneHint }}，填新号可更换</text>
            <view class="sms-row">
              <input v-model="form.contactPhone" class="field" type="number" placeholder="新手机号" />
              <button class="sms-button" :disabled="smsSending || smsCountdown > 0" @click="sendSmsCodeForHomepagePhone">
                {{ smsCountdown > 0 ? `${smsCountdown}s` : '验证码' }}
              </button>
            </view>
            <view class="sms-code-field">
              <text class="field-label field-label-small">短信验证码</text>
              <input v-model="smsCode" class="field" type="number" placeholder="请输入短信验证码" />
            </view>
          </view>
          <view v-else class="form-field">
            <text class="field-label">主页联系电话</text>
            <input v-model="form.contactPhone" class="field" type="number" placeholder="联系电话" />
          </view>
          <view class="form-field">
            <text class="field-label">主页微信</text>
            <text v-if="merchantId && contactWechatHint" class="field-helper">当前：{{ contactWechatHint }}，填新微信可更换</text>
            <input v-model="form.contactWechat" class="field" :placeholder="contactWechatPlaceholder" />
          </view>
          <view class="form-field">
            <text class="field-label">商家地址</text>
            <view class="address-row">
              <input v-model="form.addressText" class="field" placeholder="请输入地址" />
              <button class="map-button" @click="chooseMerchantLocation">地图选择</button>
            </view>
            <view class="location-status">
              <text>{{ locationSelected ? '已选地图位置' : '可选地图定位' }}</text>
              <button v-if="locationSelected" class="location-clear-button" @click="clearMerchantLocation">清除位置</button>
            </view>
          </view>
        </view>
      </view>

      <view class="form-section brand-section">
        <button class="section-toggle" @click="toggleBrandSection">
          <view>
            <view class="section-title-row">
              <text class="section-title">品牌展示</text>
            </view>
            <text class="section-summary">头像、卡片、主页图</text>
          </view>
          <text class="toggle-mark">{{ brandSectionOpen ? '收起' : '展开' }}</text>
        </button>
        <view v-if="brandSectionOpen" class="section-body brand-fields">
          <view class="form-field logo-field">
            <view class="logo-layout">
              <view class="logo-copy">
                <text class="field-label">商家 LOGO</text>
                <text class="image-helper">正方形 LOGO，裁剪后保存</text>
              </view>
              <view v-if="logoPreviewUrl" class="logo-preview-wrap">
                <button class="logo-preview-tile" @click="previewMerchantLogo">
                  <image class="logo-preview" :src="logoPreviewUrl" mode="aspectFill" />
                </button>
                <button
                  class="logo-change-button"
                  :disabled="submitting"
                  open-type="chooseAvatar"
                  @chooseavatar="onChooseMerchantLogoAvatar"
                >
                  更换 LOGO
                </button>
              </view>
              <button
                v-else
                class="logo-upload-tile"
                :disabled="submitting"
                open-type="chooseAvatar"
                @chooseavatar="onChooseMerchantLogoAvatar"
              >
                <view class="logo-plus">
                  <view class="logo-plus-icon" />
                </view>
              </button>
            </view>
          </view>
          <view class="form-field image-field">
            <view class="image-title-row">
              <text class="field-label">商家主页图片</text>
              <button class="secondary-button" :disabled="submitting" @click="uploadMerchantImage">
                选择图片
              </button>
            </view>
            <text class="image-helper">主页展示图</text>
            <view v-if="merchantImageItems.length > 0" class="image-list">
              <view v-for="item in merchantImageItems" :key="item.id" class="image-item">
                <image class="merchant-image" :src="item.url" mode="aspectFill" />
                <button class="remove-button" @click="removeMerchantImage(item)">移除</button>
              </view>
            </view>
            <text v-else class="empty-image-text">暂无图片</text>
          </view>
        </view>
      </view>
    </view>
    <view class="fixed-save-spacer" />
    <view class="fixed-save-bar">
      <button class="primary-button" :disabled="submitting" @click="submitMerchantProfile">
        {{ saveButtonText }}
      </button>
    </view>
  </view>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { onLoad, onUnload } from '@dcloudio/uni-app'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import { sendSmsCode } from '../../api/auth'
import { createMerchant, getMerchant, updateMerchant } from '../../api/merchant'
import { getLatestVerification } from '../../api/verification'
import { chooseImageFile, createImageFileFromPath, uploadSelectedImage } from '../../common/upload'
import { getMerchantId, saveMerchantId } from '../../store/session'

const merchantTypeOptions = [
  { label: '工厂', value: 'factory' },
  { label: '档口', value: 'stall' },
  { label: '库存商', value: 'stockist' },
  { label: '服务商', value: 'service_provider' },
]

const merchantId = ref('')
const submitting = ref(false)
const contactSectionOpen = ref(false)
const brandSectionOpen = ref(false)
const smsSending = ref(false)
const smsCountdown = ref(0)
const smsCode = ref('')
const contactPhoneHint = ref('')
const contactWechatHint = ref('')
const originalMerchantType = ref('factory')
const merchantVerificationStatus = ref('unverified')
const mainCategoriesText = ref('')
const imagesText = ref('')
const pendingLogoFile = ref(null)
const pendingImageFiles = ref([])
const imageUrls = computed(() => parseList(imagesText.value))
const logoPreviewUrl = computed(() => pendingLogoFile.value?.path || form.logoUrl)
const merchantImageItems = computed(() => [
  ...imageUrls.value.map((url) => ({ id: url, url, local: false })),
  ...pendingImageFiles.value.map((file) => ({ id: file.id, url: file.path, local: true })),
])
const hasPendingImages = computed(() => Boolean(pendingLogoFile.value || pendingImageFiles.value.length > 0))
const saveButtonText = computed(() => {
  if (hasPendingImages.value) return '上传并保存'
  return merchantId.value ? '保存资料' : '提交入驻'
})
let smsTimer = 0
const form = reactive({
  cityCode: DEFAULT_CITY_CODE,
  name: '',
  merchantType: 'factory',
  contactName: '',
  contactPhone: '',
  contactWechat: '',
  addressText: '',
  location: {},
  description: '',
  logoUrl: '',
})

const currentMerchantTypeLabel = computed(() => {
  const matched = merchantTypeOptions.find((item) => item.value === form.merchantType) || {}
  return matched.label || '工厂'
})
const contactWechatPlaceholder = computed(() => {
  if (!merchantId.value || !contactWechatHint.value) return '微信号'
  return '新微信号'
})
const locationSelected = computed(() => hasValidLocation(form.location))
const merchantTypeChanged = computed(() => Boolean(merchantId.value) && form.merchantType !== originalMerchantType.value)
const merchantTypeChangeNeedsReverify = computed(() => merchantTypeChanged.value && ['pending', 'verified'].includes(merchantVerificationStatus.value))

onLoad((options) => {
  merchantId.value = options.merchantId || getMerchantId()
  loadMerchant()
})

onUnload(() => {
  clearSMSCountdown()
})

async function loadMerchant() {
  if (!merchantId.value) return
  try {
    const detail = await getMerchant(merchantId.value)
    form.name = detail.name || ''
    form.cityCode = detail.cityCode || DEFAULT_CITY_CODE
    form.merchantType = detail.merchantType || 'factory'
    originalMerchantType.value = form.merchantType
    merchantVerificationStatus.value = detail.verificationStatus || 'unverified'
    const contact = detail.contact || {}
    form.contactName = contact.name || ''
    contactPhoneHint.value = contact.phoneMasked || ''
    contactWechatHint.value = contact.wechatMasked || ''
    form.contactPhone = ''
    form.contactWechat = ''
    form.addressText = detail.addressText || ''
    form.location = detail.location || {}
    form.description = detail.description || ''
    form.logoUrl = detail.logoUrl || ''
    mainCategoriesText.value = (detail.mainCategories || []).join(',')
    imagesText.value = (detail.images || []).join(',')
    brandSectionOpen.value = Boolean(form.logoUrl || imageUrls.value.length > 0)
    await loadMerchantVerificationStatus()
  } catch (err) {
    uni.showToast({ title: err.message || '商家资料加载失败', icon: 'none' })
  }
}

async function loadMerchantVerificationStatus() {
  try {
    const latestVerification = await getLatestVerification(merchantId.value)
    if (latestVerification?.status === 'pending') {
      merchantVerificationStatus.value = 'pending'
    }
  } catch (err) {
    // 没有认证记录时保持商家详情返回的认证状态即可。
  }
}

function toggleBrandSection() {
  brandSectionOpen.value = !brandSectionOpen.value
}

function toggleContactSection() {
  contactSectionOpen.value = !contactSectionOpen.value
}

function changeMerchantType(event) {
  const selected = merchantTypeOptions[Number(event.detail.value)] || {}
  form.merchantType = selected.value || 'factory'
}

async function submitMerchantProfile() {
  const mainCategories = parseList(mainCategoriesText.value)
  const normalizedContactPhone = form.contactPhone.trim()
  const normalizedSmsCode = smsCode.value.trim()
  if (!form.name.trim()) {
    uni.showToast({ title: '请填写商家名称', icon: 'none' })
    return
  }
  if (mainCategories.length === 0) {
    uni.showToast({ title: '请填写主营品类', icon: 'none' })
    return
  }
  if (merchantId.value && (normalizedContactPhone || normalizedSmsCode) && (!normalizedContactPhone || !normalizedSmsCode)) {
    uni.showToast({ title: '请填写新手机号和短信验证码', icon: 'none' })
    return
  }
  try {
    submitting.value = true
    const needsReverifyAfterSave = merchantTypeChangeNeedsReverify.value
    await uploadPendingMerchantImages()
    const images = imageUrls.value
    if (merchantId.value) {
      const patch = {
        mainCategories,
        merchantType: form.merchantType,
        description: form.description.trim(),
        logoUrl: form.logoUrl.trim(),
        images,
        contactName: form.contactName.trim(),
        addressText: form.addressText.trim(),
        location: form.location || {},
      }
      if (normalizedContactPhone) {
        patch.contactPhone = normalizedContactPhone
        patch.smsCode = normalizedSmsCode
      }
      const normalizedWechat = form.contactWechat.trim()
      if (normalizedWechat) {
        patch.contactWechat = normalizedWechat
      }
      await updateMerchant(merchantId.value, patch)
      if (needsReverifyAfterSave) {
        merchantVerificationStatus.value = 'unverified'
      }
      if (merchantTypeChanged.value) {
        originalMerchantType.value = form.merchantType
      }
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
      if (images.length > 0 || form.logoUrl.trim() || form.description.trim() || locationSelected.value) {
        await updateMerchant(resp.id, {
          mainCategories,
          merchantType: form.merchantType,
          description: form.description.trim(),
          logoUrl: form.logoUrl.trim(),
          images,
          addressText: form.addressText.trim(),
          location: form.location || {},
        })
      }
    }
    uni.showToast({ title: needsReverifyAfterSave ? '已保存，请重新提交认证' : '商家资料已保存', icon: 'none' })
  } catch (err) {
    uni.showToast({ title: err.message || '商家资料保存失败', icon: 'none' })
  } finally {
    submitting.value = false
  }
}

function chooseMerchantLocation() {
  uni.chooseLocation({
    success: (result) => {
      const latitude = Number(result.latitude)
      const longitude = Number(result.longitude)
      if (!Number.isFinite(latitude) || !Number.isFinite(longitude)) {
        uni.showToast({ title: '未获取到有效地图位置', icon: 'none' })
        return
      }
      // 地图坐标用于商家主页导航；文字地址仍保留给用户手动修正。
      form.location = {
        latitude,
        longitude,
        name: result.name || '',
        address: result.address || '',
      }
      form.addressText = result.address || result.name || form.addressText
      uni.showToast({ title: '地图位置已保存', icon: 'none' })
    },
    fail: (err) => {
      if (String(err?.errMsg || '').includes('cancel')) return
      uni.showToast({ title: '地图选择失败，请稍后重试', icon: 'none' })
    },
  })
}

function clearMerchantLocation() {
  form.location = {}
}

async function sendSmsCodeForHomepagePhone() {
  const normalizedPhone = form.contactPhone.trim()
  if (!normalizedPhone) {
    uni.showToast({ title: '请填写新主页联系电话', icon: 'none' })
    return
  }
  try {
    smsSending.value = true
    await sendSmsCode({ phone: normalizedPhone })
    uni.showToast({ title: '验证码已发送', icon: 'none' })
    startSMSCountdown()
  } catch (err) {
    uni.showToast({ title: err.message || '验证码发送失败', icon: 'none' })
  } finally {
    smsSending.value = false
  }
}

async function uploadMerchantImage() {
  try {
    const file = await chooseImageFile()
    pendingImageFiles.value = [...pendingImageFiles.value, file]
    uni.showToast({ title: '已选择，保存时上传', icon: 'none' })
  } catch (err) {
    if (String(err?.errMsg || '').includes('cancel')) return
    uni.showToast({ title: err.message || '图片选择失败，请重试', icon: 'none' })
  }
}

function onChooseMerchantLogoAvatar(e) {
  const avatarUrl = e.detail.avatarUrl
  if (!avatarUrl) {
    uni.showToast({ title: 'LOGO 选择失败，请重试', icon: 'none' })
    return
  }
  pendingLogoFile.value = createImageFileFromPath(avatarUrl)
  uni.showToast({ title: '已选择，保存时上传', icon: 'none' })
}

async function uploadPendingMerchantImages() {
  if (pendingLogoFile.value) {
    form.logoUrl = await uploadSelectedImage(pendingLogoFile.value, 'merchant-logo')
    pendingLogoFile.value = null
  }
  if (pendingImageFiles.value.length === 0) return
  const uploadedImages = []
  for (const file of pendingImageFiles.value) {
    uploadedImages.push(await uploadSelectedImage(file, 'merchant-profile'))
  }
  imagesText.value = [...imageUrls.value, ...uploadedImages].join(',')
  pendingImageFiles.value = []
}

function removeMerchantImage(item) {
  if (item.local) {
    pendingImageFiles.value = pendingImageFiles.value.filter((file) => file.id !== item.id)
    return
  }
  imagesText.value = imageUrls.value.filter((url) => url !== item.url).join(',')
}

function previewMerchantLogo() {
  if (!logoPreviewUrl.value) return
  uni.previewImage({
    urls: [logoPreviewUrl.value],
    current: logoPreviewUrl.value,
  })
}

function parseList(value) {
  return String(value || '')
    .split(/[,，\n]/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function hasValidLocation(location) {
  if (!location) return false
  return Number.isFinite(Number(location.latitude)) && Number.isFinite(Number(location.longitude))
}

function startSMSCountdown() {
  clearSMSCountdown()
  smsCountdown.value = 60
  smsTimer = setInterval(() => {
    smsCountdown.value -= 1
    if (smsCountdown.value <= 0) {
      clearSMSCountdown()
    }
  }, 1000)
}

function clearSMSCountdown() {
  if (smsTimer) {
    clearInterval(smsTimer)
    smsTimer = 0
  }
}
</script>

<style lang="scss" scoped>
.profile-page {
  min-height: 100vh;
  padding: 24rpx 24rpx 0;
  background: $wplink-bg;
}

.form-card {
  display: grid;
  gap: 20rpx;
}

.form-section {
  padding: 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
}

.form-section {
  display: grid;
  gap: 18rpx;
}

.section-heading,
.section-toggle {
  min-height: 48rpx;
}

.section-title-row,
.label-row {
  display: flex;
  gap: 10rpx;
  align-items: center;
}

.section-title {
  color: $wplink-primary;
  font-size: 30rpx;
  font-weight: 700;
}

.required-badge {
  padding: 4rpx 12rpx;
  border-radius: 999rpx;
  font-size: 22rpx;
  line-height: 1.2;
}

.required-badge {
  background: $wplink-warning-soft;
  color: $wplink-warning;
}

.section-body,
.brand-fields {
  display: grid;
  gap: 18rpx;
}

.section-toggle {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 96rpx;
  gap: 16rpx;
  align-items: center;
  width: 100%;
  padding: 0;
  background: transparent;
  color: $wplink-primary;
  line-height: 1.4;
  text-align: left;
}

.section-toggle::after {
  border: 0;
}

.section-summary {
  display: block;
  margin-top: 6rpx;
  color: $wplink-muted;
  font-size: 24rpx;
}

.toggle-mark {
  color: $wplink-primary;
  font-size: 24rpx;
  text-align: right;
}

.field,
.textarea {
  width: 100%;
  min-height: 80rpx;
  padding: 0 20rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 10rpx;
  box-sizing: border-box;
}

.form-field,
.contact-phone-card,
.sms-code-field,
.image-field {
  display: grid;
  gap: 12rpx;
}

.field-label {
  color: $wplink-primary;
  font-size: 26rpx;
  font-weight: 600;
}

.field-helper {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.45;
}

.warning-helper {
  color: $wplink-warning;
}

.field-label-small {
  font-size: 24rpx;
  font-weight: 600;
}

.sms-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 180rpx;
  gap: 14rpx;
  align-items: center;
}

.address-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 172rpx;
  gap: 14rpx;
  align-items: center;
}

.sms-button,
.map-button {
  height: 80rpx;
  border: 1rpx solid $wplink-primary;
  border-radius: 10rpx;
  background: $wplink-card;
  color: $wplink-primary;
  font-size: 26rpx;
  line-height: 80rpx;
}

.location-status {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 148rpx;
  gap: 12rpx;
  align-items: center;
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.45;
}

.location-clear-button {
  height: 56rpx;
  border-radius: 8rpx;
  background: #f8fafc;
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 56rpx;
}

.picker-field {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: $wplink-primary;
}

.textarea {
  min-height: 144rpx;
  padding: 20rpx;
}

.primary-button {
  height: 84rpx;
  border-radius: 12rpx;
  background: $wplink-primary;
  color: $wplink-card;
}

.fixed-save-bar {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 20;
  padding: 18rpx 24rpx calc(18rpx + env(safe-area-inset-bottom));
  border-top: 1rpx solid $wplink-line;
  background: rgba(255, 255, 255, 0.96);
}

.fixed-save-spacer {
  height: calc(156rpx + env(safe-area-inset-bottom));
}

.secondary-button {
  height: 84rpx;
  border: 1rpx solid $wplink-primary;
  border-radius: 12rpx;
  background: $wplink-card;
  color: $wplink-primary;
}

.image-title-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 180rpx;
  gap: 14rpx;
  align-items: center;
}

.image-title-row .secondary-button {
  height: 68rpx;
  border-radius: 10rpx;
  font-size: 24rpx;
  line-height: 68rpx;
}

.image-helper,
.empty-image-text {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.45;
}

.logo-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 112rpx;
  gap: 18rpx;
  align-items: center;
}

.logo-copy {
  display: grid;
  gap: 10rpx;
}

.logo-preview-wrap {
  position: relative;
  width: 112rpx;
  height: 112rpx;
  border-radius: 14rpx;
  overflow: hidden;
}

.logo-upload-tile,
.logo-preview-tile {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 112rpx;
  height: 112rpx;
  padding: 0;
  border: 1rpx dashed $wplink-line;
  border-radius: 14rpx;
  box-sizing: border-box;
  background: #f8fafc;
  overflow: hidden;
}

.logo-preview-tile {
  border-style: solid;
  background: #e3e8ef;
}

.logo-change-button {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  width: 100%;
  height: 34rpx;
  padding: 0;
  margin: 0;
  border-radius: 0 0 14rpx 14rpx;
  background: rgba(0, 0, 0, 0.5);
  color: #fff;
  font-size: 20rpx;
  line-height: 34rpx;
  text-align: center;
}

.logo-upload-tile::after,
.logo-preview-tile::after,
.logo-change-button::after {
  border: 0;
}

.logo-plus {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36rpx;
  height: 36rpx;
}

.logo-plus-icon,
.logo-plus-icon::after {
  display: block;
  width: 22rpx;
  height: 2rpx;
  border-radius: 999rpx;
  background: $wplink-muted;
}

.logo-plus-icon {
  position: relative;
}

.logo-plus-icon::after {
  position: absolute;
  top: 0;
  left: 0;
  content: '';
  transform: rotate(90deg);
}

.logo-preview {
  width: 100%;
  height: 100%;
  background: #e3e8ef;
}

.empty-image-text {
  padding: 18rpx 20rpx;
  border: 1rpx dashed $wplink-line;
  border-radius: 10rpx;
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
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 56rpx;
}
</style>
