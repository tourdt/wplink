<template>
  <view class="publish-page">
    <view class="form-section basic-section">
      <view class="section-head">
        <text class="section-title">基础信息</text>
        <text class="section-note">必填</text>
      </view>
      <view class="basic-progress">
        <view class="progress-copy">
          <text class="progress-title">{{ publishReadyText }}</text>
          <text class="progress-desc">标题、品类、联系人和联系电话为必填项</text>
        </view>
        <text class="completion-percent">{{ completionPercent }}%</text>
      </view>
      <view class="completion-bar">
        <view class="completion-bar-fill" :style="completionBarStyle"></view>
      </view>
      <view class="field-group">
        <text class="field-label">资源类型</text>
        <picker :range="resourceTypeNames" :value="selectedTypeIndex" @change="selectType">
          <view class="field picker-field">
            <text>{{ selectedTypeLabel }}</text>
            <text class="picker-arrow">›</text>
          </view>
        </picker>
        <text class="field-helper">资源类型会用于搜索筛选和分类展示。</text>
      </view>
      <view class="field-group">
        <text class="field-label">标题</text>
        <input v-model="form.title" class="field" placeholder="例如：童装春款现货 3000 件" />
      </view>
      <view class="field-group">
        <text class="field-label">品类</text>
        <input v-model="form.category" class="field" placeholder="例如：童装、女装、面料、加工" />
      </view>
    </view>

    <view class="form-section supply-section">
      <view class="section-head">
        <text class="section-title">供应信息</text>
        <text class="section-note">建议填写</text>
      </view>
      <view class="field-group">
        <text class="field-label">数量/产能</text>
        <input v-model="form.quantityText" class="field" placeholder="例如：3000 件、日产 800 件" />
      </view>
      <view class="field-group">
        <text class="field-label">价格描述</text>
        <input v-model="form.priceText" class="field" placeholder="例如：18-25 元/件，量大可议" />
      </view>
      <view class="field-group">
        <text class="field-label">资源描述</text>
        <textarea v-model="form.description" class="textarea" placeholder="说明货品状态、尺码颜色、交期、看样方式等关键信息" />
      </view>
    </view>

    <view v-if="dynamicFieldItems.length" class="form-section attribute-section">
      <view class="section-head">
        <text class="section-title">类型信息</text>
        <text class="section-note">按资源类型</text>
      </view>
      <view v-for="field in dynamicFieldItems" :key="field.key" class="field-group">
        <text class="field-label">{{ field.label }}</text>
        <view v-if="field.type === 'boolean'" class="toggle-group">
          <button
            :class="['toggle-option', getDynamicFieldValue(field.key) === true ? 'active' : '']"
            @click="setDynamicFieldBoolean(field.key, true)"
          >
            是
          </button>
          <button
            :class="['toggle-option', getDynamicFieldValue(field.key) === false ? 'active' : '']"
            @click="setDynamicFieldBoolean(field.key, false)"
          >
            否
          </button>
        </view>
        <picker
          v-else-if="field.type === 'select'"
          :range="field.options"
          :value="getDynamicFieldOptionIndex(field)"
          @change="setDynamicFieldSelect(field, $event)"
        >
          <view class="field picker-field">
            <text>{{ getDynamicFieldValue(field.key) || `请选择${field.label}` }}</text>
            <text class="picker-arrow">›</text>
          </view>
        </picker>
        <textarea
          v-else-if="field.type === 'textarea'"
          class="textarea"
          :value="form.attributes[field.key]"
          :placeholder="field.placeholder || `请填写${field.label}`"
          @input="setDynamicFieldValue(field.key, $event.detail.value)"
        />
        <input
          v-else
          class="field"
          :type="field.type === 'number' ? 'number' : 'text'"
          :value="form.attributes[field.key]"
          :placeholder="field.placeholder || `请填写${field.label}`"
          @input="setDynamicFieldValue(field.key, $event.detail.value)"
        />
      </view>
    </view>

    <view class="form-section image-section">
      <view class="section-head">
        <text class="section-title">资源图片</text>
        <text class="image-count">{{ resourceImageEntries.length }}/{{ resourceImageMaxCount }}</text>
      </view>
      <view class="image-grid-wrap">
        <UniGrid :column="3" :show-border="false" :square="true" @change="onResourceImageGridItemClick">
          <UniGridItem v-for="(item, index) in resourceImageGridItems" :key="item.id" :index="index">
            <view v-if="item.type === 'image'" class="upload-img-item">
              <image class="resource-image" :src="item.url" mode="aspectFill" />
              <button class="img-del" @click.stop="removeResourceImage(item)">
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

    <view class="form-section contact-section">
      <view class="section-head">
        <text class="section-title">联系信息</text>
        <text class="section-note">必填</text>
      </view>
      <view class="field-group">
        <text class="field-label">联系人</text>
        <input v-model="form.contact.name" class="field" placeholder="买家看到的联系人" />
      </view>
      <view class="field-group">
        <text class="field-label">联系电话</text>
        <input v-model="form.contact.phone" class="field" placeholder="用于买家发起联系" />
      </view>
    </view>

    <view :class="['fixed-save-spacer', { 'no-safe-area': !reserveBottomSafeArea }]" />
    <view :class="['fixed-save-bar', { 'no-safe-area': !reserveBottomSafeArea }]">
      <view class="fixed-save-actions">
        <button class="secondary-button" @click="saveDraft">保存草稿</button>
        <button :class="['primary-button', canSubmit ? '' : 'is-disabled']" @click="submit">提交审核</button>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed, reactive, ref, watch, onUnmounted } from 'vue'
