<template>
  <view class="publish-page">
    <view class="form-section basic-section">
      <view class="section-head">
        <text class="section-title">{{ directionLabels.basicTitle }}</text>
        <text class="section-note">必填</text>
      </view>
      <view class="field-group">
        <text class="field-label">标题</text>
        <input v-model="form.title" class="field" :placeholder="directionLabels.titlePlaceholder" />
      </view>
    </view>

    <view class="form-section supply-section">
      <view class="section-head">
        <text class="section-title">{{ directionLabels.detailTitle }}</text>
        <text class="section-note">建议填写</text>
      </view>
      <view class="field-group">
        <text class="field-label">{{ directionLabels.descriptionLabel }}</text>
        <textarea v-model="form.description" class="textarea" :placeholder="directionLabels.descriptionPlaceholder" />
      </view>
    </view>

    <view v-if="dynamicFieldItems.length" class="form-section attribute-section">
      <view class="section-head">
        <text class="section-title">{{ directionLabels.attributeTitle }}</text>
        <text class="section-note">{{ directionLabels.attributeNote }}</text>
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
        <view v-else-if="isDynamicFieldCustomSelect(field)" class="select-with-custom">
          <picker
            v-if="getDynamicFieldSelectOptions(field).length"
            :range="getDynamicFieldSelectOptions(field)"
            :value="getDynamicFieldOptionIndex(field)"
            @change="setDynamicFieldSelect(field, $event)"
          >
            <view class="field picker-field">
              <text>{{ getDynamicFieldValue(field.key) || `请选择${field.label}` }}</text>
              <text class="picker-arrow">›</text>
            </view>
          </picker>
          <input
            v-if="isDynamicFieldCustomSelectActive(field)"
            class="field"
            :value="form.attributes[field.key]"
            :placeholder="field.placeholder || `请填写${field.label}`"
            @input="setDynamicFieldCustomSelect(field, $event.detail.value)"
          />
        </view>
        <picker
          v-else-if="field.type === 'select' && field.options.length"
          :range="field.options"
          :value="getDynamicFieldOptionIndex(field)"
          @change="setDynamicFieldSelect(field, $event)"
        >
          <view class="field picker-field">
            <text>{{ getDynamicFieldValue(field.key) || `请选择${field.label}` }}</text>
            <text class="picker-arrow">›</text>
          </view>
        </picker>
        <input
          v-else-if="field.type === 'select'"
          class="field"
          :value="form.attributes[field.key]"
          :placeholder="field.placeholder || `请填写${field.label}`"
          @input="setDynamicFieldValue(field.key, $event.detail.value)"
        />
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
        <text class="section-title">{{ directionLabels.imageTitle }}</text>
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
        <input v-model="form.contact.name" class="field" :placeholder="directionLabels.contactNamePlaceholder" />
      </view>
      <view class="field-group">
        <text class="field-label">联系电话</text>
        <input v-model="form.contact.phone" class="field" :placeholder="directionLabels.contactPhonePlaceholder" />
      </view>
      <view class="field-group">
        <text class="field-label">微信号</text>
        <input
          v-model="form.contact.wechat"
          class="field"
          maxlength="32"
          :placeholder="directionLabels.contactWechatPlaceholder"
          @input="sanitizeContactWechat"
        />
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
import { flattenGroupedResourceTypes, groupResourceTypes } from '../common/resourceCategories'

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

const RESOURCE_DIRECTION_SUPPLY = 'supply'
const RESOURCE_DIRECTION_DEMAND = 'demand'
const CUSTOM_SELECT_OPTION_LABEL = '其他'
const summaryFieldNames = new Set(['category', 'quantityText', 'priceText'])
const customSelectOptionLabels = new Set(['其他', '其它', '自定义', '其他/自定义'])

