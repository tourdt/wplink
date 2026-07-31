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
  assert.match(source, /v-if="homeFeedReady && homeFeedState\.hasAnyContent" class="home-feed-section"/)
  assert.match(source, /v-if="activeHomeFeedTab === 'merchants'" class="recent-merchant-list"/)
  assert.match(source, /v-for="item in recentMerchants"/)
  assert.match(source, /@open="openMerchant"/)
  assert.match(source, /function openSourcingMap\(\) \{[\s\S]*uni\.switchTab\(\{ url: '\/pages\/sourcing-map\/index' \}\)[\s\S]*\}/)
  assert.match(discoveryApiSource, /url: '\/api\/v1\/home\/recent-merchants'/)
  assert.match(discoveryApiSource, /suppressErrorToast: true/)
})

test('home combines recent merchants and resources into a local tabbed feed', () => {
  assert.match(source, /import \{ getHomeFeedState \} from '\.\/homeFeedState'/)
  assert.match(source, /const activeHomeFeedTab = ref\(''\)/)
  assert.match(source, /class="home-feed-tabs"/)
  assert.match(source, />新入驻商家</)
  assert.match(source, />近期供需</)
  assert.match(source, /selectHomeFeedTab\('merchants'\)/)
  assert.match(source, /selectHomeFeedTab\('resources'\)/)
  assert.match(source, /v-if="activeHomeFeedTab === 'merchants'"/)
  assert.match(source, /v-else-if="activeHomeFeedTab === 'resources'"/)
})

test('home initializes the feed tab after parallel data loading without refetching on switch', () => {
  assert.match(source, /await Promise\.all\(\[loadHomeOperationConfig\(\), loadHomeRecentMerchants\(\), loadHomeResources\(\)\]\)/)
  assert.match(source, /const homeFeedReady = ref\(false\)/)
  assert.match(source, /activeHomeFeedTab\.value = homeFeedState\.value\.defaultTab/)
  assert.match(source, /homeFeedReady\.value = true/)
  assert.match(source, /v-if="homeFeedReady && homeFeedState\.hasAnyContent"/)
  assert.match(source, /function selectHomeFeedTab\(tab\)/)
  const switchFunction = source.match(/function selectHomeFeedTab\(tab\) \{[\s\S]*?\n\}/)?.[0] || ''
  assert.doesNotMatch(switchFunction, /loadHomeRecentMerchants|loadHomeResources|listHome/)
})

test('home degrades to one available feed and hides an empty feed container', () => {
  assert.match(source, /v-if="homeFeedReady && homeFeedState\.hasAnyContent"/)
  assert.match(source, /v-if="homeFeedState\.showSwitcher"/)
  assert.match(source, /homeFeedState\.hasMerchants/)
  assert.match(source, /homeFeedState\.value\.hasResources/)
  assert.match(source, /v-for="item in recentMerchants"/)
})
