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
          <view class="form-field logo-field">
            <view class="logo-layout">
              <view class="logo-copy">
                <text class="field-label">头像 / LOGO</text>
                <text class="image-helper">正方形头像或 LOGO，裁剪后保存</text>
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
                  更换图片
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
          <view class="form-field">
            <text class="field-label">展示名称</text>
            <input v-model="form.name" class="field" placeholder="请输入展示名称" />
          </view>
          <view class="form-field">
            <text class="field-label">主要身份</text>
            <picker :range="merchantTypeOptions" range-key="label" @change="changeMerchantType">
              <view class="field picker-field">{{ currentMerchantTypeLabel }}</view>
            </picker>
          </view>
          <view class="form-field">
            <text class="field-label">主营内容</text>
            <input v-model="mainCategoriesText" class="field" placeholder="如：童装现货、女装尾货、厂房出租、档口转让、设备转让" />
          </view>
          <view class="form-field">
            <text class="field-label">简介</text>
            <textarea v-model="form.description" class="textarea" placeholder="比如：主做童装现货，支持看样和小单补货" />
          </view>
          <view class="form-field image-field profile-images-field">
            <view class="image-title-row">
              <text class="field-label">展示图片</text>
              <text class="image-count">{{ merchantImageEntries.length }}/{{ merchantProfileImageMaxCount }}</text>
            </view>
            <text class="image-helper">公开展示图片，点击图片预览，点击最后一格添加</text>
            <view class="image-grid-wrap">
              <UniGrid :column="3" :show-border="false" :square="true" @change="onMerchantImageGridItemClick">
                <UniGridItem v-for="(item, index) in merchantImageGridItems" :key="item.id" :index="index">
                  <view v-if="item.type === 'image'" class="upload-img-item">
                    <image class="merchant-image" :src="item.url" mode="aspectFill" />
                    <button class="img-del" :disabled="submitting" @click.stop="removeMerchantImage(item)">
                      <text class="img-del-line" />
                    </button>
                  </view>
                  <view v-else class="upload-img-add-container">
                    <view class="upload-img-item-add">
                      <view class="image-add-icon" />
                    </view>
                  </view>
                </UniGridItem>
              </UniGrid>
            </view>
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
            <text class="field-label">联系人</text>
            <input v-model="form.contactName" class="field" placeholder="联系人姓名" />
          </view>
          <view class="form-field contact-phone-card">
            <text class="field-label">联系电话</text>
            <view class="phone-input-row">
              <input v-model="form.contactPhone" class="field" type="number" maxlength="20" placeholder="联系电话" @input="sanitizeContactPhone" />
              <button
                class="wechat-phone-button"
                :disabled="submitting || phoneAuthorizing"
                :loading="phoneAuthorizing"
                open-type="getPhoneNumber"
                @getphonenumber="useWechatPhoneNumber"
              >
                微信手机号
              </button>
            </view>
            <text class="field-helper">手机号需为 6-20 位数字</text>
          </view>
          <view class="form-field">
            <text class="field-label">微信</text>
            <input v-model="form.contactWechat" class="field" maxlength="32" placeholder="微信号" @input="sanitizeContactWechat" />
            <text class="field-helper">支持字母、数字、下划线和减号，最多 32 位</text>
          </view>
          <view class="form-field">
            <text class="field-label">经营地址</text>
            <input v-model="form.addressText" class="field" placeholder="请输入经营地址" @input="handleAddressTextInput" />
          </view>
          <view class="location-fields">
            <view class="form-field">
              <text class="field-label">市场名称</text>
              <input v-model="form.marketName" class="field" placeholder="如：织里童装城" />
            </view>
            <view class="form-field">
              <text class="field-label">楼栋 / 区域</text>
              <input v-model="form.buildingName" class="field" placeholder="如：一期 A 区" />
            </view>
            <view class="form-field">
              <text class="field-label">楼层</text>
              <input v-model="form.floorNo" class="field" placeholder="如：1F" />
            </view>
            <view class="form-field">
              <text class="field-label">档口号</text>
              <input v-model="form.boothNo" class="field" placeholder="如：A108" />
            </view>
          </view>
          <view class="location-status-block" :class="locationStatusTone">
            <view class="location-status-copy">
              <text v-if="locationRiskHidden" class="location-status-title">位置暂未公开</text>
              <text v-else-if="locationSelected" class="location-status-title">地图位置已设置</text>
              <text v-else-if="addressLocationNeedsReselection" class="location-status-title">地址已修改，需重新选择</text>
              <text v-else class="location-status-title">未设置地图位置</text>
              <text v-if="locationRiskHidden" class="location-status-detail">已保留原坐标，请重新选择地图位置恢复展示</text>
              <text v-else-if="locationSelected" class="location-status-detail">{{ form.locationName || '坐标有效，保存后可提供导航' }}</text>
              <text v-else-if="addressLocationNeedsReselection" class="location-status-detail">旧坐标已清除，保存前请核对并重新选点</text>
              <text v-else class="location-status-detail">设置后才会在拿货档口目录提供查看位置</text>
            </view>
            <view class="location-actions">
              <button class="map-button" :disabled="submitting" @click="chooseMerchantLocation">
                {{ locationSelected ? '重新选择' : '选择地图位置' }}
              </button>
              <button
                v-if="locationSelected"
                class="location-clear-button"
                :disabled="submitting"
                @click="clearMerchantLocation"
              >
                清除地图位置
              </button>
            </view>
          </view>
        </view>
      </view>
    </view>
    <view class="fixed-save-spacer" />
    <view class="fixed-save-bar">
      <button class="primary-button" :disabled="submitting" :loading="submitting" @click="submitMerchantProfile">
        {{ submitting ? '保存中' : saveButtonText }}
      </button>
    </view>
  </view>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import UniGrid from '../../components/uni-ui/uni-grid/uni-grid.vue'
