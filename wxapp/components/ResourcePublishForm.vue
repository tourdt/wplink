<template>
  <view class="publish-page">
    <view class="form-section basic-section">
      <view class="section-head">
        <text class="section-title">{{ directionLabels.detailTitle }}</text>
        <text class="section-note">必填</text>
      </view>
      <view class="field-group">
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
        <view v-else-if="field.type === 'address'" class="address-field">
          <view class="address-row">
            <input
              class="field"
              :value="getDynamicAddressText(field.key)"
              :placeholder="field.placeholder || `请输入${field.label}`"
              @input="setDynamicAddressText(field.key, $event.detail.value)"
            />
            <button class="map-button" @click="chooseDynamicAddress(field)">地图选择</button>
          </view>
        </view>
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

    <view v-if="resourceTagOptions.length" class="form-section tag-section">
      <view class="section-head">
        <text class="section-title">{{ resourceTagTitle }}</text>
        <text class="section-note">{{ resourceTagSectionNote }}</text>
      </view>
      <view class="tag-option-grid">
        <button
          v-for="tag in resourceTagOptions"
          :key="tag"
          :class="['tag-option', isResourceTagSelected(tag) ? 'active' : '']"
          @click="toggleResourceTag(tag)"
        >
          {{ tag }}
        </button>
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
        <text class="section-note">2项必填</text>
      </view>
      <view class="field-group">
        <view class="field-label-row">
          <text class="field-label">联系人</text>
          <text class="field-tag required">必填</text>
        </view>
        <input v-model="form.contact.name" class="field" :placeholder="directionLabels.contactNamePlaceholder" />
      </view>
      <view class="field-group">
        <view class="field-label-row">
          <text class="field-label">联系电话</text>
          <text class="field-tag required">必填</text>
        </view>
        <input v-model="form.contact.phone" class="field" :placeholder="directionLabels.contactPhonePlaceholder" />
      </view>
      <view class="field-group">
        <view class="field-label-row">
          <text class="field-label">微信号</text>
          <text class="field-tag optional">选填</text>
        </view>
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
import { DEFAULT_CITY_CODE, DEFAULT_CITY_LOCATION } from '../common/constants'
import { getMerchantId, saveMerchantId } from '../store/session'
import { listCityResourceTypes } from '../api/city'
import { reverseGeocodeLocation } from '../api/location'
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
const DEFAULT_MAX_RESOURCE_TAGS = 8
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
      typeLabel: '需求类型',
      detailTitle: '需求描述',
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
    typeLabel: '供应类型',
    detailTitle: '供应描述',
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
  const visibleConfiguredFields = configuredFields.filter((field) => !summaryFieldNames.has(field) && field !== 'title')
  return Array.from(new Set(['typeCode', 'description', 'contactName', 'contactPhone', ...visibleConfiguredFields]))
})
const requiredFieldStates = computed(() => requiredFields.value.map(isPublishFieldCompleted))
const canSubmit = computed(() => requiredFieldStates.value.every(Boolean))
const resourceTagOptions = computed(() => normalizeResourceTagOptions(currentResourceType.value.fieldSchema))
const resourceTagMaxCount = computed(() => normalizeResourceTagMaxCount(currentResourceType.value.fieldSchema))
const resourceTagsRequired = computed(() => requiredFields.value.includes('tags'))
const resourceTagTitle = computed(() => {
  const configuredTitle = normalizeSummaryText(currentResourceType.value.fieldSchema?.tagLabel)
  if (configuredTitle) return configuredTitle
  return isDemandDirection.value ? '需求标签' : '供应标签'
})
const resourceTagSectionNote = computed(() => {
  const prefix = resourceTagsRequired.value ? '必填' : '选填'
  return `${prefix}，最多${resourceTagMaxCount.value}个`
})
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
  syncTagsWithSelectedType()
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
    const images = await uploadPendingResourceImages()
    if (!editSavedAsDraft.value || editingResourceStatus.value !== 'draft') {
      // 驳回资源或有未保存改动的草稿，提交审核前先落库为草稿，保证审核使用的是当前编辑内容。
      await saveResourceDraftPayload(images)
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
  payload.tags = normalizeSelectedResourceTags(payload.tags, resourceTagOptions.value).slice(0, resourceTagMaxCount.value)
  payload.contact.wechat = sanitizeContactWechatValue(payload.contact.wechat)
  applySummaryFieldsToPayload(payload, currentResourceType.value)
  // 发布页不再让用户单独填写标题，提交前用类型摘要和描述生成稳定标题，兼容列表、分享和审核消息。
  payload.title = buildAutoResourceTitle(payload, currentResourceType.value)
  return payload
}