import UniGrid from './uni-ui/uni-grid/uni-grid.vue'
import UniGridItem from './uni-ui/uni-grid-item/uni-grid-item.vue'
import { DEFAULT_CITY_CODE } from '../common/constants'
import { getMerchantId, saveMerchantId } from '../store/session'
import { listCityResourceTypes } from '../api/city'
import { getMerchant } from '../api/merchant'
import { createResource, createResourceDraft, getEditableResource, submitResource, updateResourceDraft } from '../api/resource'
import { chooseImageFile, uploadSelectedImage } from '../common/upload'

const props = defineProps({
  initialOptions: {
    type: Object,
    default: () => ({}),
  },
  mode: {
    type: String,
    default: 'create',
  },
  reserveBottomSafeArea: {
    type: Boolean,
    default: true,
  },
})

const reserveBottomSafeArea = computed(() => props.reserveBottomSafeArea)
const resourceTypes = ref([])
const selectedTypeIndex = ref(0)
const resourceImageMaxCount = 9
const resourceImageEntries = ref([])
const publishLocalDraftStorageKey = ref('')
const autosaveReady = ref(false)
const editingResourceId = ref('')
const editingResourceStatus = ref('')
const editSavedAsDraft = ref(true)
let localDraftSaveTimer = null
const form = reactive({
  merchantId: '',
  cityCode: DEFAULT_CITY_CODE,
  typeCode: '',
  title: '',
  category: '',
  quantityText: '',
  priceText: '',
  description: '',
  attributes: {},
  tags: [],
  images: [],
  contact: {
    name: '',
    phone: '',
    wechat: '',
  },
})

const resourceTypeNames = computed(() => resourceTypes.value.map((item) => item.typeName))
const currentResourceType = computed(() => resourceTypes.value[selectedTypeIndex.value] || {})
const selectedTypeLabel = computed(() => {
  return currentResourceType.value.typeName || '请选择资源类型'
})
const dynamicFieldItems = computed(() => normalizeDynamicFieldItems(currentResourceType.value.fieldSchema))
const requiredFields = computed(() => {
  const configuredFields = Array.isArray(currentResourceType.value.requiredFields) ? currentResourceType.value.requiredFields : []
  return Array.from(new Set(['typeCode', 'title', 'category', 'contactName', 'contactPhone', ...configuredFields]))
})
const requiredFieldStates = computed(() => requiredFields.value.map(isPublishFieldCompleted))
const canSubmit = computed(() => requiredFieldStates.value.every(Boolean))
const completedRequiredCount = computed(() => requiredFieldStates.value.filter(Boolean).length)
const completionPercent = computed(() => Math.round((completedRequiredCount.value / requiredFields.value.length) * 100))
const completionBarStyle = computed(() => `width: ${completionPercent.value}%;`)
const resourceImageGridItems = computed(() => {
  const imageItems = resourceImageEntries.value.map((entry, index) => ({
    id: entry.id,
    type: 'image',
    url: getResourceImagePreviewUrl(entry),
    index,
  }))
  if (imageItems.length < resourceImageMaxCount) {
    imageItems.push({ id: 'resource-image-add', type: 'add' })
  }
  return imageItems
})
const publishReadyText = computed(() => {
  if (canSubmit.value) {
    return '必填项已完整，可提交审核'
  }
  return `还差 ${requiredFields.value.length - completedRequiredCount.value} 项必填信息`
})