import UniGridItem from '../../components/uni-ui/uni-grid-item/uni-grid-item.vue'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import {
  buildMerchantLocationPayload,
  clearMerchantLocationForm,
  normalizeMerchantLocationForm,
} from '../../common/merchantLocationForm'
import { validateMerchantName } from '../../common/merchantName'
import { bindWechatPhone } from '../../api/auth'
import { reverseGeocodeLocation } from '../../api/location'
import { getMerchant, updateMerchant } from '../../api/merchant'
import { createImageFileFromPath, uploadSelectedImage } from '../../common/upload'
import {
  MERCHANT_PROFILE_IMAGE_MAX_COUNT,
  appendMerchantImageFiles,
  createStoredMerchantImageEntry,
  getMerchantImagePreviewUrl,
  getMerchantImageUrlsForPreview,
  getStoredMerchantImageUrls,
  removeMerchantImageEntry,
  resolveImageCompressionOptions,
} from '../../common/merchantProfileImages'
import { getMerchantId } from '../../store/session'

const DEFAULT_MERCHANT_TYPE = 'individual'
const merchantTypeOptions = [
  { label: '个人', value: 'individual' },
  { label: '源头工厂', value: 'factory' },
  { label: '库存货源', value: 'stockist' },
  { label: '配套服务', value: 'service_provider' },
  { label: '采购', value: 'buyer' },
]
const legacyMerchantTypeText = {
  stall: '现货档口',
}

const merchantId = ref('')
const submitting = ref(false)
const phoneAuthorizing = ref(false)
const contactSectionOpen = ref(false)
const addressLocationNeedsReselection = ref(false)
const locationStatus = ref('hidden')
const mainCategoriesText = ref('')
const pendingLogoFile = ref(null)
const merchantImageEntries = ref([])
const merchantProfileImageMaxCount = MERCHANT_PROFILE_IMAGE_MAX_COUNT
const logoPreviewUrl = computed(() => pendingLogoFile.value?.path || form.logoUrl)
const merchantImageGridItems = computed(() => {
  const imageItems = merchantImageEntries.value.map((entry) => ({
    ...entry,
    type: 'image',
    url: getMerchantImagePreviewUrl(entry),
  }))
  if (imageItems.length < merchantProfileImageMaxCount) {
    imageItems.push({ id: 'merchant-image-add', type: 'add' })
  }
  return imageItems
})
const hasPendingImages = computed(() => Boolean(
  pendingLogoFile.value || merchantImageEntries.value.some((entry) => entry.kind === 'pending'),
))
const saveButtonText = computed(() => {
  if (hasPendingImages.value) return '上传并保存'
  return '保存资料'
})
const form = reactive({
  cityCode: DEFAULT_CITY_CODE,
  name: '',
  merchantType: DEFAULT_MERCHANT_TYPE,
  contactName: '',
  contactPhone: '',
  contactWechat: '',
  addressText: '',
  marketName: '',
  buildingName: '',
  floorNo: '',
  boothNo: '',
  latitude: null,
  longitude: null,
  locationName: '',
  description: '',
  logoUrl: '',
})

