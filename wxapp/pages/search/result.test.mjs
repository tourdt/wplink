import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/search/result.vue'), 'utf8')

function cssBlock(selector) {
  const escapedSelector = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const match = source.match(new RegExp(`${escapedSelector} \\{([\\s\\S]*?)\\n\\}`))
  return match?.[1] || ''
}

test('search result page keeps the main tools and removes explanatory copy', () => {
  for (const token of [
    'class="search-bar"',
    'class="filter-row"',
    'class="hot-row"',
    'ResourceCard',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  for (const removedText of [
    '提交采购需求',
    'openDemand',
    '/pages/demand/index',
    'empty-visual-label',
    '找货',
    '输入关键词或先选热门条件',
    '刷新保存',
    '推广资源均需审核通过',
    '平台运营会继续留意库存',
    'search-guide',
    'promotion-note',
    'saveCurrentSearch',
    'savedSearches.length',
    'createSavedSearch',
    'listSavedSearches',
    'applySavedSearch',
    '保存搜索',
    '已保存搜索',
  ]) {
    assert.doesNotMatch(source, new RegExp(removedText.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
})

test('search result page matches recommendation category browsing controls', () => {
  for (const token of [
    'visibleResourceTypes',
    'scrollIntoTypeId',
    'scrollToSelectedType',
    'getTypeButtonId',
    'scroll-into-view',
    'scroll-with-animation',
    'showTypeDrawer',
    'openTypeDrawer',
    'closeTypeDrawer',
    'type-drawer-mask',
    'type-drawer-panel',
    '全部分类',
    '常用分类',
    'drawer-type-grid',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(source, /visibleResourceTypes = computed\(\(\) => resourceTypes\.value\)/)
  assert.match(source, /v-for="item in visibleResourceTypes"[\s\S]*:id="getTypeButtonId\(item\.value\)"/)
  assert.match(source, /async function selectType\(typeCode\) \{[\s\S]*showTypeDrawer\.value = false[\s\S]*scrollToSelectedType\(typeCode\)[\s\S]*await search\(\)[\s\S]*\}/)
})

test('search result page uses recommendation category font size', () => {
  assert.match(cssBlock('.filter-button'), /font-size:\s*26rpx;/)
})

test('search result empty state gives search-aware recovery actions', () => {
  for (const token of [
    'emptyTitle',
    'emptyDesc',
    'emptySuggestions',
    'empty-actions',
    'empty-suggestions',
    '换个条件',
    'resetSearchConditions',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(source, /const emptyTitle = '暂无匹配资源'/)
  assert.match(source, /const emptySuggestions = computed\(\(\) => hotKeywords[\s\S]*slice\(0, 3\)\)/)
  assert.match(source, /async function resetSearchConditions\(\) \{[\s\S]*keyword\.value = ''[\s\S]*filters\.typeCode = ''[\s\S]*searched\.value = false[\s\S]*\}/)
})

test('search result empty state keeps copy short and illustration text-free', () => {
  assert.match(source, /const emptyTitle = '暂无匹配资源'/)
  assert.match(source, /const emptyDesc = '换个关键词或分类试试。'/)
  assert.doesNotMatch(source, /暂未找到「/)
  assert.doesNotMatch(source, /平台资源会保持更新/)
})

test('search result empty state is visually subdued', () => {
  assert.match(cssBlock('.empty-card'), /padding:\s*32rpx 24rpx;/)
  assert.doesNotMatch(cssBlock('.empty-card'), /box-shadow/)
  assert.match(cssBlock('.empty-visual'), /width:\s*168rpx;/)
  assert.match(cssBlock('.empty-visual'), /height:\s*112rpx;/)
  assert.match(cssBlock('.empty-title'), /font-size:\s*28rpx;/)
  assert.match(cssBlock('.empty-title'), /color:\s*\$wplink-text;/)
  assert.match(cssBlock('.empty-desc'), /font-size:\s*24rpx;/)
})