watch(
  () => props.initialOptions,
  (options) => {
    initializePublishForm(options || {})
  },
  { deep: true, immediate: true },
)

onUnmounted(() => {
  flushPublishLocalDraft()
})

watch(
  form,
  () => {
    markEditingResourceUnsaved()
    scheduleSavePublishLocalDraft()
  },
  { deep: true },
)

watch(
  resourceImageEntries,
  () => {
    markEditingResourceUnsaved()
    scheduleSavePublishLocalDraft()
  },
  { deep: true },
)

function markEditingResourceUnsaved() {
  if (autosaveReady.value && editingResourceId.value) {
    editSavedAsDraft.value = false
  }
}

async function initializePublishForm(options = {}) {
  autosaveReady.value = false
  editingResourceId.value = options.resourceId || ''
  if (options.repost) {
    editingResourceId.value = ''
  }
  editingResourceStatus.value = ''
  editSavedAsDraft.value = true
  Object.assign(form, createEmptyPublishForm())
  resourceImageEntries.value = []
  selectedTypeIndex.value = 0
  form.merchantId = options.merchantId || getMerchantId()
  form.typeCode = options.typeCode || ''
  publishLocalDraftStorageKey.value = buildPublishLocalDraftStorageKey(form.merchantId, editingResourceId.value)
  await loadResourceTypes()
  if (options.repost) {
    restoreRepostInitialForm()
    publishLocalDraftStorageKey.value = buildPublishLocalDraftStorageKey(form.merchantId, editingResourceId.value)
  } else if (editingResourceId.value) {
    await loadEditableResource()
  } else {
    await loadMerchantContact()
    restorePublishLocalDraft()
  }
  autosaveReady.value = true
}

async function loadResourceTypes() {
  const resp = await listCityResourceTypes(form.cityCode)
  resourceTypes.value = resp.items || []
  if (!resourceTypes.value.length) {
    form.typeCode = ''
    return
  }
  const matchIndex = resourceTypes.value.findIndex((item) => item.typeCode === form.typeCode)
  selectedTypeIndex.value = matchIndex >= 0 ? matchIndex : 0
  form.typeCode = resourceTypes.value[selectedTypeIndex.value].typeCode
  syncAttributesWithSelectedType()
}

function selectType(event) {
  selectedTypeIndex.value = Number(event.detail.value)
  const current = resourceTypes.value[selectedTypeIndex.value] || {}
  form.typeCode = current.typeCode || ''
  syncAttributesWithSelectedType()
}

async function loadMerchantContact() {
  if (!form.merchantId) return
  try {
    const detail = await getMerchant(form.merchantId)
    applyMerchantContactDefaults(detail.contact || {})
  } catch (err) {
    // 商户资料无法加载时不阻断发布，联系人继续由用户手动填写。
  }
}

async function loadEditableResource() {
  const detail = await getEditableResource(editingResourceId.value, form.merchantId)
  editingResourceStatus.value = detail.status
  editSavedAsDraft.value = detail.status === 'draft'
  Object.assign(form, createEmptyPublishForm(), {
    merchantId: detail.merchantId || form.merchantId,
    cityCode: detail.cityCode || DEFAULT_CITY_CODE,
    typeCode: detail.typeCode || '',
    title: detail.title || '',
    category: detail.category || '',
    quantityText: detail.quantityText || '',
    priceText: detail.priceText || '',
    description: detail.description || '',
    attributes: detail.attributes || {},
    tags: detail.tags || [],
    images: detail.images || [],
    contact: {
      name: detail.contact?.name || '',
      phone: detail.contact?.phone || '',
      wechat: detail.contact?.wechat || '',
    },
  })
  resourceImageEntries.value = (detail.images || []).map(createStoredResourceImageEntry)
  syncSelectedTypeIndex()
}

function restoreRepostInitialForm() {
  let initialForm = null
  try {
    initialForm = uni.getStorageSync('publish:repost-initial-form')
    uni.removeStorageSync('publish:repost-initial-form')
  } catch (err) {
    initialForm = null
  }
  if (!initialForm || typeof initialForm !== 'object') {
    return
  }
  applyInitialPublishForm(initialForm)
}