function buildAutoResourceTitle(payload, resourceType = {}) {
  const typeName = normalizeSummaryText(resourceType.typeName)
  const summaryParts = [payload.category, payload.quantityText, payload.priceText].map(normalizeSummaryText).filter(Boolean)
  const titleFromSummary = [typeName, summaryParts.join(' ')].filter(Boolean).join('｜')
  if (titleFromSummary) return truncateResourceTitle(titleFromSummary)
  return truncateResourceTitle(normalizeSummaryText(payload.description))
}

function truncateResourceTitle(value, maxLength = 30) {
  const chars = Array.from(String(value || '').trim())
  if (chars.length <= maxLength) return chars.join('')
  return `${chars.slice(0, maxLength).join('')}...`
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
  if (typeof value === 'object') return normalizeAddressText(value)
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
  if (['select', 'boolean', 'number', 'textarea', 'address'].includes(type)) {
    return type
  }
  return 'text'
}

function syncAttributesWithSelectedType() {
  const allowedKeys = new Set(dynamicFieldItems.value.map((field) => field.key))
  const nextAttributes = { ...(form.attributes || {}) }
  Object.keys(nextAttributes).forEach((key) => {
    if (!allowedKeys.has(key)) {
      delete nextAttributes[key]
    }
  })
  form.attributes = nextAttributes
  Object.keys(customSelectFieldKeys).forEach((key) => {
    if (!allowedKeys.has(key)) {
      delete customSelectFieldKeys[key]
    }
  })
}

function syncTagsWithSelectedType() {
  form.tags = normalizeSelectedResourceTags(form.tags, resourceTagOptions.value).slice(0, resourceTagMaxCount.value)
}

function normalizeResourceTagOptions(fieldSchema = {}) {
  return normalizeTagTextList(fieldSchema?.tagOptions || [])
}

function normalizeResourceTagMaxCount(fieldSchema = {}) {
  const configuredMax = Number(fieldSchema?.maxTags || fieldSchema?.tagMaxCount)
  if (!Number.isFinite(configuredMax) || configuredMax <= 0) {
    return DEFAULT_MAX_RESOURCE_TAGS
  }
  return Math.min(Math.floor(configuredMax), DEFAULT_MAX_RESOURCE_TAGS)
}

function normalizeSelectedResourceTags(tags = [], options = []) {
  const allowedTags = new Set(options)
  return normalizeTagTextList(tags).filter((tag) => !allowedTags.size || allowedTags.has(tag))
}

function normalizeTagTextList(items = []) {
  if (!Array.isArray(items)) return []
  const seen = new Set()
  const normalized = []
  for (const item of items) {
    const tag = normalizeTagText(item)
    if (!tag || seen.has(tag)) continue
    seen.add(tag)
    normalized.push(tag)
  }
  return normalized
}

function normalizeTagText(value) {
  return String(value || '').trim()
}

function isResourceTagSelected(tag) {
  return form.tags.includes(tag)
}

function toggleResourceTag(tag) {
  tag = normalizeTagText(tag)
  if (!tag) return
  const currentTags = normalizeSelectedResourceTags(form.tags, resourceTagOptions.value)
  if (currentTags.includes(tag)) {
    form.tags = currentTags.filter((item) => item !== tag)
    return
  }
  if (currentTags.length >= resourceTagMaxCount.value) {
    uni.showToast({ title: `最多选择${resourceTagMaxCount.value}个标签`, icon: 'none' })
    return
  }
  form.tags = [...currentTags, tag]
}

function getDynamicFieldValue(key) {
  return form.attributes[key]
}

function setDynamicFieldValue(key, value) {
  if (!key) return
  // 替换 attributes 对象，确保微信小程序端新增/替换对象型字段后，输入框 value 能立即刷新。
  form.attributes = {
    ...(form.attributes || {}),
    [key]: value,
  }
}

function setDynamicFieldBoolean(key, value) {
  setDynamicFieldValue(key, value)
}

function getDynamicAddressText(key) {
  return normalizeAddressText(getDynamicFieldValue(key))
}

function setDynamicAddressText(key, value) {
  const address = String(value || '').trim()
  // 用户手动修改地址后，原经纬度可能已经不再匹配新地址；丢弃旧坐标可避免详情页导航到旧位置。
  setDynamicFieldValue(key, address ? { address } : '')
}