const reserveBottomSafeArea = computed(() => props.reserveBottomSafeArea)
const resourceTypes = ref([])
const selectedTypeIndex = ref(0)
const resourceImageMaxCount = 9
const resourceImageEntries = ref([])
const publishLocalDraftStorageKey = ref('')
const initialRouteTypeCode = ref('')
const autosaveReady = ref(false)
const editingResourceId = ref('')
const editingResourceStatus = ref('')
const editSavedAsDraft = ref(true)
let localDraftSaveTimer = null
const form = reactive({
  merchantId: '',
  cityCode: DEFAULT_CITY_CODE,
  direction: RESOURCE_DIRECTION_SUPPLY,
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
const customSelectFieldKeys = reactive({})

const currentResourceType = computed(() => resourceTypes.value[selectedTypeIndex.value] || {})
const isDemandDirection = computed(() => form.direction === RESOURCE_DIRECTION_DEMAND)
const directionLabels = computed(() => {
  if (isDemandDirection.value) {
    return {
      basicTitle: '需求信息',
      typeLabel: '需求类型',
      titlePlaceholder: '例如：急找童装春款现货 3000 件',
      detailTitle: '需求说明',
      descriptionLabel: '需求描述',
      descriptionPlaceholder: '说明款式、尺码颜色、交期、验货和交付要求',
      attributeTitle: '需求字段',
      attributeNote: '按需求类型',
      imageTitle: '参考图片',
      contactNamePlaceholder: '供应商看到的联系人',
      contactPhonePlaceholder: '用于供应商发起联系',
      contactWechatPlaceholder: '选填，供应商可复制联系',
    }
  }
  return {
    basicTitle: '基础信息',
    typeLabel: '供应类型',
    titlePlaceholder: '例如：童装春款现货 3000 件',
    detailTitle: '供应说明',
    descriptionLabel: '供应描述',
    descriptionPlaceholder: '说明货品状态、尺码颜色、交期、看样方式等关键信息',
    attributeTitle: '类型字段',
    attributeNote: '按供应类型',
    imageTitle: '供应图片',
    contactNamePlaceholder: '买家看到的联系人',
    contactPhonePlaceholder: '用于买家发起联系',
    contactWechatPlaceholder: '选填，买家可复制联系',
  }
})
const dynamicFieldItems = computed(() => normalizeDynamicFieldItems(currentResourceType.value.fieldSchema))
const requiredFields = computed(() => {
  const configuredFields = Array.isArray(currentResourceType.value.requiredFields) ? currentResourceType.value.requiredFields : []
  const visibleConfiguredFields = configuredFields.filter((field) => !summaryFieldNames.has(field))
  return Array.from(new Set(['typeCode', 'title', 'contactName', 'contactPhone', ...visibleConfiguredFields]))
})
const requiredFieldStates = computed(() => requiredFields.value.map(isPublishFieldCompleted))
const canSubmit = computed(() => requiredFieldStates.value.every(Boolean))
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
  initialRouteTypeCode.value = options.typeCode || ''
  Object.assign(form, createEmptyPublishForm())
  resetCustomSelectFieldKeys()
  resourceImageEntries.value = []
  selectedTypeIndex.value = 0
  form.merchantId = options.merchantId || getMerchantId()
  form.direction = normalizePublishDirection(options.direction || '') || RESOURCE_DIRECTION_SUPPLY
  form.typeCode = options.typeCode || ''
  publishLocalDraftStorageKey.value = buildPublishLocalDraftStorageKey(form.merchantId, editingResourceId.value, initialRouteTypeCode.value)
  if (editingResourceId.value) {
    await loadEditableResource()
  } else {
    await loadResourceTypes()
  }
  if (options.repost) {
    restoreRepostInitialForm()
    publishLocalDraftStorageKey.value = buildPublishLocalDraftStorageKey(form.merchantId, editingResourceId.value, initialRouteTypeCode.value || form.typeCode)
    await loadResourceTypes()
  } else if (!editingResourceId.value) {
    await loadMerchantContact()
    restorePublishLocalDraft()
  }
  autosaveReady.value = true
}

async function loadResourceTypes() {
  const resp = await listCityResourceTypes(form.cityCode)
  resourceTypes.value = flattenGroupedResourceTypes(groupResourceTypes(resp.items || []))
  if (!resourceTypes.value.length) {
    form.typeCode = ''
    return
  }
  const matchIndex = resourceTypes.value.findIndex((item) => item.typeCode === form.typeCode)
  selectedTypeIndex.value = matchIndex >= 0 ? matchIndex : 0
  applySelectedResourceType()
}

function applySelectedResourceType() {
  const current = resourceTypes.value[selectedTypeIndex.value] || {}
  form.typeCode = current.typeCode || ''
  form.direction = normalizePublishDirection(current.direction || '') || RESOURCE_DIRECTION_SUPPLY
  syncAttributesWithSelectedType()
  syncPublishNavigationTitle()
}

function syncPublishNavigationTitle() {
  const typeTitle = normalizeSummaryText(currentResourceType.value.typeName)
  if (typeTitle) {
    uni.setNavigationBarTitle({ title: typeTitle })
    return
  }
  if (!editingResourceId.value) {
    uni.setNavigationBarTitle({ title: isDemandDirection.value ? '发布需求' : '发布供应' })
  }
}

async function loadMerchantContact() {
  if (!form.merchantId) return
  try {
    const detail = await getMerchant(form.merchantId, { suppressErrorToast: true })
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
    direction: normalizePublishDirection(detail.direction || '') || RESOURCE_DIRECTION_SUPPLY,
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
  await loadResourceTypes()
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
    direction: normalizePublishDirection(initialForm.direction || '') || form.direction,
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
}

function applyMerchantContactDefaults(contact) {
  if (!form.contact.name.trim() && contact.name) {
    form.contact.name = contact.name
  }
  if (!form.contact.phone.trim() && (contact.phone || contact.phoneMasked)) {
    form.contact.phone = contact.phone || contact.phoneMasked
  }
  if (!form.contact.wechat.trim() && contact.wechat) {
    // 微信号会被用户复制使用，不能把脱敏值写入发布内容。
    form.contact.wechat = sanitizeContactWechatValue(contact.wechat)
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
    const resp = await submitResource(editingResourceId.value, form.merchantId)
    openPublishSuccess(resp)
    clearPublishLocalDraft()
    resetPublishForm()
    uni.showToast({ title: publishSubmitToast(resp), icon: 'none' })
    return
  }
  const images = await uploadPendingResourceImages()
  const resp = await createResource(buildResourcePublishPayload(images))
  openPublishSuccess(resp)
  clearPublishLocalDraft()
  resetPublishForm()
  uni.showToast({ title: publishSubmitToast(resp), icon: 'none' })
}

function openPublishSuccess(result = {}) {
  const publishDirection = normalizePublishDirection(form.direction) || RESOURCE_DIRECTION_SUPPLY
  const query = [
    `direction=${encodeURIComponent(publishDirection)}`,
    `status=${encodeURIComponent(result.status || 'pending')}`,
    `message=${encodeURIComponent(result.message || '')}`,
  ].join('&')
  uni.navigateTo({ url: `/pages/publish-success/index?${query}` })
}

function publishSubmitToast(result = {}) {
  if (result.status === 'published') return '已发布'
  if (result.status === 'rejected') return '审核未通过'
  return '已提交审核'
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
  const payload = buildResourcePublishPayload(images)
  if (!editingResourceId.value) {
    return createResourceDraft(payload)
  }
  const resp = await updateResourceDraft(editingResourceId.value, payload)
  editingResourceStatus.value = 'draft'
  editSavedAsDraft.value = true
  return resp
}

function buildPublishLocalDraftStorageKey(merchantId, resourceId = '', typeCode = '') {
  const draftScope = resourceId || (typeCode ? `new-type-${typeCode}` : `new-${form.direction || RESOURCE_DIRECTION_SUPPLY}`)
  return `publish:local-draft:${merchantId || 'default'}:${draftScope}`
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
  if (initialRouteTypeCode.value) {
    form.typeCode = initialRouteTypeCode.value
    syncSelectedTypeIndex()
  } else {
    syncSelectedTypeIndex()
  }
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
  initialRouteTypeCode.value = ''
  Object.assign(form, createEmptyPublishForm())
  resetCustomSelectFieldKeys()
  resourceImageEntries.value = []
  selectedTypeIndex.value = 0
}

function createEmptyPublishForm() {
  return {
    merchantId: '',
    cityCode: DEFAULT_CITY_CODE,
    direction: RESOURCE_DIRECTION_SUPPLY,
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

function normalizePublishDirection(value) {
  return [RESOURCE_DIRECTION_SUPPLY, RESOURCE_DIRECTION_DEMAND].includes(value) ? value : ''
}

function clonePublishForm() {
  return JSON.parse(JSON.stringify(form))
}

function buildResourcePublishPayload(images) {
  const payload = {
    ...clonePublishForm(),
    images,
  }
  payload.contact.wechat = sanitizeContactWechatValue(payload.contact.wechat)
  applySummaryFieldsToPayload(payload, currentResourceType.value)
  return payload
}

function applySummaryFieldsToPayload(payload, resourceType = {}) {
  const summary = resourceType.displayTemplate?.summary || {}
  for (const field of summaryFieldNames) {
    const sourceKey = String(summary[field] || '').trim()
    payload[field] = sourceKey ? getPublishSummarySourceValue(payload, sourceKey) : normalizeSummaryText(payload[field])
  }
}

function getPublishSummarySourceValue(payload, sourceKey) {
  if (sourceKey in (payload.attributes || {})) {
    return normalizeSummaryText(payload.attributes[sourceKey])
  }
  return normalizeSummaryText(payload[sourceKey])
}

function normalizeSummaryText(value) {
  if (value === false) return '否'
  if (value === true) return '是'
  if (value === undefined || value === null) return ''
  return String(value).trim()
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
  applySelectedResourceType()
}

function normalizeDynamicFieldItems(fieldSchema = {}) {
  const fields = Array.isArray(fieldSchema.fields) ? fieldSchema.fields : []
  return fields
    .map((field) => ({
      key: String(field?.key || '').trim(),
      label: String(field?.label || field?.key || '').trim(),
      type: normalizeDynamicFieldType(field?.type),
      options: normalizeDynamicFieldOptions(field?.options),
      allowCustom: field?.allowCustom === true,
      required: field?.required === true,
      filterable: field?.filterable === true,
      displayIn: Array.isArray(field?.displayIn) ? field.displayIn.filter(Boolean) : [],
      placeholder: field?.placeholder || '',
    }))
    .filter((field) => field.key && field.label)
}

function normalizeDynamicFieldOptions(options) {
  if (!Array.isArray(options)) return []
  return options
    .map((item) => String(item || '').trim())
    .filter(Boolean)
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
  Object.keys(customSelectFieldKeys).forEach((key) => {
    if (!allowedKeys.has(key)) {
      delete customSelectFieldKeys[key]
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
  const option = getDynamicFieldSelectOptions(field)[index] || ''
  if (field.allowCustom && isDynamicFieldCustomOption(option)) {
    customSelectFieldKeys[field.key] = true
    if (isDynamicFieldConfiguredOption(field, getDynamicFieldValue(field.key))) {
      setDynamicFieldValue(field.key, '')
    }
    return
  }
  customSelectFieldKeys[field.key] = false
  setDynamicFieldValue(field.key, option)
}

function setDynamicFieldCustomSelect(field, value) {
  customSelectFieldKeys[field.key] = true
  setDynamicFieldValue(field.key, value)
}

function getDynamicFieldOptionIndex(field) {
  const options = getDynamicFieldSelectOptions(field)
  if (isDynamicFieldCustomSelectActive(field)) {
    const customIndex = options.findIndex(isDynamicFieldCustomOption)
    return customIndex >= 0 ? customIndex : 0
  }
  const value = getDynamicFieldValue(field.key)
  const index = options.findIndex((item) => item === value)
  return index >= 0 ? index : 0
}

function isDynamicFieldCustomSelect(field) {
  return field.type === 'select' && field.allowCustom
}

function getDynamicFieldSelectOptions(field) {
  const options = Array.isArray(field.options) ? field.options : []
  if (!field.allowCustom || options.some(isDynamicFieldCustomOption)) {
    return options
  }
  return [...options, CUSTOM_SELECT_OPTION_LABEL]
}

function isDynamicFieldCustomSelectActive(field) {
  if (!isDynamicFieldCustomSelect(field)) return false
  if (!field.options.length) return true
  if (customSelectFieldKeys[field.key] === true) return true
  const value = normalizeSummaryText(getDynamicFieldValue(field.key))
  return value !== '' && !isDynamicFieldConfiguredOption(field, value)
}

function isDynamicFieldConfiguredOption(field, value) {
  const text = normalizeSummaryText(value)
  return text !== '' && field.options.includes(text) && !isDynamicFieldCustomOption(text)
}

function isDynamicFieldCustomOption(option) {
  return customSelectOptionLabels.has(normalizeSummaryText(option))
}

function resetCustomSelectFieldKeys() {
  Object.keys(customSelectFieldKeys).forEach((key) => {
    delete customSelectFieldKeys[key]
  })
}

function sanitizeContactWechat(event) {
  form.contact.wechat = sanitizeContactWechatValue(event?.detail?.value ?? form.contact.wechat)
}

function sanitizeContactWechatValue(value) {
  return String(value || '').replace(/[^a-zA-Z0-9_-]/g, '').slice(0, 32)
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
    uni.showToast({ title: '请重新登录后发布', icon: 'none' })
    return false
  }
  if (!form.typeCode) {
    uni.showToast({ title: `请选择${directionLabels.value.typeLabel}`, icon: 'none' })
    return false
  }
  if (!form.title.trim()) {
    uni.showToast({ title: '请填写标题', icon: 'none' })
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
    typeCode: directionLabels.value.typeLabel,
    title: '标题',
    category: '分类摘要',
    quantityText: '数量摘要',
    priceText: '价格摘要',
    description: directionLabels.value.descriptionLabel,
    contactName: '联系人',
    contactPhone: '联系电话',
    contactWechat: '联系微信',
    images: directionLabels.value.imageTitle,
    tags: isDemandDirection.value ? '需求标签' : '供应标签',
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

.section-note {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.45;
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

.select-with-custom {
  display: grid;
  gap: 12rpx;
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