const currentMerchantTypeLabel = computed(() => {
  const matched = merchantTypeOptions.find((item) => item.value === form.merchantType) || {}
  return matched.label || legacyMerchantTypeText[form.merchantType] || '个人'
})
const locationSelected = computed(() => buildMerchantLocationPayload(form).locationAction === 'set')
const locationRiskHidden = computed(() => locationStatus.value === 'risk_hidden' && locationSelected.value)
const locationStatusTone = computed(() => {
  if (locationRiskHidden.value || addressLocationNeedsReselection.value) return 'attention'
  return locationSelected.value ? 'selected' : 'empty'
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
    form.merchantType = detail.merchantType || DEFAULT_MERCHANT_TYPE
    const contact = detail.contact || {}
    form.contactName = contact.name || ''
    form.contactPhone = sanitizeContactPhoneValue(contact.phone || '')
    form.contactWechat = sanitizeContactWechatValue(contact.wechat || '')
    form.addressText = detail.addressText || ''
    Object.assign(form, normalizeMerchantLocationForm(detail))
    locationStatus.value = detail.location?.status || 'hidden'
    addressLocationNeedsReselection.value = false
    form.description = detail.description || ''
    form.logoUrl = detail.logoUrl || ''
    mainCategoriesText.value = (detail.mainCategories || []).join(',')
    merchantImageEntries.value = (detail.images || [])
      .filter(Boolean)
      .map(createStoredMerchantImageEntry)
    contactSectionOpen.value = hasExistingContactInfo()
  } catch (err) {
    uni.showToast({ title: err.message || '资料加载失败', icon: 'none' })
  }
}

function toggleContactSection() {
  contactSectionOpen.value = !contactSectionOpen.value
}

function changeMerchantType(event) {
  const selected = merchantTypeOptions[Number(event.detail.value)] || {}
  form.merchantType = selected.value || DEFAULT_MERCHANT_TYPE
}

async function submitMerchantProfile() {
  if (submitting.value) return
  const mainCategories = parseList(mainCategoriesText.value)
  const normalizedContactPhone = sanitizeContactPhoneValue(form.contactPhone)
  const normalizedWechat = sanitizeContactWechatValue(form.contactWechat)
  form.contactPhone = normalizedContactPhone
  form.contactWechat = normalizedWechat
  const merchantNameMessage = validateMerchantName(form.name)
  if (merchantNameMessage) {
    uni.showToast({ title: merchantNameMessage, icon: 'none' })
    return
  }
  if (mainCategories.length === 0) {
    uni.showToast({ title: '请填写主营内容', icon: 'none' })
    return
  }
  if (normalizedContactPhone && !isValidContactPhone(normalizedContactPhone)) {
    uni.showToast({ title: '手机号需为 6-20 位数字', icon: 'none' })
    return
  }
  if (!merchantId.value) {
    uni.showToast({ title: '登录状态异常，请重新登录', icon: 'none' })
    return
  }
  try {
    submitting.value = true
    await uploadPendingMerchantImages()
    const images = getStoredMerchantImageUrls(merchantImageEntries.value)
    const locationPayload = buildMerchantLocationPayload(form)
    const patch = {
      name: form.name.trim(),
      mainCategories,
      merchantType: form.merchantType,
      description: form.description.trim(),
      logoUrl: form.logoUrl.trim(),
      images,
      contactName: form.contactName.trim(),
      addressText: form.addressText.trim(),
      ...locationPayload,
    }
    if (normalizedContactPhone) {
      patch.contactPhone = normalizedContactPhone
    }
    if (normalizedWechat) {
      patch.contactWechat = normalizedWechat
    }
    await updateMerchant(merchantId.value, patch)
    uni.showToast({ title: '资料已保存', icon: 'none' })
  } catch (err) {
    uni.showToast({ title: err.message || '资料保存失败', icon: 'none' })
  } finally {
    submitting.value = false
  }
}

