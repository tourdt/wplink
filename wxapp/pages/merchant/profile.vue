<template>
  <view class="profile-page">
    <view class="onboarding-head">
      <text class="eyebrow">商家与服务入驻</text>
      <text class="onboarding-title">建立你的业务主页</text>
      <text class="onboarding-subtitle">填写真实经营信息。保存后以主页实际状态为准。</text>
    </view>
    <view v-if="merchantLoadError" class="form-section load-error">
      <text class="field-label">业务资料加载失败</text>
      <text class="field-helper">请检查网络后重试，资料未加载前无法提交。</text>
      <button class="secondary-button" @click="loadMerchant">重新加载</button>
    </view>
    <view v-else-if="merchantProfileLoading" class="form-section">
      <text class="field-helper">正在读取业务资料…</text>
    </view>
    <view v-else class="form-card">
      <view v-if="savedState" class="saved-state" :class="savedState">
        <text class="saved-state-title">{{ savedStateText }}</text>
        <text class="field-helper">{{ savedStateHint }}</text>
        <view class="saved-actions">
          <button class="secondary-button" @click="previewStorefront">预览主页</button>
          <button v-if="['public', 'profile_public'].includes(savedState)" class="secondary-button" open-type="share">分享主页</button>
        </view>
      </view>
      <view class="form-section">
        <text class="section-title">基本资料</text>
        <view class="form-field">
          <text class="field-label">主体性质 <text class="required-mark">*</text></text>
          <view class="category-grid"><button v-for="option in ENTITY_KIND_OPTIONS" :key="option.value" class="category-chip" :class="{ selected: form.businessProfile.entityKind === option.value }" @click="form.businessProfile.entityKind = option.value">{{ option.label }}</button></view>
          <text class="field-helper">性质为自行填写，不代表平台认证。</text>
          <text v-if="fieldErrors.entityKind" class="field-error">{{ fieldErrors.entityKind }}</text>
        </view>
        <view class="form-field">
          <text class="field-label">主要业务 <text class="required-mark">*</text></text>
          <view class="category-grid"><button v-for="option in BUSINESS_ROLE_OPTIONS" :key="option.value" class="category-chip" :class="{ selected: form.businessProfile.primaryRole === option.value }" @click="selectPrimaryRole(option.value)">{{ option.label }}</button></view>
          <text v-if="fieldErrors.primaryRole" class="field-error">{{ fieldErrors.primaryRole }}</text>
          <text class="field-label">兼营业务（可多选）</text>
          <view class="category-grid"><button v-for="option in BUSINESS_ROLE_OPTIONS" :key="option.value" class="category-chip" :disabled="form.businessProfile.primaryRole === option.value" :class="{ selected: form.businessProfile.roles.includes(option.value) }" @click="toggleBusinessRole(option.value)">{{ option.label }}</button></view>
          <text v-if="fieldErrors.roles" class="field-error">{{ fieldErrors.roles }}</text>
          <text v-if="fieldErrors.businessProfile" class="field-error">{{ fieldErrors.businessProfile }}</text>
        </view>
        <view class="form-field">
          <text class="field-label">{{ businessNameCopy.label }} <text class="required-mark">*</text></text>
          <input v-model="form.name" class="field" :placeholder="businessNameCopy.placeholder" />
          <text v-if="fieldErrors.name" class="field-error">{{ fieldErrors.name }}</text>
        </view>
        <view v-if="needsBusinessCategories(form.businessProfile)" class="form-field">
          <text class="field-label">经营品类 <text class="required-mark">*</text></text>
          <text class="field-helper">可多选，让买家更快找到你</text>
          <view class="category-grid">
            <button v-for="category in merchantCategoryOptions" :key="category" class="category-chip" :class="{ selected: mainCategories.includes(category) }" @click="toggleCategory(category)">{{ category }}</button>
          </view>
          <text v-if="fieldErrors.mainCategories" class="field-error">{{ fieldErrors.mainCategories }}</text>
        </view>
        <view class="form-field">
          <text class="field-label">简介</text>
          <textarea v-model="form.description" class="textarea" placeholder="介绍你的业务、服务特色与合作方式" />
        </view>
        <view v-for="field in visibleBusinessFields" :key="field.key" class="form-field">
          <text class="field-label">{{ field.label }} <text class="required-mark">*</text></text>
          <input v-model="businessItemInputs[field.key]" class="field" :placeholder="field.placeholder" @input="changeBusinessItems(field.key, $event)" />
          <text class="field-helper">多项请用顿号或逗号分隔</text>
          <text v-if="fieldErrors[field.key]" class="field-error">{{ fieldErrors[field.key] }}</text>
        </view>
        <view class="form-field">
          <text class="field-label">服务区域（选填）</text>
          <input v-model="form.businessProfile.serviceArea" class="field" maxlength="200" placeholder="例如：织里及周边、全国" />
          <text v-if="fieldErrors.serviceArea" class="field-error">{{ fieldErrors.serviceArea }}</text>
        </view>
      </view>
      <view class="form-section">
        <text class="section-title">业务照片</text>
        <view class="form-field image-field">
          <view class="image-title-row"><text class="field-label">业务实拍（选填）</text><text class="image-count">{{ merchantImageEntries.length }}/{{ merchantProfileImageMaxCount }}</text></view>
          <text class="image-helper">首张照片会作为未设置 LOGO 时的封面。</text>
          <view class="image-grid-wrap">
            <UniGrid :column="3" :show-border="false" :square="true" @change="onMerchantImageGridItemClick">
              <UniGridItem v-for="(item, index) in merchantImageGridItems" :key="item.id" :index="index">
                <view v-if="item.type === 'image'" class="upload-img-item">
                  <image class="merchant-image" :src="item.url" mode="aspectFill" />
                  <button class="img-del" :disabled="submitting" @click.stop="removeMerchantImage(item)"><text class="img-del-line" /></button>
                </view>
                <view v-else class="upload-img-add-container"><view class="upload-img-item-add"><view class="image-add-icon" /></view></view>
              </UniGridItem>
            </UniGrid>
          </view>
          <text v-if="fieldErrors.images" class="field-error">{{ fieldErrors.images }}</text>
        </view>
        <view class="form-field logo-field">
          <view class="logo-layout">
            <view class="logo-copy"><text class="field-label">LOGO（选填）</text><text class="image-helper">可用品牌标识；未设置时使用首张实拍</text></view>
            <view v-if="logoPreviewUrl" class="logo-preview-wrap">
              <button class="logo-preview-tile" @click="previewMerchantLogo"><image class="logo-preview" :src="logoPreviewUrl" mode="aspectFill" /></button>
              <button class="logo-change-button" :disabled="submitting" open-type="chooseAvatar" @chooseavatar="onChooseMerchantLogoAvatar">更换</button>
            </view>
            <button v-else class="logo-upload-tile" :disabled="submitting" open-type="chooseAvatar" @chooseavatar="onChooseMerchantLogoAvatar"><view class="logo-plus"><view class="logo-plus-icon" /></view></button>
          </view>
        </view>
      </view>
      <view class="form-section">
        <text class="section-title">接待地点</text>
        <text v-if="draftLocationRetained" class="field-helper">草稿地点需重新选点，当前保留已保存地点；其它业务资料已恢复。</text>
        <view class="visibility-row"><text>在地图上公开接待地点</text><switch :checked="form.mapVisible" color="#214d72" @change="form.mapVisible = $event.detail.value" /></view>
        <text v-if="fieldErrors.mapVisible" class="field-error">{{ fieldErrors.mapVisible }}</text>
        <text class="field-helper">关闭后仍可公开和分享业务主页；详细地址及坐标不会向访客展示。服务区域始终可用于介绍业务。</text>
        <view class="form-field">
          <text class="field-label">地图选点 <text v-if="form.mapVisible" class="required-mark">*</text></text>
          <view class="location-status-block" :class="locationStatusTone">
            <view class="location-status-copy">
              <text v-if="locationResolving" class="location-status-title">正在确认地图位置</text>
              <text v-else-if="locationRiskHidden" class="location-status-title">位置暂未公开</text>
              <text v-else-if="locationSelected" class="location-status-title">地图位置已设置</text>
              <text v-else-if="addressLocationNeedsReselection" class="location-status-title">地址已修改，需重新选择</text>
              <text v-else class="location-status-title">未设置地图位置</text>
              <text class="location-status-detail">{{ form.locationName || '请在地图中选择准确的接待地点' }}</text>
            </view>
            <view class="location-actions">
              <button class="map-button" :disabled="submitting || locationResolving || merchantProfileLoading" :loading="locationResolving" @click="chooseMerchantLocation">{{ locationSelected ? '重新选择' : '选择位置' }}</button>
              <button v-if="locationSelected && !firstSubmission" class="location-clear-button" :disabled="submitting || merchantProfileLoading" @click="clearMerchantLocation">清除位置</button>
            </view>
          </view>
          <text v-if="fieldErrors.location" class="field-error">{{ fieldErrors.location }}</text>
        </view>
        <view class="form-field">
          <text class="field-label">详细地址 <text v-if="form.mapVisible" class="required-mark">*</text></text>
          <input v-model="form.addressText" class="field" :disabled="submitting" placeholder="门牌、街道等详细位置" @input="handleAddressTextInput" />
          <text v-if="fieldErrors.addressText" class="field-error">{{ fieldErrors.addressText }}</text>
        </view>
        <view class="location-fields">
          <view class="form-field"><text class="field-label">市场名称</text><input v-model="form.marketName" class="field" :disabled="locationResolving" placeholder="如织里童装城" /></view>
          <view class="form-field"><text class="field-label">楼栋 / 区域</text><input v-model="form.buildingName" class="field" :disabled="locationResolving" placeholder="如一期 A 区" /></view>
          <view class="form-field"><text class="field-label">楼层</text><input v-model="form.floorNo" class="field" :disabled="locationResolving" placeholder="如 1F" /></view>
          <view class="form-field"><text class="field-label">档口号</text><input v-model="form.boothNo" class="field" :disabled="locationResolving" placeholder="如 A108" /></view>
        </view>
      </view>
      <view class="form-section">
        <text class="section-title">联系方式</text>
        <text class="field-helper">联系方式由你单独决定是否展示。授权填入手机号不会自动公开。</text>
        <view class="form-field"><text class="field-label">联系人</text><input v-model="form.contactName" class="field" placeholder="联系人姓名" /></view>
        <view class="form-field contact-phone-card">
          <text class="field-label">联系电话</text>
          <view class="phone-input-row">
            <input v-model="form.contactPhone" class="field" type="number" maxlength="20" placeholder="联系电话" @input="sanitizeContactPhone" />
            <button class="wechat-phone-button" :disabled="submitting || phoneAuthorizing" :loading="phoneAuthorizing" open-type="getPhoneNumber" @getphonenumber="useWechatPhoneNumber">微信手机号</button>
          </view>
          <text class="field-helper">手机号需为 6–20 位数字</text>
          <text v-if="fieldErrors.contactPhone" class="field-error">{{ fieldErrors.contactPhone }}</text>
          <view class="visibility-row"><text>公开电话</text><switch :checked="form.publicPhone && !!form.contactPhone" :disabled="!form.contactPhone" color="#214d72" @change="form.publicPhone = $event.detail.value" /></view>
        </view>
        <view class="form-field">
          <text class="field-label">微信号</text>
          <input v-model="form.contactWechat" class="field" maxlength="32" placeholder="微信号" @input="sanitizeContactWechat" />
          <view class="visibility-row"><text>公开微信</text><switch :checked="form.publicWechat && !!form.contactWechat" :disabled="!form.contactWechat" color="#214d72" @change="form.publicWechat = $event.detail.value" /></view>
        </view>
      </view>
      <text class="draft-note">草稿只保存在当前设备，不会发布业务资料。</text>
    </view>
    <view class="fixed-save-spacer" />
    <view class="fixed-save-bar">
      <button class="secondary-button" :disabled="submitting || merchantProfileLoading || merchantLoadError" @click="saveLocalDraft">保存本机草稿</button>
      <button class="primary-button" :disabled="submitting || locationResolving || merchantProfileLoading || merchantLoadError" :loading="submitting" @click="submitMerchantProfile">{{ submitting ? '保存中' : firstSubmission ? '提交入驻' : '保存修改' }}</button>
    </view>
  </view>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { onLoad, onShareAppMessage } from '@dcloudio/uni-app'