async function chooseDynamicAddress(field) {
  if (typeof uni.chooseLocation !== 'function') {
    uni.showToast({ title: '当前环境不支持地图选点', icon: 'none' })
    return
  }
  const initialLocation = await resolveChooseLocationInitialLocation()
  uni.chooseLocation({
    latitude: initialLocation.latitude,
    longitude: initialLocation.longitude,
    success: async (result) => {
      console.log('供需表单地图选点成功', result)
      const latitude = Number(result?.latitude)
      const longitude = Number(result?.longitude)
      if (!Number.isFinite(latitude) || !Number.isFinite(longitude)) {
        uni.showToast({ title: '未获取到有效地图位置', icon: 'none' })
        return
      }
      const resolvedAddress = await resolveChooseLocationAddress(result, field.key, latitude, longitude)
      if (!resolvedAddress.address) {
        // 微信在拖动到非 POI 点位时可能只返回经纬度；反查和手填兜底都失败时，不保存无法展示的“空地址”。
        console.warn('供需表单地图未返回详细地址', {
          hasName: Boolean(result?.name),
          hasAddress: Boolean(result?.address),
          hasLatitude: Number.isFinite(latitude),
          hasLongitude: Number.isFinite(longitude),
        })
        uni.showToast({ title: '未获取到详细地址，请搜索具体地点或先填写地址', icon: 'none' })
        return
      }
      console.log('供需表单地图选点结果', {
        address: resolvedAddress.address,
        name: resolvedAddress.name,
        source: resolvedAddress.source,
        latitude,
        longitude,
      })
      setDynamicFieldValue(field.key, {
        address: resolvedAddress.address,
        name: resolvedAddress.name,
        latitude,
        longitude,
      })
      uni.showToast({ title: '地图位置已保存', icon: 'none' })
    },
    fail: (err) => {
      if (String(err?.errMsg || '').includes('cancel')) return
      // 真实设备上地图选点失败通常来自隐私接口未声明、用户拒绝授权或系统定位关闭；记录原始 errMsg 便于排查，前端只展示可操作提示。
      console.warn('供需表单地图选择失败', err)
      uni.showToast({ title: resolveChooseLocationErrorMessage(err), icon: 'none' })
    },
  })
}

async function resolveChooseLocationInitialLocation() {
  const fallback = DEFAULT_CITY_LOCATION
  const currentLocation = await getCurrentMapLocation()
  if (!currentLocation) return fallback
  const geocoded = await reverseGeocodeCurrentLocation(currentLocation.latitude, currentLocation.longitude)
  if (isZhejiangLocation(geocoded)) {
    return currentLocation
  }
  console.log('供需表单地图默认定位到织里', {
    currentProvince: geocoded.province,
    hasCurrentAddress: Boolean(geocoded.address),
  })
  return fallback
}

function getCurrentMapLocation() {
  if (typeof uni.getLocation !== 'function') {
    return Promise.resolve(null)
  }
  return new Promise((resolve) => {
    uni.getLocation({
      type: 'gcj02',
      success: (result) => {
        const latitude = Number(result?.latitude)
        const longitude = Number(result?.longitude)
        if (!Number.isFinite(latitude) || !Number.isFinite(longitude)) {
          resolve(null)
          return
        }
        resolve({ latitude, longitude })
      },
      fail: (err) => {
        // 当前位置只用于决定地图初始视野；失败不阻断选点，直接回到织里默认中心。
        console.warn('供需表单获取当前位置失败，使用织里默认地图中心', {
          errMsg: err?.errMsg || err?.message || String(err || ''),
        })
        resolve(null)
      },
    })
  })
}

async function reverseGeocodeCurrentLocation(latitude, longitude) {
  try {
    const resp = await reverseGeocodeLocation({ latitude, longitude })
    return {
      address: normalizeChooseLocationText(resp?.address),
      province: normalizeChooseLocationText(resp?.province),
    }
  } catch (err) {
    console.warn('供需表单当前位置省份判断失败，使用织里默认地图中心', {
      latitude,
      longitude,
      errMsg: err?.message || err?.errMsg || String(err || ''),
    })
    return { address: '', province: '' }
  }
}

function isZhejiangLocation(location) {
  const province = normalizeChooseLocationText(location?.province)
  const address = normalizeChooseLocationText(location?.address)
  return province.includes('浙江') || address.includes('浙江')
}

async function resolveChooseLocationAddress(result, key, latitude, longitude) {
  const selectedAddress = buildChooseLocationAddressText(result)
  const selectedName = normalizeChooseLocationText(result?.name)
  if (selectedAddress) {
    return {
      address: selectedAddress,
      name: selectedName,
      source: 'chooseLocation',
    }
  }
  const geocoded = await reverseGeocodeChooseLocation(latitude, longitude)
  if (geocoded.address) {
    return {
      address: geocoded.address,
      name: geocoded.name || selectedName,
      source: 'reverseGeocode',
    }
  }
  const manualAddress = normalizeChooseLocationText(getDynamicAddressText(key))
  if (manualAddress) {
    return {
      address: manualAddress,
      name: selectedName,
      source: 'manualInput',
    }
  }
  return { address: '', name: selectedName, source: '' }
}

