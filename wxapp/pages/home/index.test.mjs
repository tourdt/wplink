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

test('home presents recent merchants and resources as a woven business channel', () => {
  assert.match(source, /import \{ getHomeFeedState \} from '\.\/homeFeedState'/)
  assert.match(source, /const activeHomeFeedTab = ref\(''\)/)
  assert.match(source, />织里商机 · 持续更新</)
  assert.match(source, />新入驻商家</)
  assert.match(source, />近期供需</)
  assert.match(source, /v-if="activeHomeFeedTab === 'merchants'" class="recent-merchant-list"/)
  assert.match(source, /v-else class="home-resource-panel"/)
  assert.doesNotMatch(source, /v-show="activeHomeFeedTab ===/)
  assert.match(source, /\.home-feed-more\s*\{[\s\S]*min-height:\s*88rpx/)

  const tabListTag = source.match(/<view\s+[^>]*class="home-feed-tabs"[^>]*>/)?.[0] || ''
  const merchantTab = source.match(
    /<button\n\s+:class="\['home-feed-tab', 'home-feed-woven-label', \{ active: activeHomeFeedTab === 'merchants' \}\]"[\s\S]*?<\/button>/,
  )?.[0] || ''
  const resourceTab = source.match(
    /<button\n\s+:class="\['home-feed-tab', 'home-feed-woven-label', \{ active: activeHomeFeedTab === 'resources' \}\]"[\s\S]*?<\/button>/,
  )?.[0] || ''
  const singleHead = source.match(
    /<view v-else class="home-feed-single-head home-feed-woven-label active">[\s\S]*?<\/view>/,
  )?.[0] || ''

  assert.match(tabListTag, /role="tablist"/)
  assert.match(tabListTag, /aria-label="首页内容"/)
  assert.match(merchantTab, /role="tab"/)
  assert.match(merchantTab, /:aria-selected="activeHomeFeedTab === 'merchants'"/)
  assert.match(merchantTab, /selectHomeFeedTab\('merchants'\)/)
  assert.match(resourceTab, /role="tab"/)
  assert.match(resourceTab, /:aria-selected="activeHomeFeedTab === 'resources'"/)
  assert.match(resourceTab, /selectHomeFeedTab\('resources'\)/)
  assert.match(singleHead, /homeFeedState\.hasMerchants \? '新入驻商家' : '近期供需'/)
  assert.doesNotMatch(singleHead, /role="tab"|aria-selected|@click/)

  const kickerStyle = source.match(/\.home-feed-kicker\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const tabsStyle = source.match(/\.home-feed-tabs\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const labelStyle = source.match(/\.home-feed-woven-label\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const activeStyle = source.match(/\.home-feed-woven-label\.active\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const seamStyle = source.match(/\.home-feed-woven-label::before\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const activeSeamStyle = source.match(/\.home-feed-woven-label\.active::before\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const activeNotchStyle = source.match(/\.home-feed-woven-label\.active::after\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const pressedTabStyle = source.match(/\.home-feed-tab:active\s*\{([\s\S]*?)\n\}/)?.[1] || ''

  assert.match(kickerStyle, /margin-bottom:\s*12rpx/)
  assert.match(kickerStyle, /font-size:\s*22rpx/)
  assert.match(kickerStyle, /font-weight:\s*600/)
  assert.match(kickerStyle, /letter-spacing:\s*1rpx/)
  assert.match(kickerStyle, /color:\s*\$wplink-muted/)
  assert.match(tabsStyle, /display:\s*flex/)
  assert.match(tabsStyle, /gap:\s*18rpx/)
  assert.match(tabsStyle, /border-top:\s*1rpx solid \$wplink-line/)
  assert.match(tabsStyle, /border-bottom:\s*1rpx solid \$wplink-line/)
  assert.doesNotMatch(tabsStyle, /background:|border-radius:|box-shadow:/)
  assert.match(labelStyle, /display:\s*inline-flex/)
  assert.match(labelStyle, /min-height:\s*88rpx/)
  assert.match(labelStyle, /padding:\s*0 28rpx/)
  assert.match(labelStyle, /font-size:\s*27rpx/)
  assert.match(labelStyle, /font-weight:\s*700/)
  assert.match(labelStyle, /overflow:\s*visible/)
  assert.match(labelStyle, /white-space:\s*nowrap/)
  assert.match(activeStyle, /background:\s*\$wplink-primary/)
  assert.match(activeStyle, /color:\s*#ffffff/)
  assert.doesNotMatch(activeStyle, /padding:|font-size:|font-weight:/)
  assert.match(seamStyle, /top:\s*28rpx/)
  assert.match(seamStyle, /width:\s*4rpx/)
  assert.match(seamStyle, /height:\s*32rpx/)
  assert.match(activeSeamStyle, /background:\s*\$wplink-accent/)
  assert.match(activeNotchStyle, /right:\s*-14rpx/)
  assert.match(activeNotchStyle, /border-top:\s*44rpx solid transparent/)
  assert.match(activeNotchStyle, /border-bottom:\s*44rpx solid transparent/)
  assert.match(activeNotchStyle, /border-left:\s*14rpx solid \$wplink-primary/)
  assert.doesNotMatch(activeNotchStyle, /clip-path:/)
  assert.match(pressedTabStyle, /opacity:\s*0\.82/)
  assert.doesNotMatch(
    `${labelStyle}\n${activeStyle}\n${pressedTabStyle}`,
    /transform:|transition:|animation:|box-shadow:/,
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
