import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/publish/index.vue'), 'utf8')

test('publish entry requires login but does not require merchant profile completion', () => {
  assert.match(source, /import \{ requireLogin \} from '\.\.\/\.\.\/common\/auth'/)
  assert.doesNotMatch(source, /ensureMerchantProfileReady/)
  assert.match(source, /async function applyPendingPublishType\(\) \{[\s\S]*if \(!requireLogin\(\)\) return[\s\S]*navigateToPublishForm/)
  assert.match(source, /function startPublishGroup\(group\) \{[\s\S]*if \(!requireLogin\(\)\) return[\s\S]*selectedCategoryGroup\.value = group[\s\S]*showTypeSheet\.value = true/)
  assert.match(source, /function startPublishCategory\(item\) \{[\s\S]*if \(!requireLogin\(\)\) return[\s\S]*navigateToPublishForm\(\{ typeCode: item\.typeCode \}\)/)
})

test('publish entry shows pre-publish content rules and penalty notice', () => {
  assert.match(source, /<view class="publish-rule-card">/)
  for (const token of [
    '发布前须知',
    '平台审核',
    '提交后进入平台审核',
    '真实有效',
    '联系方式',
    '价格',
    '数量',
    '交期',
    '服务范围',
    '禁止违规',
    '虚假夸大',
    '重复刷屏',
    '无关推广',
    '侵权',
    '误导交易',
    '审核与权益',
    '违规内容可能下架',
    '限制功能',
    '封停账号',
    '收回相关权益',
    '费用不予退还',
  ]) {
    assert.match(source, new RegExp(token))
  }
  for (const removedText of [
    '请确认发布信息真实、合法、有效，内容应与实际供给或需求一致。',
    '禁止发布违法违规、虚假夸大、重复刷屏、无关推广或误导交易的信息。',
    '违规内容可能被下架；情节严重的，平台有权限制功能、封停账号，并收回已购买或已发放的相关权益，费用不予退还。',
  ]) {
    assert.doesNotMatch(source, new RegExp(removedText))
  }
})

test('publish entry shows primary category groups and chooses secondary type in bottom sheet', () => {
  assert.doesNotMatch(source, /<view class="entry-head">/)
  assert.doesNotMatch(source, /选择本次要发布的内容类型，后续字段会按供给或需求自动切换。/)
  for (const token of [
    'publish-category-section',
    'direction-section-title',
    '选择发布大类',
    'categoryGroups',
    'publish-category-list',
    'category-group-button',
    'category-group-name',
    'type-sheet-mask',
    'type-sheet-panel',
    'type-sheet-title',
    'type-option-list',
    'type-option-button',
    'type-option-name',
    '选择具体发布类型',
    '童装批发、厂房仓库、本地服务',
    '系统会自动匹配供应或需求表单',
    '类目加载中...',
    '暂无可发布类目，请稍后重试',
  ]) {
    assert.match(source, new RegExp(token))
  }
  assert.match(source, /import \{ groupResourceTypes \} from '\.\.\/\.\.\/common\/resourceCategories'/)
  assert.match(source, /listCityResourceTypes\(DEFAULT_CITY_CODE\)/)
  assert.match(source, /categoryGroups\.value = groupResourceTypes\(resp\.items \|\| \[\]\)/)
  assert.match(source, /v-for="group in categoryGroups"[\s\S]*@click="startPublishGroup\(group\)"/)
  assert.match(source, /v-for="item in selectedCategoryItems"[\s\S]*@click="startPublishCategory\(item\)"/)
  assert.match(source, /const selectedCategoryItems = computed\(\(\) => selectedCategoryGroup\.value\?\.items \|\| \[\]\)/)
  assert.match(source, /if \(items\.length === 1\) \{[\s\S]*navigateToPublishForm\(\{ typeCode: items\[0\]\.typeCode \}\)[\s\S]*return[\s\S]*\}/)
  assert.match(source, /function navigateToPublishForm\(options = \{\}\) \{[\s\S]*typeCode: options\.typeCode \|\| '',[\s\S]*typeCode=\$\{encodeURIComponent\(initialPublishOptions\.typeCode\)\}/)
  assert.match(source, /\.publish-category-list \{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\);/)
  assert.match(source, /\.type-sheet-mask \{[\s\S]*position: fixed;[\s\S]*align-items: flex-end;/)
  assert.match(source, /\.type-option-button \{[\s\S]*text-align: left;/)
  for (const token of [
    'category-item-grid',
    'category-item-button',
    'category-item-direction',
    "item.direction === RESOURCE_DIRECTION_DEMAND ? '需求' : '供应'",
    'publish-direction-section',
    'publish-direction-list',
    'publish-direction-row',
    'direction-marker',
    'direction-arrow',
    '<text class="direction-arrow">›</text>',
    '我能提供',
    '我想寻找',
    'publish-direction-card',
    'direction-card-body',
    'direction-icon',
    'direction-icon-text',
    "icon: '供'",
    "icon: '需'",
    'direction-meta',
    'direction-chip',
    'direction-title-row',
    'direction-action',
    '发布供给',
    '开始填写',
    '快速发布',
    '采购找货',
    '让买家看到你的货源、产能和服务',
    '适合发布现货、库存、工厂、招聘、配套服务等可供应内容',
    '让供应商看到你的采购和合作需求',
    '适合发布找现货、找库存、找工厂、找服务等采购需求',
  ]) {
    assert.doesNotMatch(source, new RegExp(token))
  }
  assert.doesNotMatch(source, /采购需求/)
})