function applyInitialPublishForm(initialForm) {
  Object.assign(form, createEmptyPublishForm(), {
    merchantId: initialForm.merchantId || form.merchantId,
    cityCode: initialForm.cityCode || DEFAULT_CITY_CODE,
    typeCode: initialForm.typeCode || '',
    title: initialForm.title || '',
    category: initialForm.category || '',
    quantityText: initialForm.quantityText || '',
    priceText: initialForm.priceText || '',
    description: initialForm.description || '',
    attributes: initialForm.attributes || {},
    tags: initialForm.tags || [],
    images: initialForm.images || [],
    contact: {
      name: initialForm.contact?.name || '',
      phone: initialForm.contact?.phone || '',
      wechat: initialForm.contact?.wechat || '',
    },
  })
  resourceImageEntries.value = (initialForm.images || []).map(createStoredResourceImageEntry)
  syncSelectedTypeIndex()
}

function applyMerchantContactDefaults(contact) {
  if (!form.contact.name.trim() && contact.name) {
    form.contact.name = contact.name
  }
  if (!form.contact.phone.trim() && (contact.phone || contact.phoneMasked)) {
    form.contact.phone = contact.phone || contact.phoneMasked
  }
}

async function submit() {
  if (!validatePublishForm()) {
    return
  }
  saveMerchantId(form.merchantId)
  if (editingResourceId.value) {
    if (!editSavedAsDraft.value || editingResourceStatus.value !== 'draft') {
      uni.showToast({ title: '请先保存草稿后再提交审核', icon: 'none' })
      return
    }
    await submitResource(editingResourceId.value, form.merchantId)
    clearPublishLocalDraft()
    resetPublishForm()
    uni.showToast({ title: '已提交审核', icon: 'none' })
    uni.navigateTo({ url: '/pages/publish-success/index' })
    return
  }
  const images = await uploadPendingResourceImages()
  await createResource({ ...form, images })
  clearPublishLocalDraft()
  resetPublishForm()
  uni.showToast({ title: '已提交审核', icon: 'none' })
  uni.navigateTo({ url: '/pages/publish-success/index' })
}

async function saveDraft() {
  if (!validatePublishForm()) {
    return
  }
  saveMerchantId(form.merchantId)
  const merchantId = form.merchantId
  const images = await uploadPendingResourceImages()
  await saveResourceDraftPayload(images)
  clearPublishLocalDraft()
  resetPublishForm()
  uni.showToast({ title: '草稿已保存', icon: 'none' })
  uni.navigateTo({ url: `/pages/my-resources/index?merchantId=${merchantId}` })
}

async function saveResourceDraftPayload(images) {
  const payload = { ...form, images }
  if (!editingResourceId.value) {
    return createResourceDraft(payload)
  }
  const resp = await updateResourceDraft(editingResourceId.value, payload)
  editingResourceStatus.value = 'draft'
  editSavedAsDraft.value = true
  return resp
}

function buildPublishLocalDraftStorageKey(merchantId, resourceId = '') {
  return `publish:local-draft:${merchantId || 'default'}:${resourceId || 'new'}`
}

function restorePublishLocalDraft() {
  if (!publishLocalDraftStorageKey.value) return
  let draft = null
  try {
    draft = uni.getStorageSync(publishLocalDraftStorageKey.value)
  } catch (err) {
    return
  }
  if (!draft || typeof draft !== 'object') return
  Object.assign(form, createEmptyPublishForm(), draft.form || {})
  resourceImageEntries.value = Array.isArray(draft.resourceImageEntries)
    ? draft.resourceImageEntries.filter((entry) => entry?.id && entry?.url)
    : []
  syncSelectedTypeIndex()
}

function scheduleSavePublishLocalDraft() {
  if (!autosaveReady.value || !publishLocalDraftStorageKey.value) return
  if (localDraftSaveTimer) {
    clearTimeout(localDraftSaveTimer)
  }
  localDraftSaveTimer = setTimeout(savePublishLocalDraft, 500)
}

function savePublishLocalDraft() {
  if (!publishLocalDraftStorageKey.value) return
  localDraftSaveTimer = null
  // 图片本地路径只作为意外退出后的辅助恢复，最终仍以保存/提交时上传 OSS 为准。
  uni.setStorageSync(publishLocalDraftStorageKey.value, {
    form: clonePublishForm(),
    resourceImageEntries: resourceImageEntries.value.map(serializeResourceImageEntry),
    savedAt: Date.now(),
  })
}

