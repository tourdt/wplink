import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('..', import.meta.url).pathname)

test('admin launch UI hides manual matching feature', () => {
  const visibleFiles = [
    'src/router/index.js',
    'src/layouts/AdminLayout.vue',
    'src/views/DashboardView.vue',
    'src/views/DemandView.vue',
  ]

  const visibleSource = visibleFiles.map((file) => fs.readFileSync(path.join(root, file), 'utf8')).join('\n')

  assert.equal(visibleSource.includes('match-cases'), false)
  assert.equal(visibleSource.includes('人工撮合'), false)
  assert.equal(visibleSource.includes('待撮合'), false)
})

test('banner topic form uses image upload with ratio guidance', () => {
  const source = fs.readFileSync(path.join(root, 'src/views/BannerTopicView.vue'), 'utf8')

  assert.match(source, /<el-upload/)
  assert.match(source, /建议比例\s*2\.2:1/)
  assert.match(source, /上传封面/)
  assert.equal(source.includes('placeholder="https://..."'), false)
})

test('banner config unifies topic entry and uses selectable non-web targets', () => {
  const source = fs.readFileSync(path.join(root, 'src/views/BannerTopicView.vue'), 'utf8')
  const layoutSource = fs.readFileSync(path.join(root, 'src/layouts/AdminLayout.vue'), 'utf8')

  assert.match(source, /<h2>首页运营位<\/h2>/)
  assert.match(layoutSource, /<span>首页运营位<\/span>/)
  assert.match(source, /v-if="form\.jumpType === 'webview'"/)
  assert.match(source, /v-else-if="form\.jumpType === 'internal'"/)
  assert.match(source, /internalPageOptions/)
  assert.match(source, /resourceOptions/)
  assert.match(source, /merchantOptions/)
  assert.match(source, /kindOptions/)
  assert.equal(source.includes('<el-option label="专题" value="topic" />'), false)
})

test('home recommend card config hides banner image fields', () => {
  const source = fs.readFileSync(path.join(root, 'src/views/BannerTopicView.vue'), 'utf8')

  assert.match(source, /home_recommend_card/)
  assert.match(source, /v-if="isBannerKind"/)
  assert.match(source, /form\.kind === 'home_recommend_card'/)
  assert.match(source, /角标/)
  assert.match(source, /buildSubmitPayload\(\)[\s\S]*payload\.coverUrl = ''/)
})

test('resource type config explains required field meanings', () => {
  const source = fs.readFileSync(path.join(root, 'src/views/ResourceTypeConfigView.vue'), 'utf8')

  assert.match(source, /必填字段控制商家发布或保存资源时必须补全的信息/)
  assert.match(source, /fieldDescriptionMap/)
  assert.match(source, /标题用于搜索、列表卡片和详情页主标题/)
  assert.match(source, /联系电话用于买家联系和平台审核核验/)
  assert.match(source, /required-field-note/)
})

test('banner target selectors support searchable remote options', () => {
  const source = fs.readFileSync(path.join(root, 'src/views/BannerTopicView.vue'), 'utf8')

  assert.match(source, /remote-method="searchResourceTargets"/)
  assert.match(source, /remote-method="searchMerchantTargets"/)
  assert.match(source, /loading="resourceTargetLoading"/)
  assert.match(source, /loading="merchantTargetLoading"/)
  assert.match(source, /keyword: query/)
})

test('hot search keywords are configurable from admin web', () => {
  const routeSource = fs.readFileSync(path.join(root, 'src/router/index.js'), 'utf8')
  const layoutSource = fs.readFileSync(path.join(root, 'src/layouts/AdminLayout.vue'), 'utf8')
  const apiSource = fs.readFileSync(path.join(root, 'src/api/hotSearchKeyword.js'), 'utf8')
  const viewSource = fs.readFileSync(path.join(root, 'src/views/HotSearchKeywordView.vue'), 'utf8')

  assert.match(routeSource, /HotSearchKeywordView/)
  assert.match(routeSource, /hot-search-keywords/)
  assert.match(layoutSource, /<span>热门搜索词<\/span>/)
  assert.match(apiSource, /\/api\/v1\/admin\/hot-search-keywords/)
  assert.match(viewSource, /<h2>热门搜索词<\/h2>/)
  assert.match(viewSource, /v-model="form\.keyword"/)
  assert.match(viewSource, /v-model="form\.status"/)
})