import UniGrid from '../../components/uni-ui/uni-grid/uni-grid.vue'
import UniGridItem from '../../components/uni-ui/uni-grid-item/uni-grid-item.vue'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import {
  applyMerchantLocationBaseline,
  buildMerchantLocationPayload,
  buildMerchantLocationSubmission,
  clearMerchantLocationForm,
  createMerchantLocationSession,
  invalidateMerchantLocationSession,
  isMerchantLocationRequestCurrent,
  normalizeMerchantLocationForm,
  reduceMerchantLocationSession,
  resolveLatestMerchantLocationRequest,
} from '../../common/merchantLocationForm'
import { buildLoginUrl } from '../../common/auth'
import { MERCHANT_CATEGORY_OPTIONS, mergeMerchantCategories, toggleMerchantCategory } from '../../common/merchantCategories'
import { clearMerchantDraft, getMerchantDraft, isMerchantDraftSupported, saveMerchantDraft } from '../../common/merchantDraft'
import { BUSINESS_ROLE_OPTIONS, ENTITY_KIND_OPTIONS, BUSINESS_ARRAY_FIELDS, getBusinessProfile, normalizeBusinessProfile, normalizeBusinessItems, needsBusinessCategories } from '../../common/merchantBusinessProfile'
import { buildMerchantContactVisibility, getMerchantOnboardingErrors, getMerchantSavedState } from '../../common/merchantOnboarding'
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
import { getSession } from '../../store/session'