function flushPublishLocalDraft() {
  if (!autosaveReady.value) return
  if (localDraftSaveTimer) {
    clearTimeout(localDraftSaveTimer)
    localDraftSaveTimer = null
  }
  savePublishLocalDraft()
}

function clearPublishLocalDraft() {
  if (localDraftSaveTimer) {
    clearTimeout(localDraftSaveTimer)
    localDraftSaveTimer = null
  }
  if (publishLocalDraftStorageKey.value) {
    uni.removeStorageSync(publishLocalDraftStorageKey.value)
  }
}

function resetPublishForm() {
  autosaveReady.value = false
  editingResourceId.value = ''
  editingResourceStatus.value = ''
  editSavedAsDraft.value = true
  Object.assign(form, createEmptyPublishForm())
  resourceImageEntries.value = []
  selectedTypeIndex.value = 0
}

function createEmptyPublishForm() {
  return {
    merchantId: '',
    cityCode: DEFAULT_CITY_CODE,
    typeCode: '',
    title: '',
    category: '',
    quantityText: '',
    priceText: '',
    description: '',
    attributes: {},
    tags: [],
    images: [],
    contact: {
      name: '',
      phone: '',
      wechat: '',
    },
  }
}

function clonePublishForm() {
  return JSON.parse(JSON.stringify(form))
}

function serializeResourceImageEntry(entry) {
  return {
    id: entry.id,
    kind: entry.kind,
    url: entry.url,
    file: entry.file,
  }
}

function syncSelectedTypeIndex() {
  if (!resourceTypes.value.length) return
  const matchIndex = resourceTypes.value.findIndex((item) => item.typeCode === form.typeCode)
  selectedTypeIndex.value = matchIndex >= 0 ? matchIndex : 0
  form.typeCode = resourceTypes.value[selectedTypeIndex.value]?.typeCode || ''
  syncAttributesWithSelectedType()
}

function normalizeDynamicFieldItems(fieldSchema = {}) {
  const fields = Array.isArray(fieldSchema.fields) ? fieldSchema.fields : []
  return fields
    .map((field) => ({
      key: String(field?.key || '').trim(),
      label: String(field?.label || field?.key || '').trim(),
      type: normalizeDynamicFieldType(field?.type),
      options: Array.isArray(field?.options) ? field.options.filter(Boolean) : [],
      placeholder: field?.placeholder || '',
    }))
    .filter((field) => field.key && field.label)
}

function normalizeDynamicFieldType(type) {
  if (['select', 'boolean', 'number', 'textarea'].includes(type)) {
    return type
  }
  return 'text'
}

function syncAttributesWithSelectedType() {
  const allowedKeys = new Set(dynamicFieldItems.value.map((field) => field.key))
  Object.keys(form.attributes || {}).forEach((key) => {
    if (!allowedKeys.has(key)) {
      delete form.attributes[key]
    }
  })
}

function getDynamicFieldValue(key) {
  return form.attributes[key]
}

function setDynamicFieldValue(key, value) {
  if (!key) return
  form.attributes[key] = value
}

function setDynamicFieldBoolean(key, value) {
  setDynamicFieldValue(key, value)
}

function setDynamicFieldSelect(field, event) {
  const index = Number(event.detail.value)
  setDynamicFieldValue(field.key, field.options[index] || '')
}

function getDynamicFieldOptionIndex(field) {
  const value = getDynamicFieldValue(field.key)
  const index = field.options.findIndex((item) => item === value)
  return index >= 0 ? index : 0
}

function isPublishFieldCompleted(field) {
  if (field === 'typeCode') return Boolean(form.typeCode)
  if (field === 'title') return Boolean(form.title.trim())
  if (field === 'category') return Boolean(form.category.trim())
  if (field === 'quantityText') return Boolean(form.quantityText.trim())
  if (field === 'priceText') return Boolean(form.priceText.trim())
  if (field === 'description') return Boolean(form.description.trim())
  if (field === 'contactName') return Boolean(form.contact.name.trim())
  if (field === 'contactPhone') return Boolean(form.contact.phone.trim())
  if (field === 'contactWechat') return Boolean(form.contact.wechat.trim())
  if (field === 'images') return resourceImageEntries.value.length > 0
  if (field === 'tags') return form.tags.length > 0
  return !isDynamicAttributeEmpty(form.attributes[field])
}

