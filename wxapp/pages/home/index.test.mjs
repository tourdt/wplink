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
  assert.match(source, /v-if="activeHomeFeedTab === 'merchants'" class="recent-merchant-list"/)
  assert.match(source, /v-else class="home-resource-panel"/)
  assert.doesNotMatch(source, /v-show="activeHomeFeedTab ===/)
  assert.match(source, /\.home-feed-more\s*\{[\s\S]*min-height:\s*88rpx/)

  const tabListTag = source.match(/<view\s+[^>]*class="home-feed-tabs"[^>]*>/)?.[0] || ''
  const merchantTab = source.match(
    /<button\n\s+:class="\['home-feed-tab', \{ active: activeHomeFeedTab === 'merchants' \}\]"[\s\S]*?<\/button>/,
  )?.[0] || ''
  const resourceTab = source.match(
    /<button\n\s+:class="\['home-feed-tab', \{ active: activeHomeFeedTab === 'resources' \}\]"[\s\S]*?<\/button>/,
  )?.[0] || ''

  assert.match(tabListTag, /role="tablist"/)
  assert.match(tabListTag, /aria-label="首页内容"/)
  assert.match(merchantTab, /role="tab"/)
  assert.match(merchantTab, /:aria-selected="activeHomeFeedTab === 'merchants'"/)
  assert.match(merchantTab, /selectHomeFeedTab\('merchants'\)/)
  assert.match(resourceTab, /role="tab"/)
  assert.match(resourceTab, /:aria-selected="activeHomeFeedTab === 'resources'"/)
  assert.match(resourceTab, /selectHomeFeedTab\('resources'\)/)

  const tabsStyle = source.match(/\.home-feed-tabs\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const tabStyle = source.match(/\.home-feed-tab\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const activeTabStyle = source.match(/\.home-feed-tab\.active\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const indicatorStyle = source.match(/\.home-feed-tab::after\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const activeIndicatorStyle = source.match(/\.home-feed-tab\.active::after\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const pressedTabStyle = source.match(/\.home-feed-tab:active\s*\{([\s\S]*?)\n\}/)?.[1] || ''

  assert.match(tabsStyle, /display:\s*flex/)
  assert.match(tabsStyle, /gap:\s*36rpx/)
  assert.match(tabsStyle, /border-bottom:\s*1rpx solid/)
  assert.doesNotMatch(tabsStyle, /grid-template-columns|background:|border-radius:|box-shadow:/)
  assert.match(tabStyle, /position:\s*relative/)
  assert.match(tabStyle, /width:\s*auto/)
  assert.match(tabStyle, /min-height:\s*88rpx/)
  assert.match(tabStyle, /background:\s*transparent/)
  assert.match(activeTabStyle, /color:\s*\$wplink-primary/)
  assert.doesNotMatch(activeTabStyle, /background:\s*#ffffff|box-shadow:\s*0\s+6rpx/)
  assert.match(indicatorStyle, /height:\s*4rpx/)
  assert.match(indicatorStyle, /right:\s*2rpx/)
  assert.match(indicatorStyle, /left:\s*2rpx/)
  assert.match(indicatorStyle, /background:\s*transparent/)
  assert.match(activeIndicatorStyle, /background:\s*\$wplink-primary/)
  assert.match(pressedTabStyle, /opacity:\s*0\.72/)
  assert.doesNotMatch(`${tabStyle}\n${activeTabStyle}\n${pressedTabStyle}`, /transform:|transition:|animation:/)
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

test('home reuses map merchant rows and the shared compact resource feed card', () => {
  const merchantCardSource = fs.readFileSync(path.join(root, 'components/HomeRecentMerchantCard.vue'), 'utf8')

  assert.match(source, /import HomeRecentMerchantCard from '\.\.\/\.\.\/components\/HomeRecentMerchantCard\.vue'/)
  assert.match(source, /import ResourceFeedCard from '\.\.\/\.\.\/components\/ResourceFeedCard\.vue'/)
  assert.match(
    source,
    /<ResourceExposure[\s\S]*v-for="item in homeResources"[\s\S]*:resource-id="item\.id"[\s\S]*source="home"[\s\S]*<ResourceFeedCard[\s\S]*:resource="item"[\s\S]*@open="openResource"/,
  )
  assert.doesNotMatch(source, /import ResourceCard/)
  assert.doesNotMatch(source, /variant="home"/)
  assert.match(merchantCardSource, /import MerchantListItem from '\.\/MerchantListItem\.vue'/)
  assert.doesNotMatch(merchantCardSource, /导航|待认领|这是我的档口/)
})
