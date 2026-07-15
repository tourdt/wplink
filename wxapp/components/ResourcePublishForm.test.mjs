import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

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

test('resource publish form includes optional contact wechat input', () => {
  assert.match(source, /<text class="field-label">微信号<\/text>/)
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
