import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import vm from 'node:vm'
import { compileTemplate, parse } from '@vue/compiler-sfc'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'

const root = path.resolve(new URL('..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'components/ResourcePublishForm.vue'), 'utf8')

test('resource publish form keeps chosen type locked without repeating category labels', () => {
  assert.doesNotMatch(source, /<picker :range="resourceTypeNames" :value="selectedTypeIndex" @change="selectType">/)
  assert.doesNotMatch(source, /function selectType\(event\)/)
  assert.doesNotMatch(source, /field-helper/)
  for (const removedToken of [
    'category-lock-card',
    'category-lock-label',
    'category-lock-main',
    'category-lock-sub',
    'category-lock-badge',
    'selectedGroupName',
    'selectedTypeLabel',
  ]) {
    assert.doesNotMatch(source, new RegExp(removedToken))
  }
  assert.match(source, /const currentResourceType = computed/)
  assert.match(source, /syncPublishNavigationTitle\(\)/)
  assert.match(source, /uni\.setNavigationBarTitle\(\{ title: typeTitle \}\)/)
})

test('resource publish form does not let local draft overwrite route type code', () => {
  assert.match(source, /const initialRouteTypeCode = ref\(''\)/)
  assert.match(source, /initialRouteTypeCode\.value = options\.typeCode \|\| ''/)
  assert.match(source, /buildPublishLocalDraftStorageKey\(form\.merchantId, editingResourceId\.value, initialRouteTypeCode\.value\)/)
  assert.match(source, /function buildPublishLocalDraftStorageKey\(merchantId, resourceId = '', typeCode = ''\)/)
  assert.match(source, /const draftScope = resourceId \|\| \(typeCode \? `new-type-\$\{typeCode\}` : `new-\$\{form\.direction \|\| RESOURCE_DIRECTION_SUPPLY\}`\)/)
  assert.match(source, /if \(initialRouteTypeCode\.value\) \{[\s\S]*form\.typeCode = initialRouteTypeCode\.value[\s\S]*syncSelectedTypeIndex\(\)[\s\S]*\}/)
})

test('resource publish form removes completion progress from the basic section', () => {
  for (const removedToken of [
    'basic-progress',
    'progress-copy',
    'progress-title',
    'completion-percent',
    'completionPercent',
    'completionBarStyle',
    'publishReadyText',
  ]) {
    assert.doesNotMatch(source, new RegExp(removedToken))
  }

  assert.match(source, /const canSubmit = computed\(\(\) => requiredFieldStates\.value\.every\(Boolean\)\)/)
})

test('resource publish form only asks for description and auto-generates title', () => {
  assert.doesNotMatch(source, /<text class="field-label">标题<\/text>/)
  assert.doesNotMatch(source, /v-model="form\.title"/)
  assert.doesNotMatch(source, /titlePlaceholder/)
  assert.doesNotMatch(source, /请填写标题/)
  assert.doesNotMatch(source, /\['typeCode', 'title', 'contactName', 'contactPhone'/)
  assert.match(source, /<text class="section-title">\{\{ directionLabels\.detailTitle \}\}<\/text>/)
  assert.match(source, /<text class="section-note">必填<\/text>/)
  assert.match(source, /detailTitle: '需求描述'/)
  assert.match(source, /detailTitle: '供应描述'/)
  assert.doesNotMatch(source, /showDescriptionLabel/)
  assert.doesNotMatch(source, /detailTitle: '需求说明'/)
  assert.doesNotMatch(source, /detailTitle: '供应说明'/)
  assert.match(source, /v-model="form\.description"/)
  assert.match(source, /\['typeCode', 'description', 'contactName', 'contactPhone'/)
  assert.match(source, /field !== 'title'/)
  assert.match(source, /payload\.title = buildAutoResourceTitle\(payload, currentResourceType\.value\)/)
  assert.match(source, /function buildAutoResourceTitle\(payload, resourceType = \{\}\)/)
  assert.match(source, /请填写\$\{directionLabels\.value\.descriptionLabel\}/)
})

test('resource publish form includes optional contact wechat input', () => {
  assert.match(source, /<text class="section-title">联系信息<\/text>/)
  assert.match(source, /<text class="section-note">2项必填<\/text>/)
  assert.match(source, /<text class="field-label">联系人<\/text>[\s\S]*?<text class="field-tag required">必填<\/text>/)
  assert.match(source, /<text class="field-label">联系电话<\/text>[\s\S]*?<text class="field-tag required">必填<\/text>/)
  assert.match(source, /<text class="field-label">微信号<\/text>[\s\S]*?<text class="field-tag optional">选填<\/text>/)
  assert.match(source, /v-model="form\.contact\.wechat"/)
  assert.match(source, /maxlength="32"/)
  assert.match(source, /@input="sanitizeContactWechat"/)
  assert.match(source, /contactWechatPlaceholder: '选填，买家可复制联系'/)
  assert.match(source, /contactWechatPlaceholder: '选填，供应商可复制联系'/)
  assert.match(source, /function sanitizeContactWechat\(event\)/)
  assert.match(source, /replace\(\/\[\^a-zA-Z0-9_-\]\/g, ''\)/)
  assert.match(source, /form\.contact\.wechat = sanitizeContactWechatValue\(contact\.wechat\)/)
  assert.match(source, /payload\.contact\.wechat = sanitizeContactWechatValue\(payload\.contact\.wechat\)/)
})

test('resource publish form submits edited rejected resources after saving current changes', () => {
  assert.doesNotMatch(source, /请先保存草稿后再提交审核/)
  assert.match(source, /if \(editingResourceId\.value\) \{[\s\S]*const images = await uploadPendingResourceImages\(\)[\s\S]*if \(!editSavedAsDraft\.value \|\| editingResourceStatus\.value !== 'draft'\) \{[\s\S]*await saveResourceDraftPayload\(images\)[\s\S]*const resp = await submitResource\(editingResourceId\.value, form\.merchantId\)/)
})

test('resource publish form supports address fields with optional map location', () => {
  assert.match(source, /field\.type === 'address'/)
  assert.match(source, /getDynamicAddressText\(field\.key\)/)
  assert.match(source, /setDynamicAddressText\(field\.key, \$event\.detail\.value\)/)
  assert.match(source, /chooseDynamicAddress\(field\)/)
  assert.doesNotMatch(source, /hasDynamicAddressLocation\(field\.key\)/)
  assert.doesNotMatch(source, /手动输入不显示地图/)
  assert.doesNotMatch(source, /清除定位/)
  assert.doesNotMatch(source, /location-status/)
  assert.doesNotMatch(source, /location-clear-button/)
  assert.doesNotMatch(source, /请选择有效地址/)
  assert.match(source, /function setDynamicAddressText\(key, value\) \{[\s\S]*原经纬度可能已经不再匹配新地址[\s\S]*setDynamicFieldValue\(key, address \? \{ address \} : ''\)/)
  assert.match(source, /function setDynamicFieldValue\(key, value\) \{[\s\S]*form\.attributes = \{[\s\S]*\.\.\.\(form\.attributes \|\| \{\}\),[\s\S]*\[key\]: value/)
  assert.match(source, /import \{ DEFAULT_CITY_CODE, DEFAULT_CITY_LOCATION \} from '\.\.\/common\/constants'/)
  assert.match(source, /import \{ reverseGeocodeLocation \} from '\.\.\/api\/location'/)
  assert.match(source, /async function chooseDynamicAddress\(field\) \{[\s\S]*resolveChooseLocationInitialLocation\(\)[\s\S]*uni\.chooseLocation\(\{[\s\S]*latitude: initialLocation\.latitude[\s\S]*longitude: initialLocation\.longitude[\s\S]*success: async \(result\)[\s\S]*未获取到详细地址，请搜索具体地点或先填写地址[\s\S]*地图位置已保存[\s\S]*resolveChooseLocationErrorMessage\(err\)/)
  assert.match(source, /async function resolveChooseLocationInitialLocation\(\) \{[\s\S]*DEFAULT_CITY_LOCATION[\s\S]*getCurrentMapLocation\(\)[\s\S]*reverseGeocodeCurrentLocation\(currentLocation\.latitude, currentLocation\.longitude\)[\s\S]*isZhejiangLocation\(geocoded\)[\s\S]*return fallback/)
  assert.match(source, /function getCurrentMapLocation\(\) \{[\s\S]*uni\.getLocation\(\{[\s\S]*type: 'gcj02'[\s\S]*供需表单获取当前位置失败，使用织里默认地图中心/)
  assert.match(source, /function isZhejiangLocation\(location\) \{[\s\S]*province\.includes\('浙江'\)[\s\S]*address\.includes\('浙江'\)/)
  assert.match(source, /async function resolveChooseLocationAddress\(result, key, latitude, longitude\) \{[\s\S]*buildChooseLocationAddressText\(result\)[\s\S]*reverseGeocodeChooseLocation\(latitude, longitude\)[\s\S]*normalizeChooseLocationText\(getDynamicAddressText\(key\)\)[\s\S]*return \{ address: '', name: selectedName, source: '' \}/)
  assert.match(source, /async function reverseGeocodeChooseLocation\(latitude, longitude\) \{[\s\S]*解析地址中[\s\S]*reverseGeocodeLocation\(\{ latitude, longitude \}\)[\s\S]*供需表单地图地址反查失败[\s\S]*uni\.hideLoading\(\)/)
  assert.match(source, /function buildChooseLocationAddressText\(result\) \{[\s\S]*normalizeChooseLocationText\(result\?\.address\)[\s\S]*normalizeChooseLocationText\(result\?\.name\)[\s\S]*return `\$\{address\}\$\{name\}`[\s\S]*return address \|\| name/)
  assert.match(source, /function normalizeChooseLocationText\(value\) \{[\s\S]*isCoordinateAddressText\(text\)[\s\S]*return text/)
  assert.match(source, /function isCoordinateAddressText\(text\) \{[\s\S]*gps\|经纬度\|坐标\|地图位置[\s\S]*纬度[\s\S]*经度/)
  assert.doesNotMatch(source, /地图位置\(\$\{latitude\.toFixed\(6\)\}, \$\{longitude\.toFixed\(6\)\}\)/)
  assert.match(source, /console\.warn\('供需表单地图选择失败', err\)/)
  assert.match(source, /function resolveChooseLocationErrorMessage\(err\) \{[\s\S]*requiredprivateinfos[\s\S]*请允许位置权限后再选择地图[\s\S]*请开启手机定位权限后再选择地图/)
  assert.doesNotMatch(source, /function clearDynamicAddressLocation\(key\)/)
  assert.match(source, /function normalizeDynamicFieldType\(type\) \{[\s\S]*'address'/)
})

test('resource publish form supports configured multi tag selection', () => {
  assert.match(source, /const DEFAULT_MAX_RESOURCE_TAGS = 8/)
  assert.match(source, /const resourceTagOptions = computed\(\(\) => normalizeResourceTagOptions\(currentResourceType\.value\.fieldSchema\)\)/)
  assert.match(source, /<view v-if="resourceTagOptions\.length" class="form-section tag-section">/)
  assert.match(source, /v-for="tag in resourceTagOptions"/)
  assert.match(source, /isResourceTagSelected\(tag\)/)
  assert.match(source, /@click="toggleResourceTag\(tag\)"/)
  assert.match(source, /payload\.tags = normalizeSelectedResourceTags\(payload\.tags, resourceTagOptions\.value\)\.slice\(0, resourceTagMaxCount\.value\)/)
  assert.match(source, /function syncTagsWithSelectedType\(\)/)
  assert.match(source, /function normalizeResourceTagOptions\(fieldSchema = \{\}\)/)
  assert.match(source, /fieldSchema\?\.tagOptions/)
  assert.match(source, /function toggleResourceTag\(tag\) \{[\s\S]*最多选择\$\{resourceTagMaxCount\.value\}个标签[\s\S]*form\.tags = \[\.\.\.currentTags, tag\]/)
  assert.match(source, /\.tag-option-grid \{[\s\S]*flex-wrap: wrap;/)
  assert.match(source, /\.tag-option\.active \{[\s\S]*border-color: \$wplink-warning;/)
})

test('resource publish form directs publish quota shortages to purchase', () => {
  assert.match(source, /import \{ QUOTA_TYPE_PUBLISH, buildQuotaPurchaseUrl, confirmQuotaPurchase \} from '\.\.\/common\/entitlementPurchase'/)
  assert.match(source, /async function handlePublishQuotaError\(err\) \{[\s\S]*err\?\.code !== 'QUOTA_NOT_ENOUGH'[\s\S]*return false[\s\S]*await confirmQuotaPurchase\(QUOTA_TYPE_PUBLISH\)[\s\S]*uni\.navigateTo\(\{ url: buildQuotaPurchaseUrl\(QUOTA_TYPE_PUBLISH\) \}\)[\s\S]*return true[\s\S]*\}/)
  assert.match(source, /async function submit\(\) \{[\s\S]*try \{[\s\S]*catch \(err\) \{[\s\S]*if \(await handlePublishQuotaError\(err\)\) return[\s\S]*throw err[\s\S]*\}/)
})

function ref(value) {
  return { value }
}

function reactive(value) {
  return value
}

function computed(getter) {
  return { get value() { return getter() } }
}

function deferred() {
  let resolve
  let reject
  const promise = new Promise((nextResolve, nextReject) => {
    resolve = nextResolve
    reject = nextReject
  })
  return { promise, resolve, reject }
}

function flushAsyncWork() {
  return new Promise((resolve) => setTimeout(resolve, 0))
}

function getPublishActionButton(clickExpression) {
  const { descriptor } = parse(source, { filename: 'ResourcePublishForm.vue' })
  const buttons = descriptor.template.content.match(/<button\b[^>]*>[\s\S]*?<\/button>/g) || []
  const button = buttons.find((item) => item.includes(`@click="${clickExpression}"`))
  assert.ok(button, `should render the ${clickExpression} action button`)
  return button
}

async function renderPublishActionButton(clickExpression, context) {
  const compiled = compileTemplate({
    source: getPublishActionButton(clickExpression),
    filename: 'ResourcePublishForm.vue',
    id: 'resource-publish-form-loading',
    compilerOptions: { mode: 'function' },
  })
  assert.equal(compiled.errors.length, 0, compiled.errors.join('\n'))
  const render = new Function('Vue', compiled.code)(await import('vue'))
  const Button = {
    props: Object.keys(context),
    setup(props) {
      return () => render(props, [])
    },
  }
  return renderToString(createSSRApp({ render: () => h(Button, context) }))
}

function loadResourcePublishForm(additions = {}) {
  const { descriptor } = parse(source, { filename: 'ResourcePublishForm.vue' })
  const script = descriptor.scriptSetup.content.replace(/^import\s+[\s\S]*?\s+from\s+['"][^'"]+['"]\s*$/gm, '')
  const sandbox = {
    console,
    Promise,
    Set,
    Date,
    encodeURIComponent,
    setTimeout,
    clearTimeout,
    ref,
    reactive,
    computed,
    watch: () => {},
    onUnmounted: () => {},
    defineProps: () => ({ initialOptions: {}, mode: 'create', reserveBottomSafeArea: true }),
    DEFAULT_CITY_CODE: 'huzhou',
    DEFAULT_CITY_LOCATION: { latitude: 30, longitude: 120 },
    getMerchantId: () => '',
    saveMerchantId: () => {},
    listCityResourceTypes: async () => ({ items: [] }),
    reverseGeocodeLocation: async () => ({}),
    getMerchant: async () => ({}),
    createResource: async () => ({}),
    createResourceDraft: async () => ({}),
    getEditableResource: async () => ({}),
    submitResource: async () => ({}),
    updateResourceDraft: async () => ({}),
    chooseImageFile: async () => ({ id: 'chosen-image', path: '/chosen-image.jpg' }),
    uploadSelectedImage: async () => 'https://example.test/uploaded.jpg',
    flattenGroupedResourceTypes: (items) => items,
    groupResourceTypes: (items) => items,
    QUOTA_TYPE_PUBLISH: 'publish',
    buildQuotaPurchaseUrl: () => '/pages/vip/index',
    confirmQuotaPurchase: async () => false,
    uni: {
      showToast: () => {},
      navigateTo: () => {},
      setNavigationBarTitle: () => {},
      previewImage: () => {},
      getStorageSync: () => null,
      setStorageSync: () => {},
      removeStorageSync: () => {},
      chooseLocation: () => {},
      getLocation: () => {},
      showLoading: () => {},
      hideLoading: () => {},
    },
    ...additions,
  }
  vm.runInNewContext(`${script}\nglobalThis.resourcePublishForm = { form, resourceImageEntries, publishAction: typeof publishAction === 'undefined' ? undefined : publishAction, publishPhase: typeof publishPhase === 'undefined' ? undefined : publishPhase, publishBusy: typeof publishBusy === 'undefined' ? undefined : publishBusy, isPublishAction: typeof isPublishAction === 'undefined' ? undefined : isPublishAction, isPublishActionLoading: typeof isPublishActionLoading === 'undefined' ? undefined : isPublishActionLoading, submit, saveDraft, onResourceImageGridItemClick, removeResourceImage }`, sandbox, { filename: 'ResourcePublishForm.vue' })
  return sandbox.resourcePublishForm
}

function fillValidPublishForm(page) {
  Object.assign(page.form, {
    merchantId: 'merchant-1',
    typeCode: 'supply-clothes',
    description: '一批现货童装',
    contact: { name: '张三', phone: '13800138000', wechat: '' },
  })
}

function assertNativeBusyButton(html, busyText) {
  const openingTag = html.match(/<button\b[^>]*>/)?.[0] || ''
  assert.match(openingTag, /\bdisabled(?:=|\s|>)/, '忙碌按钮应禁用')
  assert.match(openingTag, /\bloading="true"/, '忙碌按钮应使用原生 loading=true')
  assert.match(html, new RegExp(busyText), `忙碌按钮应显示${busyText}`)
}

function assertDisabledWithoutLoading(html, label) {
  const openingTag = html.match(/<button\b[^>]*>/)?.[0] || ''
  assert.match(openingTag, /\bdisabled(?:=|\s|>)/, `${label}在另一动作进行时应禁用`)
  assert.match(openingTag, /\bloading="false"/, `${label}不应显示另一动作的 loading`)
}

test('resource publish actions render native busy feedback for the active action', async () => {
  const noop = () => {}
  const draftContext = {
    publishBusy: true,
    canSubmit: true,
    isPublishAction: (action) => action === 'draft',
    isPublishActionLoading: (action) => action === 'draft',
    saveDraft: noop,
    submit: noop,
  }
  const submitContext = {
    publishBusy: true,
    canSubmit: true,
    isPublishAction: (action) => action === 'submit',
    isPublishActionLoading: (action) => action === 'submit',
    saveDraft: noop,
    submit: noop,
  }
  const savingHtml = await renderPublishActionButton('saveDraft', draftContext)
  const draftLocksSubmitHtml = await renderPublishActionButton('submit', draftContext)
  const submittingHtml = await renderPublishActionButton('submit', submitContext)
  const submitLocksDraftHtml = await renderPublishActionButton('saveDraft', submitContext)

  assertNativeBusyButton(savingHtml, '保存中')
  assertDisabledWithoutLoading(draftLocksSubmitHtml, '提交审核按钮')
  assertNativeBusyButton(submittingHtml, '提交中')
  assertDisabledWithoutLoading(submitLocksDraftHtml, '保存草稿按钮')
})

test('resource publish blocks duplicate and cross-action requests then clears the action after success', async () => {
  const request = deferred()
  let createCalls = 0
  let draftCalls = 0
  const navigations = []
  const toasts = []
  const page = loadResourcePublishForm({
    createResource: () => {
      createCalls += 1
      return request.promise
    },
    createResourceDraft: async () => { draftCalls += 1 },
    uni: {
      showToast: (options) => toasts.push(options),
      navigateTo: (options) => navigations.push(options),
    },
  })
  fillValidPublishForm(page)

  const firstSubmit = page.submit()
  const duplicateSubmit = page.submit()
  const concurrentDraft = page.saveDraft()
  try {
    await flushAsyncWork()
    assert.equal(createCalls, 1, '连续提交审核只应创建一次资源')
    assert.equal(draftCalls, 0, '提交审核期间不应保存草稿')
    assert.equal(page.publishAction.value, 'submit', '提交请求进行中应保留提交动作状态')
  } finally {
    request.resolve({ status: 'pending' })
    await Promise.all([firstSubmit, duplicateSubmit, concurrentDraft])
  }

  assert.equal(page.publishAction.value, '', '提交成功后应清除提交动作状态')
  assert.equal(navigations.at(-1)?.url, '/pages/publish-success/index?direction=supply&status=pending&message=', '提交成功应保留发布成功页跳转')
  assert.equal(toasts.at(-1)?.title, '已提交审核', '提交成功应保留原有成功提示')
})

test('resource draft blocks duplicate and cross-action requests then clears the action after rejection', async () => {
  const request = deferred()
  let draftCalls = 0
  let createCalls = 0
  const page = loadResourcePublishForm({
    createResource: async () => { createCalls += 1 },
    createResourceDraft: () => {
      draftCalls += 1
      return request.promise
    },
  })
  fillValidPublishForm(page)

  const firstDraft = page.saveDraft()
  const duplicateDraft = page.saveDraft()
  const concurrentSubmit = page.submit()
  const rejectedDraft = assert.rejects(firstDraft, /保存草稿失败/)
  const duplicateDraftResult = duplicateDraft.catch((err) => err)
  try {
    await flushAsyncWork()
    assert.equal(draftCalls, 1, '连续保存草稿只应请求一次')
    assert.equal(createCalls, 0, '保存草稿期间不应提交审核')
    assert.equal(page.publishAction.value, 'draft', '保存请求进行中应保留草稿动作状态')
  } finally {
    request.reject(new Error('保存草稿失败'))
  }

  await rejectedDraft
  await Promise.all([duplicateDraftResult, concurrentSubmit])
  assert.equal(page.publishAction.value, '', '草稿保存失败后应清除草稿动作状态')
})

test('resource publish locks image additions and removals while a request is in flight', async () => {
  const request = deferred()
  let chooseImageCalls = 0
  const page = loadResourcePublishForm({
    createResource: () => request.promise,
    chooseImageFile: async () => {
      chooseImageCalls += 1
      return { id: 'chosen-image', path: '/chosen-image.jpg' }
    },
  })
  fillValidPublishForm(page)
  page.resourceImageEntries.value = [{ id: 'stored:image-1', kind: 'stored', url: 'https://example.test/image-1.jpg' }]

  const submitting = page.submit()
  try {
    await flushAsyncWork()
    page.onResourceImageGridItemClick({ detail: { index: 1 } })
    page.removeResourceImage({ index: 0 })
    await flushAsyncWork()
    assert.equal(chooseImageCalls, 0, '发布期间不应打开图片新增入口')
    assert.deepEqual(page.resourceImageEntries.value, [{ id: 'stored:image-1', kind: 'stored', url: 'https://example.test/image-1.jpg' }], '发布期间不应新增或删除图片')
  } finally {
    request.resolve({ status: 'pending' })
    await submitting
  }
})

test('resource publish keeps the action idle and avoids all request work when validation fails', async () => {
  let createCalls = 0
  let draftCalls = 0
  let uploadCalls = 0
  const page = loadResourcePublishForm({
    createResource: async () => { createCalls += 1 },
    createResourceDraft: async () => { draftCalls += 1 },
    uploadSelectedImage: async () => { uploadCalls += 1 },
  })
  page.resourceImageEntries.value = [{
    id: 'pending:image-1',
    kind: 'pending',
    url: '/pending-image.jpg',
    file: { id: 'pending-image-1', path: '/pending-image.jpg' },
  }]

  await Promise.all([page.submit(), page.saveDraft()])

  assert.equal(page.publishAction.value, '', '校验失败不应进入任何发布动作状态')
  assert.equal(createCalls, 0, '校验失败不应创建资源')
  assert.equal(draftCalls, 0, '校验失败不应保存草稿')
  assert.equal(uploadCalls, 0, '校验失败不应上传图片')
})

test('resource publish clears the action and preserves quota-cancel behavior', async () => {
  const navigations = []
  let confirmCalls = 0
  const page = loadResourcePublishForm({
    createResource: async () => { throw { code: 'QUOTA_NOT_ENOUGH' } },
    confirmQuotaPurchase: async () => {
      confirmCalls += 1
      return false
    },
    uni: {
      navigateTo: (options) => navigations.push(options),
      showToast: () => {},
    },
  })
  fillValidPublishForm(page)

  await page.submit()

  assert.equal(confirmCalls, 1, '额度不足应继续请求购买确认')
  assert.deepEqual(navigations, [], '取消购买时不应跳转购买页或发布成功页')
  assert.equal(page.publishAction.value, '', '取消额度购买后应清除提交动作状态')
})

test('resource publish quota confirmation keeps one lock without a spinner and clears on cancel', async () => {
  const confirmation = deferred()
  let createCalls = 0
  let draftCalls = 0
  const page = loadResourcePublishForm({
    createResource: async () => { createCalls += 1; throw { code: 'QUOTA_NOT_ENOUGH' } },
    createResourceDraft: async () => { draftCalls += 1 },
    confirmQuotaPurchase: () => confirmation.promise,
  })
  fillValidPublishForm(page)

  const submitting = page.submit()
  await flushAsyncWork()
  assert.equal(page.publishAction.value, 'submit', '购买确认 pending 时应保留发布互斥锁')
  assert.equal(page.publishPhase?.value, 'prompting', '购买确认 pending 时应进入 prompting')
  const submitHtml = await renderPublishActionButton('submit', {
    publishBusy: true,
    canSubmit: true,
    isPublishAction: (action) => action === 'submit',
    isPublishActionLoading: page.isPublishActionLoading,
    saveDraft: () => {},
    submit: () => {},
  })
  const draftHtml = await renderPublishActionButton('saveDraft', {
    publishBusy: true,
    canSubmit: true,
    isPublishAction: (action) => action === 'submit',
    isPublishActionLoading: page.isPublishActionLoading,
    saveDraft: () => {},
    submit: () => {},
  })
  assert.match(submitHtml, />提交审核</, '购买确认期间提交按钮应恢复空闲文案')
  assert.match(submitHtml, /loading="false"/, '购买确认期间提交按钮不应显示 spinner')
  assert.match(draftHtml, /loading="false"/, '购买确认期间草稿按钮不应显示 spinner')
  await Promise.all([page.submit(), page.saveDraft()])
  assert.equal(createCalls, 1, '购买确认期间重复提交不得再次创建资源')
  assert.equal(draftCalls, 0, '购买确认期间不得穿透互斥锁保存草稿')

  confirmation.resolve(false)
  await submitting
  assert.equal(page.publishAction.value, '', '取消购买确认后应释放发布动作')
  assert.equal(page.publishPhase?.value, '', '取消购买确认后应清空发布阶段')
})

test('resource publish quota confirmation navigates once after confirmation and clears both states', async () => {
  const confirmation = deferred()
  const navigations = []
  const page = loadResourcePublishForm({
    createResource: async () => { throw { code: 'QUOTA_NOT_ENOUGH' } },
    confirmQuotaPurchase: () => confirmation.promise,
    uni: { navigateTo: (options) => navigations.push(options), showToast: () => {} },
  })
  fillValidPublishForm(page)

  const submitting = page.submit()
  await flushAsyncWork()
  confirmation.resolve(true)
  await submitting

  assert.equal(navigations.length, 1, '确认购买后应只跳转一次额度购买页')
  assert.equal(navigations[0].url, '/pages/vip/index')
  assert.equal(page.publishAction.value, '', '确认购买跳转后应释放发布动作')
  assert.equal(page.publishPhase?.value, '', '确认购买跳转后应清空发布阶段')
})