function chooseMerchantLocation() {
  uni.chooseLocation({
    success: async (result) => {
      const latitude = Number(result.latitude)
      const longitude = Number(result.longitude)
      if (buildMerchantLocationPayload({ latitude, longitude }).locationAction !== 'set') {
        Object.assign(form, clearMerchantLocationForm(form))
        locationStatus.value = 'hidden'
        addressLocationNeedsReselection.value = true
        uni.showToast({ title: '地图位置格式不正确，请重新选择', icon: 'none' })
        return
      }
      const resolvedAddress = await resolveMerchantLocationAddress(result, latitude, longitude)
      if (!resolvedAddress.address) {
        // 地图拖动到非 POI 点位时，微信可能只返回经纬度；不保存空地址，避免资料页显示无意义的位置。
        Object.assign(form, clearMerchantLocationForm(form))
        locationStatus.value = 'hidden'
        addressLocationNeedsReselection.value = true
        uni.showToast({ title: '未获取到详细地址，请搜索具体地点后重试', icon: 'none' })
        return
      }
      // 地图坐标用于发布者资料页导航，完整地址同步写入输入框，方便用户核对和补充。
      form.latitude = latitude
      form.longitude = longitude
      form.locationName = resolvedAddress.name
      form.addressText = resolvedAddress.address
      locationStatus.value = 'visible'
      addressLocationNeedsReselection.value = false
      uni.showToast({ title: '地图位置已保存', icon: 'none' })
    },
    fail: (err) => {
      if (String(err?.errMsg || '').includes('cancel')) return
      Object.assign(form, clearMerchantLocationForm(form))
      locationStatus.value = 'hidden'
      addressLocationNeedsReselection.value = true
      uni.showToast({ title: '地图选择失败，请稍后重试', icon: 'none' })
    },
  })
}

function handleAddressTextInput(event) {
  form.addressText = String(event?.detail?.value ?? '')
  if (!locationSelected.value) return

  // 手工地址与地图坐标必须作为一个整体保存；文字变化后统一清空结构字段，避免旧坐标导航到错误档口。
  Object.assign(form, clearMerchantLocationForm(form))
  locationStatus.value = 'hidden'
  addressLocationNeedsReselection.value = true
}

async function resolveMerchantLocationAddress(result, latitude, longitude) {
  const selectedAddress = buildMerchantLocationAddressText(result)
  const selectedName = normalizeMerchantLocationText(result?.name)
  if (selectedAddress) {
    return { address: selectedAddress, name: selectedName }
  }
  return reverseGeocodeMerchantLocation(latitude, longitude, selectedName)
}

async function reverseGeocodeMerchantLocation(latitude, longitude, selectedName) {
  uni.showLoading({ title: '解析地址中', mask: false })
  try {
    const result = await reverseGeocodeLocation({ latitude, longitude })
    const address = buildMerchantLocationAddressText(result)
    if (address) {
      return {
        address,
        name: normalizeMerchantLocationText(result?.name) || selectedName,
      }
    }
    console.warn('商家资料地图地址反查未返回详细地址', { latitude, longitude })
  } catch (err) {
    // 地址反查异常不暴露服务端细节；记录坐标和错误信息，便于定位地图服务或网络问题。
    console.warn('商家资料地图地址反查失败', {
      latitude,
      longitude,
      errMsg: err?.message || err?.errMsg || String(err || ''),
    })
  } finally {
    uni.hideLoading()
  }
  return { address: '', name: selectedName }
}

function buildMerchantLocationAddressText(result) {
  const address = normalizeMerchantLocationText(result?.address)
  const name = normalizeMerchantLocationText(result?.name)
  if (address && name && !address.includes(name) && !name.includes(address)) {
    return `${address}${name}`
  }
  return address || name
}

function normalizeMerchantLocationText(value) {
  const text = String(value || '').trim()
  if (!text || isCoordinateMerchantLocationText(text)) return ''
  return text
}

