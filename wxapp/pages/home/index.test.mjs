import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/home/index.vue'), 'utf8')

test('home platform picks fall back to public resources when curated feed is empty', () => {
  assert.match(source, /import \{ listResources \} from '\.\.\/\.\.\/api\/resource'/)
  assert.match(source, /async function resolveHomeResourceItems\(resp\)/)
  assert.match(source, /if \(items\.length\) return items/)
  assert.match(source, /async function loadFallbackHomeResources\(\)/)
  assert.match(source, /listResources\([\s\S]*cityCode: DEFAULT_CITY_CODE[\s\S]*pageSize: 30[\s\S]*suppressErrorToast: true[\s\S]*\)/)
})

test('home cold-start entries exclude recruitment and job seeking', () => {
  assert.doesNotMatch(source, /招聘/)
  assert.doesNotMatch(source, /求职/)
  assert.doesNotMatch(source, /groupCode:\s*'jobs'/)
})

test('home displays at most six recent onboarded merchants without blocking other feeds', () => {
  const discoveryApiSource = fs.readFileSync(path.join(root, 'api/discovery.js'), 'utf8')

  assert.match(source, /import HomeRecentMerchantCard from '\.\.\/\.\.\/components\/HomeRecentMerchantCard\.vue'/)
  assert.match(source, /import \{[\s\S]*listHomeRecentMerchants[\s\S]*\} from '\.\.\/\.\.\/api\/discovery'/)
  assert.match(source, /const RECENT_MERCHANT_LIMIT = 6/)
  assert.match(source, /Promise\.all\(\[loadHomeOperationConfig\(\), loadHomeRecentMerchants\(\), loadHomeResources\(\)\]\)/)
  assert.match(source, /recentMerchants\.value = \(resp\.items \|\| \[\]\)\.slice\(0, RECENT_MERCHANT_LIMIT\)/)
  assert.match(source, /catch \{[\s\S]*recentMerchants\.value = \[\]/)
  assert.match(source, /v-if="recentMerchants\.length" class="recent-merchant-section"/)
  assert.match(source, /v-for="item in recentMerchants"/)
  assert.match(source, /@open="openMerchant"/)
  assert.match(source, /openSourcingMap/)
  assert.match(discoveryApiSource, /url: '\/api\/v1\/home\/recent-merchants'/)
  assert.match(discoveryApiSource, /suppressErrorToast: true/)
})

test('home recent merchants appear between quick entries and recent resources', () => {
  const quickEntryIndex = source.indexOf('class="quick-action-grid"')
  const merchantSectionIndex = source.indexOf('class="recent-merchant-section"')
  const resourceSectionIndex = source.indexOf('>近期供需<')

  assert(quickEntryIndex >= 0, 'quick entry grid should exist')
  assert(merchantSectionIndex > quickEntryIndex, 'recent merchants should follow quick entries')
  assert(resourceSectionIndex > merchantSectionIndex, 'recent resources should follow recent merchants')
})