const merchantId = ref('')
const userId = ref('')
const merchantProfileLoading = ref(true)
const merchantLoadError = ref(false)
const submitting = ref(false)
const phoneAuthorizing = ref(false)
const addressLocationNeedsReselection = ref(false)
const locationStatus = ref('hidden')
// 服务端风险基线不能被本机草稿覆盖；重新选点才代表店主确认了新位置。
const serverLocationStatus = ref('hidden')
const serverMapVisible = ref(false)
const explicitLocationAction = ref('keep')
const draftLocationNeedsConfirmation = ref(false)
const draftLocationRetained = ref(false)
let serverLocationForm = {}
let serverLocationFingerprint = ''
let serverLocationDetailsFingerprint = ''
const locationSession = ref(createMerchantLocationSession())
const mainCategories = ref([])
const merchantCategoryOptions = MERCHANT_CATEGORY_OPTIONS
const fieldErrors = reactive({})
const businessItemInputs = reactive({ processingMethods: '', propertyTypes: '', serviceItems: '' })
const firstSubmission = ref(true)
const savedState = ref('')
const savedStateText = computed(() => ({ profile_public: '业务主页已公开，未上地图', public: '业务主页和地图已公开', paused: '业务主页已暂停展示', location_error: '位置待确认' })[savedState.value] || '')
const savedStateHint = computed(() => ({ profile_public: '可分享业务主页；开启地图展示需要详细地址和准确位置。', public: '访客可以查看业务主页和接待地点。', paused: '业务资料已保存，恢复展示后访客才能查看。', location_error: '业务资料已保存，地图地点暂未公开，请核对位置状态。' })[savedState.value] || '')
const pendingLogoFile = ref(null)
const merchantImageEntries = ref([])
const merchantProfileImageMaxCount = MERCHANT_PROFILE_IMAGE_MAX_COUNT
const logoPreviewUrl = computed(() => pendingLogoFile.value?.path || form.logoUrl || merchantImageEntries.value[0]?.url || '')
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
const form = reactive({
  cityCode: DEFAULT_CITY_CODE,
  name: '',
  businessProfile: normalizeBusinessProfile(),
  mapVisible: false,
  contactName: '',
  contactPhone: '',
  contactWechat: '',
  publicPhone: false,
  publicWechat: false,
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

const businessNameCopy = computed(() => ({
  stall: { label: '店铺名称', placeholder: '例如：织里小鹿童装' },
  factory: { label: '工厂名称', placeholder: '例如：小鹿服装加工厂' },
  landlord: { label: '出租方名称', placeholder: '例如：小鹿厂房出租' },
  service_provider: { label: '服务方名称', placeholder: '例如：小鹿服装设计工作室' },
  supplier: { label: '供应方名称', placeholder: '例如：小鹿童装货源' },
}[form.businessProfile.primaryRole] || { label: '对外名称', placeholder: '填写对外展示的名称' }))
const visibleBusinessFields = computed(() => BUSINESS_ARRAY_FIELDS.filter((field) => form.businessProfile.roles.includes(field.role)))
const locationSelected = computed(() => buildMerchantLocationPayload(form).locationAction === 'set')
const locationResolving = computed(() => locationSession.value.resolving)
const locationRiskHidden = computed(() => locationSession.value.intent === 'decision_required' && locationSelected.value)
const locationStatusTone = computed(() => {
  if (locationResolving.value || locationRiskHidden.value || addressLocationNeedsReselection.value) return 'attention'
  return locationSelected.value ? 'selected' : 'empty'
})

onLoad((options) => {
  const session = getSession()
  if (!session.token) {
    // 登录仅在进入编辑页时要求，回跳保留原 merchantId 参数。
    const target = `/pages/merchant/profile${options.merchantId ? `?merchantId=${encodeURIComponent(options.merchantId)}` : ''}`
    uni.redirectTo({ url: buildLoginUrl(target) })
    return
  }
  userId.value = session.userId
  merchantId.value = options.merchantId || session.merchantId
  loadMerchant()
})

onShareAppMessage(() => ({ title: form.name || '我的业务主页', path: `/pages/merchant/detail?merchantId=${encodeURIComponent(merchantId.value)}` }))

async function loadMerchant({ promptDraft = true, showError = true } = {}) {
  merchantProfileLoading.value = true
  merchantLoadError.value = false
  if (locationResolving.value) uni.hideLoading()
  // 资料基线开始加载时立即推进位置代次，保证此前地图回调即使稍后完成也不能回写新资料。
  locationSession.value = invalidateMerchantLocationSession(locationSession.value)
  try {
    if (!merchantId.value || !userId.value) throw new Error('missing merchant session')
    const detail = await getMerchant(merchantId.value)
    form.name = detail.name || ''
    form.cityCode = detail.cityCode || DEFAULT_CITY_CODE
    form.businessProfile = getBusinessProfile(detail)
    if (form.businessProfile.schemaVersion !== 1 || typeof detail.mapVisible !== 'boolean') {
      throw new Error('unsupported_business_profile')
    }
    syncBusinessItemInputs()
    // 地图开关只取服务端显式状态；草稿不能新增公开授权。
    serverMapVisible.value = detail.mapVisible
    form.mapVisible = serverMapVisible.value
    explicitLocationAction.value = 'keep'
    draftLocationNeedsConfirmation.value = false
    draftLocationRetained.value = false
    const contact = detail.contact || {}
    form.contactName = contact.name || ''
    form.contactPhone = sanitizeContactPhoneValue(contact.phone || '')
    form.contactWechat = sanitizeContactWechatValue(contact.wechat || '')
    form.publicPhone = detail.publicPhone === true
    form.publicWechat = detail.publicWechat === true
    form.addressText = detail.addressText || ''
    Object.assign(form, normalizeMerchantLocationForm(detail))
    locationStatus.value = detail.location?.status || 'hidden'
    serverLocationStatus.value = locationStatus.value
    serverLocationForm = { addressText: form.addressText, ...normalizeMerchantLocationForm(detail) }
    serverLocationFingerprint = locationFingerprint(form)
    serverLocationDetailsFingerprint = locationDetailsFingerprint(form)
    locationSession.value = applyMerchantLocationBaseline(locationSession.value, {
      status: locationStatus.value,
      form,
    })
    addressLocationNeedsReselection.value = false
    form.description = detail.description || ''
    form.logoUrl = detail.logoUrl || ''
    pendingLogoFile.value = null
    mainCategories.value = mergeMerchantCategories(detail.mainCategories || []).filter((category) => MERCHANT_CATEGORY_OPTIONS.includes(category))
    merchantImageEntries.value = (detail.images || [])
      .filter(Boolean)
      .map(createStoredMerchantImageEntry)
    firstSubmission.value = detail.profileStatus === 'incomplete' || !String(detail.name || '').trim()
    savedState.value = ''
    Object.keys(fieldErrors).forEach((key) => { delete fieldErrors[key] })
    if (promptDraft) {
      const draft = getMerchantDraft(userId.value, merchantId.value)
      if (draft) promptRestoreDraft(draft)
    }
    return detail
  } catch (err) {
    merchantLoadError.value = true
    console.warn('业务资料加载失败', { merchantId: merchantId.value, reason: 'request_failed' })
    if (showError) uni.showToast({ title: err.message === 'unsupported_business_profile' ? '业务资料格式或版本不受支持，请刷新后重试' : '资料加载失败，请重试', icon: 'none' })
  } finally {
    merchantProfileLoading.value = false
  }
}

function toggleCategory(category) {
  mainCategories.value = toggleMerchantCategory(mainCategories.value, category)
  delete fieldErrors.mainCategories
}

function selectPrimaryRole(role) {
  form.businessProfile.primaryRole = role
  if (!form.businessProfile.roles.includes(role)) form.businessProfile.roles.push(role)
}

function toggleBusinessRole(role) {
  if (role === form.businessProfile.primaryRole) return
  const roles = form.businessProfile.roles
  // 切换业务只改身份，不删除暂时隐藏的专业资料，便于恢复兼营内容。
  form.businessProfile.roles = roles.includes(role) ? roles.filter((item) => item !== role) : [...roles, role]
}

function changeBusinessItems(key, event) {
  businessItemInputs[key] = String(event?.detail?.value || '')
  form.businessProfile[key] = normalizeBusinessItems(businessItemInputs[key].split(/[、,，;；]/))
}

function syncBusinessItemInputs() {
  // 输入文本独立保存，避免实时去空去重把刚输入的分隔符删除，影响连续输入多项。
  BUSINESS_ARRAY_FIELDS.forEach(({ key }) => { businessItemInputs[key] = form.businessProfile[key].join('、') })
}

function showFieldErrors(errors) {
  Object.keys(fieldErrors).forEach((key) => { delete fieldErrors[key] })
  Object.assign(fieldErrors, errors)
}

function promptRestoreDraft(draft) {
  // 先加载服务端最新资料，再由用户选择恢复或丢弃，避免旧草稿静默覆盖新资料。
  uni.showModal({
    title: '发现本机草稿',
    content: '这份草稿只保存在当前设备。要用它替换刚加载的编辑内容吗？',
    confirmText: '恢复草稿',
    cancelText: '丢弃草稿',
    success: (res) => {
      if (res.confirm) {
        restoreDraft(draft)
      } else if (res.cancel) {
        clearMerchantDraft(userId.value, merchantId.value)
      }
    },
  })
}

function restoreDraft(draft) {
  if (!isMerchantDraftSupported(draft)) return
  const fields = ['name', 'description', 'logoUrl', 'contactName', 'contactPhone', 'contactWechat',
    'publicPhone', 'publicWechat', 'addressText', 'marketName', 'buildingName', 'floorNo', 'boothNo',
    'latitude', 'longitude', 'locationName']
  fields.forEach((key) => {
    if (Object.prototype.hasOwnProperty.call(draft, key)) form[key] = draft[key]
  })
  form.businessProfile = normalizeBusinessProfile(draft.businessProfile)
  syncBusinessItemInputs()
  // 恢复草稿不能新增地图公开授权。关闭操作可恢复，开启必须在当前编辑中明确选择。
  form.mapVisible = serverMapVisible.value && draft.mapVisible !== false
  mainCategories.value = mergeMerchantCategories(draft.mainCategories).filter((category) => MERCHANT_CATEGORY_OPTIONS.includes(category))
  merchantImageEntries.value = (draft.images || []).map(createStoredMerchantImageEntry)
  pendingLogoFile.value = null
  const draftLocationChanged = locationFingerprint(form) !== serverLocationFingerprint
  // 草稿仅恢复业务内容；地点回到当前服务端基线，让页面显示与 keep 的实际结果一致。
  // 草稿地点必须通过本次重新选点确认，不能因恢复编辑内容而获得公开授权。
  if (draftLocationChanged) Object.assign(form, serverLocationForm)
  draftLocationRetained.value = draftLocationChanged
  locationStatus.value = serverLocationStatus.value === 'risk_hidden'
    ? 'risk_hidden' : draft.locationStatus || 'hidden'
  locationSession.value = applyMerchantLocationBaseline(locationSession.value, { status: locationStatus.value, form })
  addressLocationNeedsReselection.value = Boolean(draft.addressLocationNeedsReselection)
  explicitLocationAction.value = 'keep'
  draftLocationNeedsConfirmation.value = serverLocationStatus.value === 'risk_hidden' || draftLocationChanged
  savedState.value = ''
}

function locationDetailsFingerprint(value) {
  return JSON.stringify([value.marketName, value.buildingName, value.floorNo, value.boothNo])
}

function locationFingerprint(value) {
  return JSON.stringify([value.addressText || '', ...Object.values(buildMerchantLocationPayload(value))])
}

function createDraftSnapshot() {
  return {
    name: form.name,
    businessProfile: normalizeBusinessProfile(form.businessProfile),
    mapVisible: form.mapVisible,
    mainCategories: [...mainCategories.value],
    description: form.description,
    logoUrl: form.logoUrl,
    images: getStoredMerchantImageUrls(merchantImageEntries.value),
    contactName: form.contactName,
    contactPhone: form.contactPhone,
    contactWechat: form.contactWechat,
    ...buildMerchantContactVisibility(form),
    addressText: form.addressText,
    marketName: form.marketName,
    buildingName: form.buildingName,
    floorNo: form.floorNo,
    boothNo: form.boothNo,
    latitude: form.latitude,
    longitude: form.longitude,
    locationName: form.locationName,
    locationStatus: locationStatus.value,
    addressLocationNeedsReselection: addressLocationNeedsReselection.value,
  }
}

async function saveLocalDraft() {
  if (submitting.value || merchantProfileLoading.value || merchantLoadError.value) return
  try {
    submitting.value = true
    // 本机临时路径重启后会失效，先上传为稳定 URL；此处不调用资料更新接口。
    await uploadPendingMerchantImages()
    if (!saveMerchantDraft(userId.value, merchantId.value, createDraftSnapshot())) {
      throw new Error('local_storage_failed')
    }
    uni.showToast({ title: '草稿已存到本机', icon: 'none' })
  } catch {
    console.warn('本机草稿保存失败', { merchantId: merchantId.value })
    uni.showToast({ title: '草稿保存失败，请重试', icon: 'none' })
  } finally {
    submitting.value = false
  }
}

async function submitMerchantProfile() {
  if (submitting.value || merchantProfileLoading.value || merchantLoadError.value) return
  const normalizedContactPhone = sanitizeContactPhoneValue(form.contactPhone)
  const normalizedWechat = sanitizeContactWechatValue(form.contactWechat)
  form.contactPhone = normalizedContactPhone
  form.contactWechat = normalizedWechat
  const errors = getMerchantOnboardingErrors({
    ...form, mainCategories: mainCategories.value, images: merchantImageEntries.value,
  })
  if (normalizedContactPhone && !isValidContactPhone(normalizedContactPhone)) {
    errors.contactPhone = '手机号需为 6–20 位数字'
  }
  // keep 保留服务端私有地址和风险基线。加载或恢复草稿都不是重新选点授权，
  // 只有本次明确选点/清除才提交 set/clear；风险地点普通资料保存不解除风险隐藏。
  let locationSubmission = { ok: true, payload: { locationAction: 'keep' } }
  if (locationResolving.value || explicitLocationAction.value !== 'keep') {
    locationSubmission = buildMerchantLocationSubmission(locationSession.value, form)
  } else if (locationDetailsFingerprint(form) !== serverLocationDetailsFingerprint) {
    locationSubmission = { ok: false, message: '接待地点补充信息已修改，请重新选择地图位置后保存' }
  } else if (form.mapVisible && draftLocationNeedsConfirmation.value) {
    locationSubmission = { ok: false, message: '恢复的草稿位置需重新选择地图位置后才能公开' }
  }
  if (!locationSubmission.ok) errors.location = locationSubmission.message
  showFieldErrors(errors)
  if (Object.keys(errors).length) return
  try {
    submitting.value = true
    await uploadPendingMerchantImages()
    const images = getStoredMerchantImageUrls(merchantImageEntries.value)
    const patch = {
      name: form.name.trim(),
      mainCategories: [...mainCategories.value],
      businessProfile: normalizeBusinessProfile(form.businessProfile),
      mapVisible: form.mapVisible,
      description: form.description.trim(),
      logoUrl: form.logoUrl.trim() || images[0] || '',
      images,
      contactName: form.contactName.trim(),
      addressText: form.addressText.trim(),
      ...locationSubmission.payload,
      ...buildMerchantContactVisibility(form),
    }
    if (normalizedContactPhone) patch.contactPhone = normalizedContactPhone
    if (normalizedWechat) patch.contactWechat = normalizedWechat
    await updateMerchant(merchantId.value, patch)
    clearMerchantDraft(userId.value, merchantId.value)
    // 保存响应不是最终公开状态；以重新读取的展示开关和位置状态为准。
    const detail = await loadMerchant({ promptDraft: false, showError: false })
    if (!merchantLoadError.value && detail) {
      savedState.value = getMerchantSavedState(detail)
      uni.showToast({ title: '业务资料已保存', icon: 'none' })
    } else {
      uni.showToast({ title: '资料已保存，状态获取失败，请重试', icon: 'none' })
    }
  } catch {
    console.warn('业务资料保存失败', { merchantId: merchantId.value, reason: 'request_failed' })
    uni.showToast({ title: '保存失败，请检查网络后重试', icon: 'none' })
  } finally {
    submitting.value = false
  }
}

function previewStorefront() {
  if (!merchantId.value) return
  uni.navigateTo({ url: `/pages/merchant/detail?merchantId=${encodeURIComponent(merchantId.value)}` })
}

function chooseMerchantLocation() {
  if (submitting.value || merchantProfileLoading.value) return
  if (locationResolving.value) uni.hideLoading()
  locationSession.value = reduceMerchantLocationSession(locationSession.value, { type: 'selection_started' })
  const requestToken = locationSession.value.requestToken
  uni.chooseLocation({
    success: async (result) => {
      if (!isMerchantLocationRequestCurrent(locationSession.value, requestToken)) return
      const latitude = Number(result.latitude)
      const longitude = Number(result.longitude)
      if (buildMerchantLocationPayload({ latitude, longitude }).locationAction !== 'set') {
        failMerchantLocationSelection(requestToken, '地图位置格式不正确，请重新选择')
        return
      }
      const reverseGeocoding = !buildMerchantLocationAddressText(result)
      if (reverseGeocoding) {
        uni.showLoading({ title: '解析地址中', mask: false })
      }
      const resolution = await resolveLatestMerchantLocationRequest({
        token: requestToken,
        task: () => resolveMerchantLocationAddress(result, latitude, longitude),
        getSession: () => locationSession.value,
      })
      if (!resolution.current) return
      if (reverseGeocoding) uni.hideLoading()
      if (resolution.error) {
        failMerchantLocationSelection(requestToken, '地图选择失败，请稍后重试')
        return
      }
      const resolvedAddress = resolution.value
      if (!resolvedAddress.address) {
        // 地图拖动到非 POI 点位时，微信可能只返回经纬度；不保存空地址，避免资料页显示无意义的位置。
        failMerchantLocationSelection(requestToken, '未获取到详细地址，请搜索具体地点后重试')
        return
      }
      draftLocationRetained.value = false
      explicitLocationAction.value = 'set'
      draftLocationNeedsConfirmation.value = false
      // 地图坐标用于发布者资料页导航，完整地址同步写入输入框，方便用户核对和补充。
      form.latitude = latitude
      form.longitude = longitude
      form.locationName = resolvedAddress.name
      form.addressText = resolvedAddress.address
      locationSession.value = reduceMerchantLocationSession(locationSession.value, {
        type: 'selection_succeeded',
        token: requestToken,
      })
      locationStatus.value = 'visible'
      addressLocationNeedsReselection.value = false
      delete fieldErrors.location
      delete fieldErrors.addressText
      uni.showToast({ title: '地图位置已保存', icon: 'none' })
    },
    fail: (err) => {
      if (!isMerchantLocationRequestCurrent(locationSession.value, requestToken)) return
      uni.hideLoading()
      if (String(err?.errMsg || '').includes('cancel')) {
        locationSession.value = reduceMerchantLocationSession(locationSession.value, {
          type: 'selection_cancelled',
          token: requestToken,
        })
        return
      }
      failMerchantLocationSelection(requestToken, '地图选择失败，请稍后重试')
    },
  })
}

function handleAddressTextInput(event) {
  draftLocationRetained.value = false
  explicitLocationAction.value = 'clear'
  draftLocationNeedsConfirmation.value = false
  form.addressText = String(event?.detail?.value ?? '')
  if (form.addressText.trim()) delete fieldErrors.addressText
  const shouldClearLocation = locationSelected.value || locationResolving.value || locationRiskHidden.value
  if (locationResolving.value) uni.hideLoading()
  locationSession.value = reduceMerchantLocationSession(locationSession.value, { type: 'address_changed' })
  if (!shouldClearLocation) return

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
  try {
    const result = await reverseGeocodeLocation({ latitude, longitude })
    const address = buildMerchantLocationAddressText(result)
    if (address) {
      return {
        address,
        name: normalizeMerchantLocationText(result?.name) || selectedName,
      }
    }
    console.warn('商家资料地图地址反查未返回详细地址', { merchantId: merchantId.value })
  } catch (err) {
    // 地址反查异常不输出完整响应，避免错误对象含地址或请求信息。
    console.warn('商家资料地图地址反查失败', {
      merchantId: merchantId.value,
      reason: 'reverse_geocode_failed',
    })
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
  if (submitting.value || merchantProfileLoading.value) return
  if (locationResolving.value) uni.hideLoading()
  draftLocationRetained.value = false
  explicitLocationAction.value = 'clear'
  draftLocationNeedsConfirmation.value = false
  locationSession.value = reduceMerchantLocationSession(locationSession.value, { type: 'location_cleared' })
  Object.assign(form, clearMerchantLocationForm(form))
  locationStatus.value = 'hidden'
  addressLocationNeedsReselection.value = false
}

function failMerchantLocationSelection(requestToken, message) {
  if (!isMerchantLocationRequestCurrent(locationSession.value, requestToken)) return
  uni.hideLoading()
  locationSession.value = reduceMerchantLocationSession(locationSession.value, {
    type: 'selection_failed',
    token: requestToken,
  })
  addressLocationNeedsReselection.value = !locationSelected.value
  uni.showToast({ title: message, icon: 'none' })
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

.onboarding-head {
  display: grid;
  gap: 10rpx;
  padding: 16rpx 8rpx 30rpx;
}

.eyebrow {
  color: #427495;
  font-size: 21rpx;
  font-weight: 700;
  letter-spacing: 4rpx;
}

.onboarding-title {
  color: $wplink-primary;
  font-size: 42rpx;
  font-weight: 800;
}

.onboarding-subtitle, .draft-note {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.5;
}

.required-mark, .field-error {
  color: #b54e43;
}

.field-error {
  font-size: 23rpx;
}

.category-grid, .saved-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12rpx;
}

.category-chip {
  width: auto;
  min-height: 64rpx;
  padding: 0 20rpx;
  margin: 0;
  border: 1rpx solid $wplink-line;
  border-radius: 32rpx;
  background: #f5f8fa;
  color: $wplink-primary;
  font-size: 24rpx;
  line-height: 64rpx;
}

.category-chip.selected {
  border-color: #427495;
  background: #e5f0f6;
  color: #214d72;
}

.category-chip::after { border: 0; }

.visibility-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 76rpx;
  padding-top: 8rpx;
  color: $wplink-primary;
  font-size: 25rpx;
}

.saved-state {
  display: grid;
  gap: 12rpx;
  padding: 24rpx;
  border-left: 6rpx solid #427495;
  border-radius: 12rpx;
  background: #eaf3f7;
}

.saved-state.paused, .saved-state.location_error {
  border-left-color: $wplink-warning;
  background: $wplink-warning-soft;
}

.saved-state-title {
  color: $wplink-primary;
  font-size: 29rpx;
  font-weight: 700;
}

.saved-actions .secondary-button {
  flex: 1;
  min-width: 220rpx;
  margin: 0;
}

.draft-note { padding: 0 8rpx; }

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
  display: grid;
  grid-template-columns: 220rpx minmax(0, 1fr);
  gap: 14rpx;
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