function isCoordinateMerchantLocationText(text) {
  const value = String(text || '').trim()
  if (!value) return false
  if (/^-?\d+(\.\d+)?\s*[,，]\s*-?\d+(\.\d+)?$/.test(value)) return true
  if (/^(gps|经纬度|坐标|地图位置)[:：\s(（-]*-?\d+(\.\d+)?/i.test(value)) return true
  return /纬度[:：]?\s*-?\d+(\.\d+)?[\s,，;；]+经度[:：]?\s*-?\d+(\.\d+)?/.test(value)
}

function clearMerchantLocation() {
  Object.assign(form, clearMerchantLocationForm(form))
  locationStatus.value = 'hidden'
  addressLocationNeedsReselection.value = false
}

async function uploadMerchantImage() {
  try {
    const remainingCount = merchantProfileImageMaxCount - merchantImageEntries.value.length
    if (remainingCount <= 0) {
      uni.showToast({ title: `最多上传${merchantProfileImageMaxCount}张图片`, icon: 'none' })
      return
    }
    const files = await chooseMerchantImageFiles(remainingCount)
    merchantImageEntries.value = appendMerchantImageFiles(
      merchantImageEntries.value,
      files,
      merchantProfileImageMaxCount,
    )
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
}

async function uploadPendingMerchantImages() {
  if (pendingLogoFile.value) {
    form.logoUrl = await uploadSelectedImage(pendingLogoFile.value, 'merchant-logo')
    pendingLogoFile.value = null
  }
  if (!merchantImageEntries.value.some((entry) => entry.kind === 'pending')) return
  const uploadedEntries = []
  for (const entry of merchantImageEntries.value) {
    if (entry.kind === 'stored') {
      uploadedEntries.push(entry)
      continue
    }
    const compressedFile = await compressMerchantImageFile(entry.file)
    const uploadedUrl = await uploadSelectedImage(compressedFile, 'merchant-profile')
    uploadedEntries.push(createStoredMerchantImageEntry(uploadedUrl))
  }
  merchantImageEntries.value = uploadedEntries
}

function removeMerchantImage(item) {
  merchantImageEntries.value = removeMerchantImageEntry(merchantImageEntries.value, item.id)
}

function onMerchantImageGridItemClick(event) {
  const item = merchantImageGridItems.value[Number(event.detail.index)]
  if (!item) return
  if (item.type === 'add') {
    uploadMerchantImage()
    return
  }
  previewMerchantImage(item)
}

function previewMerchantImage(item) {
  const urls = getMerchantImageUrlsForPreview(merchantImageEntries.value)
  const current = getMerchantImagePreviewUrl(item)
  if (!current || urls.length === 0) return
  uni.previewImage({
    urls,
    current,
  })
}

function previewMerchantLogo() {
  if (!logoPreviewUrl.value) return
  uni.previewImage({
    urls: [logoPreviewUrl.value],
    current: logoPreviewUrl.value,
  })
}

function chooseMerchantImageFiles(count) {
  return new Promise((resolve, reject) => {
    if (typeof uni.chooseMedia === 'function') {
      uni.chooseMedia({
        count,
        mediaType: ['image'],
        sourceType: ['album', 'camera'],
        sizeType: ['original'],
        success: (res) => {
          const files = (res.tempFiles || [])
            .map((file) => createImageFileFromPath(file.tempFilePath || file.path, file))
            .filter((file) => file.path)
          resolve(files)
        },
        fail: reject,
      })
      return
    }
    uni.chooseImage({
      count,
      sizeType: ['original'],
      sourceType: ['album', 'camera'],
      success: (res) => {
        const tempFiles = res.tempFiles || []
        const files = (res.tempFilePaths || [])
          .map((filePath, index) => createImageFileFromPath(filePath, tempFiles[index] || {}))
          .filter((file) => file.path)
        resolve(files)
      },
      fail: reject,
    })
  })
}

async function compressMerchantImageFile(file) {
  if (typeof uni.compressImage !== 'function') return file
  try {
    const imageInfo = await getImageInfo(file.path)
    const options = resolveImageCompressionOptions({
      width: imageInfo.width,
      height: imageInfo.height,
      size: file.size,
    })
    if (!options.shouldCompress) return file
    const compressedPath = await compressImage(file.path, options)
    return createImageFileFromPath(compressedPath, {
      ...file,
      size: file.size || 1,
    })
  } catch (err) {
    return file
  }
}

function getImageInfo(src) {
  return new Promise((resolve, reject) => {
    uni.getImageInfo({
      src,
      success: resolve,
      fail: reject,
    })
  })
}

function compressImage(src, options) {
  return new Promise((resolve, reject) => {
    const params = {
      src,
      quality: options.quality,
      success: (res) => resolve(res.tempFilePath || src),
      fail: reject,
    }
    if (options.compressedWidth) {
      params.compressedWidth = options.compressedWidth
    }
    uni.compressImage(params)
  })
}

function parseList(value) {
  return String(value || '')
    .split(/[,，\n]/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function hasExistingContactInfo() {
  return Boolean(
    form.contactName.trim()
      || form.contactPhone.trim()
      || form.contactWechat.trim()
      || form.addressText.trim()
      || locationSelected.value,
  )
}

function sanitizeContactPhone(event) {
  form.contactPhone = sanitizeContactPhoneValue(event?.detail?.value ?? form.contactPhone)
}

function sanitizeContactWechat(event) {
  form.contactWechat = sanitizeContactWechatValue(event?.detail?.value ?? form.contactWechat)
}

async function useWechatPhoneNumber(event) {
  const code = String(event?.detail?.code || '').trim()
  if (!code) {
    uni.showToast({ title: '未获取到微信手机号，请手动填写', icon: 'none' })
    return
  }
  if (phoneAuthorizing.value) return
  try {
    phoneAuthorizing.value = true
    const resp = await bindWechatPhone({ code })
    const phone = sanitizeContactPhoneValue(resp?.phone)
    if (!isValidContactPhone(phone)) {
      uni.showToast({ title: '未获取到微信手机号，请手动填写', icon: 'none' })
      return
    }
    form.contactPhone = phone
    uni.showToast({ title: '已填入微信手机号', icon: 'none' })
  } catch (err) {
    uni.showToast({ title: err.message || '手机号获取失败，请手动填写', icon: 'none' })
  } finally {
    phoneAuthorizing.value = false
  }
}

function sanitizeContactPhoneValue(value) {
  return String(value || '').replace(/\D/g, '').slice(0, 20)
}

function sanitizeContactWechatValue(value) {
  return String(value || '').replace(/[^a-zA-Z0-9_-]/g, '').slice(0, 32)
}

function isValidContactPhone(value) {
  return /^\d{6,20}$/.test(value)
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

.section-body {
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

.phone-input-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 176rpx;
  gap: 14rpx;
  align-items: center;
}

.wechat-phone-button {
  height: 80rpx;
  padding: 0;
  border: 1rpx solid $wplink-primary;
  border-radius: 10rpx;
  background: $wplink-card;
  color: $wplink-primary;
  font-size: 24rpx;
  line-height: 80rpx;
}

.wechat-phone-button::after {
  border: 0;
}

.location-fields {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14rpx;
}

.map-button {
  width: 100%;
  min-height: 88rpx;
  border: 1rpx solid $wplink-primary;
  border-radius: 10rpx;
  background: $wplink-card;
  color: $wplink-primary;
  font-size: 26rpx;
  line-height: 1.35;
}

.location-status-block {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 192rpx;
  gap: 18rpx;
  align-items: center;
  padding: 18rpx 0 18rpx 18rpx;
  border-top: 1rpx solid $wplink-line;
  border-bottom: 1rpx solid $wplink-line;
  border-left: 6rpx solid $wplink-line;
}

.location-status-block.selected {
  border-left-color: $wplink-success;
}

.location-status-block.attention {
  border-left-color: $wplink-warning;
}

.location-status-copy,
.location-actions {
  display: grid;
  gap: 10rpx;
}

.location-status-title {
  color: $wplink-primary;
  font-size: 28rpx;
  font-weight: 700;
  line-height: 1.35;
}

.location-status-detail {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.45;
}

.location-clear-button {
  width: 100%;
  min-height: 88rpx;
  padding: 8rpx 10rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 8rpx;
  background: $wplink-card;
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.35;
}

.map-button[disabled],
.location-clear-button[disabled] {
  opacity: 0.55;
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
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 14rpx;
  align-items: center;
}

.image-count {
  color: $wplink-muted;
  font-size: 24rpx;
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

.image-grid-wrap {
  margin-right: 160rpx;
}

.upload-img-item,
.upload-img-add-container {
  position: relative;
  width: 100%;
  height: 100%;
  padding: 8rpx;
  box-sizing: border-box;
}

.merchant-image {
  width: 100%;
  height: 100%;
  border-radius: 10rpx;
  background: #e3e8ef;
}

.img-del {
  position: absolute;
  right: 8rpx;
  top: 8rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44rpx;
  height: 44rpx;
  padding: 0;
  border-radius: 50%;
  background: rgba(15, 23, 42, 0.72);
}

.img-del[disabled] {
  background: rgba(15, 23, 42, 0.72);
  opacity: 1;
}

.img-del::after {
  border: 0;
}

.img-del-line,
.img-del-line::after {
  display: block;
  width: 22rpx;
  height: 3rpx;
  border-radius: 999rpx;
  background: #fff;
}

.img-del-line {
  transform: rotate(45deg);
}

.img-del-line::after {
  content: '';
  transform: rotate(90deg);
}

.upload-img-item-add {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  border: 1rpx dashed $wplink-line;
  border-radius: 10rpx;
  background: #f8fafc;
}

.image-add-icon,
.image-add-icon::after {
  display: block;
  width: 36rpx;
  height: 4rpx;
  border-radius: 999rpx;
  background: $wplink-muted;
}

.image-add-icon {
  position: relative;
}

.image-add-icon::after {
  position: absolute;
  top: 0;
  left: 0;
  content: '';
  transform: rotate(90deg);
}
</style>
