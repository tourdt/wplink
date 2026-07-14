import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'components/ResourcePublishForm.vue'), 'utf8')

test('resource publish form locks category chosen before entering the form', () => {
  for (const token of [
    'category-lock-card',
    'category-lock-label',
    'category-lock-main',
    'category-lock-sub',
    'selectedGroupName',
    'selectedTypeLabel',
    '发布类目',
    '已选择',
  ]) {
    assert.match(source, new RegExp(token))
  }

  assert.doesNotMatch(source, /<picker :range="resourceTypeNames" :value="selectedTypeIndex" @change="selectType">/)
  assert.doesNotMatch(source, /function selectType\(event\)/)
  assert.doesNotMatch(source, /field-helper/)
  assert.match(source, /const currentResourceType = computed/)
  assert.match(source, /const selectedGroupName = computed\(\(\) => currentResourceType\.value\.groupName \|\| currentResourceType\.value\.group\?\.name \|\| '发布大类'\)/)
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