async function reverseGeocodeChooseLocation(latitude, longitude) {
  uni.showLoading({ title: '解析地址中', mask: false })
  try {
    const resp = await reverseGeocodeLocation({ latitude, longitude })
    const address = normalizeChooseLocationText(resp?.address)
    const name = normalizeChooseLocationText(resp?.name)
    if (!address) {
      console.warn('供需表单地图地址反查未返回详细地址', { latitude, longitude })
      return { address: '', name: '' }
    }
    return { address, name }
  } catch (err) {
    console.warn('供需表单地图地址反查失败', {
      latitude,
      longitude,
      errMsg: err?.message || err?.errMsg || String(err || ''),
    })
    return { address: '', name: '' }
  } finally {
    uni.hideLoading()
  }
}

function buildChooseLocationAddressText(result) {
  const address = normalizeChooseLocationText(result?.address)
  const name = normalizeChooseLocationText(result?.name)
  if (address && name && !address.includes(name) && !name.includes(address)) {
    return `${address}${name}`
  }
  return address || name
}

function normalizeChooseLocationText(value) {
  const text = String(value || '').trim()
  if (!text || isCoordinateAddressText(text)) return ''
  return text
}

function isCoordinateAddressText(text) {
  const value = String(text || '').trim()
  if (!value) return false
  if (/^-?\d+(\.\d+)?\s*[,，]\s*-?\d+(\.\d+)?$/.test(value)) return true
  if (/^(gps|经纬度|坐标|地图位置)[:：\s(（-]*-?\d+(\.\d+)?/i.test(value)) return true
  return /纬度[:：]?\s*-?\d+(\.\d+)?[\s,，;；]+经度[:：]?\s*-?\d+(\.\d+)?/.test(value)
}

function resolveChooseLocationErrorMessage(err) {
  const errMsg = String(err?.errMsg || err?.message || '').toLowerCase()
  if (errMsg.includes('requiredprivateinfos') || errMsg.includes('declared')) {
    return '地图能力未完成配置，请联系管理员'
  }
  if (errMsg.includes('auth deny') || errMsg.includes('authorize') || errMsg.includes('scope.userlocation')) {
    return '请允许位置权限后再选择地图'
  }
  if (errMsg.includes('system permission denied') || errMsg.includes('permission denied')) {
    return '请开启手机定位权限后再选择地图'
  }
  return '地图选择失败，请稍后重试'
}

function normalizeAddressText(value) {
  if (value === undefined || value === null) return ''
  if (typeof value === 'string') return value.trim()
  if (typeof value !== 'object') return String(value).trim()
  return String(value.address || value.name || '').trim()
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
  if (value && typeof value === 'object') return !normalizeAddressText(value)
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
  if (!form.description.trim()) {
    uni.showToast({ title: `请填写${directionLabels.value.descriptionLabel}`, icon: 'none' })
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

.tag-option-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 14rpx;
}

.tag-option {
  box-sizing: border-box;
  min-width: 132rpx;
  height: 64rpx;
  padding: 0 18rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 10rpx;
  background: #fff;
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 64rpx;
}

.tag-option::after {
  border: 0;
}

.tag-option.active {
  border-color: $wplink-warning;
  background: rgba(194, 58, 0, 0.08);
  color: $wplink-warning;
  font-weight: 700;
}

.select-with-custom {
  display: grid;
  gap: 12rpx;
}

.address-field {
  display: grid;
  gap: 10rpx;
}

.address-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 148rpx;
  gap: 12rpx;
  align-items: center;
}

.map-button {
  height: 80rpx;
  padding: 0;
  border-radius: 10rpx;
  background: $wplink-primary;
  color: $wplink-card;
  font-size: 24rpx;
  font-weight: 700;
  line-height: 80rpx;
}

.map-button::after {
  border: 0;
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

.field-label-row {
  display: flex;
  align-items: center;
  gap: 10rpx;
  min-width: 0;
}

.field-tag {
  flex: 0 0 auto;
  padding: 2rpx 10rpx;
  border-radius: 999rpx;
  font-size: 22rpx;
  font-weight: 700;
  line-height: 1.4;
}

.field-tag.required {
  background: rgba(194, 58, 0, 0.1);
  color: $wplink-primary;
}

.field-tag.optional {
  background: #f8fafc;
  color: $wplink-muted;
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
