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

test('home names the merchant directory as taking-goods booths', () => {
  assert.match(source, /\{ title: '拿货档口'[\s\S]*icon: 'map'[\s\S]*action: 'sourcing-map'/)
  assert.doesNotMatch(source, /拿货地图/)
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

test('home presents the active feed as an editorial channel title', () => {
  assert.match(source, /import \{ getHomeFeedState \} from '\.\/homeFeedState'/)
  assert.match(source, /const activeHomeFeedTab = ref\(''\)/)
  assert.doesNotMatch(source, /织里商机 · 持续更新/)
  assert.match(source, /v-if="activeHomeFeedTab === 'merchants'" class="recent-merchant-list"/)
  assert.match(source, /v-else class="home-resource-panel"/)
  assert.doesNotMatch(source, /v-show="activeHomeFeedTab ===/)
  assert.match(source, /\.home-feed-more\s*\{[\s\S]*min-height:\s*88rpx/)

  const tabListTag = source.match(/<view\s+[^>]*class="home-feed-tabs"[^>]*>/)?.[0] || ''
  const activeTab = source.match(
    /<button\n\s+class="home-feed-tab home-feed-channel-title"[\s\S]*?<\/button>/,
  )?.[0] || ''
  const switchTab = source.match(
    /<button\n\s+class="home-feed-tab home-feed-channel-switch"[\s\S]*?<\/button>/,
  )?.[0] || ''
  const singleHead = source.match(
    /<view v-else class="home-feed-single-head">[\s\S]*?<\/view>\n\s*<\/view>/,
  )?.[0] || ''

  assert.match(tabListTag, /role="tablist"/)
  assert.match(tabListTag, /aria-label="首页内容"/)
  assert.match(activeTab, /role="tab"/)
  assert.match(activeTab, /aria-selected="true"/)
  assert.match(activeTab, /selectHomeFeedTab\(activeHomeFeedTab\)/)
  assert.match(activeTab, /activeHomeFeedTab === 'merchants' \? '新入驻商家' : '近期供需'/)
  assert.match(switchTab, /role="tab"/)
  assert.match(switchTab, /aria-selected="false"/)
  assert.match(
    switchTab,
    /selectHomeFeedTab\(activeHomeFeedTab === 'merchants' \? 'resources' : 'merchants'\)/,
  )
  assert.match(switchTab, /activeHomeFeedTab === 'merchants' \? '近期供需' : '新入驻商家'/)
  assert.match(switchTab, /class="home-feed-channel-arrow" aria-hidden="true">→<\/text>/)
  assert.match(singleHead, /class="home-feed-channel-title"/)
  assert.match(singleHead, /homeFeedState\.hasMerchants \? '新入驻商家' : '近期供需'/)
  assert.doesNotMatch(singleHead, /role="tab"|aria-selected|@click|home-feed-channel-arrow/)
  assert.doesNotMatch(source, /home-feed-woven-label/)

  const tabsStyle = source.match(/\.home-feed-tabs\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const tabStyle = source.match(/\.home-feed-tab\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const titleStyle = source.match(/\.home-feed-channel-title\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const markerStyle = source.match(/\.home-feed-channel-title::before\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const switchStyle = source.match(/\.home-feed-channel-switch\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const arrowStyle = source.match(/\.home-feed-channel-arrow\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const pressedSwitchStyle = source.match(/\.home-feed-channel-switch:active\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const singleHeadStyle = source.match(/\.home-feed-single-head\s*\{([\s\S]*?)\n\}/)?.[1] || ''

  assert.doesNotMatch(source, /home-feed-kicker/)
  assert.match(tabsStyle, /display:\s*flex/)
  assert.match(tabsStyle, /align-items:\s*stretch/)
  assert.match(tabsStyle, /justify-content:\s*space-between/)
  assert.match(tabsStyle, /gap:\s*18rpx/)
  assert.match(tabsStyle, /min-height:\s*88rpx/)
  assert.match(tabsStyle, /margin:\s*0 0 20rpx/)
  assert.match(tabsStyle, /border-bottom:\s*1rpx solid \$wplink-line/)
  assert.doesNotMatch(tabsStyle, /border-top:|background:|border-radius:|box-shadow:/)
  assert.match(tabStyle, /display:\s*inline-flex/)
  assert.match(tabStyle, /min-height:\s*88rpx/)
  assert.match(tabStyle, /white-space:\s*nowrap/)
  assert.match(titleStyle, /position:\s*relative/)
  assert.match(titleStyle, /padding:\s*0 0 0 14rpx/)
  assert.match(titleStyle, /font-size:\s*32rpx/)
  assert.match(titleStyle, /font-weight:\s*800/)
  assert.match(titleStyle, /color:\s*\$wplink-primary/)
  assert.match(markerStyle, /top:\s*28rpx/)
  assert.match(markerStyle, /left:\s*0/)
  assert.match(markerStyle, /width:\s*4rpx/)
  assert.match(markerStyle, /height:\s*32rpx/)
  assert.match(markerStyle, /background:\s*\$wplink-accent/)
  assert.match(switchStyle, /justify-content:\s*flex-end/)
  assert.match(switchStyle, /gap:\s*8rpx/)
  assert.match(switchStyle, /font-size:\s*25rpx/)
  assert.match(switchStyle, /font-weight:\s*700/)
  assert.match(switchStyle, /color:\s*\$wplink-muted/)
  assert.match(arrowStyle, /font-size:\s*24rpx/)
  assert.match(pressedSwitchStyle, /opacity:\s*0\.72/)
  assert.match(singleHeadStyle, /display:\s*flex/)
  assert.match(singleHeadStyle, /min-height:\s*88rpx/)
  assert.match(singleHeadStyle, /margin:\s*0 0 20rpx/)
  assert.match(singleHeadStyle, /border-bottom:\s*1rpx solid \$wplink-line/)
  assert.doesNotMatch(
    `${tabsStyle}\n${tabStyle}\n${titleStyle}\n${switchStyle}\n${pressedSwitchStyle}`,
    /transform:|transition:|animation:|box-shadow:|clip-path:/,
  )
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