function isDynamicAttributeEmpty(value) {
  if (value === false || value === 0) return false
  if (Array.isArray(value)) return value.length === 0
  if (typeof value === 'string') return !value.trim()
  return value === undefined || value === null
}

async function uploadResourceImage() {
  try {
    if (resourceImageEntries.value.length >= resourceImageMaxCount) {
      uni.showToast({ title: `最多上传${resourceImageMaxCount}张图片`, icon: 'none' })
      return
    }
    const file = await chooseImageFile()
    resourceImageEntries.value.push(createPendingResourceImageEntry(file))
  } catch (err) {
    if (String(err?.errMsg || '').includes('cancel')) return
    uni.showToast({ title: err.message || '图片选择失败，请重试', icon: 'none' })
  }
}

async function uploadPendingResourceImages() {
  if (!resourceImageEntries.value.some((entry) => entry.kind === 'pending')) {
    return getStoredResourceImageUrls(resourceImageEntries.value)
  }
  const uploadedEntries = []
  for (const entry of resourceImageEntries.value) {
    if (entry.kind === 'stored') {
      uploadedEntries.push(entry)
      continue
    }
    const uploadedUrl = await uploadSelectedImage(entry.file, 'resource')
    uploadedEntries.push(createStoredResourceImageEntry(uploadedUrl))
  }
  resourceImageEntries.value = uploadedEntries
  form.images = getStoredResourceImageUrls(uploadedEntries)
  return form.images
}

function createPendingResourceImageEntry(file) {
  return {
    id: file.id || `pending:${Date.now()}:${file.path}`,
    kind: 'pending',
    url: file.path,
    file,
  }
}

function createStoredResourceImageEntry(url) {
  return {
    id: `stored:${url}`,
    kind: 'stored',
    url,
  }
}

function getResourceImagePreviewUrl(entry) {
  return entry?.url || ''
}

function getResourceImagePreviewUrls(entries) {
  return entries.map(getResourceImagePreviewUrl).filter(Boolean)
}

function getStoredResourceImageUrls(entries) {
  return entries
    .filter((entry) => entry.kind === 'stored')
    .map((entry) => entry.url)
    .filter(Boolean)
}

function onResourceImageGridItemClick(event) {
  const item = resourceImageGridItems.value[Number(event.detail.index)]
  if (!item) return
  if (item.type === 'add') {
    uploadResourceImage()
    return
  }
  previewResourceImage(item)
}

function previewResourceImage(item) {
  const urls = getResourceImagePreviewUrls(resourceImageEntries.value)
  if (!item.url || !urls.length) return
  uni.previewImage({
    urls,
    current: item.url,
  })
}

function removeResourceImage(item) {
  const index = Number(item.index)
  if (index >= 0) {
    resourceImageEntries.value.splice(index, 1)
    form.images = getStoredResourceImageUrls(resourceImageEntries.value)
  }
}

function validatePublishForm() {
  if (!form.merchantId) {
    uni.showToast({ title: '请先完善商家资料', icon: 'none' })
    return false
  }
  if (!form.typeCode) {
    uni.showToast({ title: '请选择资源类型', icon: 'none' })
    return false
  }
  if (!form.title.trim()) {
    uni.showToast({ title: '请填写标题', icon: 'none' })
    return false
  }
  if (!form.category.trim()) {
    uni.showToast({ title: '请填写品类', icon: 'none' })
    return false
  }
  if (!form.contact.name.trim()) {
    uni.showToast({ title: '请填写联系人', icon: 'none' })
    return false
  }
  if (!form.contact.phone.trim()) {
    uni.showToast({ title: '请填写联系电话', icon: 'none' })
    return false
  }
  const missingConfiguredField = requiredFields.value.find((field) => !isPublishFieldCompleted(field))
  if (missingConfiguredField) {
    uni.showToast({ title: `请填写${getPublishFieldLabel(missingConfiguredField)}`, icon: 'none' })
    return false
  }
  return true
}

function getPublishFieldLabel(field) {
  const dynamicField = dynamicFieldItems.value.find((item) => item.key === field)
  if (dynamicField?.label) return dynamicField.label
  const labels = {
    typeCode: '资源类型',
    title: '标题',
    category: '品类',
    quantityText: '数量/产能',
    priceText: '价格描述',
    description: '资源描述',
    contactName: '联系人',
    contactPhone: '联系电话',
    contactWechat: '联系微信',
    images: '资源图片',
    tags: '资源标签',
  }
  return labels[field] || '配置字段'
}
</script>

