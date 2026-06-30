import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/search/result.vue'), 'utf8')

test('search result page keeps the main tools and removes explanatory copy', () => {
  for (const token of [
    'class="search-bar"',
    'class="filter-row"',
    'class="hot-row"',
    'ResourceCard',
    '提交采购需求',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  for (const removedText of [
    '输入关键词或先选热门条件',
    '刷新保存',
    '推广资源均需审核通过',
    '平台运营会继续留意库存',
    'search-guide',
    'promotion-note',
    'empty-desc',
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