<style lang="scss" scoped>
.publish-page {
  min-height: 100vh;
  padding: 24rpx;
  background: $wplink-bg;
}

.form-section {
  display: grid;
  gap: 20rpx;
  margin-bottom: 20rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.04);
}

.section-note,
.field-helper {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.45;
}

.basic-progress {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
  padding: 18rpx;
  border-radius: 10rpx;
  background: #f8fafc;
}

.toggle-group {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16rpx;
}

.toggle-option {
  height: 76rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 10rpx;
  background: #fff;
  color: $wplink-muted;
  font-size: 28rpx;
  line-height: 76rpx;
}

.toggle-option.active {
  border-color: $wplink-warning;
  background: rgba(194, 58, 0, 0.08);
  color: $wplink-warning;
  font-weight: 700;
}

.progress-copy {
  display: grid;
  gap: 4rpx;
  min-width: 0;
}

.progress-title {
  color: $wplink-primary;
  font-size: 26rpx;
  font-weight: 700;
  line-height: 1.35;
}

.progress-desc {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.4;
}

.completion-percent {
  flex: 0 0 auto;
  color: $wplink-primary;
  font-size: 34rpx;
  font-weight: 700;
  line-height: 1.15;
}

.completion-bar {
  width: 100%;
  height: 12rpx;
  overflow: hidden;
  border-radius: 999rpx;
  background: rgba(6, 22, 37, 0.1);
}

.completion-bar-fill {
  height: 100%;
  border-radius: inherit;
  background: $wplink-warning;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
  min-width: 0;
}

.section-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
  line-height: 1.3;
}

.section-note {
  flex: 0 0 auto;
  padding: 4rpx 12rpx;
  border-radius: 999rpx;
  background: #f8fafc;
}

.field-group {
  display: grid;
  gap: 10rpx;
}

.field-label {
  color: $wplink-primary;
  font-size: 26rpx;
  font-weight: 700;
}

.field-helper {
  display: block;
}

.field,
.textarea,
.picker-field {
  width: 100%;
  border: 1rpx solid $wplink-line;
  border-radius: 10rpx;
  background: #ffffff;
  font-size: 26rpx;
  color: $wplink-primary;
}

.field {
  min-height: 80rpx;
  padding: 0 20rpx;
}

.picker-field {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 80rpx;
  padding: 0 20rpx;
}

.picker-arrow {
  color: $wplink-muted;
  font-size: 36rpx;
  line-height: 1;
}

.textarea {
  min-height: 160rpx;
  padding: 18rpx 20rpx;
  line-height: 1.5;
}

.image-count {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.45;
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

.resource-image {
  width: 100%;
  height: 100%;
  border-radius: 10rpx;
  background: #e3e8ef;
}

.img-del {
  position: absolute;
  top: 8rpx;
  right: 8rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44rpx;
  height: 44rpx;
  padding: 0;
  border-radius: 50%;
  background: rgba(15, 23, 42, 0.72);
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
  background: #ffffff;
}

.img-del-line {
  transform: rotate(45deg);
}

.img-del-line::after {
  position: absolute;
  top: 0;
  left: 0;
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

.fixed-save-spacer {
  height: calc(102rpx + env(safe-area-inset-bottom));
}

.fixed-save-spacer.no-safe-area {
  height: 102rpx;
}

.fixed-save-bar {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 20;
  padding: 10rpx 24rpx calc(4rpx + env(safe-area-inset-bottom));
  border-top: 1rpx solid $wplink-line;
  background: rgba(255, 255, 255, 0.96);
}

.fixed-save-bar.no-safe-area {
  padding-bottom: 4rpx;
}

.fixed-save-actions {
  display: grid;
  grid-template-columns: minmax(0, 0.9fr) minmax(0, 1.1fr);
  gap: 16rpx;
}

.secondary-button,
.primary-button {
  height: 88rpx;
  border-radius: 12rpx;
  font-size: 28rpx;
  font-weight: 700;
  line-height: 1.25;
}

.secondary-button {
  background: #edf2f7;
  color: #364152;
}

.primary-button {
  background: $wplink-primary;
  color: $wplink-card;
}

.primary-button.is-disabled {
  opacity: 0.56;
}
</style>
